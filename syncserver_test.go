package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/grandcat/zeroconf"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// A phone, for the tests: it keeps the fingerprint of the computer, as the real phone app will, and talks to the real server.
type fakePhone struct {
	t      *testing.T
	base   string
	fp     string
	id     string
	key    string
	client *http.Client
}

func newFakePhone(t *testing.T, port int, fp string) *fakePhone {
	t.Helper()
	pin := func(raw [][]byte, _ [][]*x509.Certificate) error {
		if len(raw) == 0 {
			return fmt.Errorf("no certificate")
		}
		sum := sha256.Sum256(raw[0])
		if hex.EncodeToString(sum[:]) != fp {
			return fmt.Errorf("this is not the computer that was added")
		}
		return nil
	}
	return &fakePhone{
		t: t, base: fmt.Sprintf("https://127.0.0.1:%d", port), fp: fp, id: "aaaaaaaa-0000-4000-8000-000000000001",
		client: &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, VerifyPeerCertificate: pin}}},
	}
}

func (f *fakePhone) call(method, path string, body any, auth bool) (int, map[string]any) {
	f.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, f.base+path, rd)
	if auth {
		req.Header.Set("Authorization", "Bearer "+f.id+":"+f.key)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		f.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func (f *fakePhone) pair(code string) (int, map[string]any) {
	code2, out := f.call("POST", "/pair", map[string]string{"code": code, "deviceId": f.id, "name": "Pixel 7"}, false)
	if code2 == 200 {
		f.key, _ = out["key"].(string)
	}
	return code2, out
}

func startSync(t *testing.T) (*PhoneSync, *Store, *atomic.Int32) {
	t.Helper()
	s := syncStore(t)
	var events atomic.Int32
	p := NewPhoneSync(s, func(string) { events.Add(1) })
	st, err := p.Start()
	if err != nil || !st.On || st.Port == 0 {
		t.Fatalf("start: %+v %v", st, err)
	}
	t.Cleanup(p.shutdown)
	return p, s, &events
}

func addedPhone(t *testing.T) (*fakePhone, *PhoneSync, *Store, *atomic.Int32) {
	p, s, ev := startSync(t)
	info, err := p.StartPairing()
	if err != nil {
		t.Fatal(err)
	}
	st := p.Status()
	f := newFakePhone(t, st.Port, st.Fingerprint)
	if code, out := f.pair(info.Code); code != 200 {
		t.Fatalf("pair: %d %v", code, out)
	}
	return f, p, s, ev
}

func TestServerStartAndHello(t *testing.T) {
	p, _, _ := startSync(t)
	st := p.Status()
	if !st.On || len(st.Fingerprint) != 64 || st.Computer == "" {
		t.Fatalf("status: %+v", st)
	}
	f := newFakePhone(t, st.Port, st.Fingerprint)
	code, out := f.call("GET", "/hello", nil, false)
	if code != 200 || out["app"] != "Typewriter" || out["protocol"] != float64(syncProtocol) {
		t.Fatalf("hello: %d %v", code, out)
	}
	if _, has := out["project"]; has {
		t.Fatal("hello must not say anything about the notes")
	}
}

func TestPairingQR(t *testing.T) {
	p, _, _ := startSync(t)
	info, err := p.StartPairing()
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Code) != 6 || !strings.HasPrefix(info.QR, "data:image/png;base64,") || info.Expires < time.Now().Unix() {
		t.Fatalf("info: %+v", info)
	}
	var payload struct {
		Code, FP, Name string
		Port           int
		Hosts          []string
	}
	json.Unmarshal([]byte(info.Payload), &payload)
	st := p.Status()
	if payload.Code != info.Code || payload.FP != st.Fingerprint || payload.Port != st.Port || payload.Name != st.Computer {
		t.Fatalf("payload: %+v", payload)
	}
}

func TestPairingNeedsSyncOn(t *testing.T) {
	s := syncStore(t)
	p := NewPhoneSync(s, nil)
	if _, err := p.StartPairing(); err == nil {
		t.Fatal("a code with the server off")
	}
}

func TestWrongCodesKillTheCode(t *testing.T) {
	p, _, _ := startSync(t)
	info, _ := p.StartPairing()
	st := p.Status()
	f := newFakePhone(t, st.Port, st.Fingerprint)
	for i := 0; i < pairingTries; i++ {
		if code, _ := f.pair("000000"); code != 403 {
			t.Fatalf("try %d: %d", i, code)
		}
	}
	if code, _ := f.pair(info.Code); code != 403 {
		t.Fatalf("the right code works after %d wrong ones: %d", pairingTries, code)
	}
}

func TestCodeWorksOnceAndExpires(t *testing.T) {
	p, _, _ := startSync(t)
	info, _ := p.StartPairing()
	st := p.Status()
	f := newFakePhone(t, st.Port, st.Fingerprint)
	if code, out := f.pair(info.Code); code != 200 || out["key"] == "" {
		t.Fatalf("first: %d %v", code, out)
	}
	other := newFakePhone(t, st.Port, st.Fingerprint)
	other.id = "bbbbbbbb-0000-4000-8000-000000000002"
	if code, _ := other.pair(info.Code); code != 403 {
		t.Fatalf("a code worked twice: %d", code)
	}
	// an old code
	info2, _ := p.StartPairing()
	p.mu.Lock()
	p.pair.expires = time.Now().Add(-time.Second)
	p.mu.Unlock()
	if code, _ := other.pair(info2.Code); code != 403 {
		t.Fatalf("an old code worked: %d", code)
	}
}

func TestSyncNeedsAKey(t *testing.T) {
	f, _, _, _ := addedPhone(t)
	body := PhoneRequest{Notes: nil}
	f2 := *f
	f2.key = "not-the-key"
	if code, _ := f2.call("POST", "/sync", body, true); code != 401 {
		t.Fatalf("a wrong key: %d", code)
	}
	if code, _ := f.call("POST", "/sync", body, false); code != 401 {
		t.Fatalf("no key: %d", code)
	}
	f3 := *f
	f3.id = "cccccccc-0000-4000-8000-000000000003"
	if code, _ := f3.call("POST", "/sync", body, true); code != 401 {
		t.Fatalf("an unknown phone: %d", code)
	}
	if code, _ := f.call("POST", "/sync", body, true); code != 200 {
		t.Fatalf("the right key: %d", code)
	}
}

func TestSyncOverTheNetwork(t *testing.T) {
	f, p, s, events := addedPhone(t)
	addDraft(t, s, "From the computer", "<p>Hello</p>")
	req := map[string]any{"project": s.projectInfo().ID, "since": 0, "notes": []map[string]any{
		{"sid": sidA, "title": "From the phone", "text": "An **idea**", "updatedAt": time.Now().UTC().Format(time.RFC3339), "base": 0},
	}}
	code, out := f.call("POST", "/sync", req, true)
	if code != 200 {
		t.Fatalf("sync: %d %v", code, out)
	}
	if len(out["saved"].([]any)) != 1 || len(out["changed"].([]any)) != 1 {
		t.Fatalf("answer: %v", out)
	}
	if got := drafts(t, s); got["From the phone"] != "<p>An <strong>idea</strong></p>" {
		t.Fatalf("pages: %v", got)
	}
	if events.Load() == 0 {
		t.Fatal("the screen was not told")
	}
	if st := p.Status(); len(st.Devices) != 1 || st.Devices[0].Last == "" || st.Devices[0].Name != "Pixel 7" {
		t.Fatalf("devices: %+v", st.Devices)
	}
	// the note says which phone it came from
	notes, _ := s.readSyncNotes()
	var from string
	for _, n := range notes {
		if n.SID == sidA {
			from = n.From
		}
	}
	if from != "Pixel 7" {
		t.Fatalf("from %q", from)
	}
	// another project is refused without a change
	req["project"] = "other"
	if code, out := f.call("POST", "/sync", req, true); code != 200 || out["mismatch"] != true {
		t.Fatalf("mismatch: %d %v", code, out)
	}
}

func TestBadBodies(t *testing.T) {
	f, _, _, _ := addedPhone(t)
	big := map[string]any{"notes": []map[string]any{{"sid": sidA, "title": "x", "text": strings.Repeat("y", maxBody)}}}
	if code, _ := f.call("POST", "/sync", big, true); code != 400 {
		t.Fatalf("a huge body: %d", code)
	}
	req, _ := http.NewRequest("POST", f.base+"/sync", strings.NewReader("{not json"))
	req.Header.Set("Authorization", "Bearer "+f.id+":"+f.key)
	resp, err := f.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("bad json: %d", resp.StatusCode)
	}
	if code, _ := f.call("GET", "/sync", nil, true); code != 405 {
		t.Fatalf("GET: %d", code)
	}
}

func TestPinnedFingerprint(t *testing.T) {
	p, _, _ := startSync(t)
	st := p.Status()
	wrong := newFakePhone(t, st.Port, strings.Repeat("0", 64))
	if _, err := wrong.client.Get(wrong.base + "/hello"); err == nil {
		t.Fatal("a phone that pinned another certificate talked to this computer")
	}
}

func TestRemoveAPhone(t *testing.T) {
	f, p, _, _ := addedPhone(t)
	if err := p.RemoveDevice(f.id); err != nil {
		t.Fatal(err)
	}
	if code, _ := f.call("POST", "/sync", PhoneRequest{}, true); code != 401 {
		t.Fatalf("a removed phone synced: %d", code)
	}
	if len(p.Status().Devices) != 0 {
		t.Fatal("still in the list")
	}
}

func TestSwitchOffAndOnKeepsPhonesAndCertificate(t *testing.T) {
	f, p, s, _ := addedPhone(t)
	fp := p.Status().Fingerprint
	if _, err := p.Stop(); err != nil {
		t.Fatal(err)
	}
	if p.Status().On {
		t.Fatal("still on")
	}
	if _, err := f.client.Get(f.base + "/hello"); err == nil {
		t.Fatal("the computer answered while switched off")
	}
	// a new run of the app: it was off, so it stays off
	p2 := NewPhoneSync(s, nil)
	p2.resume()
	if p2.Status().On {
		t.Fatal("it came on by itself")
	}
	st, err := p2.Start()
	if err != nil || st.Fingerprint != fp || len(st.Devices) != 1 {
		t.Fatalf("after start: %+v %v", st, err)
	}
	defer p2.shutdown()
	f.base = fmt.Sprintf("https://127.0.0.1:%d", st.Port)
	if code, _ := f.call("POST", "/sync", PhoneRequest{}, true); code != 200 {
		t.Fatalf("the phone has to add itself again: %d", code)
	}
	// it was on when the app closed: it comes on again
	p2.shutdown()
	p3 := NewPhoneSync(s, nil)
	p3.resume()
	defer p3.shutdown()
	if !p3.Status().On {
		t.Fatal("it did not come back on")
	}
}

func TestKeyIsNotKept(t *testing.T) {
	f, p, _, _ := addedPhone(t)
	b, err := os.ReadFile(p.path())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), f.key) {
		t.Fatal("the key of the phone is in sync.json")
	}
	if !strings.Contains(string(b), keyHash(f.key)) {
		t.Fatal("its fingerprint should be there")
	}
}

func TestAddingTheSamePhoneAgainGivesANewKey(t *testing.T) {
	f, p, _, _ := addedPhone(t)
	old := f.key
	info, _ := p.StartPairing()
	if code, _ := f.pair(info.Code); code != 200 {
		t.Fatalf("pair again: %d", code)
	}
	if f.key == old || len(p.Status().Devices) != 1 {
		t.Fatalf("key %q, phones %d", f.key, len(p.Status().Devices))
	}
	f2 := *f
	f2.key = old
	if code, _ := f2.call("POST", "/sync", PhoneRequest{}, true); code != 401 {
		t.Fatalf("the old key still works: %d", code)
	}
}

// Finding the computer by name needs a network that lets multicast through, so a failure here only skips the test.
func TestAnnouncedOnTheNetwork(t *testing.T) {
	p, _, _ := startSync(t)
	st := p.Status()
	if !st.Announced {
		t.Skip("the computer could not announce itself on this network")
	}
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		t.Skip("no resolver: ", err)
	}
	entries := make(chan *zeroconf.ServiceEntry, 16)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	if err := resolver.Browse(ctx, "_typewriter._tcp", "local.", entries); err != nil {
		t.Skip("cannot browse: ", err)
	}
	for {
		select {
		case e := <-entries:
			if e != nil && e.Port == st.Port {
				ok := false
				for _, txt := range e.Text {
					if strings.HasPrefix(txt, "fp="+st.Fingerprint[:16]) {
						ok = true
					}
				}
				if !ok {
					t.Fatalf("the announcement has no fingerprint: %v", e.Text)
				}
				return
			}
		case <-ctx.Done():
			t.Skip("did not see the announcement on this network (multicast may be blocked)")
		}
	}
}
