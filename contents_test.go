package main

import (
	"encoding/xml"
	"io"
	"regexp"
	"strings"
	"testing"
)

func kinds2(blocks []block) []string {
	var out []string
	for _, b := range blocks {
		out = append(out, b.kind)
	}
	return out
}

func TestContentsAfterTheCover(t *testing.T) {
	in := sampleBlocks()
	got := withContents(in)

	// the input is left as it was
	for _, b := range in {
		if b.anchor != "" {
			t.Fatal("withContents changed its input")
		}
	}
	// title, author, subtitle, then a new page, the heading, two entries, and the old text goes on
	want := []string{"title", "author", "subtitle", "pagebreak", "tochead", "tocitem", "tocitem", "pagebreak", "h1"}
	if g := kinds2(got); strings.Join(g[:len(want)], ",") != strings.Join(want, ",") {
		t.Fatalf("order is %v, want it to start with %v", g, want)
	}
	items := []block{got[5], got[6]}
	if items[0].text != "Chapter 1: Start" || items[1].text != "Chapter 2" || items[0].level != 0 {
		t.Errorf("entries are wrong: %+v", items)
	}
	for i, it := range items {
		found := false
		for _, b := range got {
			if b.kind == "h1" && b.anchor == it.anchor && b.text == it.text {
				found = true
			}
		}
		if !found || it.anchor == "" {
			t.Errorf("entry %d has no heading with anchor %q", i, it.anchor)
		}
	}
}

func TestContentsForASeries(t *testing.T) {
	in := []block{
		{kind: "title", text: "Book one"}, {kind: "pagebreak"}, {kind: "h1", text: "Chapter 1"},
		{kind: "pagebreak"}, {kind: "title", text: "Book two"}, {kind: "pagebreak"}, {kind: "h1", text: "Chapter 1"},
	}
	var levels []string
	for _, b := range withContents(in) {
		if b.kind == "tocitem" {
			levels = append(levels, string(rune('0'+b.level))+":"+b.text)
		}
	}
	if strings.Join(levels, ",") != "0:Book one,1:Chapter 1,0:Book two,1:Chapter 1" {
		t.Errorf("a series lists its books with the chapters under them, got %v", levels)
	}
}

func TestNoContentsWithoutHeadings(t *testing.T) {
	in := []block{{kind: "title", text: "Only a title"}, {kind: "p", text: "text"}}
	if got := withContents(in); len(got) != len(in) || hasContents(got) {
		t.Errorf("no headings, so no contents: %v", kinds2(got))
	}
}

func TestMarkdownContents(t *testing.T) {
	md := string(buildMarkdown(withContents(sampleBlocks())))
	for _, want := range []string{
		"## Contents\n\n- [Chapter 1: Start](#chapter-1-start)\n- [Chapter 2](#chapter-2)\n\n",
		"## Chapter 1: Start\n\n",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown lacks %q in:\n%s", want, md)
		}
	}

	// two headings with the same text: the second id gets -1, as on GitHub
	twice := []block{
		{kind: "title", text: "B"}, {kind: "pagebreak"}, {kind: "h1", text: "Chapter 1"}, {kind: "p", text: "a"},
		{kind: "pagebreak"}, {kind: "h1", text: "Chapter 1"}, {kind: "p", text: "b"},
	}
	md = string(buildMarkdown(withContents(twice)))
	if !strings.Contains(md, "- [Chapter 1](#chapter-1)\n- [Chapter 1](#chapter-1-1)\n") {
		t.Errorf("same-name headings need different ids:\n%s", md)
	}
}

func TestDocxContents(t *testing.T) {
	data, err := buildDocx(withContents(sampleBlocks()), ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_, files := unzipAll(t, data)
	doc := files["word/document.xml"]
	for _, want := range []string{
		`<w:pStyle w:val="ContentsHeading"/>`,
		`<w:hyperlink w:anchor="toc1" w:history="1">`,
		`<w:hyperlink w:anchor="toc2" w:history="1">`,
		`<w:bookmarkStart w:id="1" w:name="toc1"/>`,
		`<w:bookmarkStart w:id="2" w:name="toc2"/>`,
		`<w:bookmarkEnd w:id="2"/>`,
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("document.xml lacks %s", want)
		}
	}
	if !strings.Contains(files["word/styles.xml"], `w:styleId="ContentsItem"`) {
		t.Error("the contents styles are missing")
	}
	for _, name := range []string{"word/document.xml", "word/styles.xml"} {
		dec := xml.NewDecoder(strings.NewReader(files[name]))
		for {
			if _, err := dec.Token(); err != nil {
				if err != io.EOF {
					t.Errorf("%s is not well-formed: %v", name, err)
				}
				break
			}
		}
	}
}

func TestEPUBContents(t *testing.T) {
	data, err := buildEPUB(withContents(sampleBlocks()), "My Book & More", "A. Writer")
	if err != nil {
		t.Fatal(err)
	}
	_, files := unzipAll(t, data)

	var page string
	for name, body := range files {
		if strings.HasPrefix(name, "OEBPS/part") && strings.Contains(body, "<h1>Contents</h1>") {
			page = body
		}
	}
	if page == "" {
		t.Fatal("there is no Contents page")
	}
	links := regexp.MustCompile(`<a href="(part\d+\.xhtml)">([^<]*)</a>`).FindAllStringSubmatch(page, -1)
	if len(links) != 2 {
		t.Fatalf("the Contents page has %d links, want 2:\n%s", len(links), page)
	}
	for _, l := range links {
		target, ok := files["OEBPS/"+l[1]]
		if !ok || !strings.Contains(target, "<h1>"+l[2]+"</h1>") {
			t.Errorf("the link %q points to a page that does not start with that chapter", l[2])
		}
	}
	if !strings.Contains(files["OEBPS/nav.xhtml"], ">Contents</a>") {
		t.Error("the reader's own list should include the Contents page")
	}
}

func TestPDFContentsPageNumbers(t *testing.T) {
	blocks := withContents(sampleBlocks())

	// plain: title (1), contents (2), chapter 1 (3), chapter 2 (4)
	_, pages, err := renderPDF(blocks, ExportOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if pages["toc1"] != 3 || pages["toc2"] != 4 {
		t.Errorf("plain: chapters start on pages %v, want toc1=3 and toc2=4", pages)
	}
	// print: chapters and the contents start on right-hand pages, with an empty page before each
	_, pages, err = renderPDF(blocks, ExportOptions{Print: true, PageSize: "6x9"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if pages["toc1"] != 5 || pages["toc2"] != 7 {
		t.Errorf("print: chapters start on pages %v, want toc1=5 and toc2=7", pages)
	}

	// the second run has the same layout as the first
	data, err := buildPDF(blocks, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(pageObj.FindAll(data, -1)); n != 4 {
		t.Errorf("PDF with contents has %d pages, want 4", n)
	}
}

func TestPDFLongContents(t *testing.T) {
	// many chapters: the list runs over more than one page, and the numbers still match
	blocks := []block{{kind: "title", text: "Long"}}
	for i := 1; i <= 60; i++ {
		blocks = append(blocks, block{kind: "pagebreak"}, block{kind: "h1", text: "Chapter " + strings.Repeat("x", i%7+1)}, block{kind: "p", text: "text"})
	}
	_, first, err := renderPDF(withContents(blocks), ExportOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first["toc1"] < 4 {
		t.Errorf("60 entries should need more than one contents page, chapter 1 starts on page %d", first["toc1"])
	}
	// the page where the last chapter starts is the same on a second run
	data, err := buildPDF(withContents(blocks), ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(pageObj.FindAll(data, -1)); n != first["toc60"] {
		t.Errorf("PDF has %d pages, the last chapter starts on page %d: it must be the last page", n, first["toc60"])
	}
}
