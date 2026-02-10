package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/its-jojoo/otterclip/internal/adapter/storage"
	"github.com/its-jojoo/otterclip/internal/core"
)

func TestSQLiteStore_PutListCount(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "test.db")

	st, err := Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	now := time.Now()
	it := core.Item{
		ID:          uuid.NewString(),
		Content:     "hello world",
		Type:        core.ContentTypeText,
		Fingerprint: core.Fingerprint(core.Normalize("hello world")),
		CreatedAt:   now,
		LastSeenAt:  now,
	}

	if err := st.Put(context.Background(), it, storage.PutInsert); err != nil {
		t.Fatal(err)
	}

	n, err := st.Count(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected count=1, got %d", n)
	}

	items, err := st.ListRecent(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Content != "hello world" {
		t.Fatalf("unexpected content: %q", items[0].Content)
	}
}

func TestSQLiteStore_UpsertByFingerprint(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "test.db")

	st, err := Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	fp := core.Fingerprint(core.Normalize("hello world"))

	it1 := core.Item{
		ID:          uuid.NewString(),
		Content:     "hello world",
		Type:        core.ContentTypeText,
		Fingerprint: fp,
		CreatedAt:   now,
		LastSeenAt:  now,
	}
	if err := st.Put(ctx, it1, storage.PutInsert); err != nil {
		t.Fatal(err)
	}

	it2 := core.Item{
		ID:          uuid.NewString(),
		Content:     "hello   world",
		Type:        core.ContentTypeText,
		Fingerprint: fp,
		CreatedAt:   now.Add(10 * time.Second),
		LastSeenAt:  now.Add(10 * time.Second),
	}
	if err := st.Put(ctx, it2, storage.PutInsert); err != nil {
		t.Fatal(err)
	}

	n, err := st.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected count=1, got %d", n)
	}

	items, err := st.ListRecent(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].LastSeenAt.UnixMilli() != it2.LastSeenAt.UnixMilli() {
		t.Fatalf("expected last_seen_at updated")
	}
}
