package main

import (
	"fmt"
	"strings"
)

// Front pages (copyright, dedication), back pages (acknowledgments, about the author),
// the mark between scenes, and the way a chapter heading reads.

// breakText is the mark that a scene break is printed as. "" means each format's usual mark (three stars).
func breakText(style string) string {
	switch style {
	case "hash":
		return "#"
	case "dashes":
		return "—   —   —"
	case "blank":
		return " " // a line with nothing on it, but with room above and below
	}
	return ""
}

// withBreakStyle puts the chosen mark on every scene break. It leaves the input as it was.
func withBreakStyle(blocks []block, style string) []block {
	mark := breakText(style)
	if mark == "" {
		return blocks
	}
	out := make([]block, len(blocks))
	copy(out, blocks)
	for i := range out {
		if out[i].kind == "rule" {
			out[i].text = mark
		}
	}
	return out
}

// chapterHeading is the heading of one chapter.
// style: "full" (the default) is "Chapter 3: Title", "title" is "Title", "number" is "Chapter 3", "plain" is "3. Title".
// A chapter with no title always shows its number.
func chapterHeading(order int, title, style string) string {
	title = strings.TrimSpace(title)
	switch style {
	case "title":
		if title != "" {
			return title
		}
		return fmt.Sprintf("Chapter %d", order)
	case "number":
		return fmt.Sprintf("Chapter %d", order)
	case "plain":
		if title != "" {
			return fmt.Sprintf("%d. %s", order, title)
		}
		return fmt.Sprintf("%d", order)
	}
	h := fmt.Sprintf("Chapter %d", order)
	if title != "" {
		h += ": " + title
	}
	return h
}

// centered turns text into centered paragraphs, one for each line.
func centered(text, page string, italic bool, down int) []block {
	var out []block
	for i, p := range paragraphs(text) {
		b := block{kind: "center", spans: []span{{text: p, italic: italic}}, front: true, label: page}
		if i == 0 {
			b.level = down
		}
		out = append(out, b)
	}
	return out
}

// frontPages are the copyright page and the dedication, each on its own page. They come after the title page.
func frontPages(o ExportOptions) []block {
	var out []block
	if c := centered(o.Copyright, "Copyright", false, 6); len(c) > 0 {
		out = append(out, block{kind: "pagebreak", front: true})
		out = append(out, c...)
	}
	if d := centered(o.Dedication, "Dedication", true, 6); len(d) > 0 {
		out = append(out, block{kind: "pagebreak", front: true})
		out = append(out, d...)
	}
	return out
}

// backPages are the acknowledgments and the "about the author" page, after the last chapter.
func backPages(o ExportOptions) []block {
	var out []block
	for _, page := range []struct{ heading, text string }{{"Acknowledgments", o.Thanks}, {"About the author", o.About}} {
		paras := paragraphs(page.text)
		if len(paras) == 0 {
			continue
		}
		out = append(out, block{kind: "pagebreak"}, block{kind: "h1", text: page.heading})
		for i, p := range paras {
			out = append(out, block{kind: "p", text: p, flush: i == 0})
		}
	}
	return out
}

// isCover says whether this block belongs to the cover or the front pages, which come before the contents.
func isCover(blocks []block, i int) bool {
	switch blocks[i].kind {
	case "title", "author", "subtitle":
		return true
	}
	return blocks[i].front
}
