package main

import (
	"fmt"
	"strings"
	"unicode"
)

// The table of contents is added to the blocks before a file is written.
// It sits after the cover (title, author, subtitle) and lists every chapter heading.
// A series with several books lists each book title too, with its chapters under it.
//
// Each heading that is listed gets an anchor, such as "toc3". Each format uses it its own way:
// Word makes a link to a bookmark, Markdown links to the heading, the e-book links to the page file,
// and the PDF links to the page and prints its number.

// withContents returns the blocks with a table of contents added. It leaves the input as it was.
func withContents(blocks []block) []block {
	titles := 0
	for _, b := range blocks {
		if b.kind == "title" {
			titles++
		}
	}
	out := make([]block, len(blocks))
	copy(out, blocks)

	var items []block
	for i := range out {
		level := -1
		switch out[i].kind {
		case "title":
			if titles > 1 {
				level = 0
			}
		case "h1":
			level = 0
			if titles > 1 {
				level = 1
			}
		}
		if level < 0 || strings.TrimSpace(out[i].text) == "" {
			continue
		}
		out[i].anchor = fmt.Sprintf("toc%d", len(items)+1)
		items = append(items, block{kind: "tocitem", text: out[i].text, level: level, anchor: out[i].anchor})
	}
	if len(items) == 0 {
		return blocks
	}

	at := 0 // the cover: the title, the author and the subtitle of the first book
	for at < len(out) && isCover(out, at) {
		at++
	}
	res := make([]block, 0, len(out)+len(items)+2)
	res = append(res, out[:at]...)
	res = append(res, block{kind: "pagebreak"}, block{kind: "tochead", text: "Contents"})
	res = append(res, items...)
	res = append(res, out[at:]...)
	return res
}

func hasContents(blocks []block) bool {
	for _, b := range blocks {
		if b.kind == "tochead" {
			return true
		}
	}
	return false
}

// slugify makes the heading id that GitHub-style Markdown viewers use: lower case, no marks, hyphens for spaces.
func slugify(s string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-':
			out.WriteRune(r)
		case r == ' ':
			out.WriteRune('-')
		}
	}
	return out.String()
}

// markdownSlugs gives the id of each listed heading, in the way a viewer will make it.
// Two headings with the same text get -1, -2 and so on, as GitHub does.
func markdownSlugs(blocks []block) map[string]string {
	seen := map[string]int{}
	out := map[string]string{}
	for _, b := range blocks {
		var text string
		switch b.kind {
		case "title", "tochead", "h1", "h2":
			text = b.text
		case "sub":
			text = spansText(spansOf(b))
		default:
			continue
		}
		slug := slugify(text)
		if n := seen[slug]; n > 0 {
			seen[slug]++
			slug = fmt.Sprintf("%s-%d", slug, n)
		} else {
			seen[slug] = 1
		}
		if b.anchor != "" {
			out[b.anchor] = slug
		}
	}
	return out
}
