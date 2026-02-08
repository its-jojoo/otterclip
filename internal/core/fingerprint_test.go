package core

import "testing"

func TestFingerprintStable(t *testing.T) {
	a := Fingerprint(Normalize("hello  world"))
	b := Fingerprint(Normalize("hello world"))
	if a == "" || b == "" || a != b {
		t.Fatalf("expected stable fingerprint, got %q and %q", a, b)
	}
}
