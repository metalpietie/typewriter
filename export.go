package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Exporter writes your book or story bible to a Word or PDF file.
type Exporter struct {
	store *Store
}

// block is one piece of an export, kept neutral so Word and PDF share it.
type block struct {
	kind   string // title, subtitle, h1, h2, sub, p, quote, li, rule, kv, pagebreak
	label  string // for kv
	text   string
	spans  []span // styled text; when empty, text is used
	flush  bool   // no first-line indent (first paragraph of a chapter)
	anchor string // set on a heading that the table of contents points to
	level  int    // tocitem: 0 for a chapter (or a book title), 1 for a chapter in a series. center: how far down the page it starts
	box    string // the kind of callout box that this block is inside (idea, story, example, exercise, warning, quote), or empty
	front  bool   // part of the cover or the front pages, so the table of contents comes after it
}

type rec = map[string]any

func (e *Exporter) load(kind string) []rec {
	docs, err := e.store.List(kind)
	if err != nil {
		return nil
	}
	out := make([]rec, 0, len(docs))
	for _, d := range docs {
		m := rec{}
		_ = json.Unmarshal([]byte(d.Data), &m)
		m["id"] = float64(d.ID)
		out = append(out, m)
	}
	return out
}

func str(m rec, k string) string {
	if v, ok := m[k].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func num(m rec, k string) float64 {
	v, _ := m[k].(float64)
	return v
}

func has(m rec, k string) bool { _, ok := m[k].(float64); return ok }

func byOrder(list []rec, key string) {
	sort.SliceStable(list, func(i, j int) bool { return num(list[i], key) < num(list[j], key) })
}

func nameOf(list []rec, id float64, field string) string {
	for _, m := range list {
		if num(m, "id") == id {
			return str(m, field)
		}
	}
	return ""
}

// Export asks where to save, then writes the file.
// what is "manuscript", "outline", "synopsis" or "bible". format is "docx", "pdf", "epub" or "md".
// bookID 0 means every book. It returns the saved path, or "" if you cancel.
func (e *Exporter) Export(what string, bookID int64, format string, opts ExportOptions) (string, error) {
	filters := map[string]string{
		"docx": "Word document (*.docx)", "pdf": "PDF file (*.pdf)",
		"epub": "E-book (*.epub)", "md": "Markdown file (*.md)",
	}
	filter, ok := filters[format]
	if !ok {
		return "", fmt.Errorf("unknown format %q", format)
	}
	if what != "manuscript" {
		opts.Cover = "" // only a manuscript has a cover
	}
	if opts.SceneBreak == "heading" {
		opts.SectionHeadings = true // the same choice, as it comes from the Export screen
	}
	var blocks []block
	name := "Story bible"
	switch what {
	case "manuscript":
		blocks, name = e.manuscript(bookID, opts)
		if len(blocks) == 0 {
			return "", fmt.Errorf("nothing to export yet: add a book and chapters first")
		}
	case "outline", "synopsis":
		blocks, name = e.outline(what, bookID, opts)
		if len(blocks) == 0 {
			return "", fmt.Errorf("nothing to export yet: add a book and chapters first")
		}
	case "sources":
		blocks = e.sourcesList(opts)
		name = "Sources"
		if len(blocks) == 0 {
			return "", fmt.Errorf("nothing to export yet: add a source first")
		}
	case "bible":
		blocks = e.bible()
	default:
		return "", fmt.Errorf("unknown export %q", what)
	}

	path, err := application.Get().Dialog.SaveFile().
		SetFilename(safeName(name)+"."+format).
		AddFilter(filter, "*."+format).
		PromptForSingleSelection()
	if err != nil || path == "" {
		return "", nil
	}
	if !strings.HasSuffix(strings.ToLower(path), "."+format) {
		path += "." + format
	}

	if what == "manuscript" {
		blocks = withBreakStyle(blocks, opts.SceneBreak)
		if format != "docx" { // Word has real footnotes. The other formats print the notes at the end of each chapter.
			blocks = withChapterNotes(blocks)
		}
	}
	if opts.Contents {
		blocks = withContents(blocks)
	}

	var data []byte
	switch format {
	case "docx":
		data, err = buildDocx(blocks, opts)
	case "pdf":
		data, err = buildPDF(blocks, opts)
	case "epub":
		data, err = buildEPUBWithCover(blocks, name, opts.Author, decodeCover(opts.Cover))
	case "md":
		data = buildMarkdown(blocks)
	}
	if err != nil {
		return "", err
	}
	return path, writeFileAtomic(path, data)
}

var badName = regexp.MustCompile(`[^\p{L}\p{N}_\- ]+`)

func safeName(s string) string {
	s = strings.TrimSpace(badName.ReplaceAllString(s, ""))
	if s == "" {
		return "export"
	}
	return s
}

// paragraphs splits text on line breaks and drops empty lines.
func paragraphs(s string) []string {
	var out []string
	for _, l := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func (e *Exporter) manuscript(bookID int64, opts ExportOptions) ([]block, string) {
	author := opts.Author
	books, chapters, scenes := e.load("book"), e.load("chapter"), e.load("scene")
	byOrder(books, "order")
	var out []block
	title := "Manuscript"
	count := 0
	index := map[string]*indexEntry{}
	var chapterList []chapterBlocksOf
	for _, b := range books {
		if bookID != 0 && int64(num(b, "id")) != bookID {
			continue
		}
		var chs []rec
		for _, c := range chapters {
			if num(c, "book") == num(b, "id") {
				chs = append(chs, c)
			}
		}
		if len(chs) == 0 {
			continue
		}
		byOrder(chs, "order")
		count++
		if len(out) > 0 {
			out = append(out, block{kind: "pagebreak"})
		}
		out = append(out, block{kind: "title", text: str(b, "title")})
		if a := strings.TrimSpace(author); a != "" {
			out = append(out, block{kind: "author", text: a})
		}
		if s := str(b, "synopsis"); s != "" {
			out = append(out, block{kind: "subtitle", text: s})
		}
		for _, c := range chs {
			out = append(out, block{kind: "pagebreak"})
			heading := chapterHeading(int(num(c, "order")), str(c, "title"), opts.ChapterStyle)
			out = append(out, block{kind: "h1", text: heading})
			text := numberCaptions(chapterText(c, scenes, opts.SectionHeadings), int(num(c, "order")))
			collectIndex(index, text, int(num(c, "order")))
			chapterList = append(chapterList, chapterBlocksOf{heading: heading, blocks: text})
			out = append(out, text...)
			out = append(out, takeawayBlocks(c, opts)...)
		}
		title = str(b, "title")
	}
	if bookID == 0 && count > 1 {
		title = "Complete series"
	}
	if count > 0 {
		// the front pages go after the cover of the first book, the back pages after the last chapter
		at := 0
		for at < len(out) && isCover(out, at) {
			at++
		}
		if opts.ExerciseList {
			out = append(out, exerciseList(chapterList)...)
		}
		out = e.finishCitations(out, opts)
		out = append(out[:at:at], append(frontPages(opts), out[at:]...)...)
		out = append(out, backPages(opts)...)
		if opts.IndexPage {
			out = append(out, indexBlocks(index)...)
		}
	}
	return out, title
}

// finishCitations turns the citations in the text into words, numbers the footnotes, and adds the bibliography when it was asked for.
func (e *Exporter) finishCitations(out []block, opts ExportOptions) []block {
	sources := map[int64]source{}
	for _, r := range e.load("source") {
		s := sourceOf(r)
		sources[s.id] = s
	}
	out, cited := resolveCitations(out, sources, opts.CitationStyle)
	out = numberNotes(out)
	if opts.Bibliography {
		out = append(out, bibliographyBlocks(cited, opts.CitationStyle)...)
	}
	return out
}

// sourcesList is the whole library of sources as a list.
func (e *Exporter) sourcesList(opts ExportOptions) []block {
	var all []source
	for _, r := range e.load("source") {
		all = append(all, sourceOf(r))
	}
	return sourcesBlocks(all, opts.CitationStyle)
}

func (e *Exporter) bible() []block {
	chars, rels := e.load("character"), e.load("relationship")
	places, notes, events := e.load("location"), e.load("note"), e.load("event")
	factions, members, frels := e.load("faction"), e.load("membership"), e.load("factionrel")
	items, logs := e.load("item"), e.load("itemlog")
	byOrder(logs, "order")
	out := []block{{kind: "title", text: "Story bible"}}

	// the last log entry of an item says who has it and where it is now
	lastLog := func(item float64) rec {
		var last rec
		for _, l := range logs {
			if num(l, "item") == item {
				last = l
			}
		}
		return last
	}

	kv := func(label, text string) {
		if text != "" {
			out = append(out, block{kind: "kv", label: label, text: text})
		}
	}

	out = append(out, block{kind: "pagebreak"}, block{kind: "h1", text: "Characters"})
	for _, c := range chars {
		out = append(out, block{kind: "h2", text: str(c, "name")})
		kv("Role", str(c, "role"))
		kv("Also called", str(c, "aliases"))
		kv("Age", str(c, "age"))
		if fs, ok := c["fields"].([]any); ok {
			for _, f := range fs {
				if m, ok := f.(map[string]any); ok {
					kv(fmt.Sprint(m["k"]), strings.TrimSpace(fmt.Sprint(m["v"])))
				}
			}
		}
		kv("Description", str(c, "description"))
		kv("Traits", str(c, "traits"))
		kv("Backstory", str(c, "backstory"))
		kv("Goals and fears", str(c, "goals"))
		kv("Notes", str(c, "notes"))
		var links []string
		for _, r := range rels {
			var other float64
			switch num(c, "id") {
			case num(r, "a"):
				other = num(r, "b")
			case num(r, "b"):
				other = num(r, "a")
			default:
				continue
			}
			l := str(r, "type") + ": " + nameOf(chars, other, "name")
			switch str(r, "fam") {
			case "parent":
				if num(c, "id") == num(r, "a") {
					l = "parent of " + nameOf(chars, other, "name")
				} else {
					l = "child of " + nameOf(chars, other, "name")
				}
			case "partner":
				l = "partner: " + nameOf(chars, other, "name")
			case "sibling":
				l = "sibling: " + nameOf(chars, other, "name")
			}
			if d := str(r, "description"); d != "" {
				l += " (" + d + ")"
			}
			links = append(links, l)
		}
		kv("Relationships", strings.Join(links, "; "))
		var in []string
		for _, m := range members {
			if num(m, "character") == num(c, "id") {
				t := nameOf(factions, num(m, "faction"), "name")
				if r := str(m, "rank"); r != "" {
					t += " (" + r + ")"
				}
				in = append(in, t)
			}
		}
		kv("Factions", strings.Join(in, "; "))
		var held []string
		for _, it := range items {
			if l := lastLog(num(it, "id")); l != nil && num(l, "character") == num(c, "id") && has(l, "character") {
				held = append(held, str(it, "name"))
			}
		}
		kv("Items held", strings.Join(held, "; "))
	}

	out = append(out, block{kind: "pagebreak"}, block{kind: "h1", text: "Factions"})
	for _, f := range factions {
		out = append(out, block{kind: "h2", text: str(f, "name")})
		kv("Type", str(f, "type"))
		kv("Headquarters", nameOf(places, num(f, "hq"), "name"))
		kv("Ranks", strings.Join(paragraphs(str(f, "ranks")), ", "))
		kv("Description", str(f, "description"))
		var who []string
		for _, m := range members {
			if num(m, "faction") == num(f, "id") {
				t := nameOf(chars, num(m, "character"), "name")
				if r := str(m, "rank"); r != "" {
					t += " (" + r + ")"
				}
				who = append(who, t)
			}
		}
		kv("Members", strings.Join(who, "; "))
		var ties []string
		for _, r := range frels {
			var other float64
			switch num(f, "id") {
			case num(r, "a"):
				other = num(r, "b")
			case num(r, "b"):
				other = num(r, "a")
			default:
				continue
			}
			ties = append(ties, str(r, "type")+": "+nameOf(factions, other, "name"))
		}
		kv("Relations", strings.Join(ties, "; "))
	}

	out = append(out, block{kind: "pagebreak"}, block{kind: "h1", text: "Items and artifacts"})
	for _, it := range items {
		out = append(out, block{kind: "h2", text: str(it, "name")})
		kv("Type", str(it, "type"))
		kv("Description", str(it, "description"))
		kv("History", str(it, "history"))
		var trail []string
		for _, l := range logs {
			if num(l, "item") != num(it, "id") {
				continue
			}
			t := "unknown holder"
			if has(l, "character") {
				t = nameOf(chars, num(l, "character"), "name")
			}
			if has(l, "location") {
				t += " at " + nameOf(places, num(l, "location"), "name")
			}
			if w := str(l, "when"); w != "" {
				t += " (" + w + ")"
			}
			if n := str(l, "note"); n != "" {
				t += ": " + n
			}
			trail = append(trail, t)
		}
		kv("Journey", strings.Join(trail, "; "))
	}

	out = append(out, block{kind: "pagebreak"}, block{kind: "h1", text: "Places"})
	for _, p := range places {
		out = append(out, block{kind: "h2", text: str(p, "name")})
		kv("Type", str(p, "type"))
		kv("Inside", nameOf(places, num(p, "parent"), "name"))
		kv("Description", str(p, "description"))
	}

	out = append(out, block{kind: "pagebreak"}, block{kind: "h1", text: "World and lore"})
	for _, n := range notes {
		out = append(out, block{kind: "h2", text: str(n, "title")})
		kv("Category", str(n, "category"))
		for _, p := range paragraphs(str(n, "body")) {
			out = append(out, block{kind: "p", text: p})
		}
	}

	// the glossary, in alphabetical order
	terms := e.load("term")
	sort.SliceStable(terms, func(i, j int) bool {
		return strings.ToLower(str(terms[i], "word")) < strings.ToLower(str(terms[j], "word"))
	})
	if len(terms) > 0 {
		out = append(out, block{kind: "pagebreak"}, block{kind: "h1", text: "Glossary"})
		for _, t := range terms {
			out = append(out, block{kind: "h2", text: str(t, "word")})
			kv("Type", str(t, "type"))
			kv("Said", str(t, "pronunciation"))
			kv("Meaning", str(t, "meaning"))
			kv("Also spelled", str(t, "accepted"))
			kv("Notes", str(t, "notes"))
		}
	}

	byOrder(events, "time")
	out = append(out, block{kind: "pagebreak"}, block{kind: "h1", text: "Timeline"})
	for _, ev := range events {
		when := str(ev, "when")
		if when == "" && has(ev, "time") {
			when = fmt.Sprint(num(ev, "time"))
		}
		out = append(out, block{kind: "h2", text: strings.TrimSpace(when + " " + str(ev, "title"))})
		kv("What happened", str(ev, "description"))
	}
	return out
}

// ---------- Word ----------

var xmlEsc = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

func run(text string, bold bool) string { return runOf(span{text: text, bold: bold}) }

// runOf writes one styled piece of text. A new line in the text becomes a line break.
func runOf(sp span) string {
	if sp.noteID > 0 { // a real Word footnote: the text of it is in footnotes.xml
		return fmt.Sprintf(`<w:r><w:rPr><w:rStyle w:val="FootnoteReference"/></w:rPr><w:footnoteReference w:id="%d"/></w:r>`, sp.noteID)
	}
	props := ""
	if sp.bold {
		props += "<w:b/>"
	}
	if sp.italic {
		props += "<w:i/>"
	}
	if sp.underline {
		props += `<w:u w:val="single"/>`
	}
	if props != "" {
		props = "<w:rPr>" + props + "</w:rPr>"
	}
	var out strings.Builder
	for i, part := range strings.Split(sp.text, "\n") {
		if i > 0 {
			out.WriteString("<w:r><w:br/></w:r>")
		}
		if part != "" {
			out.WriteString("<w:r>" + props + `<w:t xml:space="preserve">` + xhtmlEsc(part) + "</w:t></w:r>")
		}
	}
	return out.String()
}

func runsOf(b block) string {
	var out strings.Builder
	for _, sp := range spansOf(b) {
		out.WriteString(runOf(sp))
	}
	return out.String()
}

func buildDocx(blocks []block, opts ExportOptions) ([]byte, error) {
	sz := sizeOf(opts.PageSize)
	m := twips(sz.margin())
	var body strings.Builder
	marks := 0
	// a heading that the table of contents points to gets a bookmark
	bookmark := func(b block, inner string) string {
		if b.anchor == "" {
			return inner
		}
		marks++
		return fmt.Sprintf(`<w:bookmarkStart w:id="%d" w:name="%s"/>`, marks, b.anchor) + inner + fmt.Sprintf(`<w:bookmarkEnd w:id="%d"/>`, marks)
	}
	for _, b := range blocks {
		switch b.kind {
		case "pagebreak":
			body.WriteString(`<w:p><w:r><w:br w:type="page"/></w:r></w:p>`)
		case "tochead":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="ContentsHeading"/></w:pPr>` + run(b.text, false) + `</w:p>`)
		case "tocitem":
			body.WriteString(fmt.Sprintf(`<w:p><w:pPr><w:pStyle w:val="ContentsItem"/><w:ind w:left="%d" w:firstLine="0"/></w:pPr><w:hyperlink w:anchor="%s" w:history="1">`, b.level*480, b.anchor) + run(b.text, false) + `</w:hyperlink></w:p>`)
		case "author":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Subtitle"/></w:pPr>` + run("by "+b.text, false) + `</w:p>`)
		case "title":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Title"/></w:pPr>` + bookmark(b, run(b.text, false)) + `</w:p>`)
		case "subtitle":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Subtitle"/></w:pPr>` + run(b.text, false) + `</w:p>`)
		case "h1":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr>` + bookmark(b, run(b.text, false)) + `</w:p>`)
		case "h2":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading2"/></w:pPr>` + run(b.text, false) + `</w:p>`)
		case "kv":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Plain"/></w:pPr>` + run(b.label+": ", true) + run(b.text, false) + `</w:p>`)
		case "boxhead":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Plain"/><w:keepNext/>` + boxPr(b) + `<w:spacing w:before="200" w:after="40"/></w:pPr>` + run(b.text, true) + `</w:p>`)
		case "caption":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Caption"/></w:pPr>` + runsOf(b) + `</w:p>`)
		case "sub":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading2"/>` + boxPr(b) + `</w:pPr>` + runsOf(b) + `</w:p>`)
		case "quote":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Quote"/>` + boxPr(b) + `</w:pPr>` + runsOf(b) + `</w:p>`)
		case "li":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="ListItem"/>` + boxPr(b) + `</w:pPr>` + runsOf(b) + `</w:p>`)
		case "rule":
			mark := b.text
			if mark == "" {
				mark = "*   *   *"
			}
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Plain"/><w:spacing w:before="240" w:after="240"/><w:jc w:val="center"/></w:pPr>` + run(mark, false) + `</w:p>`)
		case "center":
			body.WriteString(fmt.Sprintf(`<w:p><w:pPr><w:pStyle w:val="Plain"/><w:spacing w:before="%d"/><w:ind w:firstLine="0"/><w:jc w:val="center"/></w:pPr>`, b.level*1000) + runsOf(b) + `</w:p>`)
		default:
			props := ""
			if b.box != "" {
				props = `<w:pPr>` + boxPr(b) + `<w:ind w:left="160" w:right="120" w:firstLine="0"/></w:pPr>`
			} else if b.flush {
				props = `<w:pPr><w:ind w:firstLine="0"/></w:pPr>`
			}
			body.WriteString(`<w:p>` + props + runsOf(b) + `</w:p>`)
		}
	}
	cover := decodeCover(opts.Cover)
	coverXML := ""
	if cover != nil {
		coverXML = docxCover(cover, sz.w-2*sz.margin(), sz.h-2*sz.margin()-10)
	}
	doc := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"><w:body>` +
		coverXML + body.String() +
		fmt.Sprintf(`<w:sectPr><w:pgSz w:w="%d" w:h="%d"/><w:pgMar w:top="%d" w:right="%d" w:bottom="%d" w:left="%d" w:header="720" w:footer="720" w:gutter="0"/></w:sectPr></w:body></w:document>`,
			twips(sz.w), twips(sz.h), m, m, m, m)

	const styles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Times New Roman" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="24"/></w:rPr></w:rPrDefault>
<w:pPrDefault><w:pPr><w:spacing w:after="120" w:line="360" w:lineRule="auto"/><w:ind w:firstLine="360"/></w:pPr></w:pPrDefault></w:docDefaults>
<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>
<w:style w:type="paragraph" w:styleId="Plain"><w:name w:val="Plain"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:after="80" w:line="276" w:lineRule="auto"/><w:ind w:firstLine="0"/></w:pPr></w:style>
<w:style w:type="paragraph" w:styleId="Quote"><w:name w:val="Quote"/><w:basedOn w:val="Normal"/><w:pPr><w:ind w:left="720" w:right="720" w:firstLine="0"/></w:pPr><w:rPr><w:i/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="ListItem"><w:name w:val="List Item"/><w:basedOn w:val="Plain"/><w:pPr><w:ind w:left="720" w:hanging="360" w:firstLine="0"/></w:pPr></w:style>
<w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="3000" w:after="360"/><w:ind w:firstLine="0"/><w:jc w:val="center"/></w:pPr><w:rPr><w:b/><w:sz w:val="64"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Subtitle"><w:name w:val="Subtitle"/><w:basedOn w:val="Normal"/><w:pPr><w:ind w:firstLine="0"/><w:jc w:val="center"/></w:pPr><w:rPr><w:i/><w:sz w:val="26"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="480" w:after="360"/><w:ind w:firstLine="0"/><w:outlineLvl w:val="0"/></w:pPr><w:rPr><w:b/><w:sz w:val="40"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="ContentsHeading"><w:name w:val="Contents Heading"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="480" w:after="360"/><w:ind w:firstLine="0"/></w:pPr><w:rPr><w:b/><w:sz w:val="40"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="ContentsItem"><w:name w:val="Contents Item"/><w:basedOn w:val="Plain"/><w:pPr><w:spacing w:after="120"/></w:pPr></w:style>
<w:style w:type="paragraph" w:styleId="Caption"><w:name w:val="caption"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="60" w:after="200" w:line="240" w:lineRule="auto"/><w:ind w:firstLine="0"/><w:jc w:val="center"/></w:pPr><w:rPr><w:i/><w:sz w:val="20"/></w:rPr></w:style>
<w:style w:type="character" w:styleId="FootnoteReference"><w:name w:val="footnote reference"/><w:rPr><w:vertAlign w:val="superscript"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="FootnoteText"><w:name w:val="footnote text"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:after="40" w:line="240" w:lineRule="auto"/><w:ind w:firstLine="0"/></w:pPr><w:rPr><w:sz w:val="20"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="320" w:after="120"/><w:ind w:firstLine="0"/><w:outlineLvl w:val="1"/></w:pPr><w:rPr><w:b/><w:sz w:val="30"/></w:rPr></w:style>
</w:styles>`

	files := []struct{ name, data string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`},
		{"word/_rels/document.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`},
		{"word/styles.xml", styles},
		{"word/document.xml", doc},
	}
	if cover != nil {
		for i := range files {
			switch files[i].name {
			case "[Content_Types].xml":
				files[i].data = strings.Replace(files[i].data, `<Default Extension="xml"`, `<Default Extension="jpg" ContentType="image/jpeg"/><Default Extension="xml"`, 1)
			case "word/_rels/document.xml.rels":
				files[i].data = strings.Replace(files[i].data, `</Relationships>`, `<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/cover.jpg"/></Relationships>`, 1)
			}
		}
		files = append(files, struct{ name, data string }{"word/media/cover.jpg", string(cover.jpeg)})
	}
	if notes := docxNotes(blocks); notes != "" {
		files = append(files, struct{ name, data string }{"word/footnotes.xml", notes}, struct{ name, data string }{"word/settings.xml", docxSettings})
		for i := range files {
			switch files[i].name {
			case "[Content_Types].xml":
				files[i].data = strings.Replace(files[i].data, `</Types>`, `<Override PartName="/word/footnotes.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footnotes+xml"/><Override PartName="/word/settings.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.settings+xml"/></Types>`, 1)
			case "word/_rels/document.xml.rels":
				files[i].data = strings.Replace(files[i].data, `</Relationships>`, `<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footnotes" Target="footnotes.xml"/><Relationship Id="rId4" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/settings" Target="settings.xml"/></Relationships>`, 1)
			}
		}
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		w, err := zw.Create(f.name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(f.data)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// SaveImage asks where to save, then writes a PNG you drew. dataURL is a "data:image/png;base64,..." string.
// It returns the saved path, or "" if you cancel.
func (e *Exporter) SaveImage(name, dataURL string) (string, error) {
	_, b64, ok := strings.Cut(dataURL, ",")
	if !ok {
		return "", fmt.Errorf("bad image data")
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	path, err := application.Get().Dialog.SaveFile().
		SetFilename(safeName(name)+".png").
		AddFilter("PNG image (*.png)", "*.png").
		PromptForSingleSelection()
	if err != nil || path == "" {
		return "", nil
	}
	if !strings.HasSuffix(strings.ToLower(path), ".png") {
		path += ".png"
	}
	return path, writeFileAtomic(path, raw)
}

// chapterText is the text of one chapter. A chapter with scenes is its scenes in order, with a
// scene break between them. An older chapter without scenes uses its own text.
func chapterText(c rec, scenes []rec, headings bool) []block {
	var mine []rec
	for _, sc := range scenes {
		if num(sc, "chapter") == num(c, "id") {
			mine = append(mine, sc)
		}
	}
	if len(mine) == 0 {
		return flushFirst(chapterBlocks(str(c, "body")))
	}
	byOrder(mine, "order")
	var out []block
	for _, sc := range mine {
		body := flushFirst(chapterBlocks(str(sc, "body")))
		if len(body) == 0 {
			continue // a planned scene with no text yet
		}
		if headings {
			// a non-fiction book: the title of each section is a sub-heading, and no mark goes between sections
			if t := strings.TrimSpace(str(sc, "title")); t != "" {
				out = append(out, block{kind: "h2", text: t})
			}
		} else if len(out) > 0 {
			out = append(out, block{kind: "rule"})
		}
		out = append(out, body...)
	}
	return out
}

// takeawayBlocks is the list "Key takeaways" at the end of a chapter, when the writer asked for it and filled it in.
func takeawayBlocks(c rec, opts ExportOptions) []block {
	if !opts.Takeaways {
		return nil
	}
	lines := paragraphs(str(c, "takeaways"))
	if len(lines) == 0 {
		return nil
	}
	out := []block{{kind: "h2", text: "Key takeaways"}}
	for _, l := range lines {
		out = append(out, block{kind: "li", text: strings.TrimSpace(strings.TrimLeft(l, "-*• "))})
	}
	return out
}

// flushFirst takes the indent off the first paragraph of a chapter or scene.
func flushFirst(body []block) []block {
	for i := range body {
		if body[i].kind == "p" {
			body[i].flush = true
			break
		}
	}
	return body
}

// writeFileAtomic writes to a temporary file first, then renames it. A disk error then never leaves a half-written file, and an old file stays whole.
func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// docxSettings tells Word which footnotes are the separators. A document with footnotes needs it.
const docxSettings = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:settings xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:footnotePr><w:footnote w:id="-1"/><w:footnote w:id="0"/></w:footnotePr></w:settings>`

// docxNotes is the part word/footnotes.xml for the footnotes in the blocks, or "" when there are none.
func docxNotes(blocks []block) string {
	var body strings.Builder
	for _, b := range blocks {
		for _, sp := range b.spans {
			if sp.noteID > 0 {
				body.WriteString(fmt.Sprintf(`<w:footnote w:id="%d"><w:p><w:pPr><w:pStyle w:val="FootnoteText"/></w:pPr><w:r><w:rPr><w:rStyle w:val="FootnoteReference"/></w:rPr><w:footnoteRef/></w:r><w:r><w:t xml:space="preserve"> %s</w:t></w:r></w:p></w:footnote>`, sp.noteID, xhtmlEsc(sp.note)))
			}
		}
	}
	if body.Len() == 0 {
		return ""
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:footnotes xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:footnote w:type="separator" w:id="-1"><w:p><w:pPr><w:spacing w:after="0" w:line="240" w:lineRule="auto"/></w:pPr><w:r><w:separator/></w:r></w:p></w:footnote><w:footnote w:type="continuationSeparator" w:id="0"><w:p><w:pPr><w:spacing w:after="0" w:line="240" w:lineRule="auto"/></w:pPr><w:r><w:continuationSeparator/></w:r></w:p></w:footnote>` + body.String() + `</w:footnotes>`
}

// boxPr is the bar at the left and the colour behind a paragraph that is inside a callout box.
func boxPr(b block) string {
	if b.box == "" {
		return ""
	}
	s := boxOf(b.box)
	return fmt.Sprintf(`<w:pBdr><w:left w:val="single" w:sz="24" w:space="8" w:color="%s"/></w:pBdr><w:shd w:val="clear" w:color="auto" w:fill="%s"/>`, s.bar, s.fill)
}
