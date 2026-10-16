package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The bookshelf is the first screen. Each project is a shelf, and the books of the project stand on it.
// It reads every project, also the ones that are not open, and it can move a book from one project to another.

// ShelfBook is one book as the bookshelf shows it.
type ShelfBook struct {
	ID       int64   `json:"id"`
	Title    string  `json:"title"`
	Order    float64 `json:"order"`
	Chapters int     `json:"chapters"`
	Words    int     `json:"words"`
	Seed     int     `json:"seed"`     // changes the generated cover (the refresh button)
	HasCover bool    `json:"hasCover"` // the book has a cover that the writer made or chose
	// The one progress bar of the book: the book goal, else the chapter goals together, else the scene goals together.
	GoalKind  string `json:"goalKind"` // "book", "chapter", "scene", or "" when there is no goal
	Goal      int    `json:"goal"`
	GoalWords int    `json:"goalWords"`
}

// Shelf is one project with its books.
type Shelf struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Current bool        `json:"current"`
	Problem string      `json:"problem"` // set when the data file of this project could not be read
	Books   []ShelfBook `json:"books"`
}

var tagRe = regexp.MustCompile(`<[^>]*>`)

// shelfWords counts the words of a scene. The text is HTML, or plain text in old data.
func shelfWords(body string) int {
	if strings.HasPrefix(strings.TrimSpace(body), "<") {
		body = tagRe.ReplaceAllString(body, " ")
	}
	return len(strings.Fields(body))
}

// withProject runs fn on the data file of a project. The open project uses the open file. Another one is opened and closed again.
func (s *Store) withProject(id string, fn func(*sql.DB) error) error {
	s.mu.RLock()
	reg, dir, cur, curDB := s.reg, s.dir, s.reg.Current, s.db
	s.mu.RUnlock()
	p, ok := reg.find(id)
	if !ok {
		return fmt.Errorf("that project does not exist")
	}
	if p.ID == cur {
		return fn(curDB)
	}
	d, _, err := openProjectIn(dir, p)
	if err != nil {
		return fmt.Errorf("could not open %q: %w", p.Name, err)
	}
	defer d.Close()
	return fn(d)
}

func readShelf(d *sql.DB) ([]ShelfBook, error) {
	rows, err := d.Query(`SELECT id, data FROM docs WHERE kind = 'book' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	books := []ShelfBook{}
	at := map[int64]int{}
	goals := map[int64]bookGoals{}
	for rows.Next() {
		var id int64
		var data string
		if err := rows.Scan(&id, &data); err != nil {
			rows.Close()
			return nil, err
		}
		var f struct {
			Title string  `json:"title"`
			Order float64 `json:"order"`
			Seed  int     `json:"shelfSeed"`
			Goal  any     `json:"goal"`
			Per   any     `json:"chapterGoal"` // the goal that a chapter has when it has none of its own
		}
		json.Unmarshal([]byte(data), &f) // a book with odd data still stands on the shelf
		at[id] = len(books)
		books = append(books, ShelfBook{ID: id, Title: f.Title, Order: f.Order, Seed: f.Seed})
		goals[id] = bookGoals{book: wholeNumber(f.Goal), perChapter: wholeNumber(f.Per)}
	}
	rows.Close()

	// the chapters, and the scenes with their words, for the chapter and scene goals
	chapterBook := map[int64]int64{}
	chapterGoal := map[int64]int{}
	cr, err := d.Query(`SELECT id, json_extract(data, '$.book'), CAST(json_extract(data, '$.goal') AS REAL) FROM docs WHERE kind = 'chapter'`)
	if err != nil {
		return nil, err
	}
	for cr.Next() {
		var id int64
		var b sql.NullInt64
		var goal sql.NullFloat64
		if err := cr.Scan(&id, &b, &goal); err != nil {
			cr.Close()
			return nil, err
		}
		if i, ok := at[b.Int64]; ok && b.Valid {
			books[i].Chapters++
			chapterBook[id] = b.Int64
			chapterGoal[id] = int(goal.Float64)
		}
	}
	cr.Close()

	chapterWords := map[int64]int{}
	sceneParts := map[int64][]Part{}
	sr, err := d.Query(`SELECT json_extract(data, '$.book'), json_extract(data, '$.chapter'), CAST(json_extract(data, '$.goal') AS REAL), json_extract(data, '$.body') FROM docs WHERE kind = 'scene'`)
	if err != nil {
		return nil, err
	}
	for sr.Next() {
		var b, ch sql.NullInt64
		var goal sql.NullFloat64
		var body sql.NullString
		if err := sr.Scan(&b, &ch, &goal, &body); err != nil {
			sr.Close()
			return nil, err
		}
		if i, ok := at[b.Int64]; ok && b.Valid {
			w := shelfWords(body.String)
			books[i].Words += w
			chapterWords[ch.Int64] += w
			sceneParts[b.Int64] = append(sceneParts[b.Int64], Part{Goal: int(goal.Float64), Words: w})
		}
	}
	sr.Close()

	chapterParts := map[int64][]Part{}
	for id, b := range chapterBook {
		g := chapterGoal[id]
		if g <= 0 {
			g = goals[b].perChapter
		}
		chapterParts[b] = append(chapterParts[b], Part{Goal: g, Words: chapterWords[id]})
	}
	for i := range books {
		id := books[i].ID
		sg := ShelfGoalFor(goals[id].book, chapterParts[id], sceneParts[id], books[i].Words)
		books[i].GoalKind, books[i].Goal, books[i].GoalWords = sg.Kind, sg.Goal, sg.Words
	}

	vr, err := d.Query(`SELECT DISTINCT json_extract(data, '$.book') FROM docs WHERE kind = 'cover'`)
	if err != nil {
		return nil, err
	}
	for vr.Next() {
		var b sql.NullInt64
		if err := vr.Scan(&b); err != nil {
			vr.Close()
			return nil, err
		}
		if i, ok := at[b.Int64]; ok && b.Valid {
			books[i].HasCover = true
		}
	}
	vr.Close()

	sort.SliceStable(books, func(i, j int) bool {
		if books[i].Order != books[j].Order {
			return books[i].Order < books[j].Order
		}
		return books[i].ID < books[j].ID
	})
	return books, nil
}

// Shelves lists every project with the books in it.
func (s *Store) Shelves() ([]Shelf, error) {
	s.mu.RLock()
	reg := s.reg
	s.mu.RUnlock()
	out := []Shelf{}
	for _, p := range reg.Projects {
		sh := Shelf{ID: p.ID, Name: p.Name, Current: p.ID == reg.Current, Books: []ShelfBook{}}
		err := s.withProject(p.ID, func(d *sql.DB) error {
			books, err := readShelf(d)
			sh.Books = books
			return err
		})
		if err != nil {
			sh.Problem = err.Error()
		}
		out = append(out, sh)
	}
	return out, nil
}

// BookCover returns the cover of a book as a picture (a data URL), or "" when the book has none.
func (s *Store) BookCover(project string, book int64) (string, error) {
	image := ""
	err := s.withProject(project, func(d *sql.DB) error {
		var data string
		err := d.QueryRow(`SELECT data FROM docs WHERE kind = 'cover' AND json_extract(data, '$.book') = ? ORDER BY id LIMIT 1`, book).Scan(&data)
		if err == sql.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		var f struct {
			Image string `json:"image"`
		}
		json.Unmarshal([]byte(data), &f)
		image = f.Image
		return nil
	})
	return image, err
}

// SetBookSeed saves the number that decides how the generated cover of a book looks.
func (s *Store) SetBookSeed(project string, book int64, seed int) error {
	return s.withProject(project, func(d *sql.DB) error {
		res, err := d.Exec(`UPDATE docs SET data = json_set(data, '$.shelfSeed', ?), updated_at = datetime('now') WHERE kind = 'book' AND id = ?`, seed, book)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("that book does not exist")
		}
		return nil
	})
}

// SetBookOrder puts the books of one shelf in the order of the list.
func (s *Store) SetBookOrder(project string, ids []int64) error {
	return s.withProject(project, func(d *sql.DB) error {
		tx, err := d.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		for i, id := range ids {
			if _, err := tx.Exec(`UPDATE docs SET data = json_set(data, '$.order', ?), updated_at = datetime('now') WHERE kind = 'book' AND id = ?`, i+1, id); err != nil {
				return err
			}
		}
		return tx.Commit()
	})
}

// AddBook puts a new, empty book on a shelf (a project that does not have to be open).
func (s *Store) AddBook(project, title string) (int64, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "New book"
	}
	var id int64
	err := s.withProject(project, func(d *sql.DB) error {
		var n int
		if err := d.QueryRow(`SELECT count(*) FROM docs WHERE kind = 'book'`).Scan(&n); err != nil {
			return err
		}
		data, _ := json.Marshal(map[string]any{"title": title, "synopsis": "", "order": n + 1})
		res, err := d.Exec(`INSERT INTO docs(kind, data) VALUES('book', ?)`, string(data))
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		return err
	})
	return id, err
}

// AddProject adds an empty shelf. Unlike NewProject, it does not open it.
func (s *Store) AddProject(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "New shelf"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("p%d", time.Now().UnixMilli())
	s.reg.Projects = append(s.reg.Projects, Project{ID: id, Name: name})
	return id, saveRegistry(s.dir, s.reg)
}

// ---- moving a book ----

// bundle is a book with everything that belongs to it.
type bundle struct {
	book                                    Doc
	passes, chapters, scenes, snaps, covers []Doc
	versions, comments, moods               []Doc
}

func readBundle(d *sql.DB, id int64, b *bundle) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = tx.QueryRow(`SELECT id, kind, data, updated_at FROM docs WHERE id = ? AND kind = 'book'`, id).Scan(&b.book.ID, &b.book.Kind, &b.book.Data, &b.book.UpdatedAt)
	if err == sql.ErrNoRows {
		return fmt.Errorf("that book does not exist")
	}
	if err != nil {
		return err
	}
	for _, g := range []struct {
		kind string
		to   *[]Doc
	}{{"pass", &b.passes}, {"chapter", &b.chapters}, {"scene", &b.scenes}, {"snapshot", &b.snaps}, {"cover", &b.covers}} {
		if *g.to, err = rowsWhere(tx, g.kind, "book", id); err != nil {
			return err
		}
	}
	for _, sc := range b.scenes {
		for _, g := range []struct {
			kind, field string
			to          *[]Doc
		}{{"version", "scene", &b.versions}, {"comment", "scene", &b.comments}, {"mood", "owner", &b.moods}} {
			found, err := rowsWhere(tx, g.kind, g.field, sc.ID)
			if err != nil {
				return err
			}
			*g.to = append(*g.to, found...)
		}
	}
	return tx.Commit()
}

// rewrite changes some fields of a saved item.
func rewrite(raw string, fn func(m map[string]any)) (string, error) {
	m := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return "", err
	}
	fn(m)
	out, err := json.Marshal(m)
	return string(out), err
}

func idOf(v any) int64 {
	f, _ := v.(float64)
	return int64(f)
}

// writeBundle puts a copy of the book in another project. Every item gets a new id, and the links between them are set again.
// Links to things that live in the old project (characters, places, arcs, sources) cannot come along. They are cleared.
func writeBundle(d *sql.DB, b *bundle) (int64, error) {
	tx, err := d.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	insert := func(d Doc, fn func(m map[string]any)) (int64, error) {
		data, err := rewrite(d.Data, fn)
		if err != nil {
			return 0, err
		}
		res, err := tx.Exec(`INSERT INTO docs(kind, data, updated_at) VALUES(?, ?, ?)`, d.Kind, data, d.UpdatedAt)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	var have int
	if err := tx.QueryRow(`SELECT count(*) FROM docs WHERE kind = 'book'`).Scan(&have); err != nil {
		return 0, err
	}
	book, err := insert(b.book, func(m map[string]any) { m["order"] = have + 1 })
	if err != nil {
		return 0, err
	}
	toBook := func(m map[string]any) { m["book"] = book }

	passes := map[int64]int64{}
	for _, p := range b.passes {
		if passes[p.ID], err = insert(p, toBook); err != nil {
			return 0, err
		}
	}
	chapters := map[int64]int64{}
	for _, c := range b.chapters {
		if chapters[c.ID], err = insert(c, toBook); err != nil {
			return 0, err
		}
	}
	scenes := map[int64]int64{}
	for _, sc := range b.scenes {
		if scenes[sc.ID], err = insert(sc, func(m map[string]any) {
			m["book"] = book
			m["chapter"] = chapters[idOf(m["chapter"])]
			m["pov"], m["location"], m["characters"] = nil, nil, []any{}
			if old, ok := m["passes"].([]any); ok {
				keep := []any{}
				for _, p := range old {
					if n, ok := passes[idOf(p)]; ok {
						keep = append(keep, n)
					}
				}
				m["passes"] = keep
			}
		}); err != nil {
			return 0, err
		}
	}
	for _, g := range []struct {
		docs  []Doc
		field string
	}{{b.versions, "scene"}, {b.comments, "scene"}, {b.moods, "owner"}} {
		for _, x := range g.docs {
			if _, err := insert(x, func(m map[string]any) { m[g.field] = scenes[idOf(m[g.field])] }); err != nil {
				return 0, err
			}
		}
	}
	for _, sn := range b.snaps { // a snapshot holds a copy of the scenes, with their old ids
		if _, err := insert(sn, func(m map[string]any) {
			m["book"] = book
			if list, ok := m["chapters"].([]any); ok {
				for _, c := range list {
					if cm, ok := c.(map[string]any); ok {
						cm["id"] = chapters[idOf(cm["id"])]
					}
				}
			}
			if list, ok := m["scenes"].([]any); ok {
				for _, c := range list {
					if cm, ok := c.(map[string]any); ok {
						cm["id"], cm["chapter"] = scenes[idOf(cm["id"])], chapters[idOf(cm["chapter"])]
					}
				}
			}
		}); err != nil {
			return 0, err
		}
	}
	for _, cv := range b.covers {
		if _, err := insert(cv, toBook); err != nil {
			return 0, err
		}
	}
	return book, tx.Commit()
}

// MoveBook moves a book, with its chapters, scenes, versions, comments, snapshots, passes and cover, to another project.
// The copy is made first. The old one is removed only after the copy is safe.
func (s *Store) MoveBook(from string, book int64, to string) (int64, error) {
	if from == to {
		return book, nil
	}
	var b bundle
	if err := s.withProject(from, func(d *sql.DB) error { return readBundle(d, book, &b) }); err != nil {
		return 0, err
	}
	var moved int64
	if err := s.withProject(to, func(d *sql.DB) (err error) { moved, err = writeBundle(d, &b); return }); err != nil {
		return 0, fmt.Errorf("nothing was moved: %w", err)
	}
	err := s.withProject(from, func(d *sql.DB) error {
		_, err := deleteIn(d, "book", book)
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("the book was copied, but the old copy could not be removed: %w", err)
	}
	return moved, nil
}

// ---- the progress bar of a book on the shelf ----

type bookGoals struct{ book, perChapter int }

// Part is a chapter or a scene: its goal (0 for none) and its words.
type Part struct{ Goal, Words int }

// ShelfGoal says what the progress bar of a book counts.
type ShelfGoal struct {
	Kind        string
	Goal, Words int
}

// ShelfGoalFor chooses the goal: the book goal, else the chapter goals added together, else the scene goals added together.
// Only the chapters (or scenes) that have a goal count, with their own words.
func ShelfGoalFor(book int, chapters, scenes []Part, bookWords int) ShelfGoal {
	if book > 0 {
		return ShelfGoal{"book", book, bookWords}
	}
	sum := func(list []Part) (goal, words int) {
		for _, p := range list {
			if p.Goal > 0 {
				goal += p.Goal
				words += p.Words
			}
		}
		return
	}
	if g, w := sum(chapters); g > 0 {
		return ShelfGoal{"chapter", g, w}
	}
	if g, w := sum(scenes); g > 0 {
		return ShelfGoal{"scene", g, w}
	}
	return ShelfGoal{}
}

// wholeNumber reads a goal that was saved as a number or as text.
func wholeNumber(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return int(f)
	}
	return 0
}
