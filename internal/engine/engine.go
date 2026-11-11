//go:build windows

// Package engine drives the rapid-fire logic: a global keyboard hook decides
// which presses to intercept and spawns per-rule workers that inject keystrokes.
package engine

import (
	"strings"
	"sync"
	"sync/atomic"
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
	byTrigger map[uint16][]*config.Rule // rules by trigger VK; several may share a key with different modifiers

	targets map[string]bool // lowercase exe names the tool acts in (empty = all apps)

	pressed map[uint16]bool                // trigger keys we currently own (consumed the press)
	engaged map[uint16]*config.Rule        // the rule each consumed press started or stopped
	workers map[*config.Rule]chan struct{} // active repeat workers, keyed by rule

	// modKeys holds the modifier keys physically held (left and right apart, as the hook reports them); held is the set they make, read by the repeat workers without the lock. Our own injected events never reach dispatch, so this is always the user's real state.
	modKeys map[uint16]bool
	held    atomic.Uint32

	// OnMasterChange is invoked when the master switch flips via the global
	// hotkey, so the UI can reflect the new state. May be nil.
	OnMasterChange func(bool)

	// output sends one key/mouse event; foreground returns the foreground exe.
	// Both are injectable so the dispatch logic can be unit-tested without syscalls.
	output     func(vk uint16, up bool)
	foreground func() string
}

// defaultOutput injects a real key or mouse event for the given virtual-key code.
func defaultOutput(vk uint16, up bool) {
	if keys.IsMouse(vk) {
		winput.SendMouseEvent(vk, up)
		return
	}
	winput.SendKeyEvent(winput.ScanCode(vk), keys.IsExtended(vk), up)
}

// New creates an engine whose master switch is toggled by masterHotkeyVK.
func New(masterHotkeyVK uint16) *Engine {
	return &Engine{
		masterHotkeyVK: masterHotkeyVK,
		byTrigger:      map[uint16][]*config.Rule{},
		targets:        map[string]bool{},
		pressed:        map[uint16]bool{},
		engaged:        map[uint16]*config.Rule{},
		workers:        map[*config.Rule]chan struct{}{},
		modKeys:        map[uint16]bool{},
		output:         defaultOutput,
		foreground:     winput.ForegroundProcessName,
	}
}

// SetTargets restricts rapid-fire to the given process exe names (case-insensitive).
// An empty list means it applies in every application.
func (e *Engine) SetTargets(names []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.targets = make(map[string]bool, len(names))
	for _, n := range names {
		if n = strings.ToLower(strings.TrimSpace(n)); n != "" {
			e.targets[n] = true
		}
	}
}

// targetActiveLocked reports whether the foreground app is one we should act in.
func (e *Engine) targetActiveLocked() bool {
	if len(e.targets) == 0 {
		return true
	}
	name := e.foreground()
	return name != "" && e.targets[name]
}

// SetRules replaces the active rule set. Any running workers are stopped, since
// their parameters may have changed.
func (e *Engine) SetRules(rules []*config.Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stopAllLocked() // also clears e.pressed
	e.rules = rules
	e.byTrigger = make(map[uint16][]*config.Rule, len(rules))
	for _, r := range rules {
		e.byTrigger[r.TriggerVK] = append(e.byTrigger[r.TriggerVK], r)
	}
}

// modOf maps a modifier key, generic or left/right-specific, to its modifier; 0 for any other key.
func modOf(vk uint16) config.Mods {
	switch vk {
	case 0x11, 0xA2, 0xA3: // VK_CONTROL, VK_LCONTROL, VK_RCONTROL
		return config.ModCtrl
	case 0x10, 0xA0, 0xA1: // VK_SHIFT, VK_LSHIFT, VK_RSHIFT
		return config.ModShift
	case 0x12, 0xA4, 0xA5: // VK_MENU, VK_LMENU, VK_RMENU
		return config.ModAlt
	case 0x5B, 0x5C: // VK_LWIN, VK_RWIN
		return config.ModWin
	}
	return 0
}

// modVK is the key injected for a modifier a rule needs but the user is not holding.
var modVK = map[config.Mods]uint16{config.ModCtrl: 0x11, config.ModShift: 0x10, config.ModAlt: 0x12, config.ModWin: keys.VKLWin}

// trackModLocked records a modifier key going down or up.
func (e *Engine) trackModLocked(vk uint16, down, up bool) {
	if modOf(vk) == 0 {
		return
	}
	if down {
		e.modKeys[vk] = true
	} else if up {
		delete(e.modKeys, vk)
	}
	var m config.Mods
	for k := range e.modKeys {
		m |= modOf(k)
	}
	e.held.Store(uint32(m))
}

func (e *Engine) heldMods() config.Mods { return config.Mods(e.held.Load()) }

// ruleForLocked picks the enabled rule a press of vk fires with the modifiers held now: one whose modifiers are exactly those held, else one without modifiers, else none.
func (e *Engine) ruleForLocked(vk uint16) *config.Rule {
	held := e.heldMods()
	var plain *config.Rule
	for _, r := range e.byTrigger[vk] {
		if !r.Enabled {
			continue
		}
		if r.Modifiers == 0 {
			if plain == nil {
				plain = r
			}
		} else if r.Modifiers == held {
			return r
		}
	}
	return plain
}

// SetMasterHotkey changes the hotkey used to toggle the master switch.
func (e *Engine) SetMasterHotkey(vk uint16) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.masterHotkeyVK = vk
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
	// Reset consumed-press tracking too. Otherwise a key whose down we consumed
	// stays "pressed" forever once its worker is stopped (e.g. master turned off
	// mid-press), and the next real keydown is misread as an auto-repeat.
	e.pressed = map[uint16]bool{}
	e.engaged = map[uint16]*config.Rule{}
}

func (e *Engine) startRepeatLocked(r *config.Rule) {
	if _, running := e.workers[r]; running {
		return
	}
	interval := r.IntervalMs
	if interval < 1 {
		interval = 1
	}
	out := r.EffectiveOutputVK()
	output := e.output
	need := r.Modifiers
	// Each pulse is sent as the rule's combination: a modifier the rule needs but the user is not holding (in toggle mode they may have let go) is pressed around the key and released after it, in reverse order; one the user holds is left alone.
	var added []config.Mods
	down := func() {
		added = added[:0]
		missing := need &^ e.heldMods()
		for _, m := range []config.Mods{config.ModCtrl, config.ModShift, config.ModAlt, config.ModWin} {
			if missing&m != 0 {
				output(modVK[m], false)
				added = append(added, m)
			}
		}
		output(out, false)
	}
	up := func() {
		output(out, true)
		for i := len(added) - 1; i >= 0; i-- {
			output(modVK[added[i]], true)
		}
	}
	stop := make(chan struct{})
	e.workers[r] = stop
	go repeatWorker(down, up, interval, stop)
}

func (e *Engine) stopRepeatLocked(r *config.Rule) {
	if stop, ok := e.workers[r]; ok {
		close(stop)
		delete(e.workers, r)
	}
}

func (e *Engine) isRepeatingLocked(r *config.Rule) bool {
	_, ok := e.workers[r]
	return ok
}

// keyHoldMs is how long each synthetic key stays down before release. A
// poll-based game (which samples key state per frame) never observes a press
// whose down and up land between two samples, so the key must be held across at
// least one frame. ~30ms safely covers 60fps and below.
const keyHoldMs = 30

// repeatWorker presses the output (down, hold, up), waits one interval, and
// repeats until stopped. Stopping during the hold still releases it, so an
// output is never left stuck down.
func repeatWorker(down, up func(), intervalMs int, stop chan struct{}) {
	hold := time.Duration(keyHoldMs) * time.Millisecond
	gap := time.Duration(intervalMs) * time.Millisecond
	for {
		down()
		select {
		case <-stop:
			up() // release on stop
			return
		case <-time.After(hold):
		}
		up()
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
	go winput.InstallHooks(e.keyboardProc, e.mouseProc)
}

// keyboardProc is the WH_KEYBOARD_LL callback.
func (e *Engine) keyboardProc(nCode, wParam, lParam uintptr) uintptr {
	if int32(nCode) < 0 {
		return winput.CallNext(nCode, wParam, lParam)
	}
	kb := (*winput.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
	if kb.DwExtraInfo == winput.Magic { // our own synthesized input
		return winput.CallNext(nCode, wParam, lParam)
	}
	vk := uint16(kb.VkCode)
	down := wParam == winput.WMKeyDown || wParam == winput.WMSysKeyDown
	up := wParam == winput.WMKeyUp || wParam == winput.WMSysKeyUp
	if e.dispatch(vk, down, up) {
		return 1
	}
	return winput.CallNext(nCode, wParam, lParam)
}

// mouseProc is the WH_MOUSE_LL callback, mapping mouse buttons to the same rules.
func (e *Engine) mouseProc(nCode, wParam, lParam uintptr) uintptr {
	if int32(nCode) < 0 {
		return winput.CallNext(nCode, wParam, lParam)
	}
	ms := (*winput.MSLLHOOKSTRUCT)(unsafe.Pointer(lParam))
	if ms.DwExtraInfo == winput.Magic { // our own synthesized input
		return winput.CallNext(nCode, wParam, lParam)
	}
	var vk uint16
	var down, up bool
	switch wParam {
	case winput.WMLButtonDown:
		vk, down = keys.VKMouseLeft, true
	case winput.WMLButtonUp:
		vk, up = keys.VKMouseLeft, true
	case winput.WMRButtonDown:
		vk, down = keys.VKMouseRight, true
	case winput.WMRButtonUp:
		vk, up = keys.VKMouseRight, true
	case winput.WMMButtonDown:
		vk, down = keys.VKMouseMiddle, true
	case winput.WMMButtonUp:
		vk, up = keys.VKMouseMiddle, true
	case winput.WMXButtonDown:
		vk, down = xbuttonVK(ms.XButton()), true
	case winput.WMXButtonUp:
		vk, up = xbuttonVK(ms.XButton()), true
	default:
		return winput.CallNext(nCode, wParam, lParam)
	}
	if e.dispatch(vk, down, up) {
		return 1
	}
	return winput.CallNext(nCode, wParam, lParam)
}

func xbuttonVK(xb uint32) uint16 {
	if xb == 2 {
		return keys.VKMouseX2
	}
	return keys.VKMouseX1
}

// dispatch applies a trigger event (from the keyboard or mouse hook) and reports
// whether the event should be swallowed. e.pressed[vk] marks a press we have
// consumed; triggers are only consumed in a target app, and once consumed the
// matching release and any auto-repeat are consumed too so nothing leaks elsewhere.
func (e *Engine) dispatch(vk uint16, down, up bool) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Modifier state is tracked whatever the master switch says, so it is right the moment it is turned on.
	e.trackModLocked(vk, down, up)

	// Master hotkey works regardless of master state.
	if vk == e.masterHotkeyVK {
		if down && !e.pressed[vk] {
			e.pressed[vk] = true
			e.setMasterLocked(!e.master, true)
		} else if up {
			delete(e.pressed, vk)
		}
		return true
	}

	if !e.master {
		return false
	}

	// The release, and any auto-repeat, of a press we consumed is ours whatever the modifiers are by then: the rule chosen at the press is the one engaged.
	if e.pressed[vk] {
		if up {
			delete(e.pressed, vk)
			if r := e.engaged[vk]; r != nil && r.Mode == config.ModeHold {
				e.stopRepeatLocked(r)
			}
			delete(e.engaged, vk)
		}
		return true
	}
	if !down {
		return false
	}

	rule := e.ruleForLocked(vk)
	if rule == nil {
		return false
	}

	switch rule.Mode {
	case config.ModeHold:
		if !e.targetActiveLocked() {
			return false // wrong app: input works normally
		}
		e.pressed[vk] = true
		e.engaged[vk] = rule
		e.startRepeatLocked(rule)
		return true
	case config.ModeToggle:
		// New press: stop from anywhere, but only start in a target app.
		running := e.isRepeatingLocked(rule)
		if !running && !e.targetActiveLocked() {
			return false
		}
		e.pressed[vk] = true
		e.engaged[vk] = rule
		if running {
			e.stopRepeatLocked(rule)
		} else {
			e.startRepeatLocked(rule)
		}
		return true
	}
	return false
}
