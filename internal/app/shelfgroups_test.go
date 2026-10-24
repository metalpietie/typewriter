package app

import (
	"database/sql"
	"strings"
	"testing"
)

func countKindIn(t *testing.T, s *Store, project, kind string) int {
	t.Helper()
	n := 0
	if err := s.withProject(project, false, func(d *sql.DB) error {
		return d.QueryRow(`SELECT COUNT(*) FROM docs WHERE kind = ?`, kind).Scan(&n)
	}); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestSplitShelfGivesEachBookItsOwnWorld(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	if err := s.RenameProject(defaultProject, "Practice novels"); err != nil {
		t.Fatal(err)
	}
	mustSave(t, s, "character", `{"name":"Mara"}`)
	mustSave(t, s, "setting", `{"key":"projectType","value":"nonfiction"}`)
	mustSave(t, s, "setting", `{"key":"dailyGoal","value":500}`)
	mustSave(t, s, "setting", `{"key":"calendar","value":{"months":[]}}`) // the world's own: it does not come along
	for i, title := range []string{"One", "Two", "Three"} {
		b := mustSave(t, s, "book", `{"title":"`+title+`","order":`+itoa(int64(i+1))+`}`)
		c := mustSave(t, s, "chapter", `{"book":`+itoa(b.ID)+`,"order":1,"title":"C"}`)
		mustSave(t, s, "scene", `{"book":`+itoa(b.ID)+`,"chapter":`+itoa(c.ID)+`,"order":1,"title":"S","body":"<p>words of `+title+`</p>"}`)
	}

	if err := s.SplitShelf(defaultProject); err != nil {
		t.Fatal(err)
	}
	shelves, err := s.Shelves()
	if err != nil || len(shelves) != 3 {
		t.Fatalf("three shelves (one per book) expected: %+v %v", shelves, err)
	}
	for i, title := range []string{"One", "Two", "Three"} {
		sh := shelves[i]
		if sh.Group != "Practice novels" {
			t.Fatalf("shelf %d must be in the group, got %q", i, sh.Group)
		}
		if sh.Name != title || len(sh.Books) != 1 || sh.Books[0].Title != title || sh.Books[0].Words != 3 {
			t.Fatalf("shelf %d must hold only %q with its text: %+v", i, title, sh)
		}
	}
	// the world stays with the first book; the new worlds start empty but work the same way
	if n := countKindIn(t, s, shelves[0].ID, "character"); n != 1 {
		t.Fatalf("the first book keeps the characters, got %d", n)
	}
	for _, sh := range shelves[1:] {
		if n := countKindIn(t, s, sh.ID, "character"); n != 0 {
			t.Fatalf("a new world starts without characters, got %d", n)
		}
		var keys []string
		s.withProject(sh.ID, false, func(d *sql.DB) error {
			r, err := d.Query(`SELECT json_extract(data, '$.key') FROM docs WHERE kind = 'setting' ORDER BY 1`)
			if err != nil {
				return err
			}
			defer r.Close()
			for r.Next() {
				var k string
				r.Scan(&k)
				keys = append(keys, k)
			}
			return nil
		})
		if strings.Join(keys, ",") != "dailyGoal,projectType" {
			t.Fatalf("a new world takes the type and the daily goal, not the calendar: %v", keys)
		}
	}
}

func TestShelfGroupNames(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	other, _ := s.AddProject("B")
	if err := s.SetProjectGroup(defaultProject, "  Practice  "); err != nil {
		t.Fatal(err)
	}
	if err := s.SetProjectGroup(other, "Practice"); err != nil {
		t.Fatal(err)
	}
	if err := s.RenameGroup("Practice", "Drafts"); err != nil {
		t.Fatal(err)
	}
	items, _ := s.Projects()
	for _, p := range items {
		if p.Group != "Drafts" {
			t.Fatalf("both projects must be in the renamed group: %+v", items)
		}
	}
	if err := s.RenameGroup("Nope", "X"); err == nil {
		t.Fatal("renaming a group that does not exist must fail")
	}
	if err := s.SetProjectGroup(other, strings.Repeat("x", 81)); err == nil {
		t.Fatal("a very long group name must be refused")
	}
	if err := s.SetProjectGroup(other, ""); err != nil {
		t.Fatal(err)
	}
	items, _ = s.Projects()
	if items[1].Group != "" {
		t.Fatal("an empty group takes the project out of the group")
	}
	// the group is kept in projects.json
	if r := loadRegistry(s.dir); r.Projects[0].Group != "Drafts" {
		t.Fatalf("the group must be saved: %+v", r.Projects)
	}
	// a new world in a group
	w, err := s.AddWorld("Drafts", "Third", defaultProject)
	if err != nil {
		t.Fatal(err)
	}
	if items, _ = s.Projects(); items[2].ID != w || items[2].Group != "Drafts" || items[2].Name != "Third" {
		t.Fatalf("the new world: %+v", items)
	}
}

func TestOrderGroup(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	a, _ := s.AddProject("A")
	solo, _ := s.AddProject("Solo") // not in the group: it keeps its place
	b, _ := s.AddProject("B")
	for _, id := range []string{a, b} {
		s.SetProjectGroup(id, "G")
	}
	if err := s.OrderGroup("G", []string{b, a}); err != nil {
		t.Fatal(err)
	}
	items, _ := s.Projects()
	got := []string{}
	for _, p := range items {
		got = append(got, p.Name)
	}
	if strings.Join(got, ",") != "My book,B,Solo,A" {
		t.Fatalf("the group keeps its places, in the new order: %v", got)
	}
	if err := s.OrderGroup("G", []string{a}); err == nil {
		t.Fatal("an order that leaves a book out must fail")
	}
	if err := s.OrderGroup("G", []string{a, solo}); err == nil {
		t.Fatal("an order with a project of another shelf must fail")
	}
}
