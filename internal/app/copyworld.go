package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

// Copying world items (characters, places, lore notes, factions, items, glossary words, sources, Notes pages) to another project,
// for example to a new world of a group. The originals stay as they are. Links between the copied items come along:
// relationships, faction members and faction ties when both ends are copied, the place a place lies in, the seat of a
// faction, @ links and citations in lore notes, and the pictures of the mood boards. A link to something that is not
// copied is let go (the text stays). Maps, the timeline and the places on a map are not copied.

// ItemRef names one item to copy.
type ItemRef struct {
	Kind string `json:"kind"`
	ID   int64  `json:"id"`
}

// CopyReport says what was copied.
type CopyReport struct {
	Items int `json:"items"` // the items that were picked
	Links int `json:"links"` // relationships, members, faction ties and mood board pictures that came along
}

var copyKinds = map[string]bool{"character": true, "location": true, "note": true, "faction": true, "item": true, "term": true, "source": true, "draft": true}

// the links between two copied items: the kind of the link row, and its two fields with the kind each points at
var copyLinks = []struct{ kind, a, aKind, b, bKind string }{
	{"relationship", "a", "character", "b", "character"},
	{"membership", "character", "character", "faction", "faction"},
	{"factionrel", "a", "faction", "b", "faction"},
}

var refValRe = regexp.MustCompile(`\sdata-ref="([a-z]+):(\d+)"`)

type copyRow struct {
	kind string
	id   int64
	data map[string]any
}

// CopyItems copies items from one project to another. Nothing in the first project changes.
func (s *Store) CopyItems(from, to string, refs []ItemRef) (CopyReport, error) {
	var rep CopyReport
	if from == to {
		return rep, fmt.Errorf("pick another shelf to copy to")
	}
	want := map[int64]string{}
	for _, r := range refs {
		if !copyKinds[r.Kind] {
			return rep, fmt.Errorf("%s cannot be copied", r.Kind)
		}
		want[r.ID] = r.Kind
	}
	if len(want) == 0 {
		return rep, nil
	}

	// read what goes: the items, the links between them, their mood board pictures, and the sources (for citations)
	var items, links []copyRow
	sources := map[int64]citeSource{}
	err := s.withProject(from, false, func(d *sql.DB) error {
		rows, err := d.Query(`SELECT id, kind, data FROM docs WHERE kind IN ('character','location','note','faction','item','term','source','draft','relationship','membership','factionrel','mood')`)
		if err != nil {
			return err
		}
		defer rows.Close()
		var all []copyRow
		for rows.Next() {
			var r copyRow
			var data string
			if err := rows.Scan(&r.id, &r.kind, &data); err != nil {
				return err
			}
			if json.Unmarshal([]byte(data), &r.data) != nil {
				continue
			}
			all = append(all, r)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		for _, r := range all {
			if r.kind == "source" {
				sources[r.id] = citeSource{Title: copyStr(r.data["title"]), Authors: copyStr(r.data["authors"]), Year: copyStr(r.data["year"])}
			}
			if k, ok := want[r.id]; ok && k == r.kind {
				items = append(items, r)
			}
		}
		for _, r := range all {
			switch r.kind {
			case "mood":
				if k := want[idOf(r.data["owner"])]; k == "character" || k == "location" {
					links = append(links, r)
				}
			default:
				for _, l := range copyLinks {
					if r.kind == l.kind && want[idOf(r.data[l.a])] == l.aKind && want[idOf(r.data[l.b])] == l.bKind {
						links = append(links, r)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return rep, err
	}
	if len(items) != len(want) {
		return rep, fmt.Errorf("%d of the picked items no longer exist", len(want)-len(items))
	}

	// write them: first the items (they get new numbers), then the fields and links that point at them
	err = s.withProject(to, false, func(d *sql.DB) error {
		tx, err := d.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		ids := map[int64]int64{} // old number -> new number
		// copied Notes pages go after the pages that are there
		var lastPage sql.NullFloat64
		if err := tx.QueryRow(`SELECT MAX(json_extract(data, '$.order')) FROM docs WHERE kind = 'draft'`).Scan(&lastPage); err != nil {
			return err
		}
		nextPage := int64(lastPage.Float64) + 1
		insert := func(kind string, data map[string]any) (int64, error) {
			b, _ := json.Marshal(data)
			res, err := tx.Exec(`INSERT INTO docs(kind, data) VALUES(?, ?)`, kind, string(b))
			if err != nil {
				return 0, err
			}
			return res.LastInsertId()
		}
		for _, r := range items {
			n, err := insert(r.kind, r.data)
			if err != nil {
				return err
			}
			ids[r.id] = n
		}
		remap := func(v any) any {
			if n, ok := ids[idOf(v)]; ok {
				return n
			}
			return nil
		}
		for _, r := range items {
			d := r.data
			switch r.kind {
			case "location":
				d["parent"] = remap(d["parent"])
				d["map"], d["x"], d["y"] = nil, nil, nil // maps are not copied
			case "faction":
				d["hq"] = remap(d["hq"])
			case "note":
				d["body"] = copyText(copyStr(d["body"]), want, ids, sources)
			case "draft":
				d["body"] = copyText(copyStr(d["body"]), want, ids, sources)
				d["order"] = nextPage
				nextPage++
			}
			b, _ := json.Marshal(d)
			if _, err := tx.Exec(`UPDATE docs SET data = ? WHERE id = ?`, string(b), ids[r.id]); err != nil {
				return err
			}
		}
		for _, r := range links {
			d := r.data
			if r.kind == "mood" {
				d["owner"] = remap(d["owner"])
			}
			for _, l := range copyLinks {
				if r.kind == l.kind {
					d[l.a], d[l.b] = remap(d[l.a]), remap(d[l.b])
				}
			}
			if _, err := insert(r.kind, d); err != nil {
				return err
			}
		}
		rep = CopyReport{Items: len(items), Links: len(links)}
		return tx.Commit()
	})
	if err != nil {
		return CopyReport{}, err
	}
	return rep, nil
}

// copyText fixes the links in the text of a copied lore note: an @ link to a copied item points at the copy, other
// @ links let go (the name stays). A citation of a copied source points at the copy; others become plain text marked
// "needs a source", as when a book moves. Comment marks are let go (notes have no comments in the new project).
func copyText(body string, kinds map[int64]string, ids map[int64]int64, sources map[int64]citeSource) string {
	if !strings.Contains(body, "data-") {
		return body
	}
	body = citeRe.ReplaceAllStringFunc(body, func(tag string) string {
		m := citeRe.FindStringSubmatch(tag)
		id, _ := strconv.ParseInt(m[1], 10, 64)
		if n, ok := ids[id]; ok && kinds[id] == "source" {
			return strings.Replace(tag, `data-cite="`+m[1]+`"`, `data-cite="`+strconv.FormatInt(n, 10)+`"`, 1)
		}
		loc := ""
		if l := citeLocRe.FindStringSubmatch(tag); l != nil {
			loc = html.UnescapeString(l[1])
		}
		src, ok := sources[id]
		return `<span class="needs" data-needs="1">` + html.EscapeString(movedCiteText(src, ok, loc)) + `</span>`
	})
	// an @ link stays only when what it points at was copied too
	keep := map[string]bool{}
	body = refValRe.ReplaceAllStringFunc(body, func(attr string) string {
		m := refValRe.FindStringSubmatch(attr)
		id, _ := strconv.ParseInt(m[2], 10, 64)
		if n, ok := ids[id]; ok && kinds[id] == m[1] {
			v := fmt.Sprintf(`%s:%d`, m[1], n)
			keep[v] = true
			return ` data-ref="` + v + `"`
		}
		return ""
	})
	body = commentRe.ReplaceAllString(body, "")
	if len(keep) == 0 {
		body = strings.ReplaceAll(body, ` class="ref"`, "")
	}
	return body
}

func copyStr(v any) string { // a text field of copied data ("" when it is missing)
	t, _ := v.(string)
	return t
}
