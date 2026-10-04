package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// Doc is one saved item. Data holds JSON text, so new fields never need a migration.
type Doc struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	Data      string `json:"data"`
	UpdatedAt string `json:"updatedAt"`
}

// Store is the service the frontend calls. It keeps all book data in one SQLite file.
type Store struct {
	mu   sync.RWMutex
	db   *sql.DB
	path string
	base string // the default folder. It always holds datadir.json, which says where the data is.
	dir  string // the data folder in use, holding every project
	reg  registry
	// problem is set when the chosen data folder could not be opened and the default one is used instead
	problem string
}

var kinds = map[string]bool{
	"book": true, "chapter": true, "character": true, "relationship": true,
	"location": true, "map": true, "pathpoint": true, "note": true, "event": true, "mapdoc": true, "scene": true, "version": true, "comment": true,
	"faction": true, "membership": true, "factionrel": true, "item": true, "itemlog": true, "plotignore": true, "wordday": true, "todo": true, "setting": true, "region": true,
}

// cascades lists, per kind, the JSON links that must go when that item is deleted.
var cascades = map[string][][2]string{
	"character": {{"relationship", "a"}, {"relationship", "b"}, {"pathpoint", "character"}, {"membership", "character"}, {"itemlog", "character"}},
	"faction":   {{"membership", "faction"}, {"factionrel", "a"}, {"factionrel", "b"}},
	"item":      {{"itemlog", "item"}},
	"map":       {{"pathpoint", "map"}, {"mapdoc", "map"}, {"region", "map"}},
	"book":      {{"chapter", "book"}, {"scene", "book"}},
	"chapter":   {{"scene", "chapter"}},
	"scene":     {{"version", "scene"}, {"comment", "scene"}},
}

// OpenStore opens the data of the project you used last.
func OpenStore() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return OpenStoreIn(filepath.Join(dir, "typewriter"))
}

// OpenStoreIn does the same with a chosen default folder (the tests use this).
func OpenStoreIn(base string) (*Store, error) {
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, err
	}
	dir, problem := resolveDataDir(base)
	s := &Store{base: base, dir: dir, problem: problem}
	s.reg = loadRegistry(dir)
	p, _ := s.reg.find(s.reg.Current)
	db, path, err := openProjectIn(dir, p)
	if err != nil {
		return nil, err
	}
	s.db, s.path = db, path
	return s, nil
}

// openDB opens one data file and makes sure the table exists.
func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS docs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		kind TEXT NOT NULL,
		data TEXT NOT NULL DEFAULT '{}',
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS docs_kind ON docs(kind);`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// d is the open database. It can change when you switch project, so always call this.
func (s *Store) d() *sql.DB {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db
}

// DataPath returns where the database file of the open project lives.
func (s *Store) DataPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.path
}

// List returns every item of one kind.
func (s *Store) List(kind string) ([]Doc, error) {
	if !kinds[kind] {
		return nil, fmt.Errorf("unknown kind %q", kind)
	}
	rows, err := s.d().Query(`SELECT id, kind, data, updated_at FROM docs WHERE kind = ? ORDER BY id`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Doc{}
	for rows.Next() {
		var d Doc
		if err := rows.Scan(&d.ID, &d.Kind, &d.Data, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// Save inserts the item when ID is 0, otherwise updates it. It returns the saved item.
func (s *Store) Save(d Doc) (Doc, error) {
	if !kinds[d.Kind] {
		return d, fmt.Errorf("unknown kind %q", d.Kind)
	}
	if d.ID == 0 {
		res, err := s.d().Exec(`INSERT INTO docs(kind, data) VALUES(?, ?)`, d.Kind, d.Data)
		if err != nil {
			return d, err
		}
		d.ID, _ = res.LastInsertId()
	} else if _, err := s.d().Exec(
		`UPDATE docs SET data = ?, updated_at = datetime('now') WHERE id = ? AND kind = ?`,
		d.Data, d.ID, d.Kind); err != nil {
		return d, err
	}
	err := s.d().QueryRow(`SELECT updated_at FROM docs WHERE id = ?`, d.ID).Scan(&d.UpdatedAt)
	return d, err
}

// ListBy returns the items of one kind whose JSON field equals id, newest first.
// History uses it to read the saved versions of one scene without loading all of them.
func (s *Store) ListBy(kind, field string, id int64) ([]Doc, error) {
	if !kinds[kind] {
		return nil, fmt.Errorf("unknown kind %q", kind)
	}
	for _, r := range field {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_') {
			return nil, fmt.Errorf("bad field %q", field)
		}
	}
	rows, err := s.d().Query(
		`SELECT id, kind, data, updated_at FROM docs WHERE kind = ? AND json_extract(data, ?) = ? ORDER BY id DESC`,
		kind, "$."+field, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Doc{}
	for rows.Next() {
		var d Doc
		if err := rows.Scan(&d.ID, &d.Kind, &d.Data, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// rowsWhere reads every item of a kind whose JSON field equals id.
func rowsWhere(tx *sql.Tx, kind, field string, id int64) ([]Doc, error) {
	rows, err := tx.Query(
		`SELECT id, kind, data, updated_at FROM docs WHERE kind = ? AND json_extract(data, ?) = ?`,
		kind, "$."+field, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Doc
	for rows.Next() {
		var d Doc
		if err := rows.Scan(&d.ID, &d.Kind, &d.Data, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// Delete removes an item and the items that point at it.
// It returns all of them as they were, so Restore can bring them back.
func (s *Store) Delete(kind string, id int64) ([]Doc, error) {
	if !kinds[kind] {
		return nil, fmt.Errorf("unknown kind %q", kind)
	}
	tx, err := s.d().Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var removed []Doc
	var self Doc
	err = tx.QueryRow(`SELECT id, kind, data, updated_at FROM docs WHERE id = ? AND kind = ?`, id, kind).
		Scan(&self.ID, &self.Kind, &self.Data, &self.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	removed = append(removed, self)

	for _, c := range cascades[kind] {
		found, err := rowsWhere(tx, c[0], c[1], id)
		if err != nil {
			return nil, err
		}
		removed = append(removed, found...)
		if _, err := tx.Exec(
			`DELETE FROM docs WHERE kind = ? AND json_extract(data, ?) = ?`,
			c[0], "$."+c[1], id); err != nil {
			return nil, err
		}
	}
	if kind == "book" || kind == "chapter" { // versions and comments of the scenes that just went
		const orphans = `kind IN ('version', 'comment') AND json_extract(data, '$.scene') NOT IN (SELECT id FROM docs WHERE kind = 'scene')`
		rows, err := tx.Query(`SELECT id, kind, data, updated_at FROM docs WHERE ` + orphans)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var d Doc
			if err := rows.Scan(&d.ID, &d.Kind, &d.Data, &d.UpdatedAt); err != nil {
				rows.Close()
				return nil, err
			}
			removed = append(removed, d)
		}
		rows.Close()
		if _, err := tx.Exec(`DELETE FROM docs WHERE ` + orphans); err != nil {
			return nil, err
		}
	}
	if kind == "map" { // places stay, but come off the deleted map
		found, err := rowsWhere(tx, "location", "map", id)
		if err != nil {
			return nil, err
		}
		removed = append(removed, found...)
		if _, err := tx.Exec(
			`UPDATE docs SET data = json_set(data, '$.map', json('null'), '$.x', json('null'), '$.y', json('null'))
			 WHERE kind = 'location' AND json_extract(data, '$.map') = ?`, id); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(`DELETE FROM docs WHERE id = ? AND kind = ?`, id, kind); err != nil {
		return nil, err
	}
	return removed, tx.Commit()
}

// Restore puts items back exactly as Delete returned them, with their old ids.
func (s *Store) Restore(docs []Doc) error {
	tx, err := s.d().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, d := range docs {
		if !kinds[d.Kind] {
			return fmt.Errorf("unknown kind %q", d.Kind)
		}
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO docs(id, kind, data, updated_at) VALUES(?, ?, ?, ?)`,
			d.ID, d.Kind, d.Data, d.UpdatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}
