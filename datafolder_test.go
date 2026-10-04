package main

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func open(t *testing.T, base string) *Store {
	t.Helper()
	s, err := OpenStoreIn(base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	return s
}

func TestVersionMatchesBuildConfig(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("build", "config.yml"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^\s+version:\s*"([^"]+)"`).FindSubmatch(b)
	if m == nil {
		t.Fatal("no version in build/config.yml")
	}
	if string(m[1]) != Version {
		t.Fatalf("version.go says %s but build/config.yml says %s: change both together", Version, m[1])
	}
}

func TestMoveDataCopiesEverything(t *testing.T) {
	base, target := t.TempDir(), filepath.Join(t.TempDir(), "mybook")
	s := open(t, base)
	s.Save(Doc{Kind: "character", Data: `{"name":"A"}`})
	id, _ := s.NewProject("Second")
	s.Save(Doc{Kind: "character", Data: `{"name":"B"}`})
	s.Save(Doc{Kind: "character", Data: `{"name":"C"}`})
	if err := s.SwitchProject(defaultProject); err != nil {
		t.Fatal(err)
	}
	if info := s.DataFolder(); !info.IsDefault || info.Path != base {
		t.Fatalf("should start in the default folder: %+v", info)
	}

	if err := s.MoveDataFolder(target, "copy"); err != nil {
		t.Fatal(err)
	}
	if info := s.DataFolder(); info.IsDefault || info.Path != target {
		t.Fatalf("should now use the new folder: %+v", info)
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
	// the old files are still there
	if _, err := os.Stat(filepath.Join(base, "typewriter.db")); err != nil {
		t.Fatal("the old data must stay:", err)
	}
	// the daily backup of today is in the new folder
	if m, _ := filepath.Glob(filepath.Join(target, "backups*", "auto-*.db")); len(m) == 0 {
		t.Fatal("expected a backup in the new folder")
	}
	s.shut()

	// after a restart the new folder is used
	s2 := open(t, base)
	if info := s2.DataFolder(); info.Path != target {
		t.Fatalf("restart should use the new folder: %+v", info)
	}
	if n := count(t, s2); n != 2 {
		t.Fatalf("restart should show the second project's 2, got %d", n)
	}
}

func TestMoveDataNeverOverwrites(t *testing.T) {
	base, other := t.TempDir(), t.TempDir()
	// the other folder already holds a book with 3 characters
	o := open(t, other)
	for _, n := range []string{"x", "y", "z"} {
		o.Save(Doc{Kind: "character", Data: `{"name":"` + n + `"}`})
	}
	o.shut()

	s := open(t, base)
	s.Save(Doc{Kind: "character", Data: `{"name":"mine"}`})
	if err := s.MoveDataFolder(other, "copy"); err == nil {
		t.Fatal("copying over existing data must be refused")
	}
	if n := count(t, s); n != 1 {
		t.Fatalf("nothing should change after a refusal, got %d", n)
	}
	if err := s.MoveDataFolder(other, "use"); err != nil {
		t.Fatal(err)
	}
	if n := count(t, s); n != 3 {
		t.Fatalf("should show the 3 characters of the other folder, got %d", n)
	}
	// go back to the default folder: my old data is still there
	if err := s.MoveDataFolder(base, "use"); err != nil {
		t.Fatal(err)
	}
	if n := count(t, s); n != 1 {
		t.Fatalf("back in the default folder there should be 1, got %d", n)
	}
	if info := s.DataFolder(); !info.IsDefault {
		t.Fatalf("should be the default folder again: %+v", info)
	}
	if _, err := os.Stat(filepath.Join(base, pointerFile)); !os.IsNotExist(err) {
		t.Fatal("the default folder needs no pointer file")
	}
}

func TestMoveDataRefusesBadFolders(t *testing.T) {
	base := t.TempDir()
	s := open(t, base)
	if _, err := s.InspectDataFolder(filepath.Join(base, "inside")); err == nil {
		t.Fatal("a folder inside the data folder must be refused")
	}
	if err := s.MoveDataFolder(t.TempDir(), "use"); err == nil {
		t.Fatal("using an empty folder must be refused")
	}
	file := filepath.Join(t.TempDir(), "afile")
	os.WriteFile(file, []byte("x"), 0o644)
	if _, err := s.InspectDataFolder(filepath.Join(file, "sub")); err == nil {
		t.Fatal("a folder that cannot be made must be refused")
	}
	if c, err := s.InspectDataFolder(base); err != nil || !c.Same {
		t.Fatalf("the current folder should count as the same: %+v %v", c, err)
	}
}

func TestMissingFolderFallsBack(t *testing.T) {
	base := t.TempDir()
	s := open(t, base)
	s.Save(Doc{Kind: "character", Data: `{"name":"A"}`})
	s.shut()
	// point at a folder that cannot be made, as a drive that is unplugged
	file := filepath.Join(t.TempDir(), "afile")
	os.WriteFile(file, []byte("x"), 0o644)
	if err := writePointer(base, filepath.Join(file, "data")); err != nil {
		t.Fatal(err)
	}
	s2 := open(t, base)
	info := s2.DataFolder()
	if info.Problem == "" || !info.IsDefault {
		t.Fatalf("should fall back to the default folder with a warning: %+v", info)
	}
	if n := count(t, s2); n != 1 {
		t.Fatalf("default data should show, got %d", n)
	}
}
