package main

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func sectionFixture() (rec, []rec) {
	c := rec{"id": 1.0, "takeaways": "- Sleep first\n* Then train\n\n"}
	scenes := []rec{
		{"id": 10.0, "chapter": 1.0, "order": 1.0, "title": "Why habits matter", "body": "<p>One.</p>"},
		{"id": 11.0, "chapter": 1.0, "order": 2.0, "title": "", "body": "<p>Two.</p>"},
		{"id": 12.0, "chapter": 1.0, "order": 3.0, "title": "Small steps", "body": "<p>Three.</p>"},
	}
	return c, scenes
}

func kindsOf(bs []block) string {
	var k []string
	for _, b := range bs {
		k = append(k, b.kind)
	}
	return strings.Join(k, ",")
}

func TestSectionHeadingsReplaceSceneBreaks(t *testing.T) {
	c, scenes := sectionFixture()
	story := chapterText(c, scenes, false)
	if strings.Contains(kindsOf(story), "h2") || strings.Count(kindsOf(story), "rule") != 2 {
		t.Fatalf("fiction keeps its breaks and has no headings: %s", kindsOf(story))
	}
	book := chapterText(c, scenes, true)
	if strings.Contains(kindsOf(book), "rule") {
		t.Fatalf("non-fiction has no break marks: %s", kindsOf(book))
	}
	var heads []string
	for _, b := range book {
		if b.kind == "h2" {
			heads = append(heads, b.text)
		}
	}
	if strings.Join(heads, "|") != "Why habits matter|Small steps" {
		t.Fatalf("a section with no title gets no heading, got %v", heads)
	}
}

func TestTakeaways(t *testing.T) {
	c, _ := sectionFixture()
	if len(takeawayBlocks(c, ExportOptions{})) != 0 {
		t.Fatal("off by default")
	}
	got := takeawayBlocks(c, ExportOptions{Takeaways: true})
	if len(got) != 3 || got[0].text != "Key takeaways" || got[1].text != "Sleep first" || got[2].text != "Then train" {
		t.Fatalf("got %+v", got)
	}
	if len(takeawayBlocks(rec{}, ExportOptions{Takeaways: true})) != 0 {
		t.Fatal("a chapter with none prints nothing")
	}
}

func zipText(t *testing.T, data []byte) string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	for _, f := range r.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		sb.Write(b)
	}
	return sb.String()
}

func TestSectionHeadingsInEveryFormat(t *testing.T) {
	c, scenes := sectionFixture()
	blocks := []block{{kind: "title", text: "Book"}, {kind: "h1", text: "Chapter 1"}}
	blocks = append(blocks, chapterText(c, scenes, true)...)
	blocks = append(blocks, takeawayBlocks(c, ExportOptions{Takeaways: true})...)
	opts := ExportOptions{}
	d, err := buildDocx(blocks, opts)
	if err != nil || !strings.Contains(zipText(t, d), "Why habits matter") || !strings.Contains(zipText(t, d), "Sleep first") {
		t.Fatalf("docx: %v", err)
	}
	e, err := buildEPUBWithCover(blocks, "Book", "", nil)
	if err != nil || !strings.Contains(zipText(t, e), "Small steps") {
		t.Fatalf("epub: %v", err)
	}
	if m := string(buildMarkdown(blocks)); !strings.Contains(m, "## Why habits matter") || !strings.Contains(m, "Key takeaways") {
		t.Fatalf("markdown: %q", m)
	}
	p, err := buildPDF(blocks, opts)
	if err != nil || len(p) < 200 {
		t.Fatalf("pdf: %v", err)
	}
}

func TestSceneWord(t *testing.T) {
	if (ExportOptions{}).sceneWord() != "Scene" || (ExportOptions{Nonfiction: true}).sceneWord() != "Section" {
		t.Fatal("the word follows the type of project")
	}
	chs := []rec{{"id": 1.0, "order": 1.0, "title": "One"}}
	scenes := []rec{{"id": 5.0, "chapter": 1.0, "order": 1.0, "title": "", "summary": "Why it matters"}}
	got := outlineBlocks(chs, scenes, nil, nil, ExportOptions{Nonfiction: true})
	var all string
	for _, b := range got {
		all += b.text
		for _, s := range b.spans {
			all += s.text
		}
	}
	if !strings.Contains(all, "Section 1") || strings.Contains(all, "Scene 1") {
		t.Fatalf("outline words: %q", all)
	}
}

func TestProjectKindsAndSettingsRoundTrip(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.shut()
	for _, d := range []Doc{
		{Kind: "setting", Data: `{"key":"projectType","value":"nonfiction"}`},
		{Kind: "setting", Data: `{"key":"projectTools","value":{"timeline":true}}`},
		{Kind: "source", Data: `{"title":"Atomic Habits","authors":"Clear, James"}`},
		{Kind: "idea", Data: `{"text":"A quote","kind":"quote"}`},
	} {
		saved, err := s.Save(d)
		if err != nil || saved.ID == 0 {
			t.Fatalf("%s: %v", d.Kind, err)
		}
	}
	for kind, n := range map[string]int{"setting": 2, "source": 1, "idea": 1} {
		if list, err := s.List(kind); err != nil || len(list) != n {
			t.Fatalf("%s: %d %v", kind, len(list), err)
		}
	}
	if _, err := s.Save(Doc{Kind: "nonsense", Data: `{}`}); err == nil {
		t.Fatal("an unknown kind is refused")
	}
}
