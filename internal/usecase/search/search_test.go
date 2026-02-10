package search

import (
	"context"
	"testing"
	"time"

	"github.com/its-jojoo/otterclip/internal/core"
)

type fakeStore struct {
	items []core.Item
}

func (f fakeStore) ListRecent(ctx context.Context, limit int) ([]core.Item, error) {
	_ = ctx
	if limit <= 0 || limit > len(f.items) {
		limit = len(f.items)
	}
	return f.items[:limit], nil
}

func TestQuery_PinnedBoost(t *testing.T) {
	now := time.Now()
	items := []core.Item{
		{ID: "1", Content: "hello world", Pinned: false, LastSeenAt: now.Add(-time.Minute), Type: core.ContentTypeText},
		{ID: "2", Content: "hello world", Pinned: true, LastSeenAt: now.Add(-time.Hour), Type: core.ContentTypeText},
	}

	svc := New(fakeStore{items: items})
	got, err := svc.Query(context.Background(), "hello", Options{ScanLimit: 10, OutLimit: 10, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	if got[0].ID != "2" {
		t.Fatalf("expected pinned item first, got %s", got[0].ID)
	}
}
