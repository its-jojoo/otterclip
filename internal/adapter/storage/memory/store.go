package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/its-jojoo/otterclip/internal/adapter/storage"
	"github.com/its-jojoo/otterclip/internal/core"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	mu   sync.RWMutex
	now  func() time.Time
	seq  int
	byID map[string]core.Item
	list []string // newest first
}

func New() *Store {
	return &Store{
		now:  time.Now,
		byID: make(map[string]core.Item),
	}
}

func (s *Store) Now() time.Time { return s.now() }

func (s *Store) Put(ctx context.Context, item core.Item, mode storage.PutMode) error {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	if item.ID == "" {
		s.seq++
		item.ID = "mem-" + itoa(s.seq)
	}

	// merge updates LastSeenAt if already exists
	if mode == storage.PutMerge {
		if existing, ok := s.byID[item.ID]; ok {
			existing.LastSeenAt = item.LastSeenAt
			existing.Content = item.Content
			existing.Type = item.Type
			existing.Fingerprint = item.Fingerprint
			s.byID[item.ID] = existing
			return nil
		}
	}

	// insert
	s.byID[item.ID] = item
	s.list = append([]string{item.ID}, s.list...)
	return nil
}

func (s *Store) ListRecent(ctx context.Context, limit int) ([]core.Item, error) {
	_ = ctx

	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.list) {
		limit = len(s.list)
	}

	out := make([]core.Item, 0, limit)
	for i := 0; i < limit; i++ {
		id := s.list[i]
		out = append(out, s.byID[id])
	}
	return out, nil
}

func (s *Store) SetPinned(ctx context.Context, id string, pinned bool) error {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	it, ok := s.byID[id]
	if !ok {
		return ErrNotFound
	}
	it.Pinned = pinned
	s.byID[id] = it
	return nil
}

func (s *Store) Delete(ctx context.Context, id string) error {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.byID[id]; !ok {
		return ErrNotFound
	}
	delete(s.byID, id)

	// remove from list
	for i := range s.list {
		if s.list[i] == id {
			s.list = append(s.list[:i], s.list[i+1:]...)
			break
		}
	}
	return nil
}

func (s *Store) Count(ctx context.Context) (int, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byID), nil
}

// tiny int->string without strconv import (keeps file minimal)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [32]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + (n % 10))
		n /= 10
	}
	return string(buf[i:])
}
