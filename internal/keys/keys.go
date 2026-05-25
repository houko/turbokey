//go:build windows

// Package keys holds the table of selectable keys and conversions between
// display names and Win32 virtual-key codes.
package keys

import "fmt"

// keyDef describes one selectable input: display name, virtual-key code, whether
// it is an "extended" key (needs KEYEVENTF_EXTENDEDKEY when injected), and whether
// it is a mouse button rather than a keyboard key.
type keyDef struct {
	name  string
	vk    uint16
	ext   bool
	mouse bool
}

// Names is the ordered list of display names shown in combo boxes.
var Names []string

// KeyboardNames is Names filtered to keyboard keys (no mouse buttons), for the
// master-hotkey picker.
var KeyboardNames []string

// Mouse-button virtual-key codes (same values Windows uses for VK_*BUTTON).
const (
	VKMouseLeft   = 0x01
	VKMouseRight  = 0x02
	VKMouseMiddle = 0x04
	VKMouseX1     = 0x05
	VKMouseX2     = 0x06
)

var (
	defs       []keyDef
	nameToVK   = map[string]uint16{}
	vkToName   = map[uint16]string{}
	vkExtended = map[uint16]bool{}
	vkMouse    = map[uint16]bool{}
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
		// Mouse buttons (language-neutral names; also persisted to config).
		keyDef{name: "Mouse Left", vk: VKMouseLeft, mouse: true},
		keyDef{name: "Mouse Right", vk: VKMouseRight, mouse: true},
		keyDef{name: "Mouse Mid", vk: VKMouseMiddle, mouse: true},
		keyDef{name: "Mouse X1", vk: VKMouseX1, mouse: true},
		keyDef{name: "Mouse X2", vk: VKMouseX2, mouse: true},
	)

	for _, k := range defs {
		Names = append(Names, k.name)
		nameToVK[k.name] = k.vk
		vkToName[k.vk] = k.name
		vkExtended[k.vk] = k.ext
		vkMouse[k.vk] = k.mouse
		if !k.mouse {
			KeyboardNames = append(KeyboardNames, k.name)
		}
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

// IsMouse reports whether a virtual-key code denotes a mouse button.
func IsMouse(vk uint16) bool {
	return vkMouse[vk]
}

// Index returns the position of a key in Names, or -1 if unknown.
func Index(vk uint16) int {
	name, ok := vkToName[vk]
	if !ok {
		return -1
	}
	for i, n := range Names {
		if n == name {
			return i
		}
	}
	return -1
}
