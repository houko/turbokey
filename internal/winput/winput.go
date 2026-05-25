//go:build windows

// Package winput wraps the Win32 calls needed for global keyboard hooking and
// synthetic key injection (SendInput with scan codes).
package winput

import (
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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

// WM_* mouse message identifiers delivered to a low-level mouse hook.
const (
	WMLButtonDown = 0x0201
	WMLButtonUp   = 0x0202
	WMRButtonDown = 0x0204
	WMRButtonUp   = 0x0205
	WMMButtonDown = 0x0207
	WMMButtonUp   = 0x0208
	WMXButtonDown = 0x020B
	WMXButtonUp   = 0x020C
)

// Magic tags every event we synthesize, so the hook can recognize and pass our
// own injected keys through instead of re-triggering on them.
const Magic uintptr = 0x1F2A3B4C

const (
	whKeyboardLL         = 13
	whMouseLL            = 14
	inputKeyboard        = 1
	inputMouse           = 0
	keyeventfExtendedKey = 0x0001
	keyeventfKeyUp       = 0x0002
	keyeventfScancode    = 0x0008
	mapvkVKToVSC         = 0

	mouseeventfLeftDown   = 0x0002
	mouseeventfLeftUp     = 0x0004
	mouseeventfRightDown  = 0x0008
	mouseeventfRightUp    = 0x0010
	mouseeventfMiddleDown = 0x0020
	mouseeventfMiddleUp   = 0x0040
	mouseeventfXDown      = 0x0080
	mouseeventfXUp        = 0x0100
	xbutton1              = 0x0001
	xbutton2              = 0x0002

	eventSystemForeground = 0x0003
	wineventOutOfContext  = 0x0000
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

// MSLLHOOKSTRUCT mirrors the Win32 struct passed to a WH_MOUSE_LL proc.
type MSLLHOOKSTRUCT struct {
	Pt          struct{ X, Y int32 }
	MouseData   uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

// XButton returns 1 (XBUTTON1) or 2 (XBUTTON2) from a mouse hook struct.
func (m *MSLLHOOKSTRUCT) XButton() uint32 { return m.MouseData >> 16 & 0xFFFF }

// mouseinput mirrors the Win32 MOUSEINPUT structure.
type mouseinput struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// mouseInput is the Win32 INPUT structure with the MOUSEINPUT variant (40 bytes
// on amd64, same union slot as the keyboard variant).
type mouseInput struct {
	Type uint32
	_    uint32
	Mi   mouseinput
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
	procEnumWindows                = user32.NewProc("EnumWindows")
	procIsWindowVisible            = user32.NewProc("IsWindowVisible")
	procGetWindowTextW             = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW       = user32.NewProc("GetWindowTextLengthW")
	procGetWindowLongPtrW          = user32.NewProc("GetWindowLongPtrW")
	procSetWinEventHook            = user32.NewProc("SetWinEventHook")
)

const (
	processQueryLimitedInformation = 0x1000
	gwlExStyle                     = ^uintptr(0) - 19 // GWL_EXSTYLE (-20)
	wsExToolWindow                 = 0x00000080
)

// processExeName returns the lowercase executable base name for a process id.
func processExeName(pid uint32) string {
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

// fgCache holds the foreground process exe name, refreshed by a WinEvent hook so
// the keyboard/mouse hooks can read it without a syscall on every keystroke.
var fgCache atomic.Value // string

// ForegroundProcessName returns the cached lowercase exe name of the foreground
// window's process (e.g. "dnfgame.exe"), or "" if unknown.
func ForegroundProcessName() string {
	if v, ok := fgCache.Load().(string); ok {
		return v
	}
	return ""
}

func queryForeground() string {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return ""
	}
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return processExeName(pid)
}

// winEventProc updates the foreground cache whenever the foreground window changes.
func winEventProc(_, _, hwnd, _, _, _, _ uintptr) uintptr {
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	fgCache.Store(processExeName(pid))
	return 0
}

var winEventCb = windows.NewCallback(winEventProc)

// WindowInfo is one entry in the running-apps picker: a window title and the
// owning process's executable name.
type WindowInfo struct {
	Title string
	Exe   string
}

// One reusable EnumWindows callback (NewCallback allocations are never freed, so
// we must not create one per call). enumAcc is guarded by enumMu.
var (
	enumMu  sync.Mutex
	enumAcc []WindowInfo
	enumCb  = windows.NewCallback(enumProc)
)

func enumProc(hwnd, _ uintptr) uintptr {
	if r, _, _ := procIsWindowVisible.Call(hwnd); r == 0 {
		return 1
	}
	if n, _, _ := procGetWindowTextLengthW.Call(hwnd); n == 0 {
		return 1
	}
	if ex, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlExStyle); ex&wsExToolWindow != 0 {
		return 1 // skip tool windows
	}
	buf := make([]uint16, 256)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	title := windows.UTF16ToString(buf)
	if title == "" {
		return 1
	}
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	exe := processExeName(pid)
	if exe == "" || exe == "turbokey.exe" {
		return 1
	}
	enumAcc = append(enumAcc, WindowInfo{Title: title, Exe: exe})
	return 1
}

// VisibleWindows lists visible top-level windows that have a title, one entry per
// distinct owning executable, for the running-apps picker.
func VisibleWindows() []WindowInfo {
	enumMu.Lock()
	defer enumMu.Unlock()
	enumAcc = nil
	procEnumWindows.Call(enumCb, 0)
	seen := map[string]bool{}
	var out []WindowInfo
	for _, w := range enumAcc {
		if seen[w.Exe] {
			continue
		}
		seen[w.Exe] = true
		out = append(out, w)
	}
	enumAcc = nil
	return out
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

// SendMouseEvent injects a single mouse button event (down or up) at the current
// cursor position, tagged with Magic so our own hooks pass it through.
func SendMouseEvent(vk uint16, keyUp bool) {
	var flags, data uint32
	switch vk {
	case 0x01: // left
		flags = mouseeventfLeftDown
		if keyUp {
			flags = mouseeventfLeftUp
		}
	case 0x02: // right
		flags = mouseeventfRightDown
		if keyUp {
			flags = mouseeventfRightUp
		}
	case 0x04: // middle
		flags = mouseeventfMiddleDown
		if keyUp {
			flags = mouseeventfMiddleUp
		}
	case 0x05, 0x06: // X1 / X2
		flags = mouseeventfXDown
		if keyUp {
			flags = mouseeventfXUp
		}
		if vk == 0x05 {
			data = xbutton1
		} else {
			data = xbutton2
		}
	default:
		return
	}
	var in mouseInput
	in.Type = inputMouse
	in.Mi = mouseinput{MouseData: data, DwFlags: flags, DwExtraInfo: Magic}
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

// HookProc is a low-level hook callback: return nonzero to swallow the event.
type HookProc func(nCode, wParam, lParam uintptr) uintptr

// InstallHooks installs the global low-level keyboard and mouse hooks plus a
// foreground-window WinEvent hook, then runs the message loop. It locks the OS
// thread and blocks, so run it in its own goroutine.
func InstallHooks(keyboard, mouse HookProc) {
	runtime.LockOSThread()

	hMod := moduleHandle()
	procSetWindowsHookExW.Call(whKeyboardLL, windows.NewCallback(keyboard), hMod, 0)
	procSetWindowsHookExW.Call(whMouseLL, windows.NewCallback(mouse), hMod, 0)
	procSetWinEventHook.Call(eventSystemForeground, eventSystemForeground, 0, winEventCb, 0, 0, wineventOutOfContext)
	fgCache.Store(queryForeground())

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
