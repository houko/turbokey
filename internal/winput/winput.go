//go:build windows

// Package winput wraps the Win32 calls needed for global keyboard hooking and
// synthetic key injection (SendInput with scan codes).
package winput

import (
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// WM_* keyboard message identifiers delivered to a low-level keyboard hook.
const (
	WMKeyDown    = 0x0100
	WMKeyUp      = 0x0101
	WMSysKeyDown = 0x0104
	WMSysKeyUp   = 0x0105
)

// Magic tags every event we synthesize, so the hook can recognize and pass our
// own injected keys through instead of re-triggering on them.
const Magic uintptr = 0x1F2A3B4C

const (
	whKeyboardLL         = 13
	inputKeyboard        = 1
	keyeventfExtendedKey = 0x0001
	keyeventfKeyUp       = 0x0002
	keyeventfScancode    = 0x0008
	mapvkVKToVSC         = 0
)

// KBDLLHOOKSTRUCT mirrors the Win32 struct passed to a WH_KEYBOARD_LL proc.
type KBDLLHOOKSTRUCT struct {
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
	user32                = windows.NewLazySystemDLL("user32.dll")
	procSetWindowsHookExW = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx    = user32.NewProc("CallNextHookEx")
	procGetMessageW       = user32.NewProc("GetMessageW")
	procTranslateMessage  = user32.NewProc("TranslateMessage")
	procDispatchMessageW  = user32.NewProc("DispatchMessageW")
	procSendInput         = user32.NewProc("SendInput")
	procMapVirtualKeyW    = user32.NewProc("MapVirtualKeyW")

	winmm               = windows.NewLazySystemDLL("winmm.dll")
	procTimeBeginPeriod = winmm.NewProc("timeBeginPeriod")

	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	procGetForegroundWindow        = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId   = user32.NewProc("GetWindowThreadProcessId")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
)

const processQueryLimitedInformation = 0x1000

// ForegroundProcessName returns the lowercase executable base name of the process
// owning the current foreground window (e.g. "dnfgame.exe"), or "" on failure.
func ForegroundProcessName() string {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return ""
	}
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return ""
	}
	h, _, _ := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer procCloseHandle.Call(h)
	buf := make([]uint16, 260)
	n := uint32(len(buf))
	r, _, _ := procQueryFullProcessImageNameW.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
	if r == 0 {
		return ""
	}
	return strings.ToLower(filepath.Base(windows.UTF16ToString(buf[:n])))
}

func moduleHandle() uintptr {
	h, _, _ := procGetModuleHandleW.Call(0)
	return h
}

// ScanCode maps a virtual-key code to its hardware scan code.
func ScanCode(vk uint16) uint16 {
	r, _, _ := procMapVirtualKeyW.Call(uintptr(vk), mapvkVKToVSC)
	return uint16(r)
}

// SendKeyEvent injects a single scancode key event (down or up), tagged with
// Magic so our own hook recognizes and passes it through. Down and up are sent
// separately so the caller can hold the key down long enough for poll-based
// games to sample it.
func SendKeyEvent(sc uint16, extended, keyUp bool) {
	flags := uint32(keyeventfScancode)
	if extended {
		flags |= keyeventfExtendedKey
	}
	if keyUp {
		flags |= keyeventfKeyUp
	}
	var in input
	in.Type = inputKeyboard
	in.Ki = keybdinput{WScan: sc, DwFlags: flags, DwExtraInfo: Magic}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}

// CallNext passes an event down the hook chain.
func CallNext(nCode, wParam, lParam uintptr) uintptr {
	r, _, _ := procCallNextHookEx.Call(0, nCode, wParam, lParam)
	return r
}

// BeginHighResTimer raises the system timer resolution to 1ms so short repeat
// intervals are honored.
func BeginHighResTimer() {
	procTimeBeginPeriod.Call(1)
}

// HookProc is a WH_KEYBOARD_LL callback: return nonzero to swallow the key.
type HookProc func(nCode, wParam, lParam uintptr) uintptr

// InstallKeyboardHook installs a global low-level keyboard hook and runs the
// message loop. It locks the OS thread and blocks, so run it in its own goroutine.
func InstallKeyboardHook(proc HookProc) {
	runtime.LockOSThread()

	cb := windows.NewCallback(proc)
	procSetWindowsHookExW.Call(whKeyboardLL, cb, moduleHandle(), 0)

	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}
