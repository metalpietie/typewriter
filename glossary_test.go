package main

import "testing"

// The story bible lists the glossary in alphabetical order, and leaves the section out when it is empty.
func TestBibleHasGlossary(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	e := &Exporter{store: s}

	headings := func() []string {
		var h []string
		inGlossary := false
		for _, b := range e.bible() {
			if b.kind == "h1" {
				inGlossary = b.text == "Glossary"
				if inGlossary {
					h = append(h, "# Glossary")
				}
				continue
			}
			if inGlossary && b.kind == "h2" {
				h = append(h, b.text)
			}
		}
		return h
	}

	if len(headings()) != 0 {
		t.Fatal("an empty glossary must not make a section")
	}
	for _, d := range []string{
		`{"word":"zephyr-kin","meaning":"wind people"}`,
		`{"word":"Ketha","meaning":"a greeting","pronunciation":"KETH-ah"}`,
		`{"word":"alder","meaning":"a measure"}`,
	} {
		if _, err := s.Save(Doc{Kind: "term", Data: d}); err != nil {
			t.Fatal(err)
		}
	}
	got := headings()
	want := []string{"# Glossary", "alder", "Ketha", "zephyr-kin"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
