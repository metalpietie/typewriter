package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

// The look matches the Word export at 12 pt: 0.25 inch first-line indent, 1.5 line spacing, 6 pt after a paragraph.
// Smaller type scales these numbers.
const (
	firstLineIndent = 6.35  // mm, at 12 pt
	lineRatio       = 0.608 // line height in mm for each point of type
	gapRatio        = 0.175 // space after a paragraph, in mm for each point of type
	gutter          = 6.0   // extra inside margin in print layout, mm
)

// pdfLook holds the sizes that depend on the page size.
type pdfLook struct {
	font, line, gap, indent float64
}

func lookFor(sz pageSize) pdfLook {
	f := sz.fontPt()
	return pdfLook{font: f, line: f * lineRatio, gap: f * gapRatio, indent: firstLineIndent * f / 12}
}

func buildPDF(blocks []block, opts ExportOptions) ([]byte, error) {
	data, pages, err := renderPDF(blocks, opts, nil)
	if err != nil || !hasContents(blocks) {
		return data, err
	}
	// The contents page has its page numbers only after one run. The second run has the same layout,
	// so every heading is on the page that the first run found.
	data, _, err = renderPDF(blocks, opts, pages)
	return data, err
}

// renderPDF writes the PDF once. pages says where each heading of the table of contents starts (nil on the first run).
// It also returns the page each heading started on.
func renderPDF(blocks []block, opts ExportOptions, pages map[string]int) ([]byte, map[string]int, error) {
	found := map[string]int{}
	sz := sizeOf(opts.PageSize)
	look := lookFor(sz)
	m := sz.margin()
	pdf := fpdf.NewCustom(&fpdf.InitType{OrientationStr: "P", UnitStr: "mm", Size: fpdf.SizeType{Wd: sz.w, Ht: sz.h}})
	pdf.SetMargins(m, m, m)
	pdf.SetAutoPageBreak(true, m)
	tr := pdf.UnicodeTranslatorFromDescriptor("") // core fonts only know Western characters

	// In print layout the inside margin is wider, so the text is not lost in the binding.
	// Odd pages are on the right of a spread, so their inside is the left side.
	left := m
	blank := map[int]bool{} // pages left empty on purpose
	pdf.SetHeaderFunc(func() {
		l, r := m, m
		if opts.Print {
			if pdf.PageNo()%2 == 1 {
				l += gutter
			} else {
				r += gutter
			}
		}
		left = l // the paragraph code reads the margin of the current page
		pdf.SetMargins(l, m, r)
		pdf.SetXY(l, m)
	})
	pdf.SetFooterFunc(func() {
		n := pdf.PageNo()
		if blank[n] || (opts.Print && n == 1) {
			return // no number on the title page or on an empty page
		}
		align := "C"
		if opts.Print {
			align = "R" // outside corner of a right-hand page
			if n%2 == 0 {
				align = "L"
			}
		}
		pdf.SetY(-15)
		pdf.SetFont("Times", "", 10)
		pdf.CellFormat(0, 10, fmt.Sprint(n), "", 0, align, false, 0, "")
	})
	pdf.AddPage()
	fresh := true // true while the current page is still empty
	links := map[string]int{}
	linkOf := func(anchor string) int {
		if id, ok := links[anchor]; ok {
			return id
		}
		links[anchor] = pdf.AddLink()
		return links[anchor]
	}
	// a heading that the table of contents points to: remember its page and make it a link target
	mark := func(b block) {
		if b.anchor != "" {
			found[b.anchor] = pdf.PageNo()
			pdf.SetLink(linkOf(b.anchor), pdf.GetY(), -1)
		}
	}

	for _, b := range blocks {
		switch b.kind {
		case "pagebreak":
			if !fresh {
				pdf.AddPage()
				fresh = true
			}
			if opts.Print && pdf.PageNo()%2 == 0 { // a new chapter starts on a right-hand page
				blank[pdf.PageNo()] = true
				pdf.AddPage()
			}
			continue
		case "title":
			mark(b)
			pdf.Ln(sz.h * 0.25)
			pdf.SetFont("Times", "B", 32)
			pdf.MultiCell(0, 14, tr(b.text), "", "C", false)
			pdf.Ln(8)
		case "author":
			pdf.SetFont("Times", "I", 14)
			pdf.MultiCell(0, 8, tr("by "+b.text), "", "C", false)
			pdf.Ln(4)
		case "subtitle":
			pdf.SetFont("Times", "I", 13)
			pdf.MultiCell(0, 7, tr(b.text), "", "C", false)
		case "tochead":
			pdf.Ln(6)
			pdf.SetFont("Times", "B", 20)
			pdf.MultiCell(0, 10, tr(b.text), "", "L", false)
			pdf.Ln(6)
		case "tocitem":
			// the title on the left, dots, and the page number on the right; the whole line is a link
			const numW = 12.0
			pageW, pageH := pdf.GetPageSize()
			_, _, right, _ := pdf.GetMargins()
			h := look.line
			x := left + float64(b.level)*6
			textW := pageW - right - numW - x
			pdf.SetFont("Times", "", look.font)
			title := tr(b.text)
			lines := pdf.SplitText(title, textW)
			if pdf.GetY()+float64(len(lines))*h > pageH-m {
				pdf.AddPage()
			}
			y0 := pdf.GetY()
			pdf.SetX(x)
			pdf.MultiCell(textW, h, title, "", "L", false)
			y1 := pdf.GetY()
			lastY := y1 - h
			if n := len(lines); n > 0 {
				from := x + pdf.GetStringWidth(lines[n-1]) + 1.5
				to := pageW - right - numW - 1
				if dot := pdf.GetStringWidth("."); to-from > 3*dot {
					pdf.SetXY(from, lastY)
					pdf.CellFormat(to-from, h, strings.Repeat(".", int((to-from)/dot)), "", 0, "L", false, 0, "")
				}
			}
			pdf.SetXY(pageW-right-numW, lastY)
			pdf.CellFormat(numW, h, fmt.Sprint(pages[b.anchor]), "", 0, "R", false, 0, "")
			pdf.Link(x, y0, pageW-right-x, y1-y0, linkOf(b.anchor))
			pdf.SetXY(left, y1+look.gap/2)
		case "h1":
			mark(b)
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
			mark := b.text
			if mark == "" {
				mark = "*   *   *"
			}
			pdf.Ln(3)
			pdf.SetFont("Times", "", look.font)
			pdf.CellFormat(0, 7, tr(mark), "", 1, "C", false, 0, "")
			pdf.Ln(3)
		case "center":
			// a centered paragraph of a front page, such as the dedication
			if b.level > 0 {
				pdf.Ln(float64(b.level) * 8)
			}
			style := ""
			if sp := spansOf(b); len(sp) > 0 && sp[0].italic {
				style = "I"
			}
			pdf.SetFont("Times", style, look.font)
			pdf.MultiCell(0, look.line, tr(spansText(spansOf(b))), "", "C", false)
			pdf.Ln(look.gap)
		case "quote", "li":
			indent := 10.0
			if b.kind == "li" {
				indent = 8
			}
			_, top, right, _ := pdf.GetMargins()
			pdf.SetMargins(left+indent, top, right)
			pdf.SetX(left + indent)
			writeSpans(pdf, tr, look, b, b.kind == "quote", 0)
			pdf.SetMargins(left, top, right)
			pdf.SetX(left)
		default:
			indent := look.indent // same indent as the Word export
			if b.flush {
				indent = 0
			}
			writeSpans(pdf, tr, look, b, false, indent)
		}
		fresh = false
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), found, nil
}

// writeSpans writes one paragraph of styled text and moves to the next line.
func writeSpans(pdf *fpdf.Fpdf, tr func(string) string, look pdfLook, b block, italic bool, firstLine float64) {
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
		pdf.SetFont("Times", style, look.font)
		pdf.Write(look.line, tr(sp.text))
	}
	pdf.Ln(look.line + look.gap)
}
