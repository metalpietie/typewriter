package main

import (
	"fmt"
	"sort"
	"strings"
)

// Sources and citations. A source is one book, article or web page that the writer used.
// A citation in the text points to a source and may name a page. These functions turn them into the
// words that are printed: "(Smith, 2020, p. 45)", "[3]", a footnote, or an entry of the bibliography.
// They are pure: no database and no file. The text of one style is in one place, so more styles can be added.

// Citation styles. The first is the start.
const (
	styleAuthorDate = "authordate" // (Smith, 2020, p. 45) in the text, and a list at the end
	styleNumbered   = "numbered"   // [1] in the text, and a numbered list at the end
	styleNotes      = "notes"      // a footnote for each citation
)

// source is one source, read from its saved fields.
type source struct {
	id                                                                 int64
	kind                                                               string // book, article, web, report, interview, other
	title                                                              string
	authors                                                            []string
	year, container, publisher, volume, issue, pages, url, edition, at string
}

func sourceOf(r rec) source {
	s := source{
		id: int64(num(r, "id")), kind: str(r, "type"), title: strings.TrimSpace(str(r, "title")),
		year: strings.TrimSpace(str(r, "year")), container: strings.TrimSpace(str(r, "container")), publisher: strings.TrimSpace(str(r, "publisher")),
		volume: strings.TrimSpace(str(r, "volume")), issue: strings.TrimSpace(str(r, "issue")), pages: strings.TrimSpace(str(r, "pages")),
		url: strings.TrimSpace(str(r, "url")), edition: strings.TrimSpace(str(r, "edition")), at: strings.TrimSpace(str(r, "accessed")),
	}
	s.authors = paragraphs(str(r, "authors"))
	if s.kind == "" {
		s.kind = "other"
	}
	return s
}

// A name is "Last, First" or "First Last". A name that ends with a comma is a group, such as "World Health Organization,".
func splitName(a string) (last, first string, group bool) {
	a = strings.TrimSpace(a)
	if strings.HasSuffix(a, ",") {
		return strings.TrimSpace(strings.TrimSuffix(a, ",")), "", true
	}
	if i := strings.Index(a, ","); i >= 0 {
		return strings.TrimSpace(a[:i]), strings.TrimSpace(a[i+1:]), false
	}
	words := strings.Fields(a)
	switch len(words) {
	case 0:
		return "", "", false
	case 1:
		return words[0], "", false
	}
	return words[len(words)-1], strings.Join(words[:len(words)-1], " "), false
}

func initials(first string) string {
	var out []string
	for _, w := range strings.Fields(first) {
		for _, part := range strings.Split(w, "-") {
			if r := []rune(part); len(r) > 0 {
				out = append(out, string(r[0])+".")
			}
		}
	}
	return strings.Join(out, " ")
}

// lastNames is the short form of the authors for a citation in the text.
func (s source) lastNames() string {
	var names []string
	for _, a := range s.authors {
		if l, _, _ := splitName(a); l != "" {
			names = append(names, l)
		}
	}
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	}
	return names[0] + " et al."
}

// fullNames is "J. Smith" for each author, joined with commas and "and".
func (s source) firstLast() string {
	var names []string
	for _, a := range s.authors {
		l, f, group := splitName(a)
		switch {
		case l == "":
		case group || f == "":
			names = append(names, l)
		default:
			names = append(names, f+" "+l)
		}
	}
	return joinNames(names, " and ")
}

// lastFirst is "Smith, J." for each author, for a list at the end.
func (s source) lastFirst() string {
	var names []string
	for _, a := range s.authors {
		l, f, group := splitName(a)
		switch {
		case l == "":
		case group || f == "":
			names = append(names, l)
		default:
			names = append(names, l+", "+initials(f))
		}
	}
	return joinNames(names, " & ")
}

func joinNames(names []string, last string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + last + names[1]
	}
	return strings.Join(names[:len(names)-1], ", ") + "," + last + names[len(names)-1]
}

func (s source) shortTitle() string {
	w := strings.Fields(s.title)
	if len(w) > 4 {
		return strings.Join(w[:4], " ") + "…"
	}
	return s.title
}

func (s source) yearText() string {
	if s.year == "" {
		return "n.d."
	}
	return s.year
}

// location turns what the writer typed ("45", "45-47", "ch. 3") into the words for a citation ("p. 45", "pp. 45–47", "ch. 3").
func location(loc string) string {
	loc = strings.TrimSpace(loc)
	if loc == "" {
		return ""
	}
	digits := true
	for _, r := range loc {
		if !(r >= '0' && r <= '9') && r != '-' && r != '–' && r != ',' && r != ' ' {
			digits = false
		}
	}
	if !digits {
		return loc
	}
	if strings.ContainsAny(loc, "-–,") {
		return "pp. " + strings.ReplaceAll(strings.ReplaceAll(loc, "-", "–"), " ", "")
	}
	return "p. " + loc
}

// citeText is a citation in the text. n is the number of the source in a numbered style.
func citeText(style string, s source, loc string, n int) string {
	l := location(loc)
	switch style {
	case styleNumbered:
		if l != "" {
			return fmt.Sprintf("[%d, %s]", n, l)
		}
		return fmt.Sprintf("[%d]", n)
	default: // author and date
		who := s.lastNames()
		if who == "" {
			who = s.shortTitle()
		}
		parts := []string{who, s.yearText()}
		if l != "" {
			parts = append(parts, l)
		}
		return "(" + strings.Join(parts, ", ") + ")"
	}
}

// noteText is the text of a footnote for a citation. The first time a source is cited the note is long. Later it is short.
func noteText(s source, loc string, short bool) string {
	l := strings.TrimPrefix(strings.TrimPrefix(location(loc), "p. "), "pp. ")
	who := s.firstLast()
	if short {
		name := s.lastNames()
		if name == "" {
			name = s.shortTitle()
			who = ""
		}
		out := name
		if who != "" {
			out += ", " + s.shortTitle()
		}
		if l != "" {
			out += ", " + l
		}
		return out + "."
	}
	var out string
	switch s.kind {
	case "article":
		out = quoted(s.title)
		if s.container != "" {
			out += " " + s.container
			if s.volume != "" {
				out += " " + s.volume
			}
			if s.issue != "" {
				out += ", no. " + s.issue
			}
		}
		if s.year != "" {
			out += " (" + s.year + ")"
		}
		if l != "" {
			out += ": " + l
		} else if s.pages != "" {
			out += ": " + s.pages
		}
	case "web":
		out = quoted(s.title)
		if s.container != "" {
			out += " " + s.container + ","
		}
		if s.year != "" {
			out += " " + s.year + ","
		}
		if s.url != "" {
			out += " " + s.url
		}
		out = strings.TrimRight(out, ", ")
	default:
		out = s.title
		var pub []string
		if s.edition != "" {
			pub = append(pub, s.edition+" ed.")
		}
		if s.publisher != "" {
			pub = append(pub, s.publisher)
		}
		if s.year != "" {
			pub = append(pub, s.year)
		}
		if len(pub) > 0 {
			out += " (" + strings.Join(pub, ", ") + ")"
		}
		if l != "" {
			out += ", " + l
		}
	}
	if who != "" {
		out = who + ", " + out
	}
	return strings.TrimRight(out, ".,") + "."
}

func quoted(t string) string {
	if t == "" {
		return ""
	}
	return "“" + strings.TrimRight(t, ".,") + ",”"
}

// entry is one line of the list at the end, with the title in italics where a title is a whole work.
func entry(style string, s source, n int) []span {
	var out []span
	add := func(t string) {
		if t != "" {
			out = append(out, span{text: t})
		}
	}
	ital := func(t string) {
		if t != "" {
			out = append(out, span{text: t, italic: true})
		}
	}
	if style == styleNumbered {
		add(fmt.Sprintf("[%d] ", n))
	}
	who := s.lastFirst()
	if style == styleNumbered {
		who = strings.ReplaceAll(s.firstLast(), " and ", " and ")
	}
	if who != "" {
		add(strings.TrimRight(who, ".") + ".")
		add(" ")
	}
	if style != styleNumbered {
		add("(" + s.yearText() + "). ")
	}
	switch s.kind {
	case "article":
		add(strings.TrimRight(s.title, ".") + ". ")
		ital(s.container)
		if s.volume != "" {
			add(", " + s.volume)
			if s.issue != "" {
				add("(" + s.issue + ")")
			}
		}
		if s.pages != "" {
			add(", " + strings.ReplaceAll(s.pages, "-", "–"))
		}
		add(".")
	case "web":
		add(strings.TrimRight(s.title, ".") + ". ")
		if s.container != "" {
			ital(s.container)
			add(". ")
		}
		add(s.url)
		if s.at != "" {
			add(" (accessed " + s.at + ")")
		}
	case "interview":
		add(strings.TrimRight(s.title, ".") + " [Interview].")
	default:
		ital(strings.TrimRight(s.title, "."))
		add(".")
		if s.edition != "" {
			add(" " + s.edition + " ed.")
		}
		if s.publisher != "" {
			add(" " + s.publisher + ".")
		}
	}
	if style == styleNumbered && s.year != "" {
		add(" " + s.year + ".")
	}
	return out
}

// sortSources puts the sources in the order of a list at the end: by the first author, then the year, then the title.
func sortSources(list []source) {
	key := func(s source) string {
		if len(s.authors) > 0 {
			l, _, _ := splitName(s.authors[0])
			return strings.ToLower(l)
		}
		return strings.ToLower(s.title)
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := key(list[i]), key(list[j])
		if a != b {
			return a < b
		}
		if list[i].year != list[j].year {
			return list[i].year < list[j].year
		}
		return strings.ToLower(list[i].title) < strings.ToLower(list[j].title)
	})
}

// cleanStyle gives a style that exists. An empty or unknown name is the start.
func cleanStyle(v string) string {
	switch v {
	case styleNumbered, styleNotes:
		return v
	}
	return styleAuthorDate
}

// citeRef is a citation in the text, as it is saved.
type citeRef struct {
	source int64
	loc    string
}

// resolveCitations turns every citation in the blocks into text for the chosen style, numbers the footnotes,
// and returns the sources that were cited, in the order of the first citation. A citation of a source that is gone becomes "[?]".
func resolveCitations(blocks []block, sources map[int64]source, style string) ([]block, []source) {
	style = cleanStyle(style)
	var cited []source
	order := map[int64]int{}
	out := make([]block, len(blocks))
	for i, b := range blocks {
		out[i] = b
		if len(b.spans) == 0 {
			continue
		}
		spans := make([]span, len(b.spans))
		for j, sp := range b.spans {
			spans[j] = sp
			if sp.cite == nil {
				continue
			}
			src, ok := sources[sp.cite.source]
			if !ok {
				spans[j] = span{text: "[?]"}
				continue
			}
			first := false
			if _, seen := order[src.id]; !seen {
				cited = append(cited, src)
				order[src.id] = len(cited)
				first = true
			}
			switch style {
			case styleNotes:
				spans[j] = span{note: noteText(src, sp.cite.loc, !first)}
			default:
				spans[j] = span{text: citeText(style, src, sp.cite.loc, order[src.id])}
			}
		}
		out[i].spans = spans
	}
	return out, cited
}

// numberNotes gives every footnote in the blocks its number, from 1, in reading order. It returns the notes.
func numberNotes(blocks []block) []block {
	n := 0
	out := make([]block, len(blocks))
	for i, b := range blocks {
		out[i] = b
		if len(b.spans) == 0 {
			continue
		}
		spans := make([]span, len(b.spans))
		copy(spans, b.spans)
		for j := range spans {
			if spans[j].note != "" {
				n++
				spans[j].noteID = n
			}
		}
		out[i].spans = spans
	}
	return out
}

// bibliographyBlocks is the heading and the list at the end. The list is for the sources that were cited.
func bibliographyBlocks(cited []source, style string) []block {
	if len(cited) == 0 {
		return nil
	}
	style = cleanStyle(style)
	list := append([]source(nil), cited...)
	out := []block{{kind: "pagebreak"}, {kind: "h1", text: "Bibliography"}}
	if style == styleNumbered {
		for i, s := range list { // already in the order of the first citation
			out = append(out, block{kind: "p", spans: entry(style, s, i+1), flush: true})
		}
		return out
	}
	sortSources(list)
	for _, s := range list {
		out = append(out, block{kind: "p", spans: entry(style, s, 0), flush: true})
	}
	return out
}

// sourcesBlocks is the whole library as a list, for the Sources export.
func sourcesBlocks(all []source, style string) []block {
	if len(all) == 0 {
		return nil
	}
	list := append([]source(nil), all...)
	sortSources(list)
	style = cleanStyle(style)
	if style == styleNotes {
		style = styleAuthorDate
	}
	out := []block{{kind: "title", text: "Sources"}}
	for i, s := range list {
		out = append(out, block{kind: "p", spans: entry(style, s, i+1), flush: true})
	}
	return out
}

// withChapterNotes puts the footnotes at the end of each chapter, for the formats that have no real footnotes.
// A chapter ends where the next page break or the next chapter heading starts.
func withChapterNotes(blocks []block) []block {
	var out []block
	var pending []block
	flush := func() {
		if len(pending) == 0 {
			return
		}
		out = append(out, block{kind: "h2", text: "Notes"})
		out = append(out, pending...)
		pending = nil
	}
	for _, b := range blocks {
		if b.kind == "pagebreak" || b.kind == "h1" {
			flush()
		}
		out = append(out, b)
		for _, sp := range b.spans {
			if sp.noteID > 0 {
				pending = append(pending, block{kind: "note", text: sp.note, level: sp.noteID})
			}
		}
	}
	flush()
	return out
}
