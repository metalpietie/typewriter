package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	ghupdate "github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

// Updates: the app asks GitHub for a newer release, downloads it, checks it, and starts again in the new version.
// Nothing else goes online, and nothing of your writing leaves the computer. See docs/releasing.md for how a release is made.

// updateRepo is where the releases are.
const updateRepo = "metalpietie/typewriter"

// signedRelease is the GitHub source of the Wails updater, with one more check: it also reads the signature of the
// file (<file>.sig, made by tools/release with the private key). A release without a good signature is refused,
// so only a file made with the release key can ever be installed, even if someone else could change the release.
type signedRelease struct {
	gh     *ghupdate.Provider
	client *http.Client
}

func (p *signedRelease) Name() string { return "github" }

func (p *signedRelease) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	rel, err := p.gh.Check(ctx, req)
	if err != nil || rel == nil {
		return rel, err
	}
	link, _ := rel.Metadata["github.asset.url"].(string)
	if link == "" {
		return nil, errors.New("the release has no download link")
	}
	text, err := p.fetch(ctx, link+".sig")
	if err != nil {
		return nil, fmt.Errorf("version %s is not signed (%v), so it is not installed", rel.Version, err)
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(text)))
	if err != nil || len(sig) != ed25519.SignatureSize {
		return nil, fmt.Errorf("the signature of version %s is broken, so it is not installed", rel.Version)
	}
	if rel.Verification == nil {
		rel.Verification = &updater.Verification{}
	}
	rel.Verification.DigestAlgo = "sha256" // the signature is made over the SHA-256 of the file
	rel.Verification.SignatureAlgo = "ed25519"
	rel.Verification.Signature = sig
	return rel, nil
}

func (p *signedRelease) Download(ctx context.Context, r *updater.Release, dst io.Writer, onProgress func(written, total int64)) error {
	return p.gh.Download(ctx, r, dst, onProgress)
}

func (p *signedRelease) fetch(ctx context.Context, link string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 4096))
}

// UpdateInfo is what the screen shows about the newest version.
type UpdateInfo struct {
	Current    string `json:"current"`
	Available  bool   `json:"available"`
	Version    string `json:"version"`
	Title      string `json:"title"`
	Notes      string `json:"notes"`      // the release text (Markdown)
	Page       string `json:"page"`       // the release page on GitHub, for a download by hand
	CanInstall bool   `json:"canInstall"` // false when the app cannot replace its own file
	Why        string `json:"why"`        // why it cannot
}

// Updates is the service the screen calls.
type Updates struct {
	store      *Store
	app        *application.App
	allowClose func() // lets the window close without asking the screen to save again (it saved already)
	initErr    error

	mu         sync.Mutex
	onClose    bool // restart into the new version when the window is closed
	restarting bool
}

// updateConfig is the set-up of the updater: this version, the signed GitHub source, and the key that the signatures
// must match. base is the address of the GitHub API; empty means GitHub itself (a test passes a local server).
func updateConfig(pub []byte, base string) (updater.Config, error) {
	if len(pub) != ed25519.PublicKeySize {
		return updater.Config{}, errors.New("the update key in the app is broken")
	}
	gh, err := ghupdate.New(ghupdate.Config{Repository: updateRepo, ChecksumAsset: "checksums.txt", BaseURL: base})
	if err != nil {
		return updater.Config{}, err
	}
	return updater.Config{
		CurrentVersion: Version,
		Providers:      []updater.Provider{&signedRelease{gh: gh, client: &http.Client{Timeout: 30 * time.Second}}},
		PublicKey:      pub,
		Window:         updater.WindowNone, // the app shows its own window
	}, nil
}

// init sets up the updater. It is called once, after the app is made.
func (u *Updates) init(app *application.App) {
	u.app = app
	pub, _ := base64.StdEncoding.DecodeString(updatePublicKey)
	cfg, err := updateConfig(pub, os.Getenv("TYPEWRITER_UPDATE_API")) // the variable is only for a test against a local server
	if err != nil {
		u.initErr = err
		return
	}
	u.initErr = app.Updater.Init(cfg)
}

// canReplaceSelf tells if the app may write a new program file next to itself.
func canReplaceSelf() (bool, string) {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		return false, "Updates in the app work on Windows and Linux."
	}
	exe, err := os.Executable()
	if err != nil {
		return false, "Typewriter could not find its own program file."
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	dir := filepath.Dir(exe)
	f, err := os.CreateTemp(dir, ".typewriter-update-test-*")
	if err != nil {
		return false, fmt.Sprintf("Typewriter is in a folder that it may not change (%s). Download the new version from the release page instead.", dir)
	}
	f.Close()
	os.Remove(f.Name())
	return true, ""
}

// friendly turns an error of the network into words.
func friendly(err error) error {
	var dns *net.DNSError
	var op *net.OpError
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return errors.New("GitHub did not answer in time. Try again later")
	case errors.As(err, &dns), errors.As(err, &op):
		return errors.New("could not reach GitHub. Check the internet connection, then try again")
	case strings.Contains(err.Error(), "HTTP 404"):
		return errors.New("no releases were found on GitHub (the repository may be private)")
	case strings.Contains(err.Error(), "HTTP 403"):
		return errors.New("GitHub asked to wait a while before asking again. Try again later")
	}
	return err
}

// CheckForUpdate asks GitHub for the newest release.
func (u *Updates) CheckForUpdate() (UpdateInfo, error) {
	info := UpdateInfo{Current: Version}
	if u.initErr != nil {
		return info, u.initErr
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rel, err := u.app.Updater.Check(ctx)
	if err != nil {
		return info, friendly(err)
	}
	if rel == nil {
		return info, nil
	}
	info.Available, info.Version, info.Notes = true, rel.Version, rel.Notes
	info.Title = rel.Name
	if info.Title == "" {
		info.Title = "Version " + rel.Version
	}
	info.Page, _ = rel.Metadata["github.release.htmlURL"].(string)
	if info.Page == "" {
		info.Page = "https://github.com/" + updateRepo + "/releases/latest"
	}
	info.CanInstall, info.Why = canReplaceSelf()
	return info, nil
}

// InstallUpdate downloads the version that CheckForUpdate found, and checks it (checksum and signature).
// The screen follows the download with the event wails:updater:download-progress. Nothing changes until RestartToUpdate.
func (u *Updates) InstallUpdate() error {
	if u.initErr != nil {
		return u.initErr
	}
	if ok, why := canReplaceSelf(); !ok {
		return errors.New(why)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	if err := u.app.Updater.DownloadAndInstall(ctx); err != nil {
		if strings.Contains(err.Error(), "digest mismatch") || strings.Contains(err.Error(), "signature") {
			return errors.New("the downloaded file did not pass the check, so it was thrown away. Nothing was changed")
		}
		return friendly(err)
	}
	return nil
}

// RestartToUpdate makes a copy of the data, then closes the app and starts the new version.
// The screen must have saved its last edits before it calls this.
func (u *Updates) RestartToUpdate() error {
	u.mu.Lock()
	if u.restarting {
		u.mu.Unlock()
		return nil
	}
	u.restarting = true
	u.mu.Unlock()
	if err := u.store.copyBeforeUpdate(); err != nil {
		u.mu.Lock()
		u.restarting = false
		u.mu.Unlock()
		return fmt.Errorf("could not make a copy of your data first, so the update waits: %w", err)
	}
	if u.allowClose != nil {
		u.allowClose()
	}
	if err := u.app.Updater.Restart(context.Background()); err != nil {
		u.mu.Lock()
		u.restarting = false
		u.mu.Unlock()
		return fmt.Errorf("could not start the new version: %w", err)
	}
	return nil
}

// RestartOnClose asks for the new version to start when the window is closed, instead of now.
func (u *Updates) RestartOnClose() {
	u.mu.Lock()
	u.onClose = true
	u.mu.Unlock()
}

// wantsRestartOnClose is asked by the window when it closes.
func (u *Updates) wantsRestartOnClose() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.onClose && !u.restarting
}

// copyBeforeUpdate keeps a copy of the open project from just before an update, next to the daily copies.
func (s *Store) copyBeforeUpdate() error {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	dir := s.backupDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "before-update-"+time.Now().Format("2006-01-02-150405")+".db")
	tmp := path + ".tmp"
	if err := s.copyTo(tmp); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	prune(dir, "before-update-", 3)
	return nil
}
