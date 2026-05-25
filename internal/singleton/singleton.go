//go:build windows

// Package singleton ensures only one TurboKey process runs at a time, so
// double-launching can't end up with two hooks fighting over the same input.
package singleton

import (
	"syscall"

	"golang.org/x/sys/windows"

	"turbokey/internal/winput"
)

const mutexName = `Local\TurboKey-SingleInstance`

// keep the handle alive for the lifetime of the process so the named mutex
// stays owned (Windows releases it on process death).
var heldMutex windows.Handle

// Release closes the held named mutex so a sibling instance can take it over.
// Used by the language-change relaunch path, where the parent must release the
// mutex BEFORE the child reaches Acquire — otherwise the child would see the
// dying parent still owning it and exit, leaving no instance running.
func Release() {
	if heldMutex != 0 {
		windows.CloseHandle(heldMutex)
		heldMutex = 0
	}
}

// Acquire returns true if this is the first instance. If another instance
// already holds the mutex it best-effort brings its window to the front and
// returns false (the caller should exit).
func Acquire() bool {
	name, _ := syscall.UTF16PtrFromString(mutexName)
	h, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		if h != 0 {
			windows.CloseHandle(h)
		}
		winput.SurfaceFirstByExe("turbokey.exe", windows.GetCurrentProcessId())
		return false
	}
	if err != nil {
		// Couldn't create the mutex at all (e.g. permissions); fail open so the
		// app still launches rather than refusing to start.
		return true
	}
	heldMutex = h
	return true
}
