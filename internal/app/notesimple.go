package app

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	xhtml "golang.org/x/net/html"
)

// The simple form of a note, for the phone. The phone edits paragraphs, headings, bold, italic, lists and quotes.
//
//	# Heading, ## Heading, ### Heading       one line each
//	- an item                                a bullet list (one item on each line)
//	1. an item                               a numbered list
//	> a quote                                a quote (one paragraph on each line)
//	**bold** and *italic*                    inside any line
//	a blank line                             between blocks
//
// A * or a \ in the text is written \* and \\, and a line that would look like one of the marks starts with a \.
// A note that has anything else in it (a link to a character, a comment, a citation, a footnote, a box, an index mark,
// underline, a rule) can be shown on the phone but not edited there, so the phone never removes it.

var unsupportedAttrs = []string{"data-ref", "data-comment", "data-cite", "data-footnote", "data-callout", "data-index", "data-caption", "data-needs"}

// htmlToSimple turns the text of a Notes page into the simple form. editable is false when the note has things that the simple form cannot keep.
func htmlToSimple(body string) (text string, editable bool) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", true
	}
	if !strings.HasPrefix(body, "<") { // an old page of plain text
		var blocks []string
		for _, p := range paragraphs(body) {
			blocks = append(blocks, escapeLine(p))
		}
		return strings.Join(blocks, "\n\n"), true
	}
	root, err := xhtml.Parse(strings.NewReader(body))
	if err != nil {
		return strings.TrimSpace(body), false
	}
	editable = true
	var blocks []string
	var walk func(n *xhtml.Node)
	walk = func(n *xhtml.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != xhtml.ElementNode {
				continue
			}
			switch c.Data {
			case "html", "head":
				walk(c)
			case "body":
				walk(c)
			case "p":
				if t := strings.TrimSpace(inlineSimple(c, &editable)); t != "" {
					blocks = append(blocks, escapeBlockStart(t))
				}
			case "h1", "h2", "h3", "h4", "h5", "h6":
				level := int(c.Data[1] - '0')
				if level > 3 {
					level = 3
				}
				if t := strings.TrimSpace(inlineSimple(c, &editable)); t != "" {
					blocks = append(blocks, strings.Repeat("#", level)+" "+t)
				}
			case "ul", "ol":
				var lines []string
				i := 0
				for li := c.FirstChild; li != nil; li = li.NextSibling {
					if li.Type != xhtml.ElementNode || li.Data != "li" {
						continue
					}
					for x := li.FirstChild; x != nil; x = x.NextSibling {
						if x.Type == xhtml.ElementNode && (x.Data == "ul" || x.Data == "ol") {
							editable = false // a list inside a list
						}
					}
					i++
					t := strings.TrimSpace(strings.ReplaceAll(inlineSimple(li, &editable), "\n", " "))
					if c.Data == "ol" {
						lines = append(lines, fmt.Sprintf("%d. %s", i, t))
					} else {
						lines = append(lines, "- "+t)
					}
				}
				if len(lines) > 0 {
					blocks = append(blocks, strings.Join(lines, "\n"))
				}
			case "blockquote":
				var lines []string
				for q := c.FirstChild; q != nil; q = q.NextSibling {
					if q.Type == xhtml.ElementNode && q.Data == "p" {
						if t := strings.TrimSpace(strings.ReplaceAll(inlineSimple(q, &editable), "\n", " ")); t != "" {
							lines = append(lines, "> "+t)
						}
					} else if q.Type == xhtml.ElementNode {
						editable = false
					}
				}
				if len(lines) > 0 {
					blocks = append(blocks, strings.Join(lines, "\n"))
				}
			case "div", "hr", "img", "table", "pre", "figure":
				editable = false
				if t := strings.TrimSpace(textOf(c)); t != "" && c.Data != "hr" {
					blocks = append(blocks, escapeBlockStart(t))
				}
			default:
				editable = false
				walk(c)
			}
		}
	}
	walk(root)
	return strings.Join(blocks, "\n\n"), editable
}

func textOf(n *xhtml.Node) string {
	var sb strings.Builder
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

func hasAttr(n *xhtml.Node, names []string) bool {
	for _, a := range n.Attr {
		for _, want := range names {
			if a.Key == want {
				return true
			}
		}
	}
	return false
}

// inlineSimple writes the text inside one block, with ** and * for bold and italic.
func inlineSimple(n *xhtml.Node, editable *bool) string {
	var sb strings.Builder
	var walk func(n *xhtml.Node, bold, italic bool)
	walk = func(n *xhtml.Node, bold, italic bool) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch c.Type {
			case xhtml.TextNode:
				t := strings.ReplaceAll(escapeInline(c.Data), "\n", " ")
				if t == "" {
					continue
				}
				mark := ""
				if bold {
					mark += "**"
				}
				if italic {
					mark += "*"
				}
				if mark == "" || strings.TrimSpace(t) == "" {
					sb.WriteString(t)
					continue
				}
				core := strings.TrimSpace(t)
				at := strings.Index(t, core)
				rev := ""
				if italic {
					rev += "*"
				}
				if bold {
					rev += "**"
				}
				sb.WriteString(t[:at] + mark + core + rev + t[at+len(core):])
			case xhtml.ElementNode:
				if hasAttr(c, unsupportedAttrs) {
					*editable = false
					walk(c, bold, italic)
					continue
				}
				switch c.Data {
				case "strong", "b":
					walk(c, true, italic)
				case "em", "i":
					walk(c, bold, true)
				case "br":
					sb.WriteString("\n")
				case "span":
					walk(c, bold, italic)
				case "p": // a paragraph inside a list item
					walk(c, bold, italic)
					sb.WriteString(" ")
				default: // underline, links and anything else
					*editable = false
					walk(c, bold, italic)
				}
			}
		}
	}
	walk(n, false, false)
	return sb.String()
}

var blockStart = regexp.MustCompile(`^(#{1,3} |- |> |\d+\. )`)

// escapeLine and escapeBlockStart keep a line of plain text from being read as a mark.
func escapeLine(s string) string { return escapeBlockStart(escapeInline(s)) }
func escapeInline(s string) string {
	return strings.NewReplacer(`\`, `\\`, "*", `\*`).Replace(s)
}
func escapeBlockStart(s string) string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if blockStart.MatchString(line) {
			line = `\` + line
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

var (
	numbered   = regexp.MustCompile(`^\d+\. `)
	headingRe  = regexp.MustCompile(`^(#{1,3}) (.*)$`)
	inlineMark = regexp.MustCompile(`\\.|\*\*|\*`)
)

// simpleToHTML turns the simple form into the HTML of a Notes page.
func simpleToHTML(text string) string {
	text = strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n")
	if text == "" {
		return ""
	}
	var out strings.Builder
	for _, block := range regexp.MustCompile(`\n\s*\n`).Split(text, -1) {
		lines := strings.Split(strings.TrimRight(block, "\n"), "\n")
		allWith := func(ok func(string) bool) bool {
			for _, l := range lines {
				if !ok(l) {
					return false
				}
			}
			return true
		}
		switch {
		case len(lines) == 1 && headingRe.MatchString(lines[0]):
			m := headingRe.FindStringSubmatch(lines[0])
			fmt.Fprintf(&out, "<h%d>%s</h%d>", len(m[1]), inlineHTML(m[2]), len(m[1]))
		case allWith(func(l string) bool { return strings.HasPrefix(l, "- ") }):
			out.WriteString("<ul>")
			for _, l := range lines {
				out.WriteString("<li><p>" + inlineHTML(strings.TrimPrefix(l, "- ")) + "</p></li>")
			}
			out.WriteString("</ul>")
		case allWith(func(l string) bool { return numbered.MatchString(l) }):
			out.WriteString("<ol>")
			for _, l := range lines {
				out.WriteString("<li><p>" + inlineHTML(numbered.ReplaceAllString(l, "")) + "</p></li>")
			}
			out.WriteString("</ol>")
		case allWith(func(l string) bool { return strings.HasPrefix(l, "> ") }):
			out.WriteString("<blockquote>")
			for _, l := range lines {
				out.WriteString("<p>" + inlineHTML(strings.TrimPrefix(l, "> ")) + "</p>")
			}
			out.WriteString("</blockquote>")
		default:
			var parts []string
			for _, l := range lines {
				l = strings.TrimPrefix(l, `\`) // an escaped start of line: \# \- \> \1.
				if strings.HasPrefix(l, `\`) { // it was a real backslash that starts the line
					l = `\` + l
				}
				parts = append(parts, inlineHTML(l))
			}
			out.WriteString("<p>" + strings.Join(parts, "<br>") + "</p>")
		}
	}
	return out.String()
}

type inlineTok struct {
	kind string // text, b (bold), i (italic)
	text string
}

// inlineHTML reads ** and * in one line. A bold or italic mark that has no partner is plain text.
func inlineHTML(s string) string {
	var toks []inlineTok
	last := 0
	for _, loc := range inlineMark.FindAllStringIndex(s, -1) {
		if loc[0] > last {
			toks = append(toks, inlineTok{"text", s[last:loc[0]]})
		}
		m := s[loc[0]:loc[1]]
		switch {
		case strings.HasPrefix(m, "\\"):
			toks = append(toks, inlineTok{"text", m[1:]})
		case m == "**":
			toks = append(toks, inlineTok{"b", m})
		default:
			toks = append(toks, inlineTok{"i", m})
		}
		last = loc[1]
	}
	if last < len(s) {
		toks = append(toks, inlineTok{"text", s[last:]})
	}
	count, lastAt := map[string]int{}, map[string]int{}
	for i, t := range toks {
		if t.kind != "text" {
			count[t.kind]++
			lastAt[t.kind] = i
		}
	}
	tags := map[string]string{"b": "strong", "i": "em"}
	open := map[string]bool{}
	var sb strings.Builder
	for i, t := range toks {
		if t.kind == "text" || (count[t.kind]%2 == 1 && i == lastAt[t.kind]) {
			sb.WriteString(html.EscapeString(t.text))
			continue
		}
		if open[t.kind] {
			sb.WriteString("</" + tags[t.kind] + ">")
		} else {
			sb.WriteString("<" + tags[t.kind] + ">")
		}
		open[t.kind] = !open[t.kind]
	}
	return sb.String()
}
