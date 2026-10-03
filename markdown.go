package main

import (
	"regexp"
	"strings"
	"unicode"
)

// mergeSpans joins neighbours that have the same style, so the output has no empty markers.
func mergeSpans(in []span) []span {
	var out []span
	for _, s := range in {
		if s.text == "" {
			continue
		}
		if n := len(out); n > 0 && out[n-1].bold == s.bold && out[n-1].italic == s.italic && out[n-1].underline == s.underline {
			out[n-1].text += s.text
			continue
		}
		out = append(out, s)
	}
	return out
}

var mdSpecial = strings.NewReplacer(`\`, `\\`, "*", `\*`, "_", `\_`, "`", "\\`", "[", `\[`, "]", `\]`, "<", `\<`)
var mdNumbered = regexp.MustCompile(`^(\d+)\.  `)
var mdLineStart = regexp.MustCompile(`^(#|>|-|\+|\d+[.)])`)

// mdInline writes styled text as Markdown. Spaces stay outside the markers, which Markdown needs.
func mdInline(spans []span) string {
	var out strings.Builder
	for _, sp := range mergeSpans(spans) {
		core := strings.TrimFunc(sp.text, unicode.IsSpace)
		if core == "" {
			out.WriteString(strings.ReplaceAll(sp.text, "\n", "  \n"))
			continue
		}
		at := strings.Index(sp.text, core)
		lead, tail := sp.text[:at], sp.text[at+len(core):]
		core = strings.ReplaceAll(mdSpecial.Replace(core), "\n", "  \n")
		if sp.underline {
			core = "<u>" + core + "</u>"
		}
		switch {
		case sp.bold && sp.italic:
			core = "***" + core + "***"
		case sp.bold:
			core = "**" + core + "**"
		case sp.italic:
			core = "*" + core + "*"
		}
		out.WriteString(strings.ReplaceAll(lead, "\n", "  \n") + core + strings.ReplaceAll(tail, "\n", "  \n"))
	}
	return strings.TrimSpace(out.String())
}

// buildMarkdown turns the export blocks into one Markdown file.
func buildMarkdown(blocks []block) []byte {
	var out strings.Builder
	inList := false
	put := func(s string) {
		if strings.TrimSpace(s) != "" {
			out.WriteString(s + "\n\n")
		}
	}
	for _, b := range blocks {
		if inList && b.kind != "li" {
			out.WriteString("\n") // a blank line ends a list
			inList = false
		}
		switch b.kind {
		case "pagebreak":
		case "title":
			put("# " + mdInline([]span{{text: b.text}}))
		case "author":
			put("*by " + mdInline([]span{{text: b.text}}) + "*")
		case "subtitle":
			put("*" + mdInline([]span{{text: b.text}}) + "*")
		case "h1":
			put("## " + mdInline([]span{{text: b.text}}))
		case "h2":
			put("### " + mdInline([]span{{text: b.text}}))
		case "sub":
			put("### " + mdInline(spansOf(b)))
		case "kv":
			put("**" + mdInline([]span{{text: b.label}}) + ":** " + mdInline([]span{{text: b.text}}))
		case "rule":
			put("* * *")
		case "quote":
			put("> " + strings.ReplaceAll(mdInline(spansOf(b)), "\n", "\n> "))
		case "li":
			sp := append([]span(nil), spansOf(b)...)
			marker := "- "
			if len(sp) > 0 {
				if t, ok := strings.CutPrefix(sp[0].text, "•  "); ok {
					sp[0].text = t
				} else if m := mdNumbered.FindStringSubmatch(sp[0].text); m != nil {
					marker = m[1] + ". "
					sp[0].text = strings.TrimPrefix(sp[0].text, m[0])
				}
			}
			out.WriteString(marker + strings.ReplaceAll(mdInline(sp), "\n", "\n  ") + "\n")
			inList = true
		default:
			t := mdInline(spansOf(b))
			if mdLineStart.MatchString(t) {
				t = `\` + t
			}
			put(t)
		}
	}
	return []byte(strings.TrimRight(out.String(), "\n") + "\n")
}
