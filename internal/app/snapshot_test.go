package app

import (
	"fmt"
	"testing"
)

// A snapshot is a copy of one book. It goes with its book, and a deleted scene or chapter does not touch it.
func TestSnapshotsGoWithTheirBook(t *testing.T) {
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
	b1 := save("book", `{"title":"One"}`)
	b2 := save("book", `{"title":"Two"}`)
	chap := save("chapter", fmt.Sprintf(`{"book":%d,"order":1}`, b1))
	sc := save("scene", fmt.Sprintf(`{"chapter":%d,"book":%d,"order":1}`, chap, b1))
	save("snapshot", fmt.Sprintf(`{"book":%d,"name":"A","scenes":[{"id":%d,"body":"<p>Hi</p>"}]}`, b1, sc))
	save("snapshot", fmt.Sprintf(`{"book":%d,"name":"B","scenes":[]}`, b1))
	save("snapshot", fmt.Sprintf(`{"book":%d,"name":"C","scenes":[]}`, b2))

	// the list for one book holds only its own snapshots
	got, err := s.ListBy("snapshot", "book", b1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("book one has 2 snapshots, got %d", len(got))
	}

	// a deleted scene and a deleted chapter leave the snapshots alone
	if _, err := s.Delete("scene", sc); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Delete("chapter", chap); err != nil {
		t.Fatal(err)
	}
	if n := countKind(t, s, "snapshot"); n != 3 {
		t.Fatalf("snapshots must stay when a scene or chapter goes, got %d", n)
	}

	// deleting a book removes its snapshots only, and Restore brings them back
	removed, err := s.Delete("book", b1)
	if err != nil {
		t.Fatal(err)
	}
	if n := countKind(t, s, "snapshot"); n != 1 {
		t.Fatalf("after the book is deleted, only the other book's snapshot should stay, got %d", n)
	}
	if err := s.Restore(removed); err != nil {
		t.Fatal(err)
	}
	if n := countKind(t, s, "snapshot"); n != 3 {
		t.Fatalf("Restore should bring the snapshots back, got %d", n)
	}
}

// A revision pass belongs to one book. It goes with that book, and Restore brings it back.
func TestPassesGoWithTheirBook(t *testing.T) {
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
	b1 := save("book", `{"title":"One"}`)
	b2 := save("book", `{"title":"Two"}`)
	save("pass", fmt.Sprintf(`{"book":%d,"name":"Plot","order":1}`, b1))
	save("pass", fmt.Sprintf(`{"book":%d,"name":"Dialogue","order":2}`, b1))
	save("pass", fmt.Sprintf(`{"book":%d,"name":"Plot","order":1}`, b2))

	removed, err := s.Delete("book", b1)
	if err != nil {
		t.Fatal(err)
	}
	if n := countKind(t, s, "pass"); n != 1 {
		t.Fatalf("only the other book's pass should stay, got %d", n)
	}
	if err := s.Restore(removed); err != nil {
		t.Fatal(err)
	}
	if n := countKind(t, s, "pass"); n != 3 {
		t.Fatalf("Restore should bring the passes back, got %d", n)
	}
}
