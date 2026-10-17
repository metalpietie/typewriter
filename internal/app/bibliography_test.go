package app

import (
	"strings"
	"testing"
)

func book() source {
	return sourceOf(rec{"id": 1.0, "type": "book", "title": "Atomic Habits", "authors": "Clear, James", "year": "2018", "publisher": "Avery"})
}
func article() source {
	return sourceOf(rec{"id": 2.0, "type": "article", "title": "How habits form", "authors": "Lally, Phillippa\nvan Jaarsveld, Cornelia H. M.\nWardle, Jane", "year": "2010",
		"container": "European Journal of Social Psychology", "volume": "40", "issue": "6", "pages": "998-1009"})
}
func web() source {
	return sourceOf(rec{"id": 3.0, "type": "web", "title": "Sleep and health", "authors": "World Health Organization,", "year": "2022", "container": "WHO", "url": "https://who.int/sleep", "accessed": "2026-01-02"})
}

func text(sp []span) string { return spansText(sp) }

func TestNames(t *testing.T) {
	cases := []struct{ in, last, first string }{
		{"Clear, James", "Clear", "James"}, {"James Clear", "Clear", "James"}, {"J. R. R. Tolkien", "Tolkien", "J. R. R."},
		{"Plato", "Plato", ""}, {"World Health Organization,", "World Health Organization", ""}, {"", "", ""},
	}
	for _, c := range cases {
		l, f, _ := splitName(c.in)
		if l != c.last || f != c.first {
			t.Errorf("%q: got %q %q", c.in, l, f)
		}
	}
	if initials("James Alan") != "J. A." || initials("Jean-Paul") != "J. P." || initials("") != "" {
		t.Error("initials")
	}
}

func TestAuthorShortForms(t *testing.T) {
	if book().lastNames() != "Clear" {
		t.Error("one author")
	}
	if article().lastNames() != "Lally et al." {
		t.Error("three authors: " + article().lastNames())
	}
	two := sourceOf(rec{"authors": "Smith, J\nJones, A"})
	if two.lastNames() != "Smith and Jones" {
		t.Error(two.lastNames())
	}
	if web().lastNames() != "World Health Organization" {
		t.Error(web().lastNames())
	}
}

func TestLocation(t *testing.T) {
	for in, want := range map[string]string{"": "", "45": "p. 45", "45-47": "pp. 45–47", "ch. 3": "ch. 3", " 7 ": "p. 7"} {
		if got := location(in); got != want {
			t.Errorf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestCitationText(t *testing.T) {
	if got := citeText(styleAuthorDate, book(), "45", 1); got != "(Clear, 2018, p. 45)" {
		t.Error(got)
	}
	if got := citeText(styleAuthorDate, book(), "", 1); got != "(Clear, 2018)" {
		t.Error(got)
	}
	nodate := sourceOf(rec{"title": "Untitled thing", "authors": "Doe, J"})
	if got := citeText(styleAuthorDate, nodate, "", 1); got != "(Doe, n.d.)" {
		t.Error(got)
	}
	noauthor := sourceOf(rec{"title": "A very long title for a report here", "year": "2001"})
	if got := citeText(styleAuthorDate, noauthor, "", 1); got != "(A very long title…, 2001)" {
		t.Error(got)
	}
	if got := citeText(styleNumbered, book(), "12", 3); got != "[3, p. 12]" {
		t.Error(got)
	}
	if got := citeText(styleNumbered, book(), "", 3); got != "[3]" {
		t.Error(got)
	}
}

func TestNotes(t *testing.T) {
	long := noteText(book(), "45", false)
	if long != "James Clear, Atomic Habits (Avery, 2018), 45." {
		t.Error(long)
	}
	short := noteText(book(), "45", true)
	if short != "Clear, Atomic Habits, 45." {
		t.Error(short)
	}
	a := noteText(article(), "", false)
	if !strings.Contains(a, "“How habits form,” European Journal of Social Psychology 40, no. 6 (2010): 998-1009") {
		t.Error(a)
	}
	w := noteText(web(), "", false)
	if !strings.HasSuffix(w, "https://who.int/sleep.") || !strings.HasPrefix(w, "World Health Organization, “Sleep and health,”") {
		t.Error(w)
	}
}

func TestEntries(t *testing.T) {
	if got := text(entry(styleAuthorDate, book(), 0)); got != "Clear, J. (2018). Atomic Habits. Avery." {
		t.Error(got)
	}
	e := entry(styleAuthorDate, book(), 0)
	italics := ""
	for _, s := range e {
		if s.italic {
			italics += s.text
		}
	}
	if italics != "Atomic Habits" {
		t.Error("the title of a book is in italics: " + italics)
	}
	a := text(entry(styleAuthorDate, article(), 0))
	if !strings.HasPrefix(a, "Lally, P., van Jaarsveld, C. H. M., & Wardle, J. (2010). How habits form. European Journal of Social Psychology, 40(6), 998–1009.") {
		t.Error(a)
	}
	if got := text(entry(styleAuthorDate, web(), 0)); got != "World Health Organization. (2022). Sleep and health. WHO. https://who.int/sleep (accessed 2026-01-02)" {
		t.Error(got)
	}
	n := text(entry(styleNumbered, book(), 4))
	if !strings.HasPrefix(n, "[4] James Clear.") || !strings.HasSuffix(n, "2018.") {
		t.Error(n)
	}
	other := sourceOf(rec{"title": "Odd thing"})
	if got := text(entry(styleAuthorDate, other, 0)); got != "(n.d.). Odd thing." {
		t.Error(got)
	}
}

func TestSortSources(t *testing.T) {
	list := []source{web(), book(), article(), sourceOf(rec{"id": 9.0, "title": "Aardvark", "authors": "Clear, James", "year": "2010"})}
	sortSources(list)
	var got []string
	for _, s := range list {
		got = append(got, s.title)
	}
	if strings.Join(got, "|") != "Aardvark|Atomic Habits|How habits form|Sleep and health" {
		t.Fatal(got)
	}
}

func citeBlocks() []block {
	return []block{
		{kind: "p", spans: []span{{text: "One."}, {cite: &citeRef{source: 1, loc: "45"}}, {text: " Two."}, {cite: &citeRef{source: 2}}}},
		{kind: "p", spans: []span{{text: "Three."}, {cite: &citeRef{source: 1}}, {cite: &citeRef{source: 99}}, {note: "A side remark."}}},
	}
}

func lib() map[int64]source { return map[int64]source{1: book(), 2: article(), 3: web()} }

func TestResolveAuthorDate(t *testing.T) {
	out, cited := resolveCitations(citeBlocks(), lib(), "")
	if text(out[0].spans) != "One.(Clear, 2018, p. 45) Two.(Lally et al., 2010)" {
		t.Error(text(out[0].spans))
	}
	if !strings.Contains(text(out[1].spans), "[?]") {
		t.Error("a source that is gone")
	}
	if len(cited) != 2 || cited[0].id != 1 || cited[1].id != 2 {
		t.Error("the sources that were cited, in order")
	}
	if citeBlocks()[0].spans[1].cite == nil {
		t.Error("the input is not changed")
	}
}

func TestResolveNumbered(t *testing.T) {
	out, _ := resolveCitations(citeBlocks(), lib(), "numbered")
	if text(out[0].spans) != "One.[1, p. 45] Two.[2]" || !strings.HasPrefix(text(out[1].spans), "Three.[1]") {
		t.Error(text(out[0].spans), text(out[1].spans))
	}
}

func TestResolveNotes(t *testing.T) {
	out, _ := resolveCitations(citeBlocks(), lib(), "notes")
	out = numberNotes(out)
	var notes []string
	var ids []int
	for _, b := range out {
		for _, sp := range b.spans {
			if sp.noteID > 0 {
				notes = append(notes, sp.note)
				ids = append(ids, sp.noteID)
			}
		}
	}
	if len(notes) != 4 { // two citations in the first block, one citation and the footnote in the second ([?] is not a note)
		t.Fatalf("%d notes: %v", len(notes), notes)
	}
	if notes[0] != "James Clear, Atomic Habits (Avery, 2018), 45." || notes[2] != "Clear, Atomic Habits." || notes[3] != "A side remark." {
		t.Error(notes)
	}
	for i, id := range ids {
		if id != i+1 {
			t.Fatalf("notes are numbered in order: %v", ids)
		}
	}
}

func TestBibliographyBlocks(t *testing.T) {
	if bibliographyBlocks(nil, "") != nil {
		t.Error("nothing cited, nothing printed")
	}
	cited := []source{web(), book()}
	b := bibliographyBlocks(cited, "authordate")
	if b[1].text != "Bibliography" || !strings.HasPrefix(text(b[2].spans), "Clear") || !strings.HasPrefix(text(b[3].spans), "World") {
		t.Error("sorted by author")
	}
	n := bibliographyBlocks(cited, "numbered")
	if !strings.HasPrefix(text(n[2].spans), "[1] ") || !strings.Contains(text(n[2].spans), "World Health Organization") {
		t.Error("numbered keeps the order of the citations")
	}
	if cited[0].id != 3 {
		t.Error("the input is not sorted in place")
	}
}

func TestChapterNotes(t *testing.T) {
	blocks := []block{
		{kind: "h1", text: "One"},
		{kind: "p", spans: []span{{text: "a"}, {note: "first", noteID: 1}}},
		{kind: "p", spans: []span{{text: "b"}, {note: "second", noteID: 2}}},
		{kind: "pagebreak"},
		{kind: "h1", text: "Two"},
		{kind: "p", spans: []span{{text: "c"}, {note: "third", noteID: 3}}},
	}
	out := withChapterNotes(blocks)
	var kinds []string
	for _, b := range out {
		kinds = append(kinds, b.kind)
	}
	if strings.Join(kinds, ",") != "h1,p,p,h2,note,note,pagebreak,h1,p,h2,note" {
		t.Fatal(kinds)
	}
	if out[4].level != 1 || out[5].text != "second" || out[10].level != 3 {
		t.Error("the notes keep their numbers")
	}
}

func TestCitationsFromHTML(t *testing.T) {
	body := `<p>Habits form slowly<span data-cite="1" data-loc="45" class="cite"></span>. A side note<span data-footnote="See the appendix &amp; more." class="fn"></span> here.</p><p><span data-cite="x"></span>Plain.</p>`
	bs := chapterBlocks(body)
	if len(bs) != 2 {
		t.Fatalf("%d blocks", len(bs))
	}
	var cite *citeRef
	note := ""
	for _, sp := range bs[0].spans {
		if sp.cite != nil {
			cite = sp.cite
		}
		if sp.note != "" {
			note = sp.note
		}
	}
	if cite == nil || cite.source != 1 || cite.loc != "45" {
		t.Errorf("cite %+v", cite)
	}
	if note != "See the appendix & more." {
		t.Errorf("note %q", note)
	}
	for _, sp := range bs[1].spans {
		if sp.cite != nil {
			t.Error("a citation with a bad id is dropped")
		}
	}
}

func fullBook() []block {
	bs := []block{{kind: "title", text: "Book"}, {kind: "pagebreak"}, {kind: "h1", text: "Chapter 1"}}
	bs = append(bs, citeBlocks()...)
	out, cited := resolveCitations(bs, lib(), "notes")
	out = numberNotes(out)
	return append(out, bibliographyBlocks(cited, "notes")...)
}

func TestNotesInEveryFormat(t *testing.T) {
	blocks := fullBook()
	d, err := buildDocx(blocks, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	all := zipText(t, d)
	for _, want := range []string{`<w:footnoteReference w:id="1"/>`, `<w:footnote w:id="4"`, "A side remark.", "footnotes.xml", `w:styleId="FootnoteReference"`, "Bibliography"} {
		if !strings.Contains(all, want) {
			t.Errorf("docx lacks %q", want)
		}
	}
	r := strings.Index(all, `<w:footnotes`)
	if r < 0 || !strings.Contains(all[r:], `</w:footnotes>`) {
		t.Error("footnotes part")
	}
	plain := withChapterNotes(blocks)
	e, err := buildEPUBWithCover(plain, "Book", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	ep := zipText(t, e)
	for _, want := range []string{`id="r1"`, `href="#n1"`, `id="n4"`, "A side remark.", "Notes", "Bibliography"} {
		if !strings.Contains(ep, want) {
			t.Errorf("epub lacks %q", want)
		}
	}
	md := string(buildMarkdown(plain))
	for _, want := range []string{"[^1]", "[^4]: A side remark.", "[^1]: James Clear, Atomic Habits (Avery, 2018), 45.", "Bibliography"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown lacks %q:\n%s", want, md)
		}
	}
	p, err := buildPDF(plain, ExportOptions{})
	if err != nil || len(p) < 500 {
		t.Fatalf("pdf: %v", err)
	}
}

func TestNoNotesNoFootnotePart(t *testing.T) {
	d, err := buildDocx([]block{{kind: "p", text: "plain"}}, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(zipText(t, d), "footnotes.xml") {
		t.Error("no footnotes, no part")
	}
}
