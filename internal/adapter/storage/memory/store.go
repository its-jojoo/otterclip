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

	fpToID map[string]string
	list   []string
}

func New() *Store {
	return &Store{
		now:    time.Now,
		byID:   make(map[string]core.Item),
		fpToID: make(map[string]string),
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

	// Upsert by fingerprint (match sqlite behavior)
	if mode == storage.PutInsert && item.Fingerprint != "" {
		if existingID, ok := s.fpToID[item.Fingerprint]; ok {
			existing := s.byID[existingID]
			existing.Content = item.Content
			existing.Type = item.Type
			existing.LastSeenAt = item.LastSeenAt
			existing.Fingerprint = item.Fingerprint
			// keep existing.CreatedAt and existing.Pinned
			s.byID[existingID] = existing

			s.moveToFront(existingID)
			return nil
		}
	}

	// Merge by ID (used by callers that explicitly want to update an existing row)
	if mode == storage.PutMerge {
		if existing, ok := s.byID[item.ID]; ok {
			existing.LastSeenAt = item.LastSeenAt
			existing.Content = item.Content
			existing.Type = item.Type
			existing.Fingerprint = item.Fingerprint
			// keep existing.CreatedAt and existing.Pinned
			s.byID[item.ID] = existing
			s.moveToFront(item.ID)
			return nil
		}
	}

	// Insert new
	s.byID[item.ID] = item
	if item.Fingerprint != "" {
		s.fpToID[item.Fingerprint] = item.ID
	}
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

	it, ok := s.byID[id]
	if !ok {
		return ErrNotFound
	}

	if it.Fingerprint != "" {
		if cur, ok := s.fpToID[it.Fingerprint]; ok && cur == id {
			delete(s.fpToID, it.Fingerprint)
		}
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

func (s *Store) moveToFront(id string) {
	for i := range s.list {
		if s.list[i] == id {
			copy(s.list[1:i+1], s.list[0:i])
			s.list[0] = id
			return
		}
	}
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
