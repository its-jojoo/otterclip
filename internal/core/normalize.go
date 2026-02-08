package core

import (
	"strings"
	"unicode"
)

const MaxContentLen = 32_000 // MVP safeguard

func Normalize(s string) string {
	// Trim + collapse whitespace to single spaces
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(s))

	space := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !space {
				b.WriteByte(' ')
				space = true
			}
			continue
		}
		space = false
		b.WriteRune(r)
	}

	out := b.String()
	if len(out) > MaxContentLen {
		out = out[:MaxContentLen]
	}
	return out
}
