package core

import "testing"

func TestNormalize(t *testing.T) {
	in := "  hello \n\t world   "
	got := Normalize(in)
	if got != "hello world" {
		t.Fatalf("expected %q, got %q", "hello world", got)
	}
}

func TestNormalizeEmpty(t *testing.T) {
	if Normalize("   \n\t ") != "" {
		t.Fatalf("expected empty")
	}
}
