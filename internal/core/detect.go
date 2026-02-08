package core

import (
	"net/url"
	"strings"
)

func DetectType(content string) ContentType {
	s := strings.TrimSpace(content)
	if s == "" {
		return ContentTypeText
	}

	// URL detection
	if u, err := url.Parse(s); err == nil && u.Scheme != "" && u.Host != "" {
		return ContentTypeURL
	}

	// Command-ish detection (very simple MVP heuristic)
	if looksLikeCommand(s) {
		return ContentTypeCommand
	}

	// Code-ish detection (very simple MVP heuristic)
	if looksLikeCode(s) {
		return ContentTypeCode
	}

	return ContentTypeText
}

func looksLikeCommand(s string) bool {
	// starts with common shell prompt or contains flags/pipes
	if strings.HasPrefix(s, "$ ") || strings.HasPrefix(s, "sudo ") {
		return true
	}
	if strings.Contains(s, " --") || strings.Contains(s, " | ") || strings.Contains(s, " && ") {
		return true
	}
	return false
}

func looksLikeCode(s string) bool {
	// tiny heuristic: braces/semicolons/keywords
	if strings.Contains(s, "{") && strings.Contains(s, "}") {
		return true
	}
	if strings.Contains(s, "function ") || strings.Contains(s, "package ") || strings.Contains(s, "import ") {
		return true
	}
	if strings.Contains(s, ";") && strings.Contains(s, "=") {
		return true
	}
	return false
}
