package main

import (
	"fmt"
	"strings"
	"testing"
)

func outlineStore(t *testing.T) (*Exporter, int64, int64) {
	t.Helper()
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
	mara := save("character", `{"name":"Mara"}`)
	shop := save("location", `{"name":"The clock shop"}`)
	b1 := save("book", `{"title":"One","order":1}`)
	b2 := save("book", `{"title":"Two","order":2}`)
	c1 := save("chapter", fmt.Sprintf(`{"book":%d,"order":1,"title":"The letter","summary":"Mara finds a letter."}`, b1))
	c2 := save("chapter", fmt.Sprintf(`{"book":%d,"order":2,"title":"The road","summary":""}`, b1))
	save("chapter", fmt.Sprintf(`{"book":%d,"order":1,"title":"Elsewhere"}`, b2))
	save("scene", fmt.Sprintf(`{"chapter":%d,"order":1,"title":"Under the clock","summary":"She opens it.","pov":%d,"location":%d}`, c1, mara, shop))
	save("scene", fmt.Sprintf(`{"chapter":%d,"order":2,"title":"","summary":""}`, c1))
	save("scene", fmt.Sprintf(`{"chapter":%d,"order":1,"title":"Leaving","summary":"She leaves town."}`, c2))
	save("scene", fmt.Sprintf(`{"chapter":%d,"order":2,"title":"Arriving","summary":"She arrives at the coast."}`, c2))
	return &Exporter{store: s}, b1, b2
}

func texts(blocks []block, kind string) []string {
	var out []string
	for _, b := range blocks {
		if b.kind == kind {
			out = append(out, b.text+spansText(b.spans))
		}
	}
	return out
}

func TestOutline(t *testing.T) {
	e, b1, _ := outlineStore(t)
	blocks, name := e.outline("outline", b1, ExportOptions{Author: "A. Writer", ChapterStyle: "plain"})
	if name != "One outline" {
		t.Errorf("name %q", name)
	}
	if got := strings.Join(texts(blocks, "h1"), "|"); got != "1. The letter|2. The road" {
		t.Errorf("chapter headings %q", got)
	}
	items := texts(blocks, "li")
	if len(items) != 3 { // the scene with no summary is left out
		t.Fatalf("items %v", items)
	}
	if items[0] != "•  Under the clock: She opens it." {
		t.Errorf("first item %q", items[0])
	}
	// the chapter summary comes after the heading
	var kinds []string
	for _, b := range blocks {
		kinds = append(kinds, b.kind)
	}
	if got := strings.Join(kinds, " "); got != "title author subtitle h1 p li h1 li li" {
		t.Errorf("order %s", got)
	}

	// details and empty scenes
	blocks, _ = e.outline("outline", b1, ExportOptions{OutlineDetails: true, OutlineEmpty: true})
	items = texts(blocks, "li")
	if len(items) != 4 {
		t.Fatalf("with empty scenes: %v", items)
	}
	if !strings.Contains(items[0], "(point of view: Mara; place: The clock shop)") {
		t.Errorf("details missing: %q", items[0])
	}
	if !strings.Contains(items[1], "Scene 2") || !strings.Contains(items[1], "no summary yet") {
		t.Errorf("empty scene: %q", items[1])
	}
}

func TestSynopsis(t *testing.T) {
	e, b1, _ := outlineStore(t)
	blocks, name := e.outline("synopsis", b1, ExportOptions{})
	if name != "One synopsis" {
		t.Errorf("name %q", name)
	}
	paras := texts(blocks, "p")
	want := []string{"Mara finds a letter.", "She leaves town. She arrives at the coast."}
	if strings.Join(paras, "|") != strings.Join(want, "|") {
		t.Errorf("paragraphs %v", paras)
	}
	for _, b := range blocks {
		if b.kind == "h1" || b.kind == "li" {
			t.Errorf("a synopsis has no headings or lists: %+v", b)
		}
		if b.kind == "p" && !b.flush {
			t.Error("synopsis paragraphs have no indent")
		}
	}
}

func TestOutlineAllBooksAndFormats(t *testing.T) {
	e, _, _ := outlineStore(t)
	blocks, name := e.outline("outline", 0, ExportOptions{})
	if name != "Complete series outline" {
		t.Errorf("name %q", name)
	}
	if len(texts(blocks, "title")) != 2 {
		t.Errorf("titles %v", texts(blocks, "title"))
	}
	if _, err := buildDocx(blocks, ExportOptions{}); err != nil {
		t.Error(err)
	}
	if _, err := buildEPUB(blocks, name, "A. Writer"); err != nil {
		t.Error(err)
	}
	if _, err := buildPDF(blocks, ExportOptions{Contents: false}); err != nil {
		t.Error(err)
	}
	if md := string(buildMarkdown(blocks)); !strings.Contains(md, "Under the clock") {
		t.Errorf("markdown: %s", md)
	}
	if got, _ := e.outline("outline", 99999, ExportOptions{}); len(got) != 0 {
		t.Error("an unknown book gives nothing")
	}
}
