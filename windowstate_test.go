package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWindowStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "window.json")
	if loadWindowState(path).Maximised {
		t.Fatal("no file: the normal window")
	}
	saveWindowState(path, windowState{Maximised: true})
	if !loadWindowState(path).Maximised {
		t.Fatal("a saved maximised window must load as maximised")
	}
	saveWindowState(path, windowState{Maximised: false})
	if loadWindowState(path).Maximised {
		t.Fatal("a saved normal window must load as normal")
	}
	os.WriteFile(path, []byte("not json"), 0o644)
	if loadWindowState(path).Maximised {
		t.Fatal("a broken file: the normal window")
	}
}
