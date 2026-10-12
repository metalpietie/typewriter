package main

import (
	"fmt"
	"strings"
)

// The outline and the synopsis of your books, made from the summaries you wrote for chapters and scenes.
//
// An outline lists every chapter with its summary, and under it each scene with its summary
// (and, when you ask, the point of view and the place).
// A synopsis is the running story: one paragraph for each chapter, with no headings. It is what an agent or an editor asks for.

// outline builds the blocks for "outline" or "synopsis". bookID 0 means every book. It returns the blocks and a file name.
func (e *Exporter) outline(what string, bookID int64, opts ExportOptions) ([]block, string) {
	books, chapters, scenes := e.load("book"), e.load("chapter"), e.load("scene")
	people, places := e.load("character"), e.load("location")
	byOrder(books, "order")
	label := "Outline"
	if what == "synopsis" {
		label = "Synopsis"
	}
	var out []block
	title := label
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
		if a := strings.TrimSpace(opts.Author); a != "" {
			out = append(out, block{kind: "author", text: a})
		}
		out = append(out, block{kind: "subtitle", text: label})
		if what == "synopsis" {
			out = append(out, synopsisBlocks(chs, scenes)...)
		} else {
			out = append(out, outlineBlocks(chs, scenes, people, places, opts)...)
		}
		title = str(b, "title") + " " + strings.ToLower(label)
	}
	if bookID == 0 && count > 1 {
		title = "Complete series " + strings.ToLower(label)
	}
	return out, title
}

// scenesOfChapter returns the scenes of one chapter, in order.
func scenesOfChapter(c rec, scenes []rec) []rec {
	var mine []rec
	for _, s := range scenes {
		if num(s, "chapter") == num(c, "id") {
			mine = append(mine, s)
		}
	}
	byOrder(mine, "order")
	return mine
}

func outlineBlocks(chs, scenes, people, places []rec, opts ExportOptions) []block {
	var out []block
	for _, c := range chs {
		out = append(out, block{kind: "h1", text: chapterHeading(int(num(c, "order")), str(c, "title"), opts.ChapterStyle)})
		if s := str(c, "summary"); s != "" {
			out = append(out, block{kind: "p", text: s, flush: true})
		}
		for i, sc := range scenesOfChapter(c, scenes) {
			name := str(sc, "title")
			if name == "" {
				name = fmt.Sprintf("Scene %d", i+1)
			}
			sum := str(sc, "summary")
			if sum == "" && !opts.OutlineEmpty {
				continue
			}
			spans := []span{{text: "•  "}, {text: name, bold: true}}
			if sum != "" {
				spans = append(spans, span{text: ": " + sum})
			} else {
				spans = append(spans, span{text: " (no summary yet)", italic: true})
			}
			if opts.OutlineDetails {
				var bits []string
				if p := nameOf(people, num(sc, "pov"), "name"); p != "" {
					bits = append(bits, "point of view: "+p)
				}
				if l := nameOf(places, num(sc, "location"), "name"); l != "" {
					bits = append(bits, "place: "+l)
				}
				if len(bits) > 0 {
					spans = append(spans, span{text: " (" + strings.Join(bits, "; ") + ")", italic: true})
				}
			}
			out = append(out, block{kind: "li", spans: spans})
		}
	}
	return out
}

func synopsisBlocks(chs, scenes []rec) []block {
	var out []block
	for _, c := range chs {
		text := str(c, "summary")
		if text == "" {
			var parts []string
			for _, sc := range scenesOfChapter(c, scenes) {
				if s := str(sc, "summary"); s != "" {
					parts = append(parts, s)
				}
			}
			text = strings.Join(parts, " ")
		}
		if text != "" {
			out = append(out, block{kind: "p", text: text, flush: true})
		}
	}
	return out
}
