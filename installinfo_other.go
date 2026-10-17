//go:build !windows

package main

// noteInstalledVersion only has work to do on Windows (see installinfo_windows.go).
func noteInstalledVersion() {}

// removeOldCopy only has work to do on Windows: on Linux the new file simply replaces the old one.
func removeOldCopy() {}
