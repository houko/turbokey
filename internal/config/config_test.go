//go:build windows

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	in := &File{
		Lang:    "en",
		Targets: []string{"dnf.exe"},
		Rules: []*Rule{
			{Name: "attack", TriggerVK: 0x4A, Mode: ModeHold, IntervalMs: 20, Enabled: true},
			{Name: "skill", TriggerVK: 0x4B, Mode: ModeToggle, IntervalMs: 50, Enabled: false},
		},
	}
	if err := saveTo(p, in); err != nil {
		t.Fatalf("saveTo: %v", err)
	}
	out, err := loadFrom(p)
	if err != nil {
		t.Fatalf("loadFrom: %v", err)
	}
	if out.Lang != "en" {
		t.Errorf("Lang = %q", out.Lang)
	}
	if len(out.Targets) != 1 || out.Targets[0] != "dnf.exe" {
		t.Errorf("Targets = %v", out.Targets)
	}
	if len(out.Rules) != 2 {
		t.Fatalf("Rules len = %d", len(out.Rules))
	}
	if r := out.Rules[0]; r.TriggerVK != 0x4A || r.Mode != ModeHold || r.IntervalMs != 20 || !r.Enabled {
		t.Errorf("rule[0] = %+v", r)
	}
	if r := out.Rules[1]; r.Mode != ModeToggle || r.Enabled {
		t.Errorf("rule[1] = %+v", r)
	}
}

func TestBOMTolerated(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	body := `{"rules":[{"name":"x","trigger":"J","output":"","mode":"hold","interval":10,"enabled":true}]}`
	if err := os.WriteFile(p, append([]byte{0xEF, 0xBB, 0xBF}, body...), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := loadFrom(p)
	if err != nil {
		t.Fatalf("BOM not tolerated: %v", err)
	}
	if len(out.Rules) != 1 {
		t.Fatalf("Rules len = %d", len(out.Rules))
	}
}

func TestLoadDefaultsAndSkips(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	body := `{"rules":[
		{"name":"bad","trigger":"NoSuchKey","mode":"hold","interval":5,"enabled":true},
		{"name":"ok","trigger":"J","output":"","mode":"weird","interval":0,"enabled":true}
	]}`
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := loadFrom(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Rules) != 1 {
		t.Fatalf("unknown trigger key should be skipped, got %d rules", len(out.Rules))
	}
	r := out.Rules[0]
	if r.Mode != ModeHold {
		t.Error("unknown mode should default to hold")
	}
	if r.IntervalMs != 10 {
		t.Errorf("interval < 1 should default to 10, got %d", r.IntervalMs)
	}
	if r.OutputVK != 0 {
		t.Error(`empty "output" should mean OutputVK 0 (same as trigger)`)
	}
}

func TestResolvedMasterHotkey(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want uint16
	}{
		{"empty defaults to F8", "", 0x77},
		{"valid F9", "F9", 0x78},
		{"unknown defaults to F8", "Bogus", 0x77},
		{"mouse button rejected -> F8", "Mouse Left", 0x77},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := (&File{MasterHotkey: c.in}).ResolvedMasterHotkey()
			if got != c.want {
				t.Errorf("ResolvedMasterHotkey(%q) = 0x%X, want 0x%X", c.in, got, c.want)
			}
		})
	}
	if (*File)(nil).ResolvedMasterHotkey() != 0x77 {
		t.Error("nil receiver should also default to F8")
	}
}

// TestSaveAtomic confirms saveTo doesn't leave a truncated config when called
// repeatedly: the result is always a fully readable file.
func TestSaveAtomic(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	for i := 0; i < 5; i++ {
		if err := saveTo(p, &File{Lang: "en"}); err != nil {
			t.Fatal(err)
		}
		out, err := loadFrom(p)
		if err != nil {
			t.Fatalf("load after save #%d: %v", i, err)
		}
		if out.Lang != "en" {
			t.Fatalf("lang after save #%d = %q", i, out.Lang)
		}
	}
}

func TestMissingFileIsEmpty(t *testing.T) {
	out, err := loadFrom(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("loadFrom missing: %v", err)
	}
	if out == nil || len(out.Rules) != 0 {
		t.Errorf("expected empty config, got %+v", out)
	}
}
