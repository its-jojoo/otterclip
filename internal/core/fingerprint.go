package core

import (
	"crypto/sha256"
	"encoding/hex"
)

func Fingerprint(normalized string) string {
	if normalized == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
