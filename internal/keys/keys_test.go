//go:build windows

package keys

import "testing"

func TestVKNameIndexRoundTrip(t *testing.T) {
	for _, name := range Names {
		vk, ok := VK(name)
		if !ok {
			t.Errorf("VK(%q) not found", name)
			continue
		}
		if got := Name(vk); got != name {
			t.Errorf("Name(VK(%q)) = %q, want %q", name, got, name)
		}
		if idx := Index(vk); idx < 0 || Names[idx] != name {
			t.Errorf("Index(VK(%q)) = %d, Names mismatch", name, idx)
		}
	}
}

func TestIsMouse(t *testing.T) {
	if !IsMouse(VKMouseLeft) {
		t.Error("VKMouseLeft should be a mouse button")
	}
	if IsMouse('A') {
		t.Error("'A' should not be a mouse button")
	}
}

func TestArrowsAreExtended(t *testing.T) {
	for _, name := range []string{"↑", "↓", "←", "→"} {
		vk, _ := VK(name)
		if !IsExtended(vk) {
			t.Errorf("%q should be an extended key", name)
		}
	}
	if a, _ := VK("A"); IsExtended(a) {
		t.Error("'A' should not be extended")
	}
}
