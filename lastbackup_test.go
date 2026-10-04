package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The app remembers when you last saved a copy yourself, so it can remind you.
func TestLastBackupIsKeptPerProject(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)

	if got, err := s.LastBackup(); err != nil || got != "" {
		t.Fatalf("before any backup: got %q, %v", got, err)
	}

	at := time.Date(2026, 10, 4, 15, 4, 5, 0, time.FixedZone("test", 2*3600))
	s.noteBackup(at)
	want := "2026-10-04T13:04:05Z" // always stored as UTC
	if got, _ := s.LastBackup(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	// another project has its own note
	other, err := s.NewProject("Series two")
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := s.LastBackup(); got != "" {
		t.Fatalf("a new project has no backup yet, got %q", got)
	}
	if err := s.SwitchProject(defaultProject); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.LastBackup(); got != want {
		t.Fatalf("the first project forgot its backup: %q", got)
	}
	_ = other
}

func TestLastBackupSurvivesCleaning(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	if _, err := s.Save(Doc{Kind: "character", Data: `{"name":"A"}`}); err != nil {
		t.Fatal(err)
	}
	s.noteBackup(time.Now())

	// ten days of automatic copies make the app delete the old ones. The note must stay.
	for i := 0; i < 10; i++ {
		if err := s.autoBackupAt(time.Now().AddDate(0, 0, -i)); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := s.LastBackup(); got == "" {
		t.Fatal("the note was deleted with the old copies")
	}
}

func TestDamagedNoteCountsAsNone(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	dir := s.backupDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, lastBackupFile), []byte("not a time"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := s.LastBackup(); err != nil || got != "" {
		t.Fatalf("got %q, %v", got, err)
	}
}

// The note travels with the data when the data folder moves.
func TestLastBackupMovesWithTheDataFolder(t *testing.T) {
	base := t.TempDir()
	s, err := OpenStoreIn(base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	if _, err := s.Save(Doc{Kind: "character", Data: `{"name":"A"}`}); err != nil {
		t.Fatal(err)
	}
	s.noteBackup(time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC))

	target := filepath.Join(t.TempDir(), "elsewhere")
	if err := s.MoveDataFolder(target, "copy"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.LastBackup(); got != "2026-10-01T08:00:00Z" {
		t.Fatalf("after the move: got %q", got)
	}
}
