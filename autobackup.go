package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	autoKeep    = 7 // daily copies kept
	safetyKeep  = 5 // "before restore" copies kept
	autoPrefix  = "auto-"
	autoLayout  = "2006-01-02"
	autoChecked = 30 * time.Minute // how often a running app checks if a new day began
)

var autoName = regexp.MustCompile(`^auto-\d{4}-\d{2}-\d{2}\.db$`)

// AutoBackup is one daily copy, as shown on the Backup screen.
type AutoBackup struct {
	Name string `json:"name"`
	Date string `json:"date"` // YYYY-MM-DD
	Size int64  `json:"size"` // bytes
}

// backupDir is per project. The first project keeps the old folder name.
func (s *Store) backupDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.reg.Current == defaultProject {
		return filepath.Join(s.dir, "backups")
	}
	return filepath.Join(s.dir, "backups-"+s.reg.Current)
}

// autoBackupAt keeps one copy per day: the data as it was when that day first used the app.
// That way a mistake made today is still absent from today's copy.
func (s *Store) autoBackupAt(now time.Time) error {
	var items int
	if err := s.d().QueryRow(`SELECT count(*) FROM docs`).Scan(&items); err != nil {
		return err
	}
	if items == 0 {
		return nil // never replace good copies with an empty one
	}
	dir := s.backupDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, autoPrefix+now.Format(autoLayout)+".db")
	if _, err := os.Stat(path); err == nil {
		return nil // already have today's
	}
	// write to a temporary name first, so a half-written file is never mistaken for a backup
	tmp := path + ".tmp"
	if err := s.copyTo(tmp); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	prune(dir, autoPrefix, autoKeep)
	prune(dir, "before-restore-", safetyKeep)
	return nil
}

// prune deletes all but the newest keep files that start with prefix. Names sort by date.
func prune(dir, prefix string, keep int) {
	files, _ := filepath.Glob(filepath.Join(dir, prefix+"*.db"))
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	for i := keep; i < len(files); i++ {
		os.Remove(files[i])
	}
}

// StartAutoBackup makes today's copy now, then checks every half hour in case a new day began.
func (s *Store) StartAutoBackup() {
	run := func() {
		if err := s.autoBackupAt(time.Now()); err != nil {
			log.Println("automatic backup failed:", err)
		}
	}
	run()
	go func() {
		for range time.Tick(autoChecked) {
			run()
		}
	}()
}

// AutoBackups lists the daily copies, newest first.
func (s *Store) AutoBackups() ([]AutoBackup, error) {
	files, _ := filepath.Glob(filepath.Join(s.backupDir(), autoPrefix+"*.db"))
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	out := []AutoBackup{}
	for _, f := range files {
		name := filepath.Base(f)
		if !autoName.MatchString(name) {
			continue
		}
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		out = append(out, AutoBackup{Name: name, Date: strings.TrimSuffix(strings.TrimPrefix(name, autoPrefix), ".db"), Size: info.Size()})
	}
	return out, nil
}

// RestoreAuto replaces all data with one of the daily copies. Your current data is saved to a safety copy first.
func (s *Store) RestoreAuto(name string) (RestoreResult, error) {
	if !autoName.MatchString(name) { // the name comes from the screen: never let it point elsewhere
		return RestoreResult{}, fmt.Errorf("that is not an automatic backup")
	}
	return s.restoreFrom(filepath.Join(s.backupDir(), name))
}

// OpenBackupFolder shows the folder with the daily copies in the file manager.
func (s *Store) OpenBackupFolder() error {
	dir := s.backupDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", dir).Start()
	case "darwin":
		return exec.Command("open", dir).Start()
	default:
		return exec.Command("xdg-open", dir).Start()
	}
}
