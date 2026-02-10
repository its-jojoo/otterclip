package search

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/its-jojoo/otterclip/internal/core"
)

type Store interface {
	ListRecent(ctx context.Context, limit int) ([]core.Item, error)
}

type Options struct {
	ScanLimit int
	OutLimit  int
	Now       time.Time // optional, for tests
}

type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Query(ctx context.Context, q string, opt Options) ([]core.Item, error) {
	if opt.ScanLimit <= 0 {
		opt.ScanLimit = 80
	}
	if opt.OutLimit <= 0 {
		opt.OutLimit = 20
	}
	now := opt.Now
	if now.IsZero() {
		now = time.Now()
	}

	items, err := s.store.ListRecent(ctx, opt.ScanLimit)
	if err != nil {
		return nil, err
	}

	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return nil, nil
	}

	type scored struct {
		it    core.Item
		score int
	}

	scoredItems := make([]scored, 0, len(items))

	for _, it := range items {
		content := strings.ToLower(it.Content)

		matchScore := scoreMatch(content, q)
		if matchScore == 0 {
			continue
		}

		score := matchScore

		// pinned boost
		if it.Pinned {
			score += 5000
		}

		// recency boost
		age := now.Sub(it.LastSeenAt)
		switch {
		case age < 10*time.Minute:
			score += 400
		case age < time.Hour:
			score += 250
		case age < 24*time.Hour:
			score += 120
		case age < 7*24*time.Hour:
			score += 40
		}

		scoredItems = append(scoredItems, scored{it: it, score: score})
	}

	sort.Slice(scoredItems, func(i, j int) bool {
		if scoredItems[i].score != scoredItems[j].score {
			return scoredItems[i].score > scoredItems[j].score
		}
		return scoredItems[i].it.LastSeenAt.After(scoredItems[j].it.LastSeenAt)
	})

	if opt.OutLimit > len(scoredItems) {
		opt.OutLimit = len(scoredItems)
	}

	out := make([]core.Item, 0, opt.OutLimit)
	for i := 0; i < opt.OutLimit; i++ {
		out = append(out, scoredItems[i].it)
	}
	return out, nil
}

func scoreMatch(s, q string) int {
	// Basic match:
	// - exact match strongest
	// - prefix strong
	// - substring ok (earlier index slightly better)
	if s == q {
		return 3000
	}
	if strings.HasPrefix(s, q) {
		return 2000
	}
	if idx := strings.Index(s, q); idx >= 0 {
		return 1000 + max(0, 200-idx)
	}
	return 0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
