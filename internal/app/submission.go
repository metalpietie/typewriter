package app

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

// A submission: the manuscript of one book in the form that agents and editors ask for (standard manuscript format).
// A title page with your contact details and the word count, then 12 pt Times, double spaced, 1 inch margins,
// a 0.5 inch first-line indent and no space between paragraphs. Every page after the first has a header at the top
// right: Surname / TITLE / page. A chapter starts on a new page, a third of the way down. A scene break is a #,
// and THE END follows the last chapter. No cover, contents or front and back pages.

type submission struct {
	title, author, surname, short string
	contact                       []string // the lines at the top left of the title page
	words                         int
	blocks                        []block
	a4                            bool
}

func (e *Exporter) submission(bookID int64, opts ExportOptions) (submission, error) {
	if bookID == 0 {
		books := 0
		for _, b := range e.load("book") {
			for _, c := range e.load("chapter") {
				if num(c, "book") == num(b, "id") {
					books++
					break
				}
			}
		}
		if books > 1 {
			return submission{}, fmt.Errorf("a submission is one book: pick the book to send")
		}
	}
	// only the chapters: the pages around them belong to a printed book, not to a submission
	plain := ExportOptions{ChapterStyle: opts.ChapterStyle, SectionHeadings: opts.SectionHeadings, Takeaways: opts.Takeaways,
		CitationStyle: opts.CitationStyle, Bibliography: opts.Bibliography, Nonfiction: opts.Nonfiction}
	blocks, title := e.manuscript(bookID, plain)
	if len(blocks) == 0 {
		return submission{}, fmt.Errorf("nothing to export yet: add a book and chapters first")
	}
	var out []block
	words := 0
	for _, b := range withChapterNotes(blocks) {
		switch b.kind {
		case "title", "author", "subtitle":
			continue // the submission makes its own title page
		case "rule":
			b.text = "#"
		case "p", "", "quote", "li", "sub", "center", "caption", "boxhead", "note":
			words += len(strings.Fields(spansText(spansOf(b))))
		}
		out = append(out, b)
	}
	for len(out) > 0 && out[0].kind == "pagebreak" {
		out = out[1:]
	}
	author := strings.TrimSpace(opts.Author)
	var contact []string
	for _, l := range strings.Split(strings.ReplaceAll(opts.Contact, "\r\n", "\n"), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			contact = append(contact, l)
		}
	}
	if len(contact) == 0 && author != "" {
		contact = []string{author}
	}
	short := strings.TrimSpace(opts.ShortTitle)
	if short == "" {
		short = shortTitle(title)
	}
	return submission{title: title, author: author, surname: surnameOf(author), short: strings.ToUpper(short), contact: contact,
		words: words, blocks: out, a4: opts.PageSize == "a4"}, nil
}

// surnameOf is the last word of a name ("Ann Lee" is "Lee").
func surnameOf(name string) string {
	f := strings.Fields(name)
	if len(f) == 0 {
		return ""
	}
	return f[len(f)-1]
}

// shortTitle is the title for the page header: up to three words, and not more than about 25 letters.
func shortTitle(title string) string {
	var out []string
	n := 0
	for _, w := range strings.Fields(title) {
		if len(out) == 3 || (len(out) > 0 && n+len(w) > 25) {
			break
		}
		out = append(out, w)
		n += len(w) + 1
	}
	return strings.Join(out, " ")
}

// aboutWords is the word count as a submission gives it: rounded to the nearest 100, or the nearest 1,000 for a novel.
func aboutWords(n int) string {
	step := 100
	if n >= 20000 {
		step = 1000
	}
	r := (n + step/2) / step * step
	if r == 0 {
		r = n
	}
	return "about " + commas(r) + " words"
}

func commas(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// header is the text before the page number at the top right of each page.
func (s submission) header() string {
	parts := []string{}
	if s.surname != "" {
		parts = append(parts, s.surname)
	}
	parts = append(parts, s.short)
	return strings.Join(parts, " / ") + " / "
}

// ---------- PDF ----------

func (s submission) pdf() ([]byte, error) {
	w, h := 215.9, 279.4
	if s.a4 {
		w, h = 210, 297
	}
	const (
		margin = 25.4 // 1 inch
		indent = 12.7 // 0.5 inch
		line   = 8.47 // 12 pt type, double spaced (24 pt)
		single = 5.0
	)
	pdf := fpdf.NewCustom(&fpdf.InitType{OrientationStr: "P", UnitStr: "mm", Size: fpdf.SizeType{Wd: w, Ht: h}})
	pdf.SetMargins(margin, margin, margin)
	pdf.SetAutoPageBreak(true, margin)
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.SetHeaderFunc(func() {
		if pdf.PageNo() > 1 {
			pdf.SetFont("Times", "", 12)
			pdf.SetXY(margin, 12.7)
			pdf.CellFormat(w-2*margin, single, tr(s.header()+fmt.Sprint(pdf.PageNo())), "", 0, "R", false, 0, "")
		}
		pdf.SetXY(margin, margin)
	})

	// the title page
	pdf.AddPage()
	pdf.SetFont("Times", "", 12)
	pdf.CellFormat(w-2*margin, single, tr(aboutWords(s.words)), "", 0, "R", false, 0, "")
	pdf.SetXY(margin, margin)
	for _, l := range s.contact {
		pdf.CellFormat((w-2*margin)*0.6, single, tr(l), "", 2, "L", false, 0, "")
	}
	pdf.SetY(h/2 - line)
	pdf.MultiCell(0, line, tr(strings.ToUpper(s.title)), "", "C", false)
	if s.author != "" {
		pdf.MultiCell(0, line, tr("by "+s.author), "", "C", false)
	}

	newPage := true
	chapterTop := func() {
		pdf.AddPage()
		pdf.SetY(h / 3)
		newPage = false
	}
	for _, b := range s.blocks {
		switch b.kind {
		case "pagebreak":
			newPage = true
			continue
		case "h1":
			if newPage {
				chapterTop()
			}
			pdf.SetFont("Times", "", 12)
			pdf.MultiCell(0, line, tr(b.text), "", "C", false)
			pdf.Ln(line)
			continue
		}
		if newPage {
			chapterTop()
		}
		switch b.kind {
		case "rule", "center", "boxhead", "caption":
			pdf.SetFont("Times", "", 12)
			pdf.MultiCell(0, line, tr(spansText(spansOf(b))), "", "C", false)
		case "sub", "h2":
			pdf.SetFont("Times", "B", 12)
			pdf.MultiCell(0, line, tr(spansText(spansOf(b))), "", "L", false)
		case "note":
			pdf.SetFont("Times", "", 12)
			pdf.MultiCell(0, line, tr(fmt.Sprintf("%d. %s", b.level, b.text)), "", "L", false)
		case "quote", "li":
			pdf.SetLeftMargin(margin + indent)
			pdf.SetX(margin + indent)
			s.pdfSpans(pdf, tr, b, b.kind == "quote", 0, line)
			pdf.SetLeftMargin(margin)
			pdf.SetX(margin)
		default:
			first := indent
			if b.flush {
				first = 0
			}
			s.pdfSpans(pdf, tr, b, false, first, line)
		}
	}
	if newPage {
		chapterTop()
	}
	pdf.Ln(line)
	pdf.SetFont("Times", "", 12)
	pdf.MultiCell(0, line, "THE END", "", "C", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s submission) pdfSpans(pdf *fpdf.Fpdf, tr func(string) string, b block, italic bool, first, line float64) {
	pdf.SetX(pdf.GetX() + first)
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
		text := sp.text
		if sp.noteID > 0 {
			text = fmt.Sprintf("[%d]", sp.noteID)
		}
		pdf.Write(line, tr(text))
	}
	pdf.Ln(line)
}

// ---------- Word ----------

func (s submission) docx() ([]byte, error) {
	pageW, pageH := 12240, 15840 // Letter, in twips
	if s.a4 {
		pageW, pageH = 11906, 16838
	}
	const margin = 1440 // 1 inch
	textW := pageW - 2*margin
	var body strings.Builder
	p := func(props, runs string) { body.WriteString("<w:p><w:pPr>" + props + "</w:pPr>" + runs + "</w:p>") }

	// the title page: contact details at the top left, the word count at the top right, the title in the middle
	first := ""
	if len(s.contact) > 0 {
		first = s.contact[0]
	}
	p(fmt.Sprintf(`<w:pStyle w:val="Single"/><w:tabs><w:tab w:val="right" w:pos="%d"/></w:tabs>`, textW),
		run(first, false)+`<w:r><w:tab/></w:r>`+run(aboutWords(s.words), false))
	for _, l := range s.contact[min(1, len(s.contact)):] {
		p(`<w:pStyle w:val="Single"/>`, run(l, false))
	}
	p(`<w:pStyle w:val="Centered"/><w:spacing w:before="4320"/>`, run(strings.ToUpper(s.title), false))
	if s.author != "" {
		p(`<w:pStyle w:val="Centered"/>`, run("by "+s.author, false))
	}

	newPage := true
	chapterTop := func() {
		body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Centered"/></w:pPr><w:r><w:br w:type="page"/></w:r></w:p>`)
		newPage = false
	}
	top := false // the next heading is the first line of a new page: it goes a third of the way down
	for _, b := range s.blocks {
		if b.kind == "pagebreak" {
			newPage = true
			continue
		}
		if newPage {
			chapterTop()
			top = true
		}
		switch b.kind {
		case "h1":
			before := ""
			if top {
				before = `<w:spacing w:before="3600"/>`
			}
			p(`<w:pStyle w:val="Centered"/><w:keepNext/>`+before, run(b.text, false))
			p(`<w:pStyle w:val="Centered"/>`, "")
		case "rule", "center", "boxhead", "caption":
			p(`<w:pStyle w:val="Centered"/>`, run(spansText(spansOf(b)), false))
		case "sub", "h2":
			p(`<w:ind w:firstLine="0"/><w:keepNext/>`, run(spansText(spansOf(b)), true))
		case "note":
			p(`<w:ind w:firstLine="0"/>`, run(fmt.Sprintf("%d. %s", b.level, b.text), false))
		case "quote", "li":
			p(`<w:ind w:left="720" w:firstLine="0"/>`, s.docxRuns(b, b.kind == "quote"))
		default:
			props := ""
			if b.flush {
				props = `<w:ind w:firstLine="0"/>`
			}
			p(props, s.docxRuns(b, false))
		}
		top = false
	}
	if newPage {
		chapterTop()
	}
	p(`<w:pStyle w:val="Centered"/><w:spacing w:before="480"/>`, run("THE END", false))

	doc := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><w:body>` +
		body.String() +
		fmt.Sprintf(`<w:sectPr><w:headerReference w:type="default" r:id="rId2"/><w:pgSz w:w="%d" w:h="%d"/><w:pgMar w:top="%d" w:right="%d" w:bottom="%d" w:left="%d" w:header="720" w:footer="720" w:gutter="0"/><w:titlePg/></w:sectPr></w:body></w:document>`,
			pageW, pageH, margin, margin, margin, margin)

	header := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:pPr><w:pStyle w:val="Single"/><w:jc w:val="right"/></w:pPr>` +
		run(s.header(), false) +
		`<w:r><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:instrText xml:space="preserve"> PAGE </w:instrText></w:r><w:r><w:fldChar w:fldCharType="separate"/></w:r><w:r><w:t>2</w:t></w:r><w:r><w:fldChar w:fldCharType="end"/></w:r></w:p></w:hdr>`

	const styles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Times New Roman" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="24"/></w:rPr></w:rPrDefault>
<w:pPrDefault><w:pPr><w:spacing w:before="0" w:after="0" w:line="480" w:lineRule="auto"/><w:ind w:firstLine="720"/></w:pPr></w:pPrDefault></w:docDefaults>
<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:pPr><w:widowControl/></w:pPr></w:style>
<w:style w:type="paragraph" w:styleId="Single"><w:name w:val="Single"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:line="240" w:lineRule="auto"/><w:ind w:firstLine="0"/></w:pPr></w:style>
<w:style w:type="paragraph" w:styleId="Centered"><w:name w:val="Centered"/><w:basedOn w:val="Normal"/><w:pPr><w:ind w:firstLine="0"/><w:jc w:val="center"/></w:pPr></w:style>
</w:styles>`

	files := []struct{ name, data string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/><Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`},
		{"word/_rels/document.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/></Relationships>`},
		{"word/styles.xml", styles},
		{"word/header1.xml", header},
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

// docxRuns writes the styled text of a paragraph. A footnote becomes its number in brackets; the notes follow the chapter.
func (s submission) docxRuns(b block, italic bool) string {
	var out strings.Builder
	for _, sp := range spansOf(b) {
		if sp.noteID > 0 {
			out.WriteString(run(fmt.Sprintf("[%d]", sp.noteID), false))
			continue
		}
		sp.italic = sp.italic || italic
		out.WriteString(runOf(sp))
	}
	return out.String()
}
