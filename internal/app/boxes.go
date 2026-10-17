package app

import (
	"fmt"
	"sort"
	"strings"
)

// Callout boxes, figure and table captions, and the index of terms in a non-fiction book.

// boxStyle is the look of one kind of callout box. Colours are hex without the #.
type boxStyle struct {
	key, label string
	bar, fill  string // the bar at the left, and the light colour behind the text
}

var boxStyles = []boxStyle{
	{"idea", "Key idea", "3B82F6", "EAF2FF"},
	{"story", "Story", "8B5CF6", "F1EBFF"},
	{"example", "Example", "10B981", "E8F8F1"},
	{"exercise", "Exercise", "F59E0B", "FEF5E1"},
	{"warning", "Warning", "EF4444", "FDECEC"},
	{"quote", "Quote", "6B7280", "F1F2F4"},
}

// boxOf finds the style of a kind. An unknown kind is a plain "Key idea" box.
func boxOf(key string) boxStyle {
	for _, b := range boxStyles {
		if b.key == key {
			return b
		}
	}
	return boxStyles[0]
}

func isBoxKind(key string) bool {
	for _, b := range boxStyles {
		if b.key == key {
			return true
		}
	}
	return false
}

// captionLabel is the word before the number of a caption.
func captionLabel(kind string) string {
	switch kind {
	case "table":
		return "Table"
	case "chart":
		return "Chart"
	}
	return "Figure"
}

// numberCaptions puts "Figure 3.2: " in front of each caption of one chapter. The count starts again in each chapter.
func numberCaptions(blocks []block, chapter int) []block {
	count := map[string]int{}
	out := make([]block, len(blocks))
	for i, b := range blocks {
		out[i] = b
		if b.kind != "caption" {
			continue
		}
		count[b.label]++
		prefix := span{text: fmt.Sprintf("%s %d.%d: ", captionLabel(b.label), chapter, count[b.label]), italic: true, bold: true}
		spans := append([]span{prefix}, b.spans...)
		for j := range spans {
			spans[j].italic = true
		}
		out[i].spans = spans
	}
	return out
}

// indexEntry is one term of the index and the chapters that it is marked in.
type indexEntry struct {
	term     string
	chapters []int
}

// collectIndex adds the terms that are marked in the blocks of one chapter to the index.
func collectIndex(index map[string]*indexEntry, blocks []block, chapter int) {
	for _, b := range blocks {
		for _, sp := range b.spans {
			term := strings.TrimSpace(sp.index)
			if term == "" {
				continue
			}
			key := strings.ToLower(term)
			e := index[key]
			if e == nil {
				e = &indexEntry{term: term}
				index[key] = e
			}
			if n := len(e.chapters); n == 0 || e.chapters[n-1] != chapter {
				e.chapters = append(e.chapters, chapter)
			}
		}
	}
}

// indexBlocks is the page at the end: each term with the chapters that it is in, in the order of the alphabet, with a letter above each group.
func indexBlocks(index map[string]*indexEntry) []block {
	if len(index) == 0 {
		return nil
	}
	keys := make([]string, 0, len(index))
	for k := range index {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []block{{kind: "pagebreak"}, {kind: "h1", text: "Index"}}
	letter := ""
	for _, k := range keys {
		e := index[k]
		first := strings.ToUpper(string([]rune(e.term)[0]))
		if first != letter {
			letter = first
			out = append(out, block{kind: "sub", spans: []span{{text: letter, bold: true}}})
		}
		var nums []string
		for _, c := range e.chapters {
			nums = append(nums, fmt.Sprint(c))
		}
		word := "chapter"
		if len(nums) > 1 {
			word = "chapters"
		}
		out = append(out, block{kind: "p", flush: true, spans: []span{{text: e.term}, {text: ", " + word + " " + strings.Join(nums, ", ")}}})
	}
	return out
}

// exerciseGroups takes the exercise boxes out of the blocks of one chapter. Each box is a group of blocks.
func exerciseGroups(blocks []block) [][]block {
	var out [][]block
	for _, b := range blocks {
		if b.box != "exercise" {
			continue
		}
		if b.kind == "boxhead" || len(out) == 0 {
			out = append(out, nil)
		}
		out[len(out)-1] = append(out[len(out)-1], b)
	}
	return out
}

// exerciseList is "All exercises": the exercise boxes of every chapter, under the title of the chapter, after the last chapter.
func exerciseList(chapters []chapterBlocksOf) []block {
	var out []block
	for _, c := range chapters {
		groups := exerciseGroups(c.blocks)
		if len(groups) == 0 {
			continue
		}
		out = append(out, block{kind: "sub", spans: []span{{text: c.heading, bold: true}}})
		for _, g := range groups {
			out = append(out, g...)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return append([]block{{kind: "pagebreak"}, {kind: "h1", text: "Exercises"}}, out...)
}

// chapterBlocksOf is the blocks of one chapter and its heading, kept for the list of exercises.
type chapterBlocksOf struct {
	heading string
	blocks  []block
}

// hexRGB reads a colour such as "3B82F6".
func hexRGB(h string) (int, int, int) {
	var r, g, b int
	fmt.Sscanf(h, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

// boxCSS is the style of the callout boxes in an e-book.
func boxCSS() string {
	var sb strings.Builder
	sb.WriteString(".box { border-left: 4px solid #6B7280; padding: .15em .8em; margin: 0; text-indent: 0; background: #F1F2F4; }\n")
	sb.WriteString("p.boxhead { font-weight: bold; margin-top: 1em; padding-top: .4em; }\n")
	sb.WriteString("p.caption { text-align: center; font-size: .9em; text-indent: 0; margin: .4em 0 1.2em; }\n")
	for _, b := range boxStyles {
		sb.WriteString(fmt.Sprintf(".box-%s { border-left-color: #%s; background: #%s; }\n", b.key, b.bar, b.fill))
	}
	return sb.String()
}
