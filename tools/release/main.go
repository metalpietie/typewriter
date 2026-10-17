// Command release builds Typewriter, signs the files and puts them in the GitHub release of the version.
//
//	go run ./tools/release keygen               make the signing key, once (see docs/releasing.md)
//	go run ./tools/release publish              build, sign, and upload to the release of the version in internal/app/version.go
//	go run ./tools/release publish -no-upload   the same, but stop before the upload (the files are in bin/release/<version>)
//	go run ./tools/release publish -no-build    sign and upload the files that are already built
//	go run ./tools/release publish -apk         also upload the phone app (bin/typewriter-notes.apk)
//	go run ./tools/release verify <file>        check a file against its .sig, with the key in the app
//
// The private key stays on this computer (keyPath). The public key is in internal/app/updatekey.go, in the app, so the app can tell
// that an update was made with this key and not by someone else.
package main

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "keygen":
		err = keygen()
	case "publish":
		err = publish(os.Args[2:])
	case "verify":
		if len(os.Args) != 3 {
			usage()
		}
		err = verifyFile(os.Args[2])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "release:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: go run ./tools/release keygen | publish [-no-upload] [-no-build] [-apk] | verify <file>")
	os.Exit(2)
}

// keyPath is where the private key is kept: outside the repository, in this user's settings folder.
func keyPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "typewriter-release", "signing.key"), nil
}

const keyFile = "internal/app/updatekey.go"

// keygen makes a new key pair. It refuses when a key exists, so a key is never replaced by mistake.
func keygen() error {
	path, err := keyPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("a signing key exists already: %s. Remove it by hand first if you really want a new one (the apps out there will then refuse updates until they are updated by hand)", path)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(priv)+"\n"), 0o600); err != nil {
		return err
	}
	if err := writePublicKey(pub); err != nil {
		return err
	}
	fmt.Println("The private key is in", path)
	fmt.Println("Make a backup of it now (a USB stick, a password manager). Never put it in the repository.")
	fmt.Println("The public key is in", keyFile, "- commit that file.")
	return nil
}

func writePublicKey(pub ed25519.PublicKey) error {
	src := `package app

// updatePublicKey is the public half of the key that signs the releases (see tools/release and docs/releasing.md).
// The app installs an update only when its signature matches this key. It is made by "go run ./tools/release keygen".
const updatePublicKey = "` + base64.StdEncoding.EncodeToString(pub) + `"
`
	return os.WriteFile(keyFile, []byte(src), 0o644)
}

func loadPrivateKey() (ed25519.PrivateKey, error) {
	path, err := keyPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no signing key (%v). Make one with: go run ./tools/release keygen", err)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil || len(raw) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("the signing key in %s is broken", path)
	}
	return ed25519.PrivateKey(raw), nil
}

var publicKeyRe = regexp.MustCompile(`updatePublicKey = "([A-Za-z0-9+/=]+)"`)

// appPublicKey reads the key that the app checks with, from internal/app/updatekey.go.
func appPublicKey() (ed25519.PublicKey, error) {
	b, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("%s is missing. Make the key with: go run ./tools/release keygen", keyFile)
	}
	m := publicKeyRe.FindSubmatch(b)
	if m == nil {
		return nil, fmt.Errorf("no public key in %s", keyFile)
	}
	raw, err := base64.StdEncoding.DecodeString(string(m[1]))
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("the public key in %s is broken", keyFile)
	}
	return ed25519.PublicKey(raw), nil
}

var versionRe = regexp.MustCompile(`const Version = "([0-9]+\.[0-9]+\.[0-9]+[^"]*)"`)

func appVersion() (string, error) {
	b, err := os.ReadFile("internal/app/version.go")
	if err != nil {
		return "", err
	}
	m := versionRe.FindSubmatch(b)
	if m == nil {
		return "", errors.New("no version in internal/app/version.go")
	}
	return string(m[1]), nil
}

// fileDigest is the SHA-256 of a file.
func fileDigest(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// sign signs the SHA-256 of a file (the updater checks the signature against the digest it computes while downloading).
func sign(priv ed25519.PrivateKey, digest []byte) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(priv, digest)) + "\n"
}

func checkSig(pub ed25519.PublicKey, digest []byte, sigText string) bool {
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sigText))
	return err == nil && ed25519.Verify(pub, digest, sig)
}

func verifyFile(path string) error {
	pub, err := appPublicKey()
	if err != nil {
		return err
	}
	digest, err := fileDigest(path)
	if err != nil {
		return err
	}
	sig, err := os.ReadFile(path + ".sig")
	if err != nil {
		return err
	}
	if !checkSig(pub, digest, string(sig)) {
		return fmt.Errorf("%s: the signature does NOT match", path)
	}
	fmt.Println(path + ": the signature matches")
	return nil
}

// run runs a command in the repository, with its output on the screen.
func run(name string, args ...string) error {
	fmt.Println(">", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.Env = withNSIS(os.Environ())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func output(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return strings.TrimSpace(string(out)), err
}

// withNSIS puts the NSIS folder on the path, because winget does not, and the installer needs makensis.
func withNSIS(env []string) []string {
	if runtime.GOOS != "windows" {
		return env
	}
	nsis := `C:\Program Files (x86)\NSIS`
	if _, err := os.Stat(filepath.Join(nsis, "makensis.exe")); err != nil {
		return env
	}
	for i, e := range env {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			env[i] = e + ";" + nsis
		}
	}
	return env
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// releaseFile is one file of a release: where it comes from and its name on GitHub.
// The name holds the platform and the processor, so the updater in the app picks the right one.
type releaseFile struct{ from, name string }

func releaseFiles(version string, apk bool) []releaseFile {
	files := []releaseFile{
		{filepath.Join("bin", "typewriter.exe"), "typewriter-" + version + "-windows-x64.exe"},
		{filepath.Join("bin", "typewriter-amd64-installer.exe"), "typewriter-" + version + "-windows-x64-installer.exe"},
		{filepath.Join("bin", "typewriter"), "typewriter-" + version + "-linux-x64"},
	}
	if apk {
		files = append(files, releaseFile{filepath.Join("bin", "typewriter-notes.apk"), "typewriter-notes-" + version + ".apk"})
	}
	return files
}

// signed tells which files get a signature: the ones that the updater can install (not the installer, not the phone app).
func signed(name string) bool {
	return !strings.Contains(name, "-installer.") && !strings.HasSuffix(name, ".apk")
}

// notesTitle is the title of the release: the first "## " line of the notes, without the "## ".
func notesTitle(notes, version string) string {
	sc := bufio.NewScanner(strings.NewReader(notes))
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); strings.HasPrefix(line, "## ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "## "))
		}
	}
	return "Version " + version
}

func publish(args []string) error {
	fs := flag.NewFlagSet("publish", flag.ExitOnError)
	noUpload := fs.Bool("no-upload", false, "build and sign, but do not upload")
	noBuild := fs.Bool("no-build", false, "use the files that are built already")
	apk := fs.Bool("apk", false, "also upload the phone app (bin/typewriter-notes.apk)")
	fs.Parse(args)

	version, err := appVersion()
	if err != nil {
		return err
	}
	tag := "v" + version
	notesPath := filepath.Join("release-notes", version+".md")
	notes, err := os.ReadFile(notesPath)
	if err != nil {
		return fmt.Errorf("the release text is missing: write %s first", notesPath)
	}
	priv, err := loadPrivateKey()
	if err != nil {
		return err
	}
	pub, err := appPublicKey()
	if err != nil {
		return err
	}
	if !pub.Equal(priv.Public()) {
		return fmt.Errorf("the signing key on this computer does not belong to the public key in %s, so the app would refuse the update", keyFile)
	}
	if !*noUpload {
		if dirty, _ := output("git", "status", "--porcelain"); dirty != "" {
			return errors.New("there are changes that are not committed. Commit them first, so the release is what is in git")
		}
		if t, _ := output("git", "tag", "--list", tag); t != tag {
			return fmt.Errorf("the tag %s does not exist. Make it first", tag)
		}
	}

	if !*noBuild {
		if err := exec.Command("docker", "info").Run(); err != nil { // the Linux build needs it: say so before building anything
			return errors.New("Docker is not running, and the Linux build needs it. Start Docker, then try again")
		}
		// Linux first: its build can remove bin\typewriter.exe, so Windows is built after it
		if err := run("wails3", "task", "linux:build"); err != nil {
			return err
		}
		if err := run("wails3", "build"); err != nil {
			return err
		}
		if err := run("wails3", "task", "windows:package", "INSTALL_SCOPE=user"); err != nil {
			return err
		}
	}

	out := filepath.Join("bin", "release", version)
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	var upload []string
	var sums strings.Builder
	for _, f := range releaseFiles(version, *apk) {
		dst := filepath.Join(out, f.name)
		if err := copyFile(f.from, dst); err != nil {
			return fmt.Errorf("%s: %w", f.from, err)
		}
		digest, err := fileDigest(dst)
		if err != nil {
			return err
		}
		fmt.Fprintf(&sums, "%s  %s\n", hex.EncodeToString(digest), f.name)
		upload = append(upload, dst)
		if signed(f.name) {
			sig := sign(priv, digest)
			if !checkSig(pub, digest, sig) { // never upload a signature that the app would refuse
				return fmt.Errorf("%s: the new signature does not check out", f.name)
			}
			if err := os.WriteFile(dst+".sig", []byte(sig), 0o644); err != nil {
				return err
			}
			upload = append(upload, dst+".sig")
		}
	}
	sumsPath := filepath.Join(out, "checksums.txt")
	if err := os.WriteFile(sumsPath, []byte(sums.String()), 0o644); err != nil {
		return err
	}
	upload = append(upload, sumsPath)
	fmt.Println("The files are in", out)
	if *noUpload {
		return nil
	}

	title := notesTitle(string(notes), version)
	if _, err := output("gh", "release", "view", tag); err == nil {
		if err := run("gh", "release", "edit", tag, "--title", title, "--notes-file", notesPath); err != nil {
			return err
		}
		return run("gh", append([]string{"release", "upload", tag, "--clobber"}, upload...)...)
	}
	return run("gh", append([]string{"release", "create", tag, "--title", title, "--notes-file", notesPath, "--verify-tag"}, upload...)...)
}
