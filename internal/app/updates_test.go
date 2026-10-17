package app

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

// fakeHost stands in for the app: the updater only needs to send events.
type fakeHost struct{}

func (fakeHost) Emit(string, ...any) bool                              { return true }
func (fakeHost) OnEvent(string, func(any)) func()                      { return func() {} }
func (fakeHost) OpenWindow(updater.WindowOptions) updater.WindowHandle { return nil }
func (fakeHost) Quit()                                                 {}

// fakeGitHub serves one release the way GitHub does: the API answer, the program file, its .sig and checksums.txt.
type fakeGitHub struct {
	tag, file string
	body      []byte // the program file that is served
	sig       string // the .sig file ("" = none)
	sums      string // checksums.txt ("" = none)
	downloads int
}

func (f *fakeGitHub) server(t *testing.T) *httptest.Server {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dl := srv.URL + "/dl/"
		switch r.URL.Path {
		case "/repos/" + updateRepo + "/releases/latest":
			assets := []map[string]any{{"id": 1, "name": f.file, "size": len(f.body), "browser_download_url": dl + f.file}}
			if f.sig != "" {
				assets = append(assets, map[string]any{"id": 2, "name": f.file + ".sig", "size": len(f.sig), "browser_download_url": dl + f.file + ".sig"})
			}
			if f.sums != "" {
				assets = append(assets, map[string]any{"id": 3, "name": "checksums.txt", "size": len(f.sums), "browser_download_url": dl + "checksums.txt"})
			}
			json.NewEncoder(w).Encode(map[string]any{"tag_name": f.tag, "name": "Version " + strings.TrimPrefix(f.tag, "v") + ": test", "body": "## New\n- a thing", "html_url": "https://example.test/release", "assets": assets})
		case "/dl/" + f.file:
			f.downloads++
			w.Write(f.body)
		case "/dl/" + f.file + ".sig":
			if f.sig == "" {
				http.NotFound(w, r)
				return
			}
			w.Write([]byte(f.sig))
		case "/dl/checksums.txt":
			w.Write([]byte(f.sums))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// release makes a signed release of body, as tools/release does.
func release(priv ed25519.PrivateKey, tag string, body []byte) *fakeGitHub {
	sum := sha256.Sum256(body)
	file := "typewriter-" + strings.TrimPrefix(tag, "v") + "-windows-x64.exe"
	if runtime.GOOS != "windows" {
		file = "typewriter-" + strings.TrimPrefix(tag, "v") + "-linux-x64"
	}
	return &fakeGitHub{
		tag: tag, file: file, body: body,
		sig:  base64.StdEncoding.EncodeToString(ed25519.Sign(priv, sum[:])) + "\n",
		sums: hex.EncodeToString(sum[:]) + "  " + file + "\n",
	}
}

func newTestUpdater(t *testing.T, pub ed25519.PublicKey, srv *httptest.Server) *updater.Updater {
	t.Helper()
	cfg, err := updateConfig(pub, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Arch = "amd64"
	u := updater.New(fakeHost{})
	if err := u.Init(cfg); err != nil {
		t.Fatal(err)
	}
	return u
}

func TestUpdateSignedReleaseInstalls(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	body := bytes.Repeat([]byte("new program "), 1000)
	gh := release(priv, "v99.0.0", body)
	u := newTestUpdater(t, pub, gh.server(t))

	rel, err := u.Check(context.Background())
	if err != nil || rel == nil || rel.Version != "99.0.0" {
		t.Fatalf("check: %+v %v", rel, err)
	}
	if rel.Verification == nil || rel.Verification.SignatureAlgo != "ed25519" || len(rel.Verification.Digest) == 0 {
		t.Fatalf("the release must carry both the checksum and the signature: %+v", rel.Verification)
	}
	if err := u.DownloadAndInstall(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Dir(u.DownloadedPath())) }) // the staged file is in a temp folder of its own
	got, err := os.ReadFile(u.DownloadedPath())
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("the staged file must be the release file: %v", err)
	}
}

func TestUpdateRefusesBadFiles(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	body := []byte("the real program")

	t.Run("changed file, checksum and signature of the real one", func(t *testing.T) {
		gh := release(priv, "v99.0.0", body)
		gh.body = []byte("a changed program")
		u := newTestUpdater(t, pub, gh.server(t))
		if _, err := u.Check(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := u.DownloadAndInstall(context.Background()); err == nil {
			t.Fatal("a changed file must be refused")
		}
	})
	t.Run("changed file with a matching checksum, but the old signature", func(t *testing.T) {
		gh := release(priv, "v99.0.0", body)
		evil := []byte("a changed program")
		sum := sha256.Sum256(evil)
		gh.body, gh.sums = evil, hex.EncodeToString(sum[:])+"  "+gh.file+"\n"
		u := newTestUpdater(t, pub, gh.server(t))
		if _, err := u.Check(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := u.DownloadAndInstall(context.Background()); err == nil {
			t.Fatal("a file whose signature does not match must be refused, even when its checksum does")
		}
	})
	t.Run("signed with another key", func(t *testing.T) {
		gh := release(other, "v99.0.0", body)
		u := newTestUpdater(t, pub, gh.server(t))
		if _, err := u.Check(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := u.DownloadAndInstall(context.Background()); err == nil {
			t.Fatal("a file signed with another key must be refused")
		}
	})
	t.Run("no signature at all", func(t *testing.T) {
		gh := release(priv, "v99.0.0", body)
		gh.sig = ""
		u := newTestUpdater(t, pub, gh.server(t))
		if rel, err := u.Check(context.Background()); err == nil || rel != nil {
			t.Fatalf("a release without a signature must not even be offered: %+v %v", rel, err)
		}
		if gh.downloads != 0 {
			t.Fatal("nothing may be downloaded")
		}
	})
	t.Run("the same or an older version", func(t *testing.T) {
		gh := release(priv, "v"+Version, body)
		u := newTestUpdater(t, pub, gh.server(t))
		if rel, err := u.Check(context.Background()); err != nil || rel != nil {
			t.Fatalf("this version is the newest: %+v %v", rel, err)
		}
	})
}

func TestUpdatePublicKeyInTheApp(t *testing.T) {
	pub, err := base64.StdEncoding.DecodeString(updatePublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		t.Fatal("updatekey.go must hold a good public key (made by go run ./tools/release keygen)")
	}
}

func TestFriendlyErrors(t *testing.T) {
	if got := friendly(context.DeadlineExceeded).Error(); !strings.Contains(got, "did not answer") {
		t.Fatal(got)
	}
	if got := friendly(errStr("github: HTTP 404")).Error(); !strings.Contains(got, "no releases") {
		t.Fatal(got)
	}
}

type errStr string

func (e errStr) Error() string { return string(e) }
