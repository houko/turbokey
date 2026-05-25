//go:build windows

package main

import (
	"runtime"
)

func main() {
	// Pin the GUI to a single OS thread.
	runtime.LockOSThread()

	engine := NewEngine(0x77) // VK_F8 master toggle hotkey

	rules, err := loadRules()
	if err != nil {
		rules = nil
	}

	if err := runUI(engine, rules); err != nil {
		panic(err)
	}
}
