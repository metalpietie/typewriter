//go:build windows

package app

import (
	"os"
	"path/filepath"
	"strings"

	winreg "golang.org/x/sys/windows/registry"
)

// The installer writes an entry for Settings > Apps, with the version (build/windows/nsis/project.nsi).
const uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\MetalPietieTypewriter`

// noteInstalledVersion keeps the version in Settings > Apps right after the app updated itself.
// Only the installed app does this: a portable copy in another folder leaves the entry alone.
func noteInstalledVersion() {
	k, err := winreg.OpenKey(winreg.CURRENT_USER, uninstallKey, winreg.QUERY_VALUE|winreg.SET_VALUE)
	if err != nil {
		return // not installed with the installer
	}
	defer k.Close()
	loc, _, err := k.GetStringValue("InstallLocation")
	if err != nil {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	if !strings.EqualFold(filepath.Clean(filepath.Dir(exe)), filepath.Clean(loc)) {
		return
	}
	if v, _, _ := k.GetStringValue("DisplayVersion"); v != Version {
		k.SetStringValue("DisplayVersion", Version)
	}
}

// removeOldCopy deletes the old program file that an update left next to the new one (typewriter.exe.old.<number>).
// Wails would only delete it at the next update; by now the old program has closed, so it can go.
func removeOldCopy() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	old, _ := filepath.Glob(exe + ".old.*")
	for _, f := range old {
		os.Remove(f) // a file that is still in use stays until next time
	}
}
