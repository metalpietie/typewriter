package app

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
)

// rowsOf reads every item of one kind in a project, as maps.
func rowsOf(t *testing.T, s *Store, project, kind string) []map[string]any {
	t.Helper()
	var out []map[string]any
	if err := s.withProject(project, false, func(d *sql.DB) error {
		r, err := d.Query(`SELECT id, data FROM docs WHERE kind = ? ORDER BY id`, kind)
		if err != nil {
			return err
		}
		defer r.Close()
		for r.Next() {
			var id int64
			var data string
			r.Scan(&id, &data)
			m := map[string]any{}
			json.Unmarshal([]byte(data), &m)
			m["_id"] = float64(id)
			out = append(out, m)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestCopyItemsKeepsTheLinksBetweenCopies(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	other, _ := s.AddProject("Book two")

	mara := mustSave(t, s, "character", `{"name":"Mara"}`)
	tom := mustSave(t, s, "character", `{"name":"Tom"}`)
	left := mustSave(t, s, "character", `{"name":"Stays behind"}`)
	town := mustSave(t, s, "location", `{"name":"Brassford","parent":null,"map":7,"x":0.4,"y":0.5}`)
	shop := mustSave(t, s, "location", `{"name":"The shop","parent":`+itoa(town.ID)+`}`)
	guild := mustSave(t, s, "faction", `{"name":"Clockmakers","hq":`+itoa(shop.ID)+`}`)
	src := mustSave(t, s, "source", `{"title":"Atomic Habits","authors":"Clear, James","year":"2018"}`)
	other1 := mustSave(t, s, "source", `{"title":"Not copied","authors":"Smith, Ann","year":"2001"}`)
	mustSave(t, s, "relationship", `{"a":`+itoa(mara.ID)+`,"b":`+itoa(tom.ID)+`,"type":"friend"}`)
	mustSave(t, s, "relationship", `{"a":`+itoa(mara.ID)+`,"b":`+itoa(left.ID)+`,"type":"rival"}`) // one end stays: not copied
	mustSave(t, s, "membership", `{"faction":`+itoa(guild.ID)+`,"character":`+itoa(mara.ID)+`,"rank":"master"}`)
	mustSave(t, s, "mood", `{"owner":`+itoa(mara.ID)+`,"text":"brass"}`)
	note := mustSave(t, s, "note", `{"title":"The guild","body":"<p><span data-ref=\"character:`+itoa(mara.ID)+`\">Mara</span> and <span data-ref=\"character:`+itoa(left.ID)+`\">Rex</span> <span class=\"cite\" data-cite=\"`+itoa(src.ID)+`\" data-loc=\"p. 4\"></span><span class=\"cite\" data-cite=\"`+itoa(other1.ID)+`\"></span></p>"}`)

	pick := []ItemRef{{"character", mara.ID}, {"character", tom.ID}, {"location", town.ID}, {"location", shop.ID}, {"faction", guild.ID}, {"source", src.ID}, {"note", note.ID}}
	rep, err := s.CopyItems(defaultProject, other, pick)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Items != 7 || rep.Links != 3 { // friend, membership, mood picture
		t.Fatalf("report: %+v", rep)
	}

	// the originals are all still there
	if n := len(rowsOf(t, s, defaultProject, "character")); n != 3 {
		t.Fatalf("the originals must stay, got %d characters", n)
	}

	byName := func(kind, name string) map[string]any {
		for _, r := range rowsOf(t, s, other, kind) {
			if r["name"] == name || r["title"] == name {
				return r
			}
		}
		t.Fatalf("no copy of %s %q", kind, name)
		return nil
	}
	cm, ct := byName("character", "Mara"), byName("character", "Tom")
	ctown, cshop := byName("location", "Brassford"), byName("location", "The shop")
	cguild, csrc := byName("faction", "Clockmakers"), byName("source", "Atomic Habits")
	if cshop["parent"] != ctown["_id"] || ctown["map"] != nil || ctown["x"] != nil {
		t.Fatalf("a place keeps its parent (the copy) and leaves the map: %v %v", cshop, ctown)
	}
	if cguild["hq"] != cshop["_id"] {
		t.Fatalf("the seat of the faction must be the copied place: %v", cguild)
	}
	rels := rowsOf(t, s, other, "relationship")
	if len(rels) != 1 || rels[0]["a"] != cm["_id"] || rels[0]["b"] != ct["_id"] {
		t.Fatalf("only the relationship with both ends copied comes along, pointing at the copies: %v", rels)
	}
	if m := rowsOf(t, s, other, "membership"); len(m) != 1 || m[0]["character"] != cm["_id"] || m[0]["faction"] != cguild["_id"] {
		t.Fatalf("membership: %v", m)
	}
	if m := rowsOf(t, s, other, "mood"); len(m) != 1 || m[0]["owner"] != cm["_id"] {
		t.Fatalf("mood board picture: %v", m)
	}
	body := copyStr(byName("note", "The guild")["body"])
	if !strings.Contains(body, `data-ref="character:`+itoa(int64(cm["_id"].(float64)))+`"`) {
		t.Fatalf("the @ link to Mara must point at her copy: %s", body)
	}
	if strings.Contains(body, `character:`+itoa(left.ID)) || !strings.Contains(body, ">Rex<") {
		t.Fatalf("the @ link to a character that stayed lets go, the name stays: %s", body)
	}
	if !strings.Contains(body, `data-cite="`+itoa(int64(csrc["_id"].(float64)))+`" data-loc="p. 4"`) {
		t.Fatalf("a citation of a copied source must point at the copy: %s", body)
	}
	if !strings.Contains(body, `data-needs="1">(Smith, 2001)<`) {
		t.Fatalf("a citation of a source that stayed becomes plain text: %s", body)
	}
}

func TestCopyItemsRefuses(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	other, _ := s.AddProject("B")
	c := mustSave(t, s, "character", `{"name":"Mara"}`)
	b := mustSave(t, s, "book", `{"title":"A book"}`)
	if _, err := s.CopyItems(defaultProject, defaultProject, []ItemRef{{"character", c.ID}}); err == nil {
		t.Fatal("copying to the same shelf must fail")
	}
	if _, err := s.CopyItems(defaultProject, other, []ItemRef{{"book", b.ID}}); err == nil {
		t.Fatal("a book is not a world item")
	}
	if _, err := s.CopyItems(defaultProject, other, []ItemRef{{"character", 9999}}); err == nil {
		t.Fatal("a missing item must be reported")
	}
	if n := len(rowsOf(t, s, other, "character")); n != 0 {
		t.Fatalf("nothing may be written after a refusal, got %d", n)
	}
}

func TestCopyNotesPages(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	other, _ := s.AddProject("B")
	// the other shelf already has two pages: the copy goes after them
	for _, o := range []string{"1", "2"} {
		s.withProject(other, false, func(d *sql.DB) error {
			_, err := d.Exec(`INSERT INTO docs(kind, data) VALUES('draft', ?)`, `{"title":"There","body":"","order":`+o+`}`)
			return err
		})
	}
	mara := mustSave(t, s, "character", `{"name":"Mara"}`)
	page := mustSave(t, s, "draft", `{"title":"Ideas","order":5,"body":"<p><span data-ref=\"character:`+itoa(mara.ID)+`\">Mara</span> finds a key.</p>"}`)
	if _, err := s.CopyItems(defaultProject, other, []ItemRef{{"draft", page.ID}}); err != nil {
		t.Fatal(err)
	}
	pages := rowsOf(t, s, other, "draft")
	if len(pages) != 3 || pages[2]["title"] != "Ideas" || pages[2]["order"] != float64(3) {
		t.Fatalf("the copied page goes after the pages that are there: %v", pages)
	}
	if body := copyStr(pages[2]["body"]); strings.Contains(body, "data-ref") || !strings.Contains(body, "Mara finds a key") && !strings.Contains(body, ">Mara</span> finds a key") {
		t.Fatalf("the @ link to a character that was not copied lets go, the name stays: %s", body)
	}
	if n := len(rowsOf(t, s, defaultProject, "draft")); n != 1 {
		t.Fatal("the original page stays")
	}
}
