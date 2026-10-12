package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"fmt"
	"strings"
	"time"
)

// xhtmlEsc makes text safe for XML. It also drops characters XML does not allow.
func xhtmlEsc(s string) string {
	s = strings.Map(func(r rune) rune {
		// XML does not allow most control characters, U+FFFE, U+FFFF or lone surrogates
		if (r < 0x20 && r != '\t' && r != '\n' && r != '\r') || r == 0xFFFE || r == 0xFFFF || (r >= 0xD800 && r <= 0xDFFF) {
			return -1
		}
		return r
	}, s)
	return xmlEsc.Replace(s)
}

// xhtmlSpans writes styled text as XHTML. A new line in the text becomes a line break.
func xhtmlSpans(spans []span) string {
	var out strings.Builder
	for _, sp := range mergeSpans(spans) {
		t := strings.ReplaceAll(xhtmlEsc(sp.text), "\n", "<br/>")
		if sp.underline {
			t = `<span class="u">` + t + "</span>"
		}
		if sp.italic {
			t = "<em>" + t + "</em>"
		}
		if sp.bold {
			t = "<strong>" + t + "</strong>"
		}
		out.WriteString(t)
	}
	return out.String()
}

// epubPart is one page file of the e-book: a title page or a chapter.
type epubPart struct {
	file  string
	label string // the entry in the table of contents
	body  strings.Builder
}

const epubCSS = `body { font-family: serif; line-height: 1.5; margin: 0 5%; }
h1 { font-size: 1.6em; margin: 2em 0 1em; text-align: left; }
h1.title { font-size: 2.4em; text-align: center; margin-top: 30%; }
h2 { font-size: 1.2em; margin: 1.6em 0 .4em; }
p { margin: 0; text-indent: 1.2em; }
p.first, p.kv, p.li, p.subtitle, p.author { text-indent: 0; }
p.kv, p.li { margin: 0 0 .4em; }
p.li { margin-left: 1.5em; }
p.subtitle, p.author { text-align: center; font-style: italic; margin-top: .8em; }
p.toc { text-indent: 0; margin: .5em 0; }
p.toc.l1 { margin-left: 1.5em; }
p.toc a { text-decoration: none; color: inherit; }
div.cover { text-align: center; margin: 0; }
div.cover img { max-width: 100%; max-height: 100vh; }
p.rule { text-align: center; text-indent: 0; margin: 1.2em 0; }
p.center { text-align: center; text-indent: 0; margin: 0 0 .6em; }
blockquote { margin: .8em 1.5em; font-style: italic; }
blockquote p { text-indent: 0; }
.u { text-decoration: underline; }
`

// buildEPUB writes an EPUB 3 e-book. Each title and each chapter heading starts a new page file.
func buildEPUB(blocks []block, title, author string) ([]byte, error) {
	return buildEPUBWithCover(blocks, title, author, nil)
}

// buildEPUBWithCover is buildEPUB with a cover. The cover is the first page, and e-readers show it in the library.
func buildEPUBWithCover(blocks []block, title, author string, cover *coverData) ([]byte, error) {
	var parts []*epubPart
	var cur *epubPart
	start := func(label string) *epubPart {
		cur = &epubPart{file: fmt.Sprintf("part%03d.xhtml", len(parts)+1), label: label}
		parts = append(parts, cur)
		return cur
	}
	anchorFile := map[string]string{} // heading anchor -> the page file it starts
	var tocPart *epubPart
	var tocItems []block
	afterBreak := false // the last block was a page break
	for _, b := range blocks {
		newPage := afterBreak
		afterBreak = b.kind == "pagebreak"
		switch b.kind {
		case "pagebreak":
			continue
		case "center":
			// a front page, such as the dedication, is a page file of its own
			if newPage && b.front {
				start(b.label)
			}
		case "title":
			p := start(b.text)
			p.body.WriteString(`<h1 class="title">` + xhtmlEsc(b.text) + "</h1>\n")
			if b.anchor != "" {
				anchorFile[b.anchor] = p.file
			}
			continue
		case "h1":
			p := start(b.text)
			p.body.WriteString("<h1>" + xhtmlEsc(b.text) + "</h1>\n")
			if b.anchor != "" {
				anchorFile[b.anchor] = p.file
			}
			continue
		case "tochead":
			tocPart = start(b.text)
			tocPart.body.WriteString("<h1>" + xhtmlEsc(b.text) + "</h1>\n")
			continue
		case "tocitem":
			tocItems = append(tocItems, b) // written after the loop, when every page file is known
			continue
		}
		if cur == nil {
			start(title)
		}
		w := &cur.body
		switch b.kind {
		case "author":
			w.WriteString(`<p class="author">by ` + xhtmlEsc(b.text) + "</p>\n")
		case "subtitle":
			w.WriteString(`<p class="subtitle">` + xhtmlEsc(b.text) + "</p>\n")
		case "h2":
			w.WriteString("<h2>" + xhtmlEsc(b.text) + "</h2>\n")
		case "sub":
			w.WriteString("<h2>" + xhtmlSpans(spansOf(b)) + "</h2>\n")
		case "kv":
			w.WriteString(`<p class="kv"><strong>` + xhtmlEsc(b.label) + ":</strong> " + strings.ReplaceAll(xhtmlEsc(b.text), "\n", "<br/>") + "</p>\n")
		case "rule":
			mark := `*&#160;&#160;&#160;*&#160;&#160;&#160;*`
			if b.text != "" {
				mark = strings.ReplaceAll(xhtmlEsc(b.text), "   ", "&#160;&#160;&#160;")
			}
			w.WriteString(`<p class="rule">` + mark + "</p>\n")
		case "center":
			style := ""
			if b.level > 0 {
				style = fmt.Sprintf(` style="margin-top:%dem"`, b.level)
			}
			w.WriteString(`<p class="center"` + style + ">" + xhtmlSpans(spansOf(b)) + "</p>\n")
		case "quote":
			w.WriteString("<blockquote><p>" + xhtmlSpans(spansOf(b)) + "</p></blockquote>\n")
		case "li":
			w.WriteString(`<p class="li">` + xhtmlSpans(spansOf(b)) + "</p>\n")
		default:
			class := ""
			if b.flush {
				class = ` class="first"`
			}
			w.WriteString("<p" + class + ">" + xhtmlSpans(spansOf(b)) + "</p>\n")
		}
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("nothing to put in the e-book")
	}
	if tocPart != nil {
		for _, it := range tocItems {
			file, ok := anchorFile[it.anchor]
			if !ok {
				continue
			}
			tocPart.body.WriteString(fmt.Sprintf(`<p class="toc l%d"><a href="%s">%s</a></p>`+"\n", it.level, file, xhtmlEsc(it.text)))
		}
	}
	if cover != nil {
		cp := &epubPart{file: "cover.xhtml", label: "Cover"}
		cp.body.WriteString(`<div class="cover"><img src="images/cover.jpg" alt="Cover"/></div>` + "\n")
		parts = append([]*epubPart{cp}, parts...)
	}
	for _, p := range parts {
		if strings.TrimSpace(p.label) == "" {
			p.label = "Untitled"
		}
	}

	id := newUUID()
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	const head = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"

	var manifest, spine, navItems, ncxPoints strings.Builder
	for i, p := range parts {
		manifest.WriteString(fmt.Sprintf(`<item id="p%d" href="%s" media-type="application/xhtml+xml"/>`+"\n", i+1, p.file))
		spine.WriteString(fmt.Sprintf(`<itemref idref="p%d"/>`+"\n", i+1))
		navItems.WriteString(fmt.Sprintf(`<li><a href="%s">%s</a></li>`+"\n", p.file, xhtmlEsc(p.label)))
		ncxPoints.WriteString(fmt.Sprintf(`<navPoint id="n%d" playOrder="%d"><navLabel><text>%s</text></navLabel><content src="%s"/></navPoint>`+"\n", i+1, i+1, xhtmlEsc(p.label), p.file))
	}
	creator := ""
	if strings.TrimSpace(author) != "" {
		creator = "<dc:creator>" + xhtmlEsc(strings.TrimSpace(author)) + "</dc:creator>\n"
	}

	coverItem, coverMeta := "", ""
	if cover != nil {
		coverItem = `<item id="cover-image" href="images/cover.jpg" media-type="image/jpeg" properties="cover-image"/>` + "\n"
		coverMeta = `<meta name="cover" content="cover-image"/>` + "\n"
	}
	files := []struct {
		name, data string
		store      bool
	}{
		{"mimetype", "application/epub+zip", true},
		{"META-INF/container.xml", head + `<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`, false},
		{"OEBPS/content.opf", head + `<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="bookid">
<metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
<dc:identifier id="bookid">urn:uuid:` + id + `</dc:identifier>
<dc:title>` + xhtmlEsc(title) + `</dc:title>
` + creator + coverMeta + `<dc:language>en</dc:language>
<meta property="dcterms:modified">` + now + `</meta>
</metadata>
<manifest>
<item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
<item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
<item id="css" href="style.css" media-type="text/css"/>
` + coverItem + manifest.String() + `</manifest>
<spine toc="ncx">
` + spine.String() + `</spine>
</package>`, false},
		{"OEBPS/nav.xhtml", head + `<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" xml:lang="en"><head><meta charset="utf-8"/><title>Contents</title><link rel="stylesheet" type="text/css" href="style.css"/></head>
<body><nav epub:type="toc" id="toc"><h1>Contents</h1><ol>
` + navItems.String() + `</ol></nav></body></html>`, false},
		{"OEBPS/toc.ncx", head + `<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1"><head><meta name="dtb:uid" content="urn:uuid:` + id + `"/><meta name="dtb:depth" content="1"/><meta name="dtb:totalPageCount" content="0"/><meta name="dtb:maxPageNumber" content="0"/></head><docTitle><text>` + xhtmlEsc(title) + `</text></docTitle><navMap>
` + ncxPoints.String() + `</navMap></ncx>`, false},
		{"OEBPS/style.css", epubCSS, false},
	}
	for _, p := range parts {
		files = append(files, struct {
			name, data string
			store      bool
		}{"OEBPS/" + p.file, head + `<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en"><head><meta charset="utf-8"/><title>` + xhtmlEsc(p.label) + `</title><link rel="stylesheet" type="text/css" href="style.css"/></head>
<body>
` + p.body.String() + `</body></html>`, false})
	}

	if cover != nil {
		files = append(files, struct {
			name, data string
			store      bool
		}{"OEBPS/images/cover.jpg", string(cover.jpeg), true})
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		// the "mimetype" file must come first and must not be compressed
		method := zip.Deflate
		if f.store {
			method = zip.Store
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: f.name, Method: method})
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

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
