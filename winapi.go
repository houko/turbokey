//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// Win32 constants used by the keyboard hook and SendInput.
const (
	whKeyboardLL = 13

	wmKeyDown    = 0x0100
	wmKeyUp      = 0x0101
	wmSysKeyDown = 0x0104
	wmSysKeyUp   = 0x0105

	inputKeyboard = 1

	keyeventfExtendedKey = 0x0001
	keyeventfKeyUp       = 0x0002
	keyeventfScancode    = 0x0008

	mapvkVKToVSC = 0
)

// magicExtra tags every event we synthesize, so our own low-level hook can
// recognize and pass them through instead of re-triggering on them.
const magicExtra uintptr = 0x1F2A3B4C

// kbdllhookstruct mirrors the Win32 KBDLLHOOKSTRUCT passed to a WH_KEYBOARD_LL proc.
type kbdllhookstruct struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

// keybdinput mirrors the Win32 KEYBDINPUT structure.
type keybdinput struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// input mirrors the Win32 INPUT structure. On amd64 the union is 8-byte aligned
// (offset 8) and the whole struct is 40 bytes; the trailing pad makes the
// keyboard variant fill the larger MOUSEINPUT union slot.
type input struct {
	Type uint32
	_    uint32
	Ki   keybdinput
	_    [8]byte
}

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procSendInput           = user32.NewProc("SendInput")
	procMapVirtualKeyW      = user32.NewProc("MapVirtualKeyW")

	winmm               = windows.NewLazySystemDLL("winmm.dll")
	procTimeBeginPeriod = winmm.NewProc("timeBeginPeriod")

	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
)

// moduleHandle returns the base handle of the current executable.
func moduleHandle() uintptr {
	h, _, _ := procGetModuleHandleW.Call(0)
	return h
}

// scanCode maps a virtual-key code to its hardware scan code.
func scanCode(vk uint16) uint16 {
	r, _, _ := procMapVirtualKeyW.Call(uintptr(vk), mapvkVKToVSC)
	return uint16(r)
}

// sendKeyEvent injects a single scancode key event (down or up), tagged with
// magicExtra so our own hook recognizes and passes it through. Down and up are
// sent separately (not as one atomic pair) so the caller can hold the key down
// long enough for poll-based games to sample it.
func sendKeyEvent(sc uint16, extended, keyUp bool) {
	flags := uint32(keyeventfScancode)
	if extended {
		flags |= keyeventfExtendedKey
	}
	if keyUp {
		flags |= keyeventfKeyUp
	}
	var in input
	in.Type = inputKeyboard
	in.Ki = keybdinput{WScan: sc, DwFlags: flags, DwExtraInfo: magicExtra}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}

func callNextHook(nCode, wParam, lParam uintptr) uintptr {
	r, _, _ := procCallNextHookEx.Call(0, nCode, wParam, lParam)
	return r
}

func timeBeginPeriod(ms uint32) {
	procTimeBeginPeriod.Call(uintptr(ms))
}
