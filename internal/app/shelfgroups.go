package app

import (
	"database/sql"
	"fmt"
	"strings"
)

// Shelf groups: several projects on one shelf of the bookshelf. Each project keeps its own world (characters, places,
// notes, maps), so the books in a group share nothing. The group is only a name in projects.json; the data files do not change.

// settings that a new world in a group takes over from the shelf it comes from: how the project looks and works,
// not what is in its world (a calendar, for one, stays behind)
var groupSettings = []string{"projectType", "projectTools", "dailyGoal"}

const maxGroupName = 80

func cleanGroup(name string) (string, error) {
	name = strings.TrimSpace(name)
	if len([]rune(name)) > maxGroupName {
		return "", fmt.Errorf("the name of a group can be %d letters at most", maxGroupName)
	}
	return name, nil
}

// SetProjectGroup puts a project in a group (a shelf of its own worlds), or takes it out with "".
func (s *Store) SetProjectGroup(id, group string) error {
	group, err := cleanGroup(group)
	if err != nil {
		return err
	}
	return s.changeProjects(func(list []Project) (bool, error) {
		for i := range list {
			if list[i].ID == id {
				list[i].Group = group
				return true, nil
			}
		}
		return false, fmt.Errorf("that project does not exist")
	})
}

// RenameGroup gives every project of a group the new group name.
func (s *Store) RenameGroup(old, name string) error {
	name, err := cleanGroup(name)
	if err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("a group needs a name")
	}
	return s.changeProjects(func(list []Project) (bool, error) {
		found := false
		for i := range list {
			if list[i].Group == old && old != "" {
				list[i].Group, found = name, true
			}
		}
		if !found {
			return false, fmt.Errorf("that group does not exist")
		}
		return true, nil
	})
}

// changeProjects changes a copy of the project list and saves it (readers may still hold the old list).
func (s *Store) changeProjects(fn func([]Project) (bool, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := append([]Project{}, s.reg.Projects...)
	changed, err := fn(list)
	if err != nil || !changed {
		return err
	}
	s.reg.Projects = list
	return saveRegistry(s.dir, s.reg)
}

// AddWorld makes a new project in a group, for one book with a world of its own. It takes over the project type,
// the tools and the daily goal of the project like (another project of the group). It does not open the new project.
func (s *Store) AddWorld(group, name, like string) (string, error) {
	group, err := cleanGroup(group)
	if err != nil {
		return "", err
	}
	if group == "" {
		return "", fmt.Errorf("a world needs a group")
	}
	id, err := s.AddProject(name)
	if err != nil {
		return "", err
	}
	if err := s.SetProjectGroup(id, group); err != nil {
		return id, err
	}
	if like == "" {
		return id, nil
	}
	var rows []string
	if err := s.withProject(like, false, func(d *sql.DB) error {
		var err error
		rows, err = settingRows(d)
		return err
	}); err != nil {
		return id, nil // the new world simply starts with the defaults
	}
	return id, s.withProject(id, false, func(d *sql.DB) error {
		for _, data := range rows {
			if _, err := d.Exec(`INSERT INTO docs(kind, data) VALUES('setting', ?)`, data); err != nil {
				return err
			}
		}
		return nil
	})
}

// settingRows reads the settings of a project that a new world takes over (see groupSettings).
func settingRows(d *sql.DB) ([]string, error) {
	q := `SELECT data FROM docs WHERE kind = 'setting' AND json_extract(data, '$.key') IN (` + strings.TrimSuffix(strings.Repeat("?,", len(groupSettings)), ",") + `) ORDER BY id`
	args := make([]any, len(groupSettings))
	for i, k := range groupSettings {
		args[i] = k
	}
	r, err := d.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var out []string
	for r.Next() {
		var data string
		if err := r.Scan(&data); err != nil {
			return nil, err
		}
		out = append(out, data)
	}
	return out, r.Err()
}

// SplitShelf gives every book of a shelf its own world: the shelf becomes a group, and each book after the first moves
// to a new project of that group (as MoveBook does: the text comes along, the world stays with the first book).
// The project keeps the first book and is named after it. It stops at the first book that cannot move; the books that
// moved before stay moved.
func (s *Store) SplitShelf(id string) error {
	s.mu.RLock()
	p, ok := s.reg.find(id)
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("that project does not exist")
	}
	var books []ShelfBook
	if err := s.withProject(id, false, func(d *sql.DB) error {
		var err error
		books, err = readShelf(d)
		return err
	}); err != nil {
		return err
	}
	group := p.Group
	if group == "" {
		group = p.Name
	}
	if err := s.SetProjectGroup(id, group); err != nil {
		return err
	}
	if len(books) == 0 {
		return nil
	}
	if err := s.RenameProject(id, bookName(books[0].Title)); err != nil {
		return err
	}
	for _, b := range books[1:] {
		to, err := s.AddWorld(group, bookName(b.Title), id)
		if err != nil {
			return fmt.Errorf("could not make a world for %q: %w", bookName(b.Title), err)
		}
		if _, err := s.MoveBook(id, b.ID, to); err != nil {
			s.RemoveProject(to) // the empty world goes again; the book stays where it was
			return fmt.Errorf("could not move %q: %w", bookName(b.Title), err)
		}
	}
	return nil
}

func bookName(title string) string {
	if t := strings.TrimSpace(title); t != "" {
		return t
	}
	return "Untitled"
}

// OrderGroup puts the projects (worlds) of a group in this order on its shelf. ids must be exactly the projects of the
// group. Projects outside the group keep their places.
func (s *Store) OrderGroup(group string, ids []string) error {
	return s.changeProjects(func(list []Project) (bool, error) {
		var at []int // the places the group holds in the list
		in := map[string]Project{}
		for i, p := range list {
			if p.Group == group && group != "" {
				at = append(at, i)
				in[p.ID] = p
			}
		}
		if len(ids) != len(at) {
			return false, fmt.Errorf("the order must name every book of the group once")
		}
		seen := map[string]bool{}
		for i, id := range ids {
			p, ok := in[id]
			if !ok || seen[id] {
				return false, fmt.Errorf("the order must name every book of the group once")
			}
			seen[id] = true
			list[at[i]] = p
		}
		return true, nil
	})
}
