//go:build windows

// Package autostart toggles launching TurboKey at logon. Because the app requires
// administrator rights, it uses a Scheduled Task with highest privileges (which
// runs elevated at logon without a UAC prompt) rather than the HKCU Run key.
package autostart

import (
	"os"
	"os/exec"
	"syscall"
)

const taskName = "TurboKey"

func schtasks(args ...string) error {
	cmd := exec.Command("schtasks", args...)
	// Don't flash a console window.
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	return cmd.Run()
}

// Enabled reports whether the logon task exists.
func Enabled() bool {
	return schtasks("/query", "/tn", taskName) == nil
}

// Set creates or removes the logon task.
func Set(on bool) error {
	if !on {
		return schtasks("/delete", "/tn", taskName, "/f")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return schtasks("/create", "/tn", taskName, "/tr", exe, "/sc", "onlogon", "/rl", "highest", "/f")
}
