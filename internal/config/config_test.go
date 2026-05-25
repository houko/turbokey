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

func TestMissingFileIsEmpty(t *testing.T) {
	out, err := loadFrom(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("loadFrom missing: %v", err)
	}
	if out == nil || len(out.Rules) != 0 {
		t.Errorf("expected empty config, got %+v", out)
	}
}
