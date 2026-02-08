package capture

import (
	"context"
	"testing"

	"github.com/its-jojoo/otterclip/internal/adapter/storage/memory"
	"github.com/its-jojoo/otterclip/internal/core"
)

func TestProcessText_IgnoresEmpty(t *testing.T) {
	st := memory.New()
	svc := New(st, nil, Config{MaxItems: 10})

	got, saved, err := svc.ProcessText(context.Background(), "   \n\t ")
	if err != nil {
		t.Fatal(err)
	}
	if saved || got != nil {
		t.Fatalf("expected not saved")
	}
}

func TestProcessText_PrivacyIgnore(t *testing.T) {
	st := memory.New()
	pf, err := core.NewPrivacyFilter([]string{"token="}, false)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, pf, Config{MaxItems: 10})

	_, saved, err := svc.ProcessText(context.Background(), "my token=abc")
	if err != nil {
		t.Fatal(err)
	}
	if saved {
		t.Fatalf("expected ignored by privacy filter")
	}
}

func TestProcessText_DedupeConsecutive(t *testing.T) {
	st := memory.New()
	svc := New(st, nil, Config{MaxItems: 10, DedupeConsecutive: true})

	_, saved1, _ := svc.ProcessText(context.Background(), "hello  world")
	_, saved2, _ := svc.ProcessText(context.Background(), "hello world")

	if !saved1 {
		t.Fatalf("expected first saved")
	}
	if saved2 {
		t.Fatalf("expected second not saved due to consecutive dedupe")
	}
}

func TestRetention_EvictsOldestNonPinned(t *testing.T) {
	st := memory.New()
	svc := New(st, nil, Config{MaxItems: 2})

	// 1
	i1, saved, err := svc.ProcessText(context.Background(), "one")
	if err != nil || !saved || i1 == nil {
		t.Fatalf("expected saved one")
	}
	// 2
	i2, saved, err := svc.ProcessText(context.Background(), "two")
	if err != nil || !saved || i2 == nil {
		t.Fatalf("expected saved two")
	}

	// Pin oldest (i1 might not have ID in this minimal pipeline; memory store assigns one)
	// We need to list to get actual IDs.
	items, _ := st.ListRecent(context.Background(), 10)
	if len(items) != 2 {
		t.Fatalf("expected 2 items")
	}
	// items[1] is oldest
	if err := st.SetPinned(context.Background(), items[1].ID, true); err != nil {
		t.Fatal(err)
	}

	// 3 triggers eviction, should evict "two" (oldest non-pinned after pinning "one")
	_, saved, err = svc.ProcessText(context.Background(), "three")
	if err != nil || !saved {
		t.Fatalf("expected saved three")
	}

	items, _ = st.ListRecent(context.Background(), 10)
	if len(items) != 2 {
		t.Fatalf("expected 2 items after retention, got %d", len(items))
	}

	// Ensure pinned one still exists
	foundOne := false
	for _, it := range items {
		if it.Content == "one" {
			foundOne = true
		}
	}
	if !foundOne {
		t.Fatalf("expected pinned 'one' to remain")
	}
}
