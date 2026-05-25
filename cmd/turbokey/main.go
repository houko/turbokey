//go:build windows

// Command turbokey is a Windows key auto-fire (rapid-fire) tool with a native GUI.
package main

import (
	"runtime"

	"turbokey/internal/config"
	"turbokey/internal/engine"
	"turbokey/internal/i18n"
	"turbokey/internal/ui"
)

func main() {
	// Pin the GUI to a single OS thread.
	runtime.LockOSThread()

	i18n.Init()

	e := engine.New(0x77) // VK_F8 master toggle hotkey

	rules, err := config.Load()
	if err != nil {
		rules = nil
	}

	if err := ui.Run(e, rules); err != nil {
		panic(err)
	}
}
