//go:build windows

// Package config defines the rapid-fire rule model and its JSON persistence.
package config

import (
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
	Rules []ruleDTO `json:"rules"`
}

func path() string {
	exe, err := os.Executable()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(filepath.Dir(exe), "config.json")
}

// Load reads rules from config.json next to the executable. A missing file
// yields an empty slice, not an error.
func Load() ([]*Rule, error) {
	data, err := os.ReadFile(path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
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
	return rules, nil
}

// Save writes rules to config.json next to the executable.
func Save(rules []*Rule) error {
	dto := configDTO{Rules: make([]ruleDTO, 0, len(rules))}
	for _, r := range rules {
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
