package capture

import (
	"context"

	"github.com/its-jojoo/otterclip/internal/adapter/storage"
	"github.com/its-jojoo/otterclip/internal/core"
)

type Config struct {
	MaxItems           int
	DedupeConsecutive  bool
	PrivacyIgnoreEmpty bool
}

type Service struct {
	store   storage.Store
	privacy *core.PrivacyFilter
	cfg     Config

	lastFingerprint string
}

func New(store storage.Store, privacy *core.PrivacyFilter, cfg Config) *Service {
	if cfg.MaxItems <= 0 {
		cfg.MaxItems = 5000
	}
	if !cfg.DedupeConsecutive {
		cfg.DedupeConsecutive = true
	}
	if !cfg.PrivacyIgnoreEmpty {
		cfg.PrivacyIgnoreEmpty = true
	}
	return &Service{store: store, privacy: privacy, cfg: cfg}
}

func (s *Service) ProcessText(ctx context.Context, raw string) (*core.Item, bool, error) {
	normalized := core.Normalize(raw)
	if s.cfg.PrivacyIgnoreEmpty && normalized == "" {
		return nil, false, nil
	}
	if s.privacy != nil && s.privacy.ShouldIgnore(normalized) {
		return nil, false, nil
	}

	fp := core.Fingerprint(normalized)
	if s.cfg.DedupeConsecutive && fp != "" && fp == s.lastFingerprint {
		return nil, false, nil
	}

	now := s.store.Now()
	item := core.Item{
		Content:     normalized,
		Type:        core.DetectType(normalized),
		Fingerprint: fp,
		CreatedAt:   now,
		LastSeenAt:  now,
	}

	// Save
	if err := s.store.Put(ctx, item, storage.PutInsert); err != nil {
		return nil, false, err
	}

	s.lastFingerprint = fp

	// Retention (evict oldest non-pinned if exceeded)
	if err := s.enforceRetention(ctx); err != nil {
		return nil, false, err
	}

	return &item, true, nil
}

func (s *Service) enforceRetention(ctx context.Context) error {
	items, err := s.store.ListRecent(ctx, s.cfg.MaxItems+200) // small window to find eviction candidates
	if err != nil {
		return err
	}
	if len(items) <= s.cfg.MaxItems {
		return nil
	}

	// items are newest->oldest in our memory store; eviction should target oldest non-pinned
	over := len(items) - s.cfg.MaxItems

	for i := len(items) - 1; i >= 0 && over > 0; i-- {
		it := items[i]
		if it.Pinned {
			continue
		}
		if it.ID != "" {
			if err := s.store.Delete(ctx, it.ID); err != nil {
				// ignore not found in case store changed
				continue
			}
			over--
		}
	}
	return nil
}
