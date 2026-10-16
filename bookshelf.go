package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
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

var (
	tagRe = regexp.MustCompile(`<[^>]*>`)
	// the ends of blocks and line breaks count as a space, as in the app (db.svelte.ts plainText)
	blockEndRe = regexp.MustCompile(`(?i)</(p|h[1-6]|li|blockquote)>|<br\s*/?>|<hr\s*/?>`)
)

// shelfWords counts the words of a text the same way as the app does: a tag inside a word does not split it,
// and &nbsp; and the like are read as the characters they stand for.
func shelfWords(body string) int {
	if strings.HasPrefix(strings.TrimSpace(body), "<") {
		body = blockEndRe.ReplaceAllString(body, " ")
		body = tagRe.ReplaceAllString(body, "")
		body = html.UnescapeString(body)
	}
	return len(strings.Fields(body))
}

// withProject runs fn on the data file of a project. The open project uses the open file. Another one is opened and closed again.
// It holds opMu, so the open project cannot be switched (and its file closed) while fn uses it.
// When the file of a project that is not open is missing, it is an error, unless create is true (a book is being added to it).
func (s *Store) withProject(id string, create bool, fn func(*sql.DB) error) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()
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
	path := projectFile(dir, p.ID)
	if _, err := os.Stat(path); os.IsNotExist(err) && !create {
		return fmt.Errorf("its data file is missing (%s). If you moved it, put it back", path)
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
	chapterOwn := map[int64]int{} // the words of an old chapter that still holds its own text
	cr, err := d.Query(`SELECT id, json_extract(data, '$.book'), CAST(json_extract(data, '$.goal') AS REAL), json_extract(data, '$.body') FROM docs WHERE kind = 'chapter'`)
	if err != nil {
		return nil, err
	}
	for cr.Next() {
		var id int64
		var b sql.NullInt64
		var goal sql.NullFloat64
		var body sql.NullString
		if err := cr.Scan(&id, &b, &goal, &body); err != nil {
			cr.Close()
			return nil, err
		}
		if i, ok := at[b.Int64]; ok && b.Valid {
			books[i].Chapters++
			chapterBook[id] = b.Int64
			chapterGoal[id] = int(goal.Float64)
			chapterOwn[id] = shelfWords(body.String)
		}
	}
	cr.Close()

	chapterWords := map[int64]int{}
	hasScenes := map[int64]bool{}
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
			hasScenes[ch.Int64] = true
			sceneParts[b.Int64] = append(sceneParts[b.Int64], Part{Goal: int(goal.Float64), Words: w})
		}
	}
	sr.Close()

	chapterParts := map[int64][]Part{}
	for id, b := range chapterBook {
		words := chapterWords[id]
		if !hasScenes[id] { // as in the app: a chapter with no scenes counts its own text
			words = chapterOwn[id]
			books[at[b]].Words += words
		}
		g := chapterGoal[id]
		if g <= 0 {
			g = goals[b].perChapter
		}
		chapterParts[b] = append(chapterParts[b], Part{Goal: g, Words: words})
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
		err := s.withProject(p.ID, false, func(d *sql.DB) error {
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
	err := s.withProject(project, false, func(d *sql.DB) error {
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
	return s.withProject(project, false, func(d *sql.DB) error {
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
	return s.withProject(project, false, func(d *sql.DB) error {
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

// nextOrder is the order for a book that is put at the end of a shelf: one more than the highest.
func nextOrder(q interface {
	QueryRow(string, ...any) *sql.Row
}) (int, error) {
	var top sql.NullFloat64
	err := q.QueryRow(`SELECT MAX(CAST(json_extract(data, '$.order') AS REAL)) FROM docs WHERE kind = 'book'`).Scan(&top)
	return int(top.Float64) + 1, err
}

// AddBook puts a new, empty book on a shelf (a project that does not have to be open).
func (s *Store) AddBook(project, title string) (int64, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "New book"
	}
	var id int64
	err := s.withProject(project, true, func(d *sql.DB) error {
		order, err := nextOrder(d)
		if err != nil {
			return err
		}
		data, _ := json.Marshal(map[string]any{"title": title, "synopsis": "", "order": order})
		res, err := d.Exec(`INSERT INTO docs(kind, data) VALUES('book', ?)`, string(data))
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		return err
	})
	return id, err
}

// AddProject adds an empty shelf. Unlike NewProject, it does not open it. Its data file is made at once,
// so a file that goes missing later can be told apart from a shelf that is new.
func (s *Store) AddProject(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "New shelf"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := Project{ID: newProjectID(s.reg, s.dir), Name: name}
	d, _, err := openProjectIn(s.dir, p)
	if err != nil {
		return "", fmt.Errorf("could not make the data file of the new shelf: %w", err)
	}
	d.Close()
	s.reg.Projects = append(append([]Project{}, s.reg.Projects...), p) // a new list: readers may still hold the old one
	return p.ID, saveRegistry(s.dir, s.reg)
}

// ---- moving a book ----

// bundle is a book with everything that belongs to it.
type bundle struct {
	book                                    Doc
	passes, chapters, scenes, snaps, covers []Doc
	versions, comments, moods               []Doc
	sources                                 map[int64]citeSource // the sources of the old project, to write out the citations
}

// citeSource is what a citation needs to be written as plain text.
type citeSource struct{ Title, Authors, Year string }

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
	b.sources = map[int64]citeSource{}
	rows, err := tx.Query(`SELECT id, data FROM docs WHERE kind = 'source'`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var sid int64
		var data string
		if err := rows.Scan(&sid, &data); err != nil {
			rows.Close()
			return err
		}
		var f citeSource
		var raw struct {
			Title   string `json:"title"`
			Authors string `json:"authors"`
			Year    any    `json:"year"`
		}
		json.Unmarshal([]byte(data), &raw)
		f.Title, f.Authors = raw.Title, raw.Authors
		if raw.Year != nil {
			f.Year = strings.TrimSpace(fmt.Sprint(raw.Year))
		}
		b.sources[sid] = f
	}
	rows.Close()
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

var (
	citeRe    = regexp.MustCompile(`<span[^>]*\sdata-cite="(\d+)"[^>]*>(?:</span>)?`)
	citeLocRe = regexp.MustCompile(`\sdata-loc="([^"]*)"`)
	refAttrRe = regexp.MustCompile(`\s(?:data-ref="[^"]*"|class="ref")`)
	commentRe = regexp.MustCompile(`\sdata-comment="(\d+)"`)
)

// movedCiteText is a citation written out as plain text, such as (Clear, 2018, p. 45): who wrote it (or the title), the year, the place.
func movedCiteText(src citeSource, ok bool, loc string) string {
	who := ""
	if ok {
		first := strings.TrimSpace(strings.SplitN(src.Authors, "\n", 2)[0])
		who = strings.TrimSpace(strings.SplitN(first, ",", 2)[0])
		if who == "" {
			who = strings.TrimSpace(src.Title)
		}
	}
	if who == "" {
		who = "source"
	}
	parts := []string{who}
	if ok && src.Year != "" {
		parts = append(parts, src.Year)
	}
	if loc = strings.TrimSpace(loc); loc != "" {
		if _, err := strconv.Atoi(strings.SplitN(loc, "-", 2)[0]); err == nil {
			loc = "p. " + loc
		}
		parts = append(parts, loc)
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// moveText fixes the links inside the text of a moved scene, version or snapshot:
//   - a citation points at a source of the old project: it becomes plain text, marked "needs a source", so the writer can cite again
//   - an @ link points at a character or place of the old project: the link is taken off, the name stays
//   - a comment mark gets the new id of its comment; a mark whose comment did not come along is taken off
func moveText(body string, sources map[int64]citeSource, comments map[int64]int64) string {
	if !strings.Contains(body, "data-") {
		return body
	}
	body = citeRe.ReplaceAllStringFunc(body, func(tag string) string {
		m := citeRe.FindStringSubmatch(tag)
		id, _ := strconv.ParseInt(m[1], 10, 64)
		loc := ""
		if l := citeLocRe.FindStringSubmatch(tag); l != nil {
			loc = html.UnescapeString(l[1])
		}
		src, ok := sources[id]
		return `<span class="needs" data-needs="1">` + html.EscapeString(movedCiteText(src, ok, loc)) + `</span>`
	})
	body = refAttrRe.ReplaceAllString(body, "")
	body = commentRe.ReplaceAllStringFunc(body, func(attr string) string {
		id, _ := strconv.ParseInt(commentRe.FindStringSubmatch(attr)[1], 10, 64)
		if n, ok := comments[id]; ok {
			return fmt.Sprintf(` data-comment="%d"`, n)
		}
		return ""
	})
	return body
}

// writeBundle puts a copy of the book in another project. Every item gets a new id, and the links between them are set again.
// Links to things that live in the old project (characters, places, arcs, sources) cannot come along. They are cleared,
// also inside the text (see moveText). It returns the id of the new book.
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
	order, err := nextOrder(tx)
	if err != nil {
		return 0, err
	}
	book, err := insert(b.book, func(m map[string]any) { m["order"] = order })
	if err != nil {
		return 0, err
	}
	toBook := func(m map[string]any) { m["book"] = book }
	noCast := func(m map[string]any) { // characters and places live in the old project
		for _, k := range []string{"pov", "location"} {
			if _, ok := m[k]; ok {
				m[k] = nil
			}
		}
		if _, ok := m["characters"]; ok {
			m["characters"] = []any{}
		}
	}

	passes := map[int64]int64{}
	for _, p := range b.passes {
		if passes[p.ID], err = insert(p, toBook); err != nil {
			return 0, err
		}
	}
	chapters := map[int64]int64{}
	for _, c := range b.chapters {
		if chapters[c.ID], err = insert(c, func(m map[string]any) { m["book"] = book; noCast(m) }); err != nil {
			return 0, err
		}
	}
	scenes := map[int64]int64{}
	for _, sc := range b.scenes {
		if scenes[sc.ID], err = insert(sc, func(m map[string]any) {
			m["book"] = book
			m["chapter"] = chapters[idOf(m["chapter"])]
			noCast(m)
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
	versions := map[int64]int64{}
	for _, v := range b.versions {
		if versions[v.ID], err = insert(v, func(m map[string]any) { m["scene"] = scenes[idOf(m["scene"])] }); err != nil {
			return 0, err
		}
	}
	comments := map[int64]int64{}
	for _, c := range b.comments {
		if comments[c.ID], err = insert(c, func(m map[string]any) { m["scene"] = scenes[idOf(m["scene"])] }); err != nil {
			return 0, err
		}
	}
	for _, x := range b.moods {
		if _, err := insert(x, func(m map[string]any) { m["owner"] = scenes[idOf(m["owner"])] }); err != nil {
			return 0, err
		}
	}

	// the text of the scenes and versions: only now are the new ids of the comments known
	fixBody := func(m map[string]any) {
		if body, ok := m["body"].(string); ok {
			m["body"] = moveText(body, b.sources, comments)
		}
	}
	for _, group := range []struct {
		docs []Doc
		ids  map[int64]int64
	}{{b.scenes, scenes}, {b.chapters, chapters}, {b.versions, versions}} {
		for _, x := range group.docs {
			// the item as it was saved above, with its other links already set; only the body changes here
			var now string
			if err := tx.QueryRow(`SELECT data FROM docs WHERE id = ?`, group.ids[x.ID]).Scan(&now); err != nil {
				return 0, err
			}
			fixed, err := rewrite(now, fixBody)
			if err != nil {
				return 0, err
			}
			if _, err := tx.Exec(`UPDATE docs SET data = ? WHERE id = ?`, fixed, group.ids[x.ID]); err != nil {
				return 0, err
			}
		}
	}

	// A snapshot holds a copy of the scenes, with their old ids. A scene or chapter that was deleted since the snapshot
	// has no new id: it gets a negative one of its own, so two of them are never taken for the same.
	gone := map[string]int64{}
	newID := func(kind string, ids map[int64]int64, old int64) int64 {
		if n, ok := ids[old]; ok {
			return n
		}
		key := kind + strconv.FormatInt(old, 10)
		if n, ok := gone[key]; ok {
			return n
		}
		gone[key] = -int64(len(gone) + 1)
		return gone[key]
	}
	for _, sn := range b.snaps {
		if _, err := insert(sn, func(m map[string]any) {
			m["book"] = book
			if list, ok := m["chapters"].([]any); ok {
				for _, c := range list {
					if cm, ok := c.(map[string]any); ok {
						cm["id"] = newID("c", chapters, idOf(cm["id"]))
					}
				}
			}
			if list, ok := m["scenes"].([]any); ok {
				for _, c := range list {
					if cm, ok := c.(map[string]any); ok {
						cm["id"], cm["chapter"] = newID("s", scenes, idOf(cm["id"])), newID("c", chapters, idOf(cm["chapter"]))
						if body, ok := cm["body"].(string); ok {
							cm["body"] = moveText(body, b.sources, comments)
						}
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
// The copy is made first. The old one is removed only after the copy is safe. If the old one cannot be removed,
// the copy is taken out again, so the book is never in two places.
func (s *Store) MoveBook(from string, book int64, to string) (int64, error) {
	if from == to {
		return book, nil
	}
	var b bundle
	if err := s.withProject(from, false, func(d *sql.DB) error { return readBundle(d, book, &b) }); err != nil {
		return 0, err
	}
	var moved int64
	if err := s.withProject(to, true, func(d *sql.DB) (err error) { moved, err = writeBundle(d, &b); return }); err != nil {
		return 0, fmt.Errorf("nothing was moved: %w", err)
	}
	err := s.withProject(from, false, func(d *sql.DB) error {
		_, err := deleteIn(d, "book", book)
		return err
	})
	if err != nil {
		undo := s.withProject(to, false, func(d *sql.DB) error {
			_, err := deleteIn(d, "book", moved)
			return err
		})
		if undo != nil {
			return 0, fmt.Errorf("the book was copied, but the old copy could not be removed (%v), and the new copy could not be taken out again (%v). It is now on both shelves", err, undo)
		}
		return 0, fmt.Errorf("nothing was moved: the old copy could not be removed: %w", err)
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
