//go:build windows

package engine

import (
	"sync"
	"testing"

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
