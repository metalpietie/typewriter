package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSignAndCheck(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	other, _, _ := ed25519.GenerateKey(rand.Reader)
	path := filepath.Join(t.TempDir(), "app.exe")
	os.WriteFile(path, []byte("the program"), 0o644)
	digest, err := fileDigest(path)
	if err != nil || len(digest) != 32 {
		t.Fatal(err)
	}
	sig := sign(priv, digest)
	if !checkSig(pub, digest, sig) {
		t.Fatal("a good signature must check out")
	}
	if checkSig(other, digest, sig) {
		t.Fatal("another key must not accept it")
	}
	os.WriteFile(path, []byte("the program, changed"), 0o644)
	changed, _ := fileDigest(path)
	if checkSig(pub, changed, sig) {
		t.Fatal("a changed file must not accept the old signature")
	}
	if checkSig(pub, digest, "not base64!") {
		t.Fatal("a broken signature must not check out")
	}
}

func TestReleaseFileNames(t *testing.T) {
	files := releaseFiles("1.11.0", true)
	names := []string{}
	for _, f := range files {
		names = append(names, f.name)
	}
	want := "typewriter-1.11.0-windows-x64.exe typewriter-1.11.0-windows-x64-installer.exe typewriter-1.11.0-linux-x64 typewriter-notes-1.11.0.apk"
	if strings.Join(names, " ") != want {
		t.Fatalf("names: %v", names)
	}
	if !signed(names[0]) || signed(names[1]) || !signed(names[2]) || signed(names[3]) {
		t.Fatal("only the files the updater installs are signed")
	}
}

func TestNotesTitle(t *testing.T) {
	if got := notesTitle("```\n## Version 1.11.0: updates\n### New\n", "1.11.0"); got != "Version 1.11.0: updates" {
		t.Fatalf("title: %q", got)
	}
	if got := notesTitle("no heading", "1.2.3"); got != "Version 1.2.3" {
		t.Fatalf("fallback: %q", got)
	}
}
