package app

import (
	"os"
	"testing"
)

func count(t *testing.T, s *Store) int {
	t.Helper()
	l, err := s.List("character")
	if err != nil {
		t.Fatal(err)
	}
	return len(l)
}

func TestProjectsAreSeparate(t *testing.T) {
	dir := t.TempDir()
	s, err := OpenStoreIn(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	if _, err := s.Save(Doc{Kind: "character", Data: `{"name":"A"}`}); err != nil {
		t.Fatal(err)
	}
	id, err := s.NewProject("Series two")
	if err != nil {
		t.Fatal(err)
	}
	if count(t, s) != 0 {
		t.Fatal("new project should be empty")
	}
	s.Save(Doc{Kind: "character", Data: `{"name":"B"}`})
	s.Save(Doc{Kind: "character", Data: `{"name":"C"}`})
	if err := s.SwitchProject(defaultProject); err != nil {
		t.Fatal(err)
	}
	if n := count(t, s); n != 1 {
		t.Fatalf("first project should hold 1, got %d", n)
	}
	if err := s.SwitchProject(id); err != nil {
		t.Fatal(err)
	}
	if n := count(t, s); n != 2 {
		t.Fatalf("second project should hold 2, got %d", n)
	}

	// the choice is remembered after a restart
	s2, err := OpenStoreIn(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s2.shut)
	if n := count(t, s2); n != 2 {
		t.Fatalf("restart should open the second project, got %d", n)
	}
}

func TestProjectRenameRemove(t *testing.T) {
	dir := t.TempDir()
	s, _ := OpenStoreIn(dir)
	t.Cleanup(s.shut)
	id, _ := s.NewProject("  Second  ")
	list, _ := s.Projects()
	if len(list) != 2 || list[1].Name != "Second" || !list[1].Current {
		t.Fatalf("bad list: %+v", list)
	}
	if err := s.RenameProject(id, "Renamed"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RemoveProject(id); err == nil {
		t.Fatal("must not remove the open project")
	}
	s.SwitchProject(defaultProject)
	moved, err := s.RemoveProject(id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(moved); err != nil {
		t.Fatal("removed file should be kept:", err)
	}
	list, _ = s.Projects()
	if len(list) != 1 {
		t.Fatalf("expected 1 project, got %+v", list)
	}
}

func TestBackupFolderPerProject(t *testing.T) {
	s, _ := OpenStoreIn(t.TempDir())
	t.Cleanup(s.shut)
	a := s.backupDir()
	s.NewProject("Other")
	if s.backupDir() == a {
		t.Fatal("each project needs its own backup folder")
	}
}
