//go:build windows

package engine

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"turbokey/internal/config"
)

type fakeOut struct {
	mu    sync.Mutex
	count int
}

func (f *fakeOut) fire(vk uint16, up bool) {
	f.mu.Lock()
	f.count++
	f.mu.Unlock()
}

// newTest builds an engine with fake output/foreground so dispatch can be driven
// without any real input injection. The returned *string sets the foreground exe.
func newTest(t *testing.T) (*Engine, *fakeOut, *string) {
	e := New(0x77)
	out := &fakeOut{}
	e.output = out.fire
	fg := ""
	e.foreground = func() string { return fg }
	t.Cleanup(func() { e.SetRules(nil) }) // stop any started workers
	return e, out, &fg
}

func active(e *Engine) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.workers)
}

func masterOn(e *Engine) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.master
}

func holdRule(vk uint16) *config.Rule {
	return &config.Rule{TriggerVK: vk, Mode: config.ModeHold, IntervalMs: 1000, Enabled: true}
}

func TestMasterHotkeyToggles(t *testing.T) {
	e, _, _ := newTest(t)
	if !e.dispatch(0x77, true, false) {
		t.Fatal("F8 down should be swallowed")
	}
	if !masterOn(e) {
		t.Fatal("F8 should enable master")
	}
	e.dispatch(0x77, false, true)
	if !e.dispatch(0x77, true, false) {
		t.Fatal("F8 down should be swallowed")
	}
	if masterOn(e) {
		t.Fatal("second F8 should disable master")
	}
}

func TestMasterOffPassesThrough(t *testing.T) {
	e, _, _ := newTest(t)
	e.SetRules([]*config.Rule{holdRule(0x4A)})
	if e.dispatch(0x4A, true, false) {
		t.Error("with master off, trigger should pass through")
	}
	if active(e) != 0 {
		t.Error("no worker should start with master off")
	}
}

func TestHoldStartsAndStops(t *testing.T) {
	e, _, _ := newTest(t)
	e.SetMaster(true)
	e.SetRules([]*config.Rule{holdRule(0x4A)})

	if !e.dispatch(0x4A, true, false) {
		t.Fatal("trigger down should be swallowed")
	}
	if active(e) != 1 {
		t.Fatalf("expected 1 worker, got %d", active(e))
	}
	// OS auto-repeat while held: swallowed, still one worker.
	if !e.dispatch(0x4A, true, false) {
		t.Error("auto-repeat should be swallowed")
	}
	if active(e) != 1 {
		t.Errorf("auto-repeat should not start another worker, got %d", active(e))
	}
	if !e.dispatch(0x4A, false, true) {
		t.Error("release should be swallowed")
	}
	if active(e) != 0 {
		t.Errorf("release should stop the worker, got %d", active(e))
	}
}

func TestToggleStartsThenStops(t *testing.T) {
	e, _, _ := newTest(t)
	e.SetMaster(true)
	e.SetRules([]*config.Rule{{TriggerVK: 0x4B, Mode: config.ModeToggle, IntervalMs: 1000, Enabled: true}})

	e.dispatch(0x4B, true, false)
	e.dispatch(0x4B, false, true)
	if active(e) != 1 {
		t.Fatalf("first toggle press should start firing, got %d", active(e))
	}
	e.dispatch(0x4B, true, false)
	e.dispatch(0x4B, false, true)
	if active(e) != 0 {
		t.Fatalf("second toggle press should stop firing, got %d", active(e))
	}
}

func TestTargetGating(t *testing.T) {
	e, _, fg := newTest(t)
	e.SetMaster(true)
	e.SetTargets([]string{"game.exe"})
	e.SetRules([]*config.Rule{holdRule(0x4A)})

	*fg = "other.exe"
	if e.dispatch(0x4A, true, false) {
		t.Error("outside target app, trigger should pass through")
	}
	if active(e) != 0 {
		t.Error("no worker outside target app")
	}

	*fg = "game.exe"
	if !e.dispatch(0x4A, true, false) {
		t.Error("inside target app, trigger should be swallowed")
	}
	if active(e) != 1 {
		t.Error("worker should start inside target app")
	}
}

// Regression: turning the master switch off mid-press used to leave pressed[vk]
// set, so the next real press was misread as an OS auto-repeat and the worker
// never started.
func TestMasterOffClearsPressedState(t *testing.T) {
	e, _, _ := newTest(t)
	e.SetMaster(true)
	e.SetRules([]*config.Rule{holdRule(0x4A)})

	e.dispatch(0x4A, true, false) // press (consumed, worker started)
	if active(e) != 1 {
		t.Fatalf("expected 1 worker, got %d", active(e))
	}

	e.SetMaster(false) // user toggles master off WHILE the key is held
	if active(e) != 0 {
		t.Fatalf("master off should stop workers, got %d", active(e))
	}

	// Re-enable master, press the same key again — must start a worker.
	e.SetMaster(true)
	if !e.dispatch(0x4A, true, false) {
		t.Fatal("trigger down after re-enabling master should be swallowed")
	}
	if active(e) != 1 {
		t.Fatalf("trigger down after re-enabling master should start a worker, got %d", active(e))
	}
}

func TestDisabledAndUnknownPassThrough(t *testing.T) {
	e, _, _ := newTest(t)
	e.SetMaster(true)
	e.SetRules([]*config.Rule{{TriggerVK: 0x4A, Mode: config.ModeHold, IntervalMs: 1000, Enabled: false}})

	if e.dispatch(0x4A, true, false) {
		t.Error("disabled rule should pass through")
	}
	if e.dispatch(0x99, true, false) {
		t.Error("key with no rule should pass through")
	}
	if active(e) != 0 {
		t.Error("no workers expected")
	}
}

// recOut records every injected event in order, for checking how a combination is sent.
type recOut struct {
	mu     sync.Mutex
	events []string
	first  chan struct{} // closed after the first key-up of the output, i.e. one full pulse
	outVK  uint16
	once   sync.Once
}

func (r *recOut) fire(vk uint16, up bool) {
	r.mu.Lock()
	dir := "down"
	if up {
		dir = "up"
	}
	r.events = append(r.events, fmt.Sprintf("%X %s", vk, dir))
	r.mu.Unlock()
	if up && vk == r.outVK {
		r.once.Do(func() { close(r.first) })
	}
}

func (r *recOut) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.events...)
}

func modRule(vk uint16, mods config.Mods, mode config.Mode) *config.Rule {
	return &config.Rule{TriggerVK: vk, Modifiers: mods, Mode: mode, IntervalMs: 1000, Enabled: true}
}

const (
	vkE     = 0x45
	vkLCtrl = 0xA2
	vkLShft = 0xA0
	vkRCtrl = 0xA3
)

// The issue's case: with only a Ctrl+E rule, E alone types normally and Ctrl+E fires.
func TestModifierRuleNeedsItsModifiers(t *testing.T) {
	e, _, _ := newTest(t)
	e.SetMaster(true)
	e.SetRules([]*config.Rule{modRule(vkE, config.ModCtrl, config.ModeHold)})

	if e.dispatch(vkE, true, false) {
		t.Fatal("E without Ctrl should pass through")
	}
	e.dispatch(vkE, false, true)
	if active(e) != 0 {
		t.Fatal("E without Ctrl should not start a worker")
	}

	if e.dispatch(vkLCtrl, true, false) {
		t.Fatal("the modifier itself must never be swallowed")
	}
	if !e.dispatch(vkE, true, false) {
		t.Fatal("Ctrl+E should be swallowed")
	}
	if active(e) != 1 {
		t.Fatal("Ctrl+E should start the worker")
	}
	e.dispatch(vkLCtrl, false, true) // letting go of Ctrl first still ends the hold on E's release
	if !e.dispatch(vkE, false, true) {
		t.Fatal("the release of a consumed press should be swallowed")
	}
	if active(e) != 0 {
		t.Fatal("releasing E should stop the worker")
	}
}

// The modifiers must match exactly: Ctrl+Shift+E is not Ctrl+E, and right Ctrl counts as Ctrl.
func TestModifiersMatchExactly(t *testing.T) {
	e, _, _ := newTest(t)
	e.SetMaster(true)
	e.SetRules([]*config.Rule{modRule(vkE, config.ModCtrl, config.ModeHold)})

	e.dispatch(vkLCtrl, true, false)
	e.dispatch(vkLShft, true, false)
	if e.dispatch(vkE, true, false) {
		t.Fatal("Ctrl+Shift+E should not fire a Ctrl+E rule")
	}
	e.dispatch(vkE, false, true)
	e.dispatch(vkLShft, false, true)
	e.dispatch(vkLCtrl, false, true)

	e.dispatch(vkRCtrl, true, false)
	if !e.dispatch(vkE, true, false) {
		t.Fatal("right Ctrl+E should fire a Ctrl+E rule")
	}
	e.dispatch(vkE, false, true)
	e.dispatch(vkRCtrl, false, true)
}

// A plain rule keeps firing with modifiers held (as before modifiers existed), but a rule naming exactly the held modifiers wins over it.
func TestExactModifierRuleWinsOverPlain(t *testing.T) {
	e, _, _ := newTest(t)
	e.SetMaster(true)
	plain := modRule(vkE, 0, config.ModeHold)
	ctrl := modRule(vkE, config.ModCtrl, config.ModeHold)
	e.SetRules([]*config.Rule{plain, ctrl})

	e.dispatch(vkLShft, true, false)
	if !e.dispatch(vkE, true, false) {
		t.Fatal("Shift+E should fall back to the plain rule")
	}
	if !isRunning(e, plain) || isRunning(e, ctrl) {
		t.Fatal("Shift+E should run the plain rule only")
	}
	e.dispatch(vkE, false, true)
	e.dispatch(vkLShft, false, true)

	e.dispatch(vkLCtrl, true, false)
	e.dispatch(vkE, true, false)
	if !isRunning(e, ctrl) || isRunning(e, plain) {
		t.Fatal("Ctrl+E should run the Ctrl rule only")
	}
	e.dispatch(vkE, false, true)
	e.dispatch(vkLCtrl, false, true)
}

func isRunning(e *Engine, r *config.Rule) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.isRepeatingLocked(r)
}

// In toggle mode the user may let go of Ctrl after starting; each pulse then presses Ctrl around E itself, so the output is still Ctrl+E, and Ctrl is never left down.
func TestToggleSendsMissingModifiers(t *testing.T) {
	e, _, _ := newTest(t)
	rec := &recOut{first: make(chan struct{}), outVK: vkE}
	e.output = rec.fire
	e.SetMaster(true)
	r := modRule(vkE, config.ModCtrl, config.ModeToggle)
	r.IntervalMs = 1
	e.SetRules([]*config.Rule{r})

	e.dispatch(vkLCtrl, true, false)
	e.dispatch(vkE, true, false) // starts repeating; the first pulse may still see Ctrl held
	e.dispatch(vkE, false, true)
	e.dispatch(vkLCtrl, false, true) // the user lets go of Ctrl; the toggle keeps running
	<-rec.first

	// Every pulse that starts from here on has no Ctrl held, so it must carry its own.
	mark := len(rec.snapshot())
	deadline := time.Now().Add(2 * time.Second)
	for {
		if ev := rec.snapshot()[mark:]; strings.Count(strings.Join(ev, ","), "45 up") >= 2 || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	e.SetRules(nil) // stop the worker
	time.Sleep(50 * time.Millisecond)

	all := rec.snapshot()
	tail := strings.Join(all[mark:], ",")
	if !strings.Contains(tail, "11 down,45 down,45 up,11 up") {
		t.Fatalf("a pulse after Ctrl was released should be Ctrl down, E down, E up, Ctrl up: %v", all[mark:])
	}
	if strings.Count(strings.Join(all, ","), "11 down") != strings.Count(strings.Join(all, ","), "11 up") {
		t.Fatalf("every injected Ctrl down needs its up: %v", all)
	}
}

// With Ctrl physically held the worker sends E alone: the user's Ctrl already makes it Ctrl+E, and injecting Ctrl up would release it under them.
func TestHeldModifierIsNotInjected(t *testing.T) {
	e, _, _ := newTest(t)
	rec := &recOut{first: make(chan struct{}), outVK: vkE}
	e.output = rec.fire
	e.SetMaster(true)
	e.SetRules([]*config.Rule{modRule(vkE, config.ModCtrl, config.ModeHold)})

	e.dispatch(vkLCtrl, true, false)
	e.dispatch(vkE, true, false)
	<-rec.first
	e.dispatch(vkE, false, true)
	e.dispatch(vkLCtrl, false, true)
	for _, x := range rec.snapshot() {
		if x == "11 down" || x == "11 up" {
			t.Fatalf("held Ctrl must not be injected: %v", rec.snapshot())
		}
	}
}
