package main

import (
	"fmt"
	"testing"
)

// Mood board cards belong to a character, a place or a scene, and go with it.
func TestMoodCardsGoWithTheirOwner(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	save := func(kind, data string) int64 {
		d, err := s.Save(Doc{Kind: kind, Data: data})
		if err != nil {
			t.Fatal(err)
		}
		return d.ID
	}
	count := func(owner int64) int {
		got, err := s.ListBy("mood", "owner", owner)
		if err != nil {
			t.Fatal(err)
		}
		return len(got)
	}

	ch := save("character", `{"name":"Aldric"}`)
	other := save("character", `{"name":"Brenna"}`)
	place := save("location", `{"name":"Harbor"}`)
	book := save("book", `{"title":"One"}`)
	chap := save("chapter", fmt.Sprintf(`{"book":%d,"order":1}`, book))
	sc1 := save("scene", fmt.Sprintf(`{"chapter":%d,"book":%d,"order":1}`, chap, book))
	sc2 := save("scene", fmt.Sprintf(`{"chapter":%d,"book":%d,"order":2}`, chap, book))
	for _, owner := range []int64{ch, ch, other, place, sc1, sc2} {
		save("mood", fmt.Sprintf(`{"owner":%d,"text":"x","image":"data:image/webp;base64,AAAA","order":1}`, owner))
	}

	// a character goes: its cards go, and Undo brings them back
	removed, err := s.Delete("character", ch)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 3 || count(ch) != 0 || count(other) != 1 {
		t.Fatalf("character: removed %d, cards left %d, other has %d", len(removed), count(ch), count(other))
	}
	if err := s.Restore(removed); err != nil || count(ch) != 2 {
		t.Fatalf("undo: %v, cards %d", err, count(ch))
	}

	// a place goes
	if _, err := s.Delete("location", place); err != nil || count(place) != 0 {
		t.Fatalf("place: %v, cards %d", err, count(place))
	}

	// one scene goes
	if _, err := s.Delete("scene", sc1); err != nil || count(sc1) != 0 || count(sc2) != 1 {
		t.Fatalf("scene: %v, %d, %d", err, count(sc1), count(sc2))
	}

	// a whole chapter goes: the cards of its scenes must not stay behind
	if _, err := s.Delete("chapter", chap); err != nil {
		t.Fatal(err)
	}
	if count(sc2) != 0 {
		t.Fatalf("a card of a scene of a deleted chapter stayed: %d", count(sc2))
	}
	if count(other) != 1 {
		t.Fatal("the card of an unrelated character was removed")
	}
}
