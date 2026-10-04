package main

import (
	"fmt"
	"testing"
)

func countKind(t *testing.T, s *Store, kind string) int {
	t.Helper()
	l, err := s.List(kind)
	if err != nil {
		t.Fatal(err)
	}
	return len(l)
}

// An arc link joins an arc and a scene. It must go when either one goes.
func TestArcLinksGoWithTheirArcOrScene(t *testing.T) {
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
	book := save("book", `{"title":"B"}`)
	chap := save("chapter", fmt.Sprintf(`{"book":%d,"order":1}`, book))
	s1 := save("scene", fmt.Sprintf(`{"chapter":%d,"book":%d,"order":1}`, chap, book))
	s2 := save("scene", fmt.Sprintf(`{"chapter":%d,"book":%d,"order":2}`, chap, book))
	arc := save("arc", `{"name":"The Crown"}`)
	save("arcscene", fmt.Sprintf(`{"arc":%d,"scene":%d,"beat":"start"}`, arc, s1))
	save("arcscene", fmt.Sprintf(`{"arc":%d,"scene":%d,"beat":""}`, arc, s2))

	// deleting one scene removes only its link
	if _, err := s.Delete("scene", s1); err != nil {
		t.Fatal(err)
	}
	if n := countKind(t, s, "arcscene"); n != 1 {
		t.Fatalf("after a scene is deleted, 1 link should stay, got %d", n)
	}
	// deleting the arc removes the rest of its links
	if _, err := s.Delete("arc", arc); err != nil {
		t.Fatal(err)
	}
	if n := countKind(t, s, "arcscene"); n != 0 {
		t.Fatalf("after the arc is deleted, no link should stay, got %d", n)
	}

	// deleting a book removes the links of all its scenes, but not the arc
	arc2 := save("arc", `{"name":"War"}`)
	s3 := save("scene", fmt.Sprintf(`{"chapter":%d,"book":%d,"order":3}`, chap, book))
	save("arcscene", fmt.Sprintf(`{"arc":%d,"scene":%d,"beat":""}`, arc2, s3))
	if _, err := s.Delete("book", book); err != nil {
		t.Fatal(err)
	}
	if n := countKind(t, s, "arcscene"); n != 0 {
		t.Fatalf("after the book is deleted, no link should stay, got %d", n)
	}
	if n := countKind(t, s, "arc"); n != 1 {
		t.Fatalf("an arc must stay when a book goes, got %d", n)
	}
}
