package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// windowState is how the window was when it closed, so the next start opens it the same way.
// It lives in window.json in the settings folder (next to projects.json of the default folder).
type windowState struct {
	Maximised bool `json:"maximised"`
}

func windowStateFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "typewriter", "window.json")
}

// loadWindowState reads window.json. A missing or broken file gives the normal window.
func loadWindowState(path string) windowState {
	var st windowState
	if path == "" {
		return st
	}
	if b, err := os.ReadFile(path); err == nil {
		json.Unmarshal(b, &st)
	}
	return st
}

// saveWindowState writes window.json. It is a nice-to-have: an error is ignored.
func saveWindowState(path string, st windowState) {
	if path == "" {
		return
	}
	b, _ := json.Marshal(st)
	tmp := path + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		os.Rename(tmp, path)
	}
}
