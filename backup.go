package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// RestoreResult tells the screen what happened.
type RestoreResult struct {
	Restored   bool   `json:"restored"`   // false when you cancelled
	Items      int    `json:"items"`      // how many items the backup held
	SafetyCopy string `json:"safetyCopy"` // where your old data was saved first
}

// copyTo writes a consistent copy of the database to path.
// VACUUM INTO is safe to run while the app is using the data.
func (s *Store) copyTo(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	_, err := s.d().Exec(`VACUUM INTO ?`, path)
	return err
}

// Backup asks where to save, then writes a copy of all your data.
// It returns the saved path, or "" if you cancel.
func (s *Store) Backup() (string, error) {
	name := "typewriter-backup-" + time.Now().Format("2006-01-02") + ".db"
	path, err := application.Get().Dialog.SaveFile().
		SetFilename(name).
		AddFilter("Typewriter backup (*.db)", "*.db").
		PromptForSingleSelection()
	if err != nil || path == "" {
		return "", nil
	}
	if !strings.HasSuffix(strings.ToLower(path), ".db") {
		path += ".db"
	}
	if err := s.copyTo(path); err != nil {
		return "", fmt.Errorf("could not write the backup: %w", err)
	}
	s.noteBackup(time.Now())
	return path, nil
}

// RestoreBackup asks for a backup file and replaces all current data with it.
// Your current data is saved to a safety copy first.
func (s *Store) RestoreBackup() (RestoreResult, error) {
	path, err := application.Get().Dialog.OpenFile().
		SetTitle("Choose a Typewriter backup").
		AddFilter("Typewriter backup (*.db)", "*.db").
		PromptForSingleSelection()
	if err != nil || path == "" {
		return RestoreResult{}, nil
	}
	return s.restoreFrom(path)
}

// restoreFrom replaces all data with the contents of the backup file at path.
func (s *Store) restoreFrom(path string) (RestoreResult, error) {
	// Attach the backup and check it looks like ours before touching anything.
	if _, err := s.d().Exec(`ATTACH DATABASE ? AS bk`, path); err != nil {
		return RestoreResult{}, fmt.Errorf("could not open that file: %w", err)
	}
	defer s.d().Exec(`DETACH DATABASE bk`)

	var items int
	if err := s.d().QueryRow(`SELECT count(*) FROM bk.docs`).Scan(&items); err != nil {
		return RestoreResult{}, fmt.Errorf("that file is not a Typewriter backup")
	}
	var bad int
	if err := s.d().QueryRow(
		`SELECT count(*) FROM bk.docs WHERE json_valid(data) = 0 OR kind NOT IN (` + kindList() + `)`).Scan(&bad); err != nil || bad > 0 {
		return RestoreResult{}, fmt.Errorf("that backup holds data this version does not understand")
	}

	// Keep what you have now, in case you picked the wrong file.
	dir := s.backupDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return RestoreResult{}, err
	}
	safety := filepath.Join(dir, "before-restore-"+time.Now().Format("2006-01-02-150405")+".db")
	if err := s.copyTo(safety); err != nil {
		return RestoreResult{}, fmt.Errorf("could not save a safety copy, so nothing was changed: %w", err)
	}

	tx, err := s.d().Begin()
	if err != nil {
		return RestoreResult{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM main.docs`); err != nil {
		return RestoreResult{}, err
	}
	if _, err := tx.Exec(`INSERT INTO main.docs(id, kind, data, updated_at) SELECT id, kind, data, updated_at FROM bk.docs`); err != nil {
		return RestoreResult{}, err
	}
	// Keep new ids growing past the restored ones.
	if _, err := tx.Exec(`UPDATE sqlite_sequence SET seq = (SELECT COALESCE(MAX(id), 0) FROM main.docs) WHERE name = 'docs'`); err != nil {
		return RestoreResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return RestoreResult{}, err
	}
	return RestoreResult{Restored: true, Items: items, SafetyCopy: safety}, nil
}

// kindList is the allowed kinds as a SQL list, built from the same table the rest of the app uses.
func kindList() string {
	var q []string
	for k := range kinds {
		q = append(q, "'"+k+"'")
	}
	return strings.Join(q, ",")
}

// lastBackupFile sits in the backup folder of the project. It says when you last saved a copy yourself.
const lastBackupFile = "last-manual-backup.txt"

// noteBackup records that you just saved a copy with the Backup button.
func (s *Store) noteBackup(at time.Time) {
	dir := s.backupDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	// a note that cannot be written only means the reminder asks again, so the error is not passed on
	_ = os.WriteFile(filepath.Join(dir, lastBackupFile), []byte(at.UTC().Format(time.RFC3339)), 0o644)
}

// LastBackup tells when you last saved a copy with the Backup button, as a time such as
// "2026-10-04T15:04:05Z". It is "" when you never did. Each project has its own.
func (s *Store) LastBackup() (string, error) {
	b, err := os.ReadFile(filepath.Join(s.backupDir(), lastBackupFile))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(b)))
	if err != nil {
		return "", nil // a damaged note counts as no note
	}
	return t.UTC().Format(time.RFC3339), nil
}
