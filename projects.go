package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Each project (a book or a series) has its own data file.
// The first project keeps the old file name, typewriter.db, so nothing has to move.
const defaultProject = "default"

// Project is one entry in the list of projects.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type registry struct {
	Current  string    `json:"current"`
	Projects []Project `json:"projects"`
}

// ProjectItem is one project as the screen shows it.
type ProjectItem struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Size    int64  `json:"size"` // bytes on disk
	Current bool   `json:"current"`
}

func (r registry) find(id string) (Project, bool) {
	for _, p := range r.Projects {
		if p.ID == id {
			return p, true
		}
	}
	if len(r.Projects) > 0 {
		return r.Projects[0], false
	}
	return Project{ID: defaultProject, Name: "My book"}, false
}

func registryPath(dir string) string { return filepath.Join(dir, "projects.json") }

func loadRegistry(dir string) registry {
	var r registry
	if b, err := os.ReadFile(registryPath(dir)); err == nil {
		json.Unmarshal(b, &r)
	}
	if len(r.Projects) == 0 {
		r = registry{Current: defaultProject, Projects: []Project{{ID: defaultProject, Name: "My book"}}}
	}
	if _, ok := r.find(r.Current); !ok {
		r.Current = r.Projects[0].ID
	}
	return r
}

func saveRegistry(dir string, r registry) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	tmp := registryPath(dir) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, registryPath(dir))
}

func projectFile(dir, id string) string {
	if id == defaultProject {
		return filepath.Join(dir, "typewriter.db")
	}
	return filepath.Join(dir, "projects", id+".db")
}

func (s *Store) openProject(p Project) (*sql.DB, string, error) { return openProjectIn(s.dir, p) }

func openProjectIn(dir string, p Project) (*sql.DB, string, error) {
	path := projectFile(dir, p.ID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, "", err
	}
	db, err := openDB(path)
	return db, path, err
}

// Projects lists every project and says which one is open.
func (s *Store) Projects() ([]ProjectItem, error) {
	s.mu.RLock()
	reg, dir := s.reg, s.dir
	s.mu.RUnlock()
	out := []ProjectItem{}
	for _, p := range reg.Projects {
		var size int64
		if info, err := os.Stat(projectFile(dir, p.ID)); err == nil {
			size = info.Size()
		}
		out = append(out, ProjectItem{ID: p.ID, Name: p.Name, Size: size, Current: p.ID == reg.Current})
	}
	return out, nil
}

// switchTo opens another project. The screen must reload after this, so it shows the new data.
// It waits for a phone sync and for bookshelf jobs that use the open file, so none of them is cut off half way.
func (s *Store) switchTo(id string) error {
	s.syncMu.Lock()
	s.opMu.Lock()
	release := func() { s.opMu.Unlock(); s.syncMu.Unlock() }
	s.mu.Lock()
	p, ok := s.reg.find(id)
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("that project does not exist")
	}
	if p.ID == s.reg.Current {
		s.mu.Unlock()
		release()
		return nil
	}
	db, path, err := s.openProject(p)
	if err != nil {
		s.mu.Unlock()
		release()
		return fmt.Errorf("could not open that project: %w", err)
	}
	old := s.db
	s.db, s.path = db, path
	s.reg.Current = p.ID
	err = saveRegistry(s.dir, s.reg)
	s.mu.Unlock()
	old.Close() // this also folds the old file's write log into the file
	release()
	s.autoBackupAt(time.Now())
	return err
}

// SwitchProject opens another project.
func (s *Store) SwitchProject(id string) error { return s.switchTo(id) }

// NewProject makes an empty project and opens it.
func (s *Store) NewProject(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "New project"
	}
	s.mu.Lock()
	id := newProjectID(s.reg, s.dir)
	s.reg.Projects = append(append([]Project{}, s.reg.Projects...), Project{ID: id, Name: name}) // a new list: readers may still hold the old one
	err := saveRegistry(s.dir, s.reg)
	s.mu.Unlock()
	if err != nil {
		return "", err
	}
	return id, s.switchTo(id)
}

// RenameProject changes the name shown in the list.
func (s *Store) RenameProject(id, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("a project needs a name")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.reg.Projects {
		if s.reg.Projects[i].ID == id {
			list := append([]Project{}, s.reg.Projects...) // a new list: readers may still hold the old one
			list[i].Name = name
			s.reg.Projects = list
			return saveRegistry(s.dir, s.reg)
		}
	}
	return fmt.Errorf("that project does not exist")
}

// RemoveProject takes a project out of the list. The file is not deleted.
// It moves to the folder projects/removed, so you can still get it back by hand.
func (s *Store) RemoveProject(id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == s.reg.Current {
		return "", fmt.Errorf("open another project first, then remove this one")
	}
	idx := -1
	for i, p := range s.reg.Projects {
		if p.ID == id {
			idx = i
		}
	}
	if idx < 0 {
		return "", fmt.Errorf("that project does not exist")
	}
	src := projectFile(s.dir, id)
	dest := filepath.Join(s.dir, "projects", "removed", id+"-"+time.Now().Format("20060102-150405")+".db")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	if _, err := os.Stat(src); err == nil {
		if err := os.Rename(src, dest); err != nil {
			return "", fmt.Errorf("could not move the file, so nothing was removed: %w", err)
		}
	} else {
		dest = ""
	}
	list := append([]Project{}, s.reg.Projects[:idx]...) // a new list: readers may still hold the old one
	s.reg.Projects = append(list, s.reg.Projects[idx+1:]...)
	return dest, saveRegistry(s.dir, s.reg)
}

// newProjectID makes an id that no project has, and whose file does not exist yet.
func newProjectID(reg registry, dir string) string {
	base := fmt.Sprintf("p%d", time.Now().UnixMilli())
	id := base
	for n := 2; ; n++ {
		_, taken := reg.find(id)
		if _, err := os.Stat(projectFile(dir, id)); !taken && os.IsNotExist(err) {
			return id
		}
		id = fmt.Sprintf("%s-%d", base, n)
	}
}

// shut closes the open data file (the tests use this).
func (s *Store) shut() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Close()
}
