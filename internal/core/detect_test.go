package core

import "testing"

func TestDetectTypeURL(t *testing.T) {
	if DetectType("https://example.com/path") != ContentTypeURL {
		t.Fatalf("expected url")
	}
}

func TestDetectTypeCommand(t *testing.T) {
	if DetectType("sudo apt update && sudo apt upgrade") != ContentTypeCommand {
		t.Fatalf("expected command")
	}
}

func TestDetectTypeCode(t *testing.T) {
	if DetectType("package main\n\nfunc main() { println(\"hi\") }") != ContentTypeCode {
		t.Fatalf("expected code")
	}
}
