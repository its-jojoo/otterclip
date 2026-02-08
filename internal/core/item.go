package core

import (
	"time"
)

type Item struct {
	ID          string      `json:"id"`
	Content     string      `json:"content"`
	ContentType ContentType `json:"content_type"`
	Fingerprint string      `json:"fingerprint"`

	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`

	Pinned bool `json:"pinned"`
}
