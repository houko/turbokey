//go:build windows

// Package engine drives the rapid-fire logic: a global keyboard hook decides
// which presses to intercept and spawns per-rule workers that inject keystrokes.
package engine

import (
	"sync"
	"time"
	"unsafe"

	"turbokey/internal/config"
	"turbokey/internal/keys"
	"turbokey/internal/winput"
)

// Engine owns the keyboard hook, the active rule set, and the per-rule repeat
// workers. The hook callback runs on a dedicated OS thread with its own message
// loop; all shared state is guarded by mu.
type Engine struct {
	mu sync.Mutex

	master         bool
	masterHotkeyVK uint16
	started        bool

	rules     []*config.Rule
	byTrigger map[uint16]*config.Rule

	pressed map[uint16]bool          // trigger keys currently held (debounces OS auto-repeat)
	workers map[uint16]chan struct{} // active repeat workers, keyed by trigger VK

	// OnMasterChange is invoked when the master switch flips via the global
	// hotkey, so the UI can reflect the new state. May be nil.
	OnMasterChange func(bool)
}

// New creates an engine whose master switch is toggled by masterHotkeyVK.
func New(masterHotkeyVK uint16) *Engine {
	return &Engine{
		masterHotkeyVK: masterHotkeyVK,
		byTrigger:      map[uint16]*config.Rule{},
		pressed:        map[uint16]bool{},
		workers:        map[uint16]chan struct{}{},
	}
}

// SetRules replaces the active rule set. Any running workers are stopped, since
// their parameters may have changed.
func (e *Engine) SetRules(rules []*config.Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stopAllLocked()
	e.pressed = map[uint16]bool{}
	e.rules = rules
	e.byTrigger = make(map[uint16]*config.Rule, len(rules))
	for _, r := range rules {
		e.byTrigger[r.TriggerVK] = r
	}
}

// SetMaster sets the master switch from the UI (no callback fired back).
func (e *Engine) SetMaster(on bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.setMasterLocked(on, false)
}

func (e *Engine) setMasterLocked(on, notify bool) {
	if e.master == on {
		return
	}
	e.master = on
	if !on {
		e.stopAllLocked()
	}
	if notify && e.OnMasterChange != nil {
		e.OnMasterChange(on)
	}
}

func (e *Engine) stopAllLocked() {
	for vk, stop := range e.workers {
		close(stop)
		delete(e.workers, vk)
	}
}

func (e *Engine) startRepeatLocked(r *config.Rule) {
	if _, running := e.workers[r.TriggerVK]; running {
		return
	}
	sc := winput.ScanCode(r.EffectiveOutputVK())
	ext := keys.IsExtended(r.EffectiveOutputVK())
	interval := r.IntervalMs
	if interval < 1 {
		interval = 1
	}
	stop := make(chan struct{})
	e.workers[r.TriggerVK] = stop
	go repeatWorker(sc, ext, interval, stop)
}

func (e *Engine) stopRepeatLocked(triggerVK uint16) {
	if stop, ok := e.workers[triggerVK]; ok {
		close(stop)
		delete(e.workers, triggerVK)
	}
}

func (e *Engine) isRepeatingLocked(triggerVK uint16) bool {
	_, ok := e.workers[triggerVK]
	return ok
}

// keyHoldMs is how long each synthetic key stays down before release. A
// poll-based game (which samples key state per frame) never observes a press
// whose down and up land between two samples, so the key must be held across at
// least one frame. ~30ms safely covers 60fps and below.
const keyHoldMs = 30

// repeatWorker presses the output key (down, hold, up), waits one interval, and
// repeats until stopped. Stopping during the hold still releases the key, so a
// key is never left stuck down.
func repeatWorker(sc uint16, ext bool, intervalMs int, stop chan struct{}) {
	hold := time.Duration(keyHoldMs) * time.Millisecond
	gap := time.Duration(intervalMs) * time.Millisecond
	for {
		winput.SendKeyEvent(sc, ext, false) // down
		select {
		case <-stop:
			winput.SendKeyEvent(sc, ext, true) // release on stop
			return
		case <-time.After(hold):
		}
		winput.SendKeyEvent(sc, ext, true) // up
		select {
		case <-stop:
			return
		case <-time.After(gap):
		}
	}
}

// Start raises the system timer resolution and installs the keyboard hook on a
// dedicated thread. It is idempotent.
func (e *Engine) Start() {
	e.mu.Lock()
	if e.started {
		e.mu.Unlock()
		return
	}
	e.started = true
	e.mu.Unlock()

	winput.BeginHighResTimer()
	go winput.InstallKeyboardHook(e.hookProc)
}

// hookProc is the WH_KEYBOARD_LL callback. Return 1 to swallow the key, or pass
// it down the hook chain otherwise.
func (e *Engine) hookProc(nCode, wParam, lParam uintptr) uintptr {
	if int32(nCode) < 0 {
		return winput.CallNext(nCode, wParam, lParam)
	}
	kb := (*winput.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))

	// Our own synthesized input: never act on it.
	if kb.DwExtraInfo == winput.Magic {
		return winput.CallNext(nCode, wParam, lParam)
	}

	vk := uint16(kb.VkCode)
	down := wParam == winput.WMKeyDown || wParam == winput.WMSysKeyDown
	up := wParam == winput.WMKeyUp || wParam == winput.WMSysKeyUp

	e.mu.Lock()
	defer e.mu.Unlock()

	// Master hotkey works regardless of master state.
	if vk == e.masterHotkeyVK {
		if down && !e.pressed[vk] {
			e.pressed[vk] = true
			e.setMasterLocked(!e.master, true)
		} else if up {
			delete(e.pressed, vk)
		}
		return 1
	}

	if !e.master {
		return winput.CallNext(nCode, wParam, lParam)
	}

	rule, ok := e.byTrigger[vk]
	if !ok || !rule.Enabled {
		return winput.CallNext(nCode, wParam, lParam)
	}

	switch rule.Mode {
	case config.ModeHold:
		if down && !e.pressed[vk] {
			e.pressed[vk] = true
			e.startRepeatLocked(rule)
		} else if up {
			delete(e.pressed, vk)
			e.stopRepeatLocked(rule.TriggerVK)
		}
		return 1
	case config.ModeToggle:
		if down && !e.pressed[vk] {
			e.pressed[vk] = true
			if e.isRepeatingLocked(rule.TriggerVK) {
				e.stopRepeatLocked(rule.TriggerVK)
			} else {
				e.startRepeatLocked(rule)
			}
		} else if up {
			delete(e.pressed, vk)
		}
		return 1
	}
	return winput.CallNext(nCode, wParam, lParam)
}
