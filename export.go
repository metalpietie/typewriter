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

	"github.com/go-pdf/fpdf"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Exporter writes your book or story bible to a Word or PDF file.
type Exporter struct {
	store *Store
}

// block is one piece of an export, kept neutral so Word and PDF share it.
type block struct {
	kind  string // title, subtitle, h1, h2, sub, p, quote, li, rule, kv, pagebreak
	label string // for kv
	text  string
	spans []span // styled text; when empty, text is used
	flush bool   // no first-line indent (first paragraph of a chapter)
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
// what is "manuscript" or "bible". format is "docx" or "pdf".
// bookID 0 means every book. It returns the saved path, or "" if you cancel.
func (e *Exporter) Export(what string, bookID int64, format string) (string, error) {
	if format != "docx" && format != "pdf" {
		return "", fmt.Errorf("unknown format %q", format)
	}
	var blocks []block
	name := "Story bible"
	switch what {
	case "manuscript":
		blocks, name = e.manuscript(bookID)
		if len(blocks) == 0 {
			return "", fmt.Errorf("nothing to export yet: add a book and chapters first")
		}
	case "bible":
		blocks = e.bible()
	default:
		return "", fmt.Errorf("unknown export %q", what)
	}

	filter := "Word document (*.docx)"
	if format == "pdf" {
		filter = "PDF file (*.pdf)"
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

	var data []byte
	if format == "docx" {
		data, err = buildDocx(blocks)
	} else {
		data, err = buildPDF(blocks)
	}
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, data, 0o644)
}

var badName = regexp.MustCompile(`[^\w\- ]+`)

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

func (e *Exporter) manuscript(bookID int64) ([]block, string) {
	books, chapters, scenes := e.load("book"), e.load("chapter"), e.load("scene")
	byOrder(books, "order")
	var out []block
	title := "Manuscript"
	count := 0
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
		if s := str(b, "synopsis"); s != "" {
			out = append(out, block{kind: "subtitle", text: s})
		}
		for _, c := range chs {
			out = append(out, block{kind: "pagebreak"})
			h := fmt.Sprintf("Chapter %d", int(num(c, "order")))
			if t := str(c, "title"); t != "" {
				h += ": " + t
			}
			out = append(out, block{kind: "h1", text: h})
			out = append(out, chapterText(c, scenes)...)
		}
		title = str(b, "title")
	}
	if bookID == 0 && count > 1 {
		title = "Complete series"
	}
	return out, title
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
			out.WriteString("<w:r>" + props + `<w:t xml:space="preserve">` + xmlEsc.Replace(part) + "</w:t></w:r>")
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

func buildDocx(blocks []block) ([]byte, error) {
	var body strings.Builder
	for _, b := range blocks {
		switch b.kind {
		case "pagebreak":
			body.WriteString(`<w:p><w:r><w:br w:type="page"/></w:r></w:p>`)
		case "title":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Title"/></w:pPr>` + run(b.text, false) + `</w:p>`)
		case "subtitle":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Subtitle"/></w:pPr>` + run(b.text, false) + `</w:p>`)
		case "h1":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr>` + run(b.text, false) + `</w:p>`)
		case "h2":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading2"/></w:pPr>` + run(b.text, false) + `</w:p>`)
		case "kv":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Plain"/></w:pPr>` + run(b.label+": ", true) + run(b.text, false) + `</w:p>`)
		case "sub":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading2"/></w:pPr>` + runsOf(b) + `</w:p>`)
		case "quote":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Quote"/></w:pPr>` + runsOf(b) + `</w:p>`)
		case "li":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="ListItem"/></w:pPr>` + runsOf(b) + `</w:p>`)
		case "rule":
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Plain"/><w:spacing w:before="240" w:after="240"/><w:jc w:val="center"/></w:pPr>` + run("*   *   *", false) + `</w:p>`)
		default:
			props := ""
			if b.flush {
				props = `<w:pPr><w:ind w:firstLine="0"/></w:pPr>`
			}
			body.WriteString(`<w:p>` + props + runsOf(b) + `</w:p>`)
		}
	}
	doc := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		body.String() +
		`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="720" w:footer="720" w:gutter="0"/></w:sectPr></w:body></w:document>`

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

// ---------- PDF ----------

func buildPDF(blocks []block) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(25.4, 25.4, 25.4)
	pdf.SetAutoPageBreak(true, 25.4)
	tr := pdf.UnicodeTranslatorFromDescriptor("") // core fonts only know Western characters
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Times", "", 10)
		pdf.CellFormat(0, 10, fmt.Sprint(pdf.PageNo()), "", 0, "C", false, 0, "")
	})
	pdf.AddPage()
	fresh := true // true while the current page is still empty

	for _, b := range blocks {
		switch b.kind {
		case "pagebreak":
			if !fresh {
				pdf.AddPage()
				fresh = true
			}
			continue
		case "title":
			pdf.Ln(70)
			pdf.SetFont("Times", "B", 32)
			pdf.MultiCell(0, 14, tr(b.text), "", "C", false)
			pdf.Ln(8)
		case "subtitle":
			pdf.SetFont("Times", "I", 13)
			pdf.MultiCell(0, 7, tr(b.text), "", "C", false)
		case "h1":
			pdf.Ln(6)
			pdf.SetFont("Times", "B", 20)
			pdf.MultiCell(0, 10, tr(b.text), "", "L", false)
			pdf.Ln(6)
		case "h2":
			pdf.Ln(4)
			pdf.SetFont("Times", "B", 14)
			pdf.MultiCell(0, 8, tr(b.text), "", "L", false)
			pdf.Ln(1)
		case "kv":
			pdf.SetFont("Times", "B", 11)
			label := tr(b.label + ": ")
			pdf.CellFormat(pdf.GetStringWidth(label)+0.5, 6, label, "", 0, "L", false, 0, "")
			pdf.SetFont("Times", "", 11)
			pdf.MultiCell(0, 6, tr(b.text), "", "L", false)
			pdf.Ln(1)
		case "sub":
			pdf.Ln(3)
			pdf.SetFont("Times", "B", 14)
			pdf.MultiCell(0, 8, tr(spansText(spansOf(b))), "", "L", false)
			pdf.Ln(1)
		case "rule":
			pdf.Ln(3)
			pdf.SetFont("Times", "", 12)
			pdf.CellFormat(0, 7, "*   *   *", "", 1, "C", false, 0, "")
			pdf.Ln(3)
		case "quote", "li":
			indent := 10.0
			if b.kind == "li" {
				indent = 8
			}
			pdf.SetLeftMargin(25.4 + indent)
			pdf.SetX(25.4 + indent)
			writeSpans(pdf, tr, b, b.kind == "quote", 0)
			pdf.SetLeftMargin(25.4)
			pdf.SetX(25.4)
		default:
			indent := firstLineIndent // same indent as the Word export
			if b.flush {
				indent = 0
			}
			writeSpans(pdf, tr, b, false, indent)
		}
		fresh = false
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeSpans writes one paragraph of styled text and moves to the next line.
func writeSpans(pdf *fpdf.Fpdf, tr func(string) string, b block, italic bool, firstLine float64) {
	pdf.SetX(pdf.GetX() + firstLine)
	for _, sp := range spansOf(b) {
		style := ""
		if sp.bold {
			style += "B"
		}
		if sp.italic || italic {
			style += "I"
		}
		if sp.underline {
			style += "U"
		}
		pdf.SetFont("Times", style, 12)
		pdf.Write(lineHeight, tr(sp.text))
	}
	pdf.Ln(lineHeight + paraGap)
}

// These match the Word export: 0.25 inch first-line indent, 1.5 line spacing, 6 pt after a paragraph.
const (
	firstLineIndent = 6.35 // mm
	lineHeight      = 7.3  // mm
	paraGap         = 2.1  // mm
)

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
	return path, os.WriteFile(path, raw, 0o644)
}

// chapterText is the text of one chapter. A chapter with scenes is its scenes in order, with a
// scene break between them. An older chapter without scenes uses its own text.
func chapterText(c rec, scenes []rec) []block {
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
		if len(out) > 0 {
			out = append(out, block{kind: "rule"})
		}
		out = append(out, body...)
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
