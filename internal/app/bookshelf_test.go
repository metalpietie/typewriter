package app

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
)

func mustSave(t *testing.T, s *Store, kind, data string) Doc {
	t.Helper()
	d, err := s.Save(Doc{Kind: kind, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func fieldOf(t *testing.T, d Doc, name string) any {
	t.Helper()
	m := map[string]any{}
	if err := json.Unmarshal([]byte(d.Data), &m); err != nil {
		t.Fatal(err)
	}
	return m[name]
}

func TestBookshelfMoveBook(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	other, err := s.AddProject("Series two") // a shelf that is not open
	if err != nil {
		t.Fatal(err)
	}
	if cur := s.reg.Current; cur != defaultProject {
		t.Fatalf("AddProject must not open the new shelf, open is %q", cur)
	}

	// a book with a chapter, two scenes, a version, a comment, a pass, a snapshot and a cover
	pov := mustSave(t, s, "character", `{"name":"Mara"}`)
	book := mustSave(t, s, "book", `{"title":"The clock","order":1}`)
	pass := mustSave(t, s, "pass", `{"book":`+itoa(book.ID)+`,"name":"Dialogue","order":1}`)
	chap := mustSave(t, s, "chapter", `{"book":`+itoa(book.ID)+`,"order":1,"title":"One"}`)
	sc1 := mustSave(t, s, "scene", `{"book":`+itoa(book.ID)+`,"chapter":`+itoa(chap.ID)+`,"order":1,"title":"A","body":"<p>one two three</p>","pov":`+itoa(pov.ID)+`,"characters":[`+itoa(pov.ID)+`],"passes":[`+itoa(pass.ID)+`]}`)
	sc2 := mustSave(t, s, "scene", `{"book":`+itoa(book.ID)+`,"chapter":`+itoa(chap.ID)+`,"order":2,"title":"B","body":"four five"}`)
	mustSave(t, s, "version", `{"scene":`+itoa(sc1.ID)+`,"body":"old"}`)
	mustSave(t, s, "comment", `{"scene":`+itoa(sc2.ID)+`,"text":"hm"}`)
	mustSave(t, s, "snapshot", `{"book":`+itoa(book.ID)+`,"name":"S","chapters":[{"id":`+itoa(chap.ID)+`}],"scenes":[{"id":`+itoa(sc1.ID)+`,"chapter":`+itoa(chap.ID)+`}]}`)
	mustSave(t, s, "cover", `{"book":`+itoa(book.ID)+`,"image":"data:image/jpeg;base64,AAAA","kind":"picture"}`)
	keep := mustSave(t, s, "book", `{"title":"Stays","order":2}`)

	shelves, err := s.Shelves()
	if err != nil || len(shelves) != 2 {
		t.Fatalf("shelves: %v %v", shelves, err)
	}
	b := shelves[0].Books[0]
	if b.Title != "The clock" || b.Chapters != 1 || b.Words != 5 || !b.HasCover {
		t.Fatalf("book on the shelf: %+v", b)
	}
	if len(shelves[1].Books) != 0 || shelves[1].Name != "Series two" {
		t.Fatalf("the other shelf: %+v", shelves[1])
	}
	if img, _ := s.BookCover(defaultProject, book.ID); img != "data:image/jpeg;base64,AAAA" {
		t.Fatalf("cover: %q", img)
	}

	newID, err := s.MoveBook(defaultProject, book.ID, other)
	if err != nil {
		t.Fatal(err)
	}

	// the old shelf only has the other book, and no leftovers
	left, _ := s.List("book")
	if len(left) != 1 || left[0].ID != keep.ID {
		t.Fatalf("old shelf: %+v", left)
	}
	for _, k := range []string{"chapter", "scene", "version", "comment", "snapshot", "cover", "pass"} {
		if l, _ := s.List(k); len(l) != 0 {
			t.Fatalf("%s left behind: %d", k, len(l))
		}
	}
	if l, _ := s.List("character"); len(l) != 1 {
		t.Fatal("characters of the old project must stay")
	}

	// the new shelf has everything, with the links set again
	if err := s.SwitchProject(other); err != nil {
		t.Fatal(err)
	}
	books, _ := s.List("book")
	chaps, _ := s.List("chapter")
	scenes, _ := s.List("scene")
	if len(books) != 1 || len(chaps) != 1 || len(scenes) != 2 {
		t.Fatalf("moved: %d %d %d", len(books), len(chaps), len(scenes))
	}
	if books[0].ID != newID {
		t.Fatalf("MoveBook returned %d, the book is %d", newID, books[0].ID)
	}
	nb, nc := float64(books[0].ID), float64(chaps[0].ID)
	if fieldOf(t, chaps[0], "book") != nb || fieldOf(t, scenes[0], "book") != nb || fieldOf(t, scenes[0], "chapter") != nc {
		t.Fatalf("links: %s / %s", chaps[0].Data, scenes[0].Data)
	}
	if fieldOf(t, scenes[0], "pov") != nil || len(fieldOf(t, scenes[0], "characters").([]any)) != 0 {
		t.Fatalf("links to the old project must be cleared: %s", scenes[0].Data)
	}
	passes, _ := s.List("pass")
	if got := fieldOf(t, scenes[0], "passes").([]any); len(got) != 1 || got[0] != float64(passes[0].ID) {
		t.Fatalf("pass link: %s", scenes[0].Data)
	}
	vers, _ := s.List("version")
	coms, _ := s.List("comment")
	if len(vers) != 1 || fieldOf(t, vers[0], "scene") != float64(scenes[0].ID) || len(coms) != 1 || fieldOf(t, coms[0], "scene") != float64(scenes[1].ID) {
		t.Fatalf("versions and comments: %v %v", vers, coms)
	}
	snaps, _ := s.List("snapshot")
	if len(snaps) != 1 || !strings.Contains(snaps[0].Data, itoa(scenes[0].ID)) || fieldOf(t, snaps[0], "book") != nb {
		t.Fatalf("snapshot: %v", snaps)
	}
	if fieldOf(t, books[0], "order") != float64(1) {
		t.Fatalf("order: %s", books[0].Data)
	}
}

func TestBookshelfSeedOrderAndAdd(t *testing.T) {
	s, _ := OpenStoreIn(t.TempDir())
	t.Cleanup(s.shut)
	other, _ := s.AddProject("Two")
	a, err := s.AddBook(other, "  First ") // into a shelf that is not open
	if err != nil {
		t.Fatal(err)
	}
	b, _ := s.AddBook(other, "")
	if err := s.SetBookSeed(other, a, 7); err != nil {
		t.Fatal(err)
	}
	if err := s.SetBookOrder(other, []int64{b, a}); err != nil {
		t.Fatal(err)
	}
	sh, _ := s.Shelves()
	books := sh[1].Books
	if len(books) != 2 || books[0].ID != b || books[0].Title != "New book" || books[1].Title != "First" || books[1].Seed != 7 {
		t.Fatalf("%+v", books)
	}
	if err := s.SetBookSeed(other, 9999, 1); err == nil {
		t.Fatal("a missing book must be an error")
	}
	if _, err := s.MoveBook(other, 9999, defaultProject); err == nil {
		t.Fatal("moving a missing book must be an error")
	}
}

func TestBookshelfGoals(t *testing.T) {
	s, _ := OpenStoreIn(t.TempDir())
	t.Cleanup(s.shut)
	add := func(title, bookData string, chapters [][2]string) int64 { // chapters: goal, then the scene "goal:words"
		b := mustSave(t, s, "book", `{"title":"`+title+`",`+bookData+`}`)
		for _, c := range chapters {
			ch := mustSave(t, s, "chapter", `{"book":`+itoa(b.ID)+`,"goal":`+c[0]+`}`)
			parts := strings.Split(c[1], ":")
			mustSave(t, s, "scene", `{"book":`+itoa(b.ID)+`,"chapter":`+itoa(ch.ID)+`,"goal":`+parts[0]+`,"body":"<p>`+strings.Repeat("w ", mustInt(parts[1]))+`</p>"}`)
		}
		return b.ID
	}
	bookGoal := add("A", `"goal":1000,"chapterGoal":10`, [][2]string{{"5", "7:20"}})
	chapGoal := add("B", `"goal":0`, [][2]string{{"100", "0:30"}, {"0", "0:50"}, {"200", "0:10"}})
	defGoal := add("C", `"goal":"","chapterGoal":"40"`, [][2]string{{"0", "0:60"}, {"0", "0:5"}}) // chapters use the default of the book
	sceneGoal := add("D", `"title2":1`, [][2]string{{"null", "50:20"}, {"0", "50:80"}, {"0", "0:9"}})
	none := add("E", `"x":1`, [][2]string{{"0", "0:5"}})

	sh, _ := s.Shelves()
	by := map[int64]ShelfBook{}
	for _, b := range sh[0].Books {
		by[b.ID] = b
	}
	check := func(id int64, kind string, goal, words int) {
		t.Helper()
		b := by[id]
		if b.GoalKind != kind || b.Goal != goal || b.GoalWords != words {
			t.Fatalf("%s: got %s %d/%d, want %s %d/%d", b.Title, b.GoalKind, b.GoalWords, b.Goal, kind, words, goal)
		}
	}
	check(bookGoal, "book", 1000, 20)
	check(chapGoal, "chapter", 300, 40) // chapters with a goal: 30 + 10 words of 100 + 200
	check(defGoal, "chapter", 80, 65)   // both chapters get the default goal of 40
	check(sceneGoal, "scene", 100, 100) // scenes with a goal: 20 + 80
	check(none, "", 0, 0)
}

func mustInt(s string) int { n, _ := strconv.Atoi(s); return n }

func TestBookshelfMoveFixesLinksInText(t *testing.T) {
	s, _ := OpenStoreIn(t.TempDir())
	t.Cleanup(s.shut)
	other, _ := s.AddProject("Two")
	src := mustSave(t, s, "source", `{"title":"Atomic Habits","authors":"Clear, James","year":"2018"}`)
	mara := mustSave(t, s, "character", `{"name":"Mara"}`)
	book := mustSave(t, s, "book", `{"title":"B","order":1}`)
	chap := mustSave(t, s, "chapter", `{"book":`+itoa(book.ID)+`,"order":1}`)
	sc := mustSave(t, s, "scene", `{"book":`+itoa(book.ID)+`,"chapter":`+itoa(chap.ID)+`,"order":1,"body":""}`)
	com := mustSave(t, s, "comment", `{"scene":`+itoa(sc.ID)+`,"text":"hm"}`)
	body := `<p>Habits <span class="cite" data-cite="` + itoa(src.ID) + `" data-loc="45"></span> matter to <span class="ref" data-ref="character:` + itoa(mara.ID) + `">Mara</span>. ` +
		`<span class="cmt" data-comment="` + itoa(com.ID) + `">Look</span> and <span class="cmt" data-comment="999">gone</span>.</p>`
	data, _ := json.Marshal(map[string]any{"book": book.ID, "chapter": chap.ID, "order": 1, "body": body})
	if _, err := s.Save(Doc{ID: sc.ID, Kind: "scene", Data: string(data)}); err != nil {
		t.Fatal(err)
	}
	mustSave(t, s, "version", `{"scene":`+itoa(sc.ID)+`,"body":"<p><span data-comment=\"`+itoa(com.ID)+`\">x</span></p>"}`)

	if _, err := s.MoveBook(defaultProject, book.ID, other); err != nil {
		t.Fatal(err)
	}
	s.SwitchProject(other)
	scenes, _ := s.List("scene")
	coms, _ := s.List("comment")
	vers, _ := s.List("version")
	got := fieldOf(t, scenes[0], "body").(string)
	if strings.Contains(got, "data-cite") || strings.Contains(got, "data-ref") || strings.Contains(got, `class="ref"`) {
		t.Fatalf("links to the old project must go: %s", got)
	}
	if !strings.Contains(got, `<span class="needs" data-needs="1">(Clear, 2018, p. 45)</span>`) {
		t.Fatalf("the citation must become text that needs a source: %s", got)
	}
	if !strings.Contains(got, ">Mara</span>") {
		t.Fatalf("the name must stay: %s", got)
	}
	if !strings.Contains(got, `data-comment="`+itoa(coms[0].ID)+`"`) || strings.Contains(got, `data-comment="999"`) {
		t.Fatalf("comment marks: %s", got)
	}
	if v := fieldOf(t, vers[0], "body").(string); !strings.Contains(v, `data-comment="`+itoa(coms[0].ID)+`"`) {
		t.Fatalf("version comment mark: %s", v)
	}
}

func TestBookshelfSnapshotOfDeletedScenes(t *testing.T) {
	s, _ := OpenStoreIn(t.TempDir())
	t.Cleanup(s.shut)
	other, _ := s.AddProject("Two")
	book := mustSave(t, s, "book", `{"title":"B","order":1}`)
	chap := mustSave(t, s, "chapter", `{"book":`+itoa(book.ID)+`,"order":1}`)
	// the snapshot knows two scenes and a chapter that no longer exist (ids 901, 902, 950)
	mustSave(t, s, "snapshot", `{"book":`+itoa(book.ID)+`,"chapters":[{"id":`+itoa(chap.ID)+`},{"id":950}],"scenes":[{"id":901,"chapter":950},{"id":902,"chapter":950}]}`)
	if _, err := s.MoveBook(defaultProject, book.ID, other); err != nil {
		t.Fatal(err)
	}
	s.SwitchProject(other)
	snaps, _ := s.List("snapshot")
	var snap struct {
		Chapters []struct{ ID int64 }          `json:"chapters"`
		Scenes   []struct{ ID, Chapter int64 } `json:"scenes"`
	}
	json.Unmarshal([]byte(snaps[0].Data), &snap)
	a, b := snap.Scenes[0], snap.Scenes[1]
	if a.ID >= 0 || b.ID >= 0 || a.ID == b.ID {
		t.Fatalf("deleted scenes need their own negative ids: %+v", snap.Scenes)
	}
	if a.Chapter != b.Chapter || a.Chapter >= 0 || snap.Chapters[1].ID != a.Chapter {
		t.Fatalf("the deleted chapter must keep one id everywhere: %+v", snap)
	}
}

func TestBookshelfMissingFileAndOrder(t *testing.T) {
	dir := t.TempDir()
	s, _ := OpenStoreIn(dir)
	t.Cleanup(s.shut)
	other, _ := s.AddProject("Two")
	if _, err := os.Stat(projectFile(dir, other)); err != nil {
		t.Fatal("AddProject must make the data file at once")
	}
	gone, _ := s.AddProject("Gone")
	os.Remove(projectFile(dir, gone))
	sh, _ := s.Shelves()
	if !strings.Contains(sh[2].Problem, "missing") {
		t.Fatalf("a missing file must show as a problem: %+v", sh[2])
	}
	if _, err := os.Stat(projectFile(dir, gone)); err == nil {
		t.Fatal("reading the shelf must not make an empty file")
	}
	// orders 1, 2, 3; the first moves away; a new book goes after the highest, not to a taken number
	a, _ := s.AddBook(other, "a")
	s.AddBook(other, "b")
	s.AddBook(other, "c")
	if _, err := s.MoveBook(other, a, defaultProject); err != nil {
		t.Fatal(err)
	}
	s.AddBook(other, "d")
	sh, _ = s.Shelves()
	var orders []float64
	for _, b := range sh[1].Books {
		orders = append(orders, b.Order)
	}
	if len(orders) != 3 || orders[0] != 2 || orders[1] != 3 || orders[2] != 4 {
		t.Fatalf("orders: %v", orders)
	}
	// two shelves made at once get different ids
	x, _ := s.AddProject("x")
	y, _ := s.AddProject("y")
	if x == y {
		t.Fatal("project ids must differ")
	}
}

func TestShelfWordsMatchTheApp(t *testing.T) {
	for in, want := range map[string]int{
		"<p>wo<em>rd</em> two</p>":                2,
		"<p>a&nbsp;b</p>":                         2,
		"<p>one</p><p>two</p>":                    2,
		"<p>line<br>break</p>":                    2,
		`<p>fact<span data-cite="1"></span>.</p>`: 1,
		"plain old text":                          3,
	} {
		if got := shelfWords(in); got != want {
			t.Errorf("%q: got %d, want %d", in, got, want)
		}
	}
}

func TestDeleteClearsLinksToGoneScenes(t *testing.T) {
	s, _ := OpenStoreIn(t.TempDir())
	t.Cleanup(s.shut)
	book := mustSave(t, s, "book", `{"title":"B"}`)
	chap := mustSave(t, s, "chapter", `{"book":`+itoa(book.ID)+`}`)
	sc := mustSave(t, s, "scene", `{"book":`+itoa(book.ID)+`,"chapter":`+itoa(chap.ID)+`}`)
	log := mustSave(t, s, "itemlog", `{"item":1,"scene":`+itoa(sc.ID)+`,"note":"found"}`)
	step := mustSave(t, s, "pathpoint", `{"character":1,"chapter":`+itoa(chap.ID)+`}`)
	mustSave(t, s, "itemlog", `{"item":1,"scene":null}`) // a log entry with no scene stays as it is
	removed, err := s.Delete("book", book.ID)
	if err != nil {
		t.Fatal(err)
	}
	logs, _ := s.List("itemlog")
	steps, _ := s.List("pathpoint")
	if len(logs) != 2 || len(steps) != 1 {
		t.Fatal("the log entries and path steps themselves must stay")
	}
	for _, d := range append(logs, steps...) {
		if (d.ID == log.ID && fieldOf(t, d, "scene") != nil) || (d.ID == step.ID && fieldOf(t, d, "chapter") != nil) {
			t.Fatalf("the link to the gone scene or chapter must be cleared: %s", d.Data)
		}
	}
	if err := s.Restore(removed); err != nil {
		t.Fatal(err)
	}
	logs, _ = s.List("itemlog")
	for _, d := range logs {
		if d.ID == log.ID && fieldOf(t, d, "scene") != float64(sc.ID) {
			t.Fatalf("undo must put the link back: %s", d.Data)
		}
	}
}
