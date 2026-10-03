package main

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// span is a piece of text with its style.
type span struct {
	text                    string
	bold, italic, underline bool
}

// spansOf returns the styled text of a block. Plain blocks have one unstyled span.
func spansOf(b block) []span {
	if len(b.spans) > 0 {
		return b.spans
	}
	return []span{{text: b.text}}
}

func spansText(sp []span) string {
	var s strings.Builder
	for _, x := range sp {
		s.WriteString(x.text)
	}
	return s.String()
}

// chapterBlocks turns a chapter body into export blocks.
// New chapters are HTML from the editor. Older chapters are plain text.
func chapterBlocks(body string) []block {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	plain := func() []block {
		var out []block
		for _, p := range paragraphs(body) {
			out = append(out, block{kind: "p", text: p})
		}
		return out
	}
	if !strings.HasPrefix(body, "<") {
		return plain()
	}
	root, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return plain()
	}
	var out []block
	var find func(*html.Node) *html.Node
	find = func(n *html.Node) *html.Node {
		if n.Type == html.ElementNode && n.Data == "body" {
			return n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if f := find(c); f != nil {
				return f
			}
		}
		return nil
	}
	if b := find(root); b != nil {
		blocksFrom(b, "p", &out)
	}
	return out
}

func blocksFrom(n *html.Node, kind string, out *[]block) {
	add := func(k string, sp []span) {
		if strings.TrimSpace(spansText(sp)) != "" {
			*out = append(*out, block{kind: k, spans: sp})
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		switch c.Data {
		case "p":
			add(kind, inlineSpans(c, span{}))
		case "h1", "h2", "h3", "h4", "h5", "h6":
			add("sub", inlineSpans(c, span{}))
		case "blockquote":
			blocksFrom(c, "quote", out)
		case "ul", "ol":
			i := 0
			for li := c.FirstChild; li != nil; li = li.NextSibling {
				if li.Type != html.ElementNode || li.Data != "li" {
					continue
				}
				i++
				prefix := "•  "
				if c.Data == "ol" {
					prefix = fmt.Sprintf("%d.  ", i)
				}
				add("li", append([]span{{text: prefix}}, inlineSpans(li, span{})...))
			}
		case "hr":
			*out = append(*out, block{kind: "rule"})
		}
	}
}

// inlineSpans walks the text inside one block and keeps bold, italic and underline.
func inlineSpans(n *html.Node, style span) []span {
	var out []span
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case html.TextNode:
			out = append(out, span{text: c.Data, bold: style.bold, italic: style.italic, underline: style.underline})
		case html.ElementNode:
			st := style
			switch c.Data {
			case "strong", "b":
				st.bold = true
			case "em", "i":
				st.italic = true
			case "u":
				st.underline = true
			case "br":
				out = append(out, span{text: "\n"})
				continue
			case "p":
				if len(out) > 0 {
					out = append(out, span{text: " "})
				}
			}
			out = append(out, inlineSpans(c, st)...)
		}
	}
	return out
}
