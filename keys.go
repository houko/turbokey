//go:build windows

package main

import "fmt"

// keyDef describes one selectable key: display name, virtual-key code, and
// whether it is an "extended" key (needs KEYEVENTF_EXTENDEDKEY when injected).
type keyDef struct {
	Name string
	VK   uint16
	Ext  bool
}

var keyDefs []keyDef

// keyNames is the ordered list of display names shown in combo boxes.
var keyNames []string

var nameToVK = map[string]uint16{}
var vkToName = map[uint16]string{}
var vkExtended = map[uint16]bool{}

func init() {
	// Letters A-Z (VK 0x41-0x5A).
	for c := byte('A'); c <= 'Z'; c++ {
		keyDefs = append(keyDefs, keyDef{Name: string(c), VK: uint16(c)})
	}
	// Digits 0-9 (VK 0x30-0x39).
	for c := byte('0'); c <= '9'; c++ {
		keyDefs = append(keyDefs, keyDef{Name: string(c), VK: uint16(c)})
	}
	// Function keys F1-F12 (VK 0x70-0x7B).
	for i := 0; i < 12; i++ {
		keyDefs = append(keyDefs, keyDef{Name: fmt.Sprintf("F%d", i+1), VK: uint16(0x70 + i)})
	}
	// Common control / whitespace keys.
	keyDefs = append(keyDefs,
		keyDef{Name: "空格", VK: 0x20},
		keyDef{Name: "回车", VK: 0x0D},
		keyDef{Name: "ESC", VK: 0x1B},
		keyDef{Name: "Tab", VK: 0x09},
		// Arrow keys are extended.
		keyDef{Name: "↑", VK: 0x26, Ext: true},
		keyDef{Name: "↓", VK: 0x28, Ext: true},
		keyDef{Name: "←", VK: 0x25, Ext: true},
		keyDef{Name: "→", VK: 0x27, Ext: true},
		keyDef{Name: "Ctrl", VK: 0x11},
		keyDef{Name: "Alt", VK: 0x12},
		keyDef{Name: "Shift", VK: 0x10},
	)

	for _, k := range keyDefs {
		keyNames = append(keyNames, k.Name)
		nameToVK[k.Name] = k.VK
		vkToName[k.VK] = k.Name
		vkExtended[k.VK] = k.Ext
	}
}

func vkName(vk uint16) string {
	if n, ok := vkToName[vk]; ok {
		return n
	}
	return fmt.Sprintf("0x%X", vk)
}

func isExtended(vk uint16) bool {
	return vkExtended[vk]
}
