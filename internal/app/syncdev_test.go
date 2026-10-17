package app

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// A real server for trying the phone app (or its live test) on this computer. It does nothing in a normal test run.
// Run:  TYPEWRITER_PHONE_DEV=C:\path\dev.json go test -run TestDevServerForPhoneApp -timeout 10m ./internal/app
// It writes the address, the fingerprint and a pairing code to that file, and stops when a file with the same name and ".stop" appears.
func TestDevServerForPhoneApp(t *testing.T) {
	out := os.Getenv("TYPEWRITER_PHONE_DEV")
	if out == "" {
		t.Skip("set TYPEWRITER_PHONE_DEV to run a server for the phone app")
	}
	s := syncStore(t)
	p := NewPhoneSync(s, nil)
	st, err := p.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer p.shutdown()
	addDraft(t, s, "Desktop note", "<p>Hello from the <strong>computer</strong>.</p>")
	addDraft(t, s, "Linked note", `<p>Talk to <span class="ref" data-ref="character:3">Mara</span></p>`)
	info, err := p.StartPairing()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]any{"port": st.Port, "fingerprint": st.Fingerprint, "code": info.Code, "host": "127.0.0.1", "project": s.projectInfo(), "payload": info.Payload})
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(9 * time.Minute)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(out + ".stop"); err == nil {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
}
