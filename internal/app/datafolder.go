package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// The data folder holds typewriter.db, projects.json, the projects folder and the backups.
// Where it is lives in one small file in the default folder, datadir.json.
// Without that file, the default folder is used.
const pointerFile = "datadir.json"

type pointer struct {
	Dir string `json:"dir"`
}

func readPointer(base string) string {
	b, err := os.ReadFile(filepath.Join(base, pointerFile))
	if err != nil {
		return ""
	}
	var p pointer
	json.Unmarshal(b, &p)
	return p.Dir
}

// writePointer saves where the data lives. The default folder means no file.
func writePointer(base, dir string) error {
	path := filepath.Join(base, pointerFile)
	if dir == "" {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	b, err := json.MarshalIndent(pointer{Dir: dir}, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// writable checks that the folder exists (it is made if needed) and that files can be written there.
func writable(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".typewriter-check-*")
	if err != nil {
		return err
	}
	name := f.Name()
	f.Close()
	return os.Remove(name)
}

// resolveDataDir finds the data folder. If the chosen folder cannot be used (a drive that is not
// plugged in, for example), it uses the default folder and says why, so the screen can warn you.
func resolveDataDir(base string) (dir, problem string) {
	chosen := readPointer(base)
	if chosen == "" {
		return base, ""
	}
	if err := writable(chosen); err != nil {
		return base, fmt.Sprintf("The data folder %s cannot be used (%v). The default folder is open instead. Your data is still in %s.", chosen, err, chosen)
	}
	return chosen, ""
}

func sameDir(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// within says whether child is the same as parent or inside it.
func within(parent, child string) bool {
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// DataFolderInfo is what the screen shows about where your data lives.
type DataFolderInfo struct {
	Path      string `json:"path"`      // the folder in use
	Default   string `json:"default"`   // the folder used when you have chosen none
	IsDefault bool   `json:"isDefault"` // true when the folder in use is the default one
	Problem   string `json:"problem"`   // not empty when your chosen folder could not be opened
}

func (s *Store) DataFolder() DataFolderInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return DataFolderInfo{Path: s.dir, Default: s.base, IsDefault: sameDir(s.dir, s.base), Problem: s.problem}
}

// FolderCheck is what a folder you picked holds.
type FolderCheck struct {
	Path    string `json:"path"`
	HasData bool   `json:"hasData"` // it already holds Typewriter data
	Same    bool   `json:"same"`    // it is the folder in use already
}

func hasData(dir string) bool {
	for _, n := range []string{"typewriter.db", "projects.json"} {
		if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
			return true
		}
	}
	return false
}

// InspectDataFolder checks a folder before the data moves there.
func (s *Store) InspectDataFolder(path string) (FolderCheck, error) {
	path = filepath.Clean(path)
	cur := s.DataFolder().Path
	if sameDir(path, cur) {
		return FolderCheck{Path: path, Same: true, HasData: true}, nil
	}
	if within(cur, path) {
		return FolderCheck{}, fmt.Errorf("pick a folder outside the current data folder")
	}
	if err := writable(path); err != nil {
		return FolderCheck{}, fmt.Errorf("the app cannot write to that folder: %w", err)
	}
	return FolderCheck{Path: path, HasData: hasData(path)}, nil
}

// ChooseDataFolder asks for a folder. It returns "" if you cancel.
func (s *Store) ChooseDataFolder() (string, error) {
	path, err := application.Get().Dialog.OpenFile().
		SetTitle("Choose the folder for your Typewriter data").
		CanChooseFiles(false).
		CanChooseDirectories(true).
		CanCreateDirectories(true).
		PromptForSingleSelection()
	if err != nil || path == "" {
		return "", nil
	}
	return path, nil
}

// MoveDataFolder changes where the data lives.
// mode "copy": copy all your data into the folder (it must not hold data yet), then use it.
// mode "use": use the data that is already in the folder.
// The old files are never deleted. You can remove them yourself when you are sure.
func (s *Store) MoveDataFolder(path, mode string) error {
	path = filepath.Clean(path)
	check, err := s.InspectDataFolder(path)
	if err != nil {
		return err
	}
	if check.Same {
		return nil
	}
	switch mode {
	case "copy":
		if check.HasData {
			return fmt.Errorf("that folder already holds Typewriter data, so nothing was copied")
		}
		if err := s.copyDataTo(path); err != nil {
			return fmt.Errorf("could not copy your data, so nothing was changed: %w", err)
		}
	case "use":
		if !check.HasData {
			return fmt.Errorf("that folder holds no Typewriter data")
		}
	default:
		return fmt.Errorf("unknown choice")
	}
	return s.switchDir(path)
}

// copyDataTo writes a full copy of the data: every project, the project list, and the backups.
func (s *Store) copyDataTo(dst string) error {
	s.mu.RLock()
	reg, cur, src := s.reg, s.reg.Current, s.dir
	s.mu.RUnlock()

	for _, p := range reg.Projects {
		to := projectFile(dst, p.ID)
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return err
		}
		if p.ID == cur {
			if err := s.copyTo(to); err != nil { // safe while the app uses it
				return err
			}
			continue
		}
		from := projectFile(src, p.ID)
		if _, err := os.Stat(from); err != nil {
			continue // a project with no file yet
		}
		if err := vacuumFile(from, to); err != nil {
			return err
		}
	}
	if err := saveRegistry(dst, reg); err != nil {
		return err
	}
	entries, _ := os.ReadDir(src)
	for _, e := range entries {
		if e.IsDir() && (e.Name() == "backups" || strings.HasPrefix(e.Name(), "backups-")) {
			if err := copyFiles(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// vacuumFile writes a clean copy of a data file that is not open.
func vacuumFile(from, to string) error {
	db, err := openDB(from)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := os.Remove(to); err != nil && !os.IsNotExist(err) {
		return err
	}
	_, err = db.Exec(`VACUUM INTO ?`, to)
	return err
}

func copyFiles(from, to string) error {
	if err := os.MkdirAll(to, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(from)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		in, err := os.Open(filepath.Join(from, e.Name()))
		if err != nil {
			return err
		}
		out, err := os.Create(filepath.Join(to, e.Name()))
		if err != nil {
			in.Close()
			return err
		}
		_, err = io.Copy(out, in)
		in.Close()
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// switchDir starts using the data in dir. The screen must reload after this.
func (s *Store) switchDir(dir string) error {
	reg := loadRegistry(dir)
	p, _ := reg.find(reg.Current)
	db, path, err := openProjectIn(dir, p)
	if err != nil {
		return fmt.Errorf("could not open the data in that folder: %w", err)
	}
	remember := dir
	if sameDir(dir, s.base) {
		remember = ""
	}
	if err := writePointer(s.base, remember); err != nil {
		db.Close()
		return fmt.Errorf("could not save the new folder, so nothing was changed: %w", err)
	}
	s.syncMu.Lock() // wait for a phone sync and for bookshelf jobs that use the open file (see switchTo)
	s.opMu.Lock()
	s.mu.Lock()
	old := s.db
	s.db, s.path, s.dir, s.reg, s.problem = db, path, dir, reg, ""
	s.mu.Unlock()
	old.Close()
	s.opMu.Unlock()
	s.syncMu.Unlock()
	s.autoBackupAt(time.Now())
	return nil
}
