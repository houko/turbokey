//go:build windows

// Package config defines the rapid-fire rule model and its JSON persistence.
package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"turbokey/internal/keys"
)

// Mode is the rapid-fire trigger style for a rule.
type Mode int

const (
	// ModeHold repeats while the trigger key is physically held.
	ModeHold Mode = iota
	// ModeToggle flips repeating on/off on each trigger press.
	ModeToggle
)

// Rule is a single rapid-fire binding (runtime form, keyed by VK code).
type Rule struct {
	Name       string
	TriggerVK  uint16
	OutputVK   uint16 // 0 means: same as TriggerVK
	Mode       Mode
	IntervalMs int
	Enabled    bool
}

// EffectiveOutputVK returns the key actually sent: the trigger key itself when no
// distinct output key is configured.
func (r *Rule) EffectiveOutputVK() uint16 {
	if r.OutputVK == 0 {
		return r.TriggerVK
	}
	return r.OutputVK
}

// File is the persisted configuration: rules plus an optional UI language code
// ("" = follow the OS, otherwise "zh" / "en").
type File struct {
	Lang  string
	Rules []*Rule
}

// --- JSON serialization (human-readable, keyed by key name) ---

type ruleDTO struct {
	Name     string `json:"name"`
	Trigger  string `json:"trigger"`
	Output   string `json:"output"` // "" => same as trigger
	Mode     string `json:"mode"`   // "hold" | "toggle"
	Interval int    `json:"interval"`
	Enabled  bool   `json:"enabled"`
}

type configDTO struct {
	Lang  string    `json:"lang,omitempty"`
	Rules []ruleDTO `json:"rules"`
}

func path() string {
	exe, err := os.Executable()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(filepath.Dir(exe), "config.json")
}

// Load reads the config next to the executable. A missing file yields an empty
// config, not an error.
func Load() (*File, error) {
	data, err := os.ReadFile(path())
	if err != nil {
		if os.IsNotExist(err) {
			return &File{}, nil
		}
		return nil, err
	}
	// Tolerate a UTF-8 BOM (e.g. from editing config.json in Notepad).
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	var dto configDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}
	rules := make([]*Rule, 0, len(dto.Rules))
	for _, d := range dto.Rules {
		tvk, ok := keys.VK(d.Trigger)
		if !ok {
			continue // unknown key name, skip defensively
		}
		var ovk uint16
		if d.Output != "" {
			ovk, _ = keys.VK(d.Output)
		}
		mode := ModeHold
		if d.Mode == "toggle" {
			mode = ModeToggle
		}
		interval := d.Interval
		if interval < 1 {
			interval = 10
		}
		rules = append(rules, &Rule{
			Name:       d.Name,
			TriggerVK:  tvk,
			OutputVK:   ovk,
			Mode:       mode,
			IntervalMs: interval,
			Enabled:    d.Enabled,
		})
	}
	return &File{Lang: dto.Lang, Rules: rules}, nil
}

// Save writes the config to config.json next to the executable.
func Save(f *File) error {
	dto := configDTO{Lang: f.Lang, Rules: make([]ruleDTO, 0, len(f.Rules))}
	for _, r := range f.Rules {
		out := ""
		if r.OutputVK != 0 {
			out = keys.Name(r.OutputVK)
		}
		mode := "hold"
		if r.Mode == ModeToggle {
			mode = "toggle"
		}
		dto.Rules = append(dto.Rules, ruleDTO{
			Name:     r.Name,
			Trigger:  keys.Name(r.TriggerVK),
			Output:   out,
			Mode:     mode,
			Interval: r.IntervalMs,
			Enabled:  r.Enabled,
		})
	}
	data, err := json.MarshalIndent(dto, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path(), data, 0644)
}
