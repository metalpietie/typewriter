package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestChapterHeadingStyles(t *testing.T) {
	cases := []struct {
		style, title, want string
	}{
		{"", "Start", "Chapter 3: Start"},
		{"full", "Start", "Chapter 3: Start"},
		{"full", "", "Chapter 3"},
		{"title", "Start", "Start"},
		{"title", "", "Chapter 3"},
		{"number", "Start", "Chapter 3"},
		{"plain", "Start", "3. Start"},
		{"plain", "", "3"},
		{"unknown", "Start", "Chapter 3: Start"},
	}
	for _, c := range cases {
		if got := chapterHeading(3, c.title, c.style); got != c.want {
			t.Errorf("style %q title %q: got %q, want %q", c.style, c.title, got, c.want)
		}
	}
}

func TestSceneBreakStyles(t *testing.T) {
	in := []block{{kind: "p", text: "a"}, {kind: "rule"}}
	if got := withBreakStyle(in, "stars"); got[1].text != "" {
		t.Error("stars keeps each format's own mark")
	}
	for style, want := range map[string]string{"hash": "#", "dashes": "—   —   —", "blank": "\u00a0"} {
		got := withBreakStyle(in, style)
		if got[1].text != want || got[0].text != "a" {
			t.Errorf("%s: got %q", style, got[1].text)
		}
	}
	if in[1].text != "" {
		t.Error("the input must not change")
	}
}

func matterBlocks(o ExportOptions) []block {
	out := []block{{kind: "title", text: "Book"}, {kind: "author", text: "A. Writer"}}
	out = append(out, frontPages(o)...)
	out = append(out, block{kind: "pagebreak"}, block{kind: "h1", text: "Chapter 1"}, block{kind: "p", text: "Text one."}, block{kind: "rule"}, block{kind: "p", text: "Text two."})
	out = append(out, backPages(o)...)
	return withBreakStyle(out, o.SceneBreak)
}

func TestFrontAndBackPages(t *testing.T) {
	none := matterBlocks(ExportOptions{})
	for _, b := range none {
		if b.kind == "center" || b.text == "Acknowledgments" || b.text == "About the author" {
			t.Fatalf("no pages were asked for, but got %+v", b)
		}
	}
	o := ExportOptions{Copyright: "© 2026 A. Writer\nAll rights reserved.", Dedication: "For my sister.", Thanks: "Thank you all.\n\nEspecially you.", About: "A. Writer lives by the sea.", Contents: true, SceneBreak: "hash"}
	blocks := matterBlocks(o)
	var kinds []string
	for _, b := range blocks {
		kinds = append(kinds, b.kind)
	}
	got := strings.Join(kinds, " ")
	for _, want := range []string{"title author pagebreak center center pagebreak center pagebreak h1 p rule p pagebreak h1 p p pagebreak h1 p"} {
		if got != want {
			t.Fatalf("order of blocks:\n got  %s\n want %s", got, want)
		}
	}

	// the contents come after the front pages, and list the back pages too
	withToc := withContents(blocks)
	var after []string
	for _, b := range withToc[:12] {
		after = append(after, b.kind)
	}
	toc := -1
	last := -1
	for i, b := range withToc {
		if b.kind == "tochead" {
			toc = i
		}
		if b.kind == "center" {
			last = i
		}
	}
	if toc < last {
		t.Fatalf("the contents must come after the dedication: contents at %d, last front page at %d (%v)", toc, last, after)
	}
	items := 0
	for _, b := range withToc {
		if b.kind == "tocitem" {
			items++
		}
	}
	if items != 3 { // Chapter 1, Acknowledgments, About the author
		t.Errorf("contents has %d items, want 3", items)
	}

	// every format writes it without an error, and the scene break shows
	if data, err := buildDocx(withToc, ExportOptions{}); err != nil || !strings.Contains(string(mustFile(t, data, "word/document.xml")), "For my sister.") {
		t.Errorf("docx: %v", err)
	}
	if data, err := buildEPUB(withToc, "Book", "A. Writer"); err != nil {
		t.Errorf("epub: %v", err)
	} else {
		_, files := unzipAll(t, data)
		found := false
		for name, body := range files {
			if strings.HasSuffix(name, ".xhtml") && strings.Contains(body, "For my sister.") && !strings.Contains(body, "<h1") {
				found = true // the dedication has a page file of its own
			}
		}
		if !found {
			t.Error("epub: the dedication must be on its own page file")
		}
	}
	for _, opts := range []ExportOptions{{}, {Print: true}, {Print: true, Contents: true}} {
		if _, err := buildPDF(withToc, opts); err != nil {
			t.Errorf("pdf: %v", err)
		}
	}
	md := string(buildMarkdown(blocks))
	if !strings.Contains(md, "\\#\n") || strings.Contains(md, "* * *") {
		t.Errorf("markdown: scene break should be an escaped #:\n%s", md)
	}
	if !strings.Contains(string(buildMarkdown(matterBlocks(ExportOptions{SceneBreak: "blank"}))), "&nbsp;") {
		t.Error("markdown: a blank scene break needs a non-breaking space")
	}
}

func mustFile(t *testing.T, data []byte, name string) []byte {
	t.Helper()
	_, files := unzipAll(t, data)
	s, ok := files[name]
	if !ok {
		t.Fatalf("no file %s", name)
	}
	return []byte(s)
}

// The manuscript of a real project: the pages sit in the right places and the chapter style is used.
func TestManuscriptUsesOptions(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	save := func(kind, data string) int64 {
		d, err := s.Save(Doc{Kind: kind, Data: data})
		if err != nil {
			t.Fatal(err)
		}
		return d.ID
	}
	b := save("book", `{"title":"One","order":1}`)
	c := save("chapter", fmt.Sprintf(`{"book":%d,"order":1,"title":"Start"}`, b))
	save("scene", fmt.Sprintf(`{"chapter":%d,"book":%d,"order":1,"body":"<p>First.</p>"}`, c, b))
	save("scene", fmt.Sprintf(`{"chapter":%d,"book":%d,"order":2,"body":"<p>Second.</p>"}`, c, b))

	e := &Exporter{store: s}
	blocks, name := e.manuscript(0, ExportOptions{Author: "A. Writer", Dedication: "For you.", About: "About me.", ChapterStyle: "plain"})
	if name != "One" {
		t.Errorf("name %q", name)
	}
	var texts []string
	for _, b := range blocks {
		switch b.kind {
		case "title", "author", "h1":
			texts = append(texts, b.text)
		case "center":
			texts = append(texts, spansText(spansOf(b)))
		}
	}
	if got, want := strings.Join(texts, "|"), "One|A. Writer|For you.|1. Start|About the author"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
