//go:build windows

// Command turbokey is a Windows key auto-fire (rapid-fire) tool with a native GUI.
package main

import (
	"runtime"

	"turbokey/internal/config"
	"turbokey/internal/engine"
	"turbokey/internal/i18n"
	"turbokey/internal/keys"
	"turbokey/internal/singleton"
	"turbokey/internal/ui"
)

func main() {
	// Pin the GUI to a single OS thread.
	runtime.LockOSThread()

	// Refuse to start a second instance; surface the running one instead.
	if !singleton.Acquire() {
		return
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = &config.File{}
	}

	i18n.Init(cfg.Lang)

	hotkey := uint16(0x77) // VK_F8 default
	if cfg.MasterHotkey != "" {
		if vk, ok := keys.VK(cfg.MasterHotkey); ok {
			hotkey = vk
		}
	}
	e := engine.New(hotkey)

	if err := ui.Run(e, cfg); err != nil {
		panic(err)
	}
}
