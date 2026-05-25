//go:build windows

// Package keys holds the table of selectable keys and conversions between
// display names and Win32 virtual-key codes.
package keys

import "fmt"

// keyDef describes one selectable key: display name, virtual-key code, and
// whether it is an "extended" key (needs KEYEVENTF_EXTENDEDKEY when injected).
type keyDef struct {
	name string
	vk   uint16
	ext  bool
}

// Names is the ordered list of display names shown in combo boxes.
var Names []string

var (
	defs       []keyDef
	nameToVK   = map[string]uint16{}
	vkToName   = map[uint16]string{}
	vkExtended = map[uint16]bool{}
)

func init() {
	// Letters A-Z (VK 0x41-0x5A).
	for c := byte('A'); c <= 'Z'; c++ {
		defs = append(defs, keyDef{name: string(c), vk: uint16(c)})
	}
	// Digits 0-9 (VK 0x30-0x39).
	for c := byte('0'); c <= '9'; c++ {
		defs = append(defs, keyDef{name: string(c), vk: uint16(c)})
	}
	// Function keys F1-F12 (VK 0x70-0x7B).
	for i := 0; i < 12; i++ {
		defs = append(defs, keyDef{name: fmt.Sprintf("F%d", i+1), vk: uint16(0x70 + i)})
	}
	// Common control / whitespace keys; arrow keys are extended. Names are kept
	// language-neutral (English / symbols) because they are also the stable
	// identifiers persisted in config.json.
	defs = append(defs,
		keyDef{name: "Space", vk: 0x20},
		keyDef{name: "Enter", vk: 0x0D},
		keyDef{name: "Esc", vk: 0x1B},
		keyDef{name: "Tab", vk: 0x09},
		keyDef{name: "↑", vk: 0x26, ext: true},
		keyDef{name: "↓", vk: 0x28, ext: true},
		keyDef{name: "←", vk: 0x25, ext: true},
		keyDef{name: "→", vk: 0x27, ext: true},
		keyDef{name: "Ctrl", vk: 0x11},
		keyDef{name: "Alt", vk: 0x12},
		keyDef{name: "Shift", vk: 0x10},
	)

	for _, k := range defs {
		Names = append(Names, k.name)
		nameToVK[k.name] = k.vk
		vkToName[k.vk] = k.name
		vkExtended[k.vk] = k.ext
	}
}

// VK returns the virtual-key code for a display name.
func VK(name string) (uint16, bool) {
	vk, ok := nameToVK[name]
	return vk, ok
}

// Name returns the display name for a virtual-key code.
func Name(vk uint16) string {
	if n, ok := vkToName[vk]; ok {
		return n
	}
	return fmt.Sprintf("0x%X", vk)
}

// IsExtended reports whether a key needs KEYEVENTF_EXTENDEDKEY when injected.
func IsExtended(vk uint16) bool {
	return vkExtended[vk]
}
