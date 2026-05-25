//go:build windows

package i18n

import "testing"

// TestCatalogsHaveSameKeys guards against locale files drifting out of sync.
func TestCatalogsHaveSameKeys(t *testing.T) {
	en := dict[EN]
	if len(en) == 0 {
		t.Fatal("English catalog is empty")
	}
	for lang, m := range dict {
		for k := range en {
			if _, ok := m[k]; !ok {
				t.Errorf("[%s] missing key %q", lang, k)
			}
		}
		for k := range m {
			if _, ok := en[k]; !ok {
				t.Errorf("[%s] has extra key %q (not in en)", lang, k)
			}
		}
	}
}

func TestTFallback(t *testing.T) {
	current = EN
	if T("app.title") == "" {
		t.Error("app.title should resolve")
	}
	if got := T("no.such.key"); got != "no.such.key" {
		t.Errorf("missing key should fall back to itself, got %q", got)
	}
}

func TestRegistryHasCatalogs(t *testing.T) {
	for _, l := range langs {
		if l.code == "" {
			continue // Auto
		}
		if _, ok := dict[Lang(l.code)]; !ok {
			t.Errorf("language %q listed but has no catalog", l.code)
		}
	}
}
