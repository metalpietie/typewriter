package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"regexp"
	"strings"
	"testing"
)

func sampleBlocks() []block {
	return []block{
		{kind: "title", text: "My Book & More"},
		{kind: "author", text: "A. Writer"},
		{kind: "subtitle", text: "A tale"},
		{kind: "pagebreak"},
		{kind: "h1", text: "Chapter 1: Start"},
		{kind: "p", flush: true, spans: []span{{text: "He said "}, {text: "run", bold: true}, {text: " and "}, {text: " she ran *fast*_", italic: true}, {text: " <now>\nnext line"}}},
		{kind: "p", text: "# not a heading"},
		{kind: "rule"},
		{kind: "quote", text: "A quote"},
		{kind: "li", spans: []span{{text: "•  "}, {text: "apple"}}},
		{kind: "li", spans: []span{{text: "2.  "}, {text: "pear"}}},
		{kind: "p", text: "After the list."},
		{kind: "pagebreak"},
		{kind: "h1", text: "Chapter 2"},
		{kind: "p", text: "Second chapter text.\x01"},
	}
}

func unzipAll(t *testing.T, data []byte) (*zip.Reader, map[string]string) {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		files[f.Name] = string(b)
	}
	return zr, files
}

func TestEPUB(t *testing.T) {
	data, err := buildEPUB(sampleBlocks(), "My Book & More", "A. Writer")
	if err != nil {
		t.Fatal(err)
	}
	zr, files := unzipAll(t, data)
	first := zr.File[0]
	if first.Name != "mimetype" || first.Method != zip.Store || files["mimetype"] != "application/epub+zip" {
		t.Fatalf("mimetype must be the first file, stored: %q method %d", first.Name, first.Method)
	}
	for name, body := range files {
		if name == "mimetype" || strings.HasSuffix(name, ".css") {
			continue
		}
		dec := xml.NewDecoder(strings.NewReader(body))
		for {
			if _, err := dec.Token(); err != nil {
				if err != io.EOF {
					t.Errorf("%s is not well-formed XML: %v", name, err)
				}
				break
			}
		}
	}
	opf := files["OEBPS/content.opf"]
	for _, id := range regexp.MustCompile(`idref="(\w+)"`).FindAllStringSubmatch(opf, -1) {
		if !strings.Contains(opf, `id="`+id[1]+`"`) {
			t.Errorf("spine item %s is not in the manifest", id[1])
		}
	}
	for _, href := range regexp.MustCompile(`href="([^"]+)"`).FindAllStringSubmatch(opf, -1) {
		if _, ok := files["OEBPS/"+href[1]]; !ok {
			t.Errorf("manifest file %s is missing", href[1])
		}
	}
	// title page, chapter 1, chapter 2
	if !strings.Contains(opf, `idref="p3"`) || strings.Contains(opf, `idref="p4"`) {
		t.Errorf("expected 3 parts in the spine:\n%s", opf)
	}
	for _, want := range []string{"<dc:title>My Book &amp; More</dc:title>", "<dc:creator>A. Writer</dc:creator>", "<dc:language>en</dc:language>", "dcterms:modified"} {
		if !strings.Contains(opf, want) {
			t.Errorf("content.opf lacks %s", want)
		}
	}
	nav := files["OEBPS/nav.xhtml"]
	if !strings.Contains(nav, `epub:type="toc"`) || !strings.Contains(nav, "Chapter 1: Start") || !strings.Contains(nav, "My Book &amp; More") {
		t.Errorf("nav is wrong:\n%s", nav)
	}
	ch1 := files["OEBPS/part002.xhtml"]
	for _, want := range []string{`<p class="first">He said <strong>run</strong> and <em> she ran *fast*_</em> &lt;now&gt;<br/>next line</p>`, `<p class="rule">`, "<blockquote>"} {
		if !strings.Contains(ch1, want) {
			t.Errorf("chapter 1 lacks %s:\n%s", want, ch1)
		}
	}
	if strings.Contains(files["OEBPS/part003.xhtml"], "\x01") {
		t.Error("a control character reached the XHTML")
	}
}

func TestEPUBEmpty(t *testing.T) {
	if _, err := buildEPUB(nil, "x", ""); err == nil {
		t.Error("an empty book should be an error")
	}
}

func TestMarkdown(t *testing.T) {
	md := string(buildMarkdown(sampleBlocks()))
	for _, want := range []string{
		"# My Book & More\n\n*by A. Writer*\n\n*A tale*\n\n",
		"## Chapter 1: Start\n\n",
		"He said **run** and  *she ran \\*fast\\*\\_* \\<now>  \nnext line\n\n",
		"\\# not a heading\n\n",
		"* * *\n\n> A quote\n\n",
		"- apple\n2. pear\n\nAfter the list.\n\n",
		"## Chapter 2\n\n",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown lacks %q in:\n%s", want, md)
		}
	}
	if strings.Contains(md, "**** ") || strings.Contains(md, "** ") && strings.Contains(md, " **") && strings.Count(md, "**")%2 != 0 {
		t.Errorf("bad emphasis markers:\n%s", md)
	}
}

var pageObj = regexp.MustCompile(`/Type /Page\n`)

func TestPDFPrintLayout(t *testing.T) {
	plain, err := buildPDF(sampleBlocks(), ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	print, err := buildPDF(sampleBlocks(), ExportOptions{Print: true, PageSize: "6x9"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(plain, []byte("%PDF")) || !bytes.HasPrefix(print, []byte("%PDF")) {
		t.Fatal("not a PDF")
	}
	// plain: title, chapter 1, chapter 2 = 3 pages.
	// print: title (1), blank (2), chapter 1 (3), blank (4), chapter 2 (5): chapters start on right-hand pages.
	if n := len(pageObj.FindAll(plain, -1)); n != 3 {
		t.Errorf("plain PDF has %d pages, want 3", n)
	}
	if n := len(pageObj.FindAll(print, -1)); n != 5 {
		t.Errorf("print PDF has %d pages, want 5", n)
	}
	if !bytes.Contains(print, []byte("/MediaBox [0 0 432.00 648.00]")) {
		t.Errorf("6 x 9 inch page size is missing")
	}
}

func TestDocxPageSize(t *testing.T) {
	data, err := buildDocx(sampleBlocks(), ExportOptions{PageSize: "a5"})
	if err != nil {
		t.Fatal(err)
	}
	_, files := unzipAll(t, data)
	doc := files["word/document.xml"]
	if !strings.Contains(doc, `<w:pgSz w:w="8391" w:h="11906"/>`) {
		t.Errorf("A5 size not set:\n%s", doc[len(doc)-300:])
	}
	if !strings.Contains(doc, "by A. Writer") {
		t.Error("author line is missing")
	}
	dec := xml.NewDecoder(strings.NewReader(doc))
	for {
		if _, err := dec.Token(); err != nil {
			if err != io.EOF {
				t.Errorf("document.xml is not well-formed: %v", err)
			}
			break
		}
	}
}
