package main

import (
	"strings"
	"testing"
)

const boxHTML = `<p>Intro text.</p>` +
	`<div data-callout="exercise" class="callout"><p>Write down three habits.</p><ul><li>one</li><li>two</li></ul></div>` +
	`<div data-callout="weird"><p>Falls back to a key idea.</p></div>` +
	`<div data-callout="story"></div>` +
	`<p data-caption="figure">The habit loop</p>` +
	`<p data-caption="table"></p>` +
	`<p>After <span data-index="habit loop">the loop</span> and <span data-index="cue"><strong>cue</strong></span>.</p>`

func TestCalloutsFromHTML(t *testing.T) {
	bs := chapterBlocks(boxHTML)
	var kinds, boxes []string
	for _, b := range bs {
		kinds = append(kinds, b.kind)
		boxes = append(boxes, b.box)
	}
	if strings.Join(kinds, ",") != "p,boxhead,p,li,li,boxhead,p,caption,p" {
		t.Fatalf("kinds %v", kinds)
	}
	if strings.Join(boxes, ",") != ",exercise,exercise,exercise,exercise,idea,idea,,,"[:len(strings.Join(boxes, ","))] && strings.Join(boxes, ",") != ",exercise,exercise,exercise,exercise,idea,idea,," {
		t.Fatalf("boxes %v", boxes)
	}
	if bs[1].text != "Exercise" || bs[5].text != "Key idea" {
		t.Errorf("labels %q %q", bs[1].text, bs[5].text)
	}
	if bs[7].label != "figure" || spansText(bs[7].spans) != "The habit loop" {
		t.Errorf("caption %+v", bs[7])
	}
}

func TestCaptionNumbers(t *testing.T) {
	bs := []block{
		{kind: "caption", label: "figure", spans: []span{{text: "One"}}},
		{kind: "p", text: "x"},
		{kind: "caption", label: "table", spans: []span{{text: "Two"}}},
		{kind: "caption", label: "figure", spans: []span{{text: "Three"}}},
	}
	out := numberCaptions(bs, 3)
	got := []string{spansText(out[0].spans), spansText(out[2].spans), spansText(out[3].spans)}
	want := []string{"Figure 3.1: One", "Table 3.1: Two", "Figure 3.2: Three"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %q want %q", got[i], want[i])
		}
	}
	if spansText(bs[0].spans) != "One" {
		t.Error("the input is not changed")
	}
	again := numberCaptions(bs, 4)
	if spansText(again[0].spans) != "Figure 4.1: One" {
		t.Error("the count starts again in each chapter")
	}
}

func TestIndex(t *testing.T) {
	idx := map[string]*indexEntry{}
	collectIndex(idx, chapterBlocks(boxHTML), 2)
	collectIndex(idx, []block{{kind: "p", spans: []span{{text: "Cue", index: "Cue"}, {text: "again", index: "Cue"}}}}, 2)
	collectIndex(idx, []block{{kind: "p", spans: []span{{text: "x", index: "cue"}, {text: "y", index: "Zebra"}}}}, 5)
	if len(idx) != 3 || len(idx["cue"].chapters) != 2 || idx["cue"].chapters[1] != 5 {
		t.Fatalf("%+v", idx)
	}
	out := indexBlocks(idx)
	var lines []string
	for _, b := range out {
		lines = append(lines, spansText(spansOf(b)))
	}
	want := "C|cue, chapters 2, 5|H|habit loop, chapter 2|Z|Zebra, chapter 5"
	if strings.Join(lines[2:], "|") != want {
		t.Fatalf("got %q", strings.Join(lines[2:], "|"))
	}
	if indexBlocks(nil) != nil {
		t.Error("no marks, no index")
	}
}

func TestExerciseList(t *testing.T) {
	one := chapterBlocks(boxHTML)
	list := exerciseList([]chapterBlocksOf{{heading: "Chapter 1", blocks: one}, {heading: "Chapter 2", blocks: []block{{kind: "p", text: "none here"}}}})
	if list[1].text != "Exercises" || spansText(list[2].spans) != "Chapter 1" || list[3].text != "Exercise" {
		t.Fatalf("%+v", list[:4])
	}
	for _, b := range list[3:] {
		if b.box != "exercise" {
			t.Errorf("only the exercise: %+v", b)
		}
	}
	if exerciseList([]chapterBlocksOf{{heading: "x", blocks: []block{{kind: "p", text: "a"}}}}) != nil {
		t.Error("no exercises, no list")
	}
}

func TestBoxesInEveryFormat(t *testing.T) {
	bs := []block{{kind: "title", text: "Book"}, {kind: "pagebreak"}, {kind: "h1", text: "Chapter 1"}}
	bs = append(bs, numberCaptions(chapterBlocks(boxHTML), 1)...)
	d, err := buildDocx(bs, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	doc := zipText(t, d)
	for _, want := range []string{`w:fill="FEF5E1"`, `w:color="F59E0B"`, `w:styleId="Caption"`, "Figure 1.1: ", "Exercise", `w:fill="EAF2FF"`} {
		if !strings.Contains(doc, want) {
			t.Errorf("docx lacks %q", want)
		}
	}
	e, err := buildEPUBWithCover(bs, "Book", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	ep := zipText(t, e)
	for _, want := range []string{`class="boxhead box box-exercise"`, `class="li box box-exercise"`, `class="caption"`, ".box-exercise { border-left-color: #F59E0B"} {
		if !strings.Contains(ep, want) {
			t.Errorf("epub lacks %q", want)
		}
	}
	md := string(buildMarkdown(bs))
	for _, want := range []string{"> **Exercise**", "> Write down three habits.", "> - one", "***Figure 1.1:*** *The habit loop*"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown lacks %q:\n%s", want, md)
		}
	}
	p, err := buildPDF(bs, ExportOptions{})
	if err != nil || len(p) < 500 {
		t.Fatalf("pdf: %v", err)
	}
}
