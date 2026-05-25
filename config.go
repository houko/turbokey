//go:build windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func (r *Rule) effectiveOutputVK() uint16 {
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

func configPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(filepath.Dir(exe), "config.json")
}

// loadRules reads rules from config.json next to the executable.
// A missing file yields an empty slice, not an error.
func loadRules() ([]*Rule, error) {
	data, err := os.ReadFile(configPath())
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
		tvk, ok := nameToVK[d.Trigger]
		if !ok {
			continue // unknown key name, skip defensively
		}
		var ovk uint16
		if d.Output != "" {
			ovk = nameToVK[d.Output]
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

// saveRules writes rules to config.json next to the executable.
func saveRules(rules []*Rule) error {
	dto := configDTO{Rules: make([]ruleDTO, 0, len(rules))}
	for _, r := range rules {
		out := ""
		if r.OutputVK != 0 {
			out = vkName(r.OutputVK)
		}
		mode := "hold"
		if r.Mode == ModeToggle {
			mode = "toggle"
		}
		dto.Rules = append(dto.Rules, ruleDTO{
			Name:     r.Name,
			Trigger:  vkName(r.TriggerVK),
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
	return os.WriteFile(configPath(), data, 0644)
}
