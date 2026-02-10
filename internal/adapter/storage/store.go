package storage

import (
	"context"
	"time"

	"github.com/its-jojoo/otterclip/internal/core"
)

type PutMode int

const (
	PutInsert PutMode = iota
	PutMerge          // used when dedupe wants to update LastSeenAt
)

type Store interface {
	Put(ctx context.Context, item core.Item, mode PutMode) error
	ListRecent(ctx context.Context, limit int) ([]core.Item, error)

	SetPinned(ctx context.Context, id string, pinned bool) error
	Delete(ctx context.Context, id string) error

	Count(ctx context.Context) (int, error)
	Now() time.Time
}
