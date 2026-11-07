package domain

import "testing"

func TestNewID(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		id := NewID()
		if !IsID(id) {
			t.Fatalf("NewID() = %q is not a UUID", id)
		}
		if id[14] != '4' {
			t.Fatalf("NewID() = %q is not version 4", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}

func TestIsID(t *testing.T) {
	for _, s := range []string{"", "abc", "00000000-0000-0000-0000-00000000000", "../../etc/passwd"} {
		if IsID(s) {
			t.Errorf("IsID(%q) = true", s)
		}
	}
	if !IsID("00000000-0000-0000-0000-000000000000") {
		t.Error("IsID(zero UUID) = false")
	}
}
