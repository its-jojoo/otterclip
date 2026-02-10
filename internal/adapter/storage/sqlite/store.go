package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"

	"github.com/its-jojoo/otterclip/internal/adapter/storage"
	"github.com/its-jojoo/otterclip/internal/core"
)

type Store struct {
	db  *sql.DB
	now func() time.Time
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	s := &Store{db: db, now: time.Now}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	// Sensible pragmas for desktop app
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL; PRAGMA foreign_keys=ON;`); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error   { return s.db.Close() }
func (s *Store) Now() time.Time { return s.now() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS items (
  id           TEXT PRIMARY KEY,
  content      TEXT NOT NULL,
  type         TEXT NOT NULL,
  fingerprint  TEXT NOT NULL,
  created_at   INTEGER NOT NULL,
  last_seen_at INTEGER NOT NULL,
  pinned       INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_items_created_at ON items(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_items_last_seen ON items(last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_items_pinned    ON items(pinned DESC);
CREATE INDEX IF NOT EXISTS idx_items_fp        ON items(fingerprint);
`)
	return err
}

func (s *Store) Put(ctx context.Context, item core.Item, mode storage.PutMode) error {
	if item.ID == "" {
		return errors.New("item ID required")
	}
	if item.Content == "" {
		return errors.New("empty content")
	}

	switch mode {
	case storage.PutInsert:
		_, err := s.db.ExecContext(ctx, `
INSERT INTO items(id, content, type, fingerprint, created_at, last_seen_at, pinned)
VALUES(?, ?, ?, ?, ?, ?, ?)
`, item.ID, item.Content, string(item.Type), item.Fingerprint,
			item.CreatedAt.UnixMilli(), item.LastSeenAt.UnixMilli(), boolToInt(item.Pinned))
		return err

	case storage.PutMerge:
		// Update last_seen_at + content/type/fingerprint (keep pinned)
		_, err := s.db.ExecContext(ctx, `
UPDATE items
SET content=?, type=?, fingerprint=?, last_seen_at=?
WHERE id=?
`, item.Content, string(item.Type), item.Fingerprint, item.LastSeenAt.UnixMilli(), item.ID)
		return err

	default:
		return errors.New("unknown put mode")
	}
}

func (s *Store) ListRecent(ctx context.Context, limit int) ([]core.Item, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT id, content, type, fingerprint, created_at, last_seen_at, pinned
FROM items
ORDER BY last_seen_at DESC
LIMIT ?
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]core.Item, 0, limit)
	for rows.Next() {
		var it core.Item
		var cAt, lsAt int64
		var pinned int
		var typ string

		if err := rows.Scan(&it.ID, &it.Content, &typ, &it.Fingerprint, &cAt, &lsAt, &pinned); err != nil {
			return nil, err
		}
		it.Type = core.ContentType(typ)
		it.CreatedAt = time.UnixMilli(cAt)
		it.LastSeenAt = time.UnixMilli(lsAt)
		it.Pinned = pinned == 1
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) SetPinned(ctx context.Context, id string, pinned bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE items SET pinned=? WHERE id=?`, boolToInt(pinned), id)
	return err
}

func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM items WHERE id=?`, id)
	return err
}

func (s *Store) Count(ctx context.Context) (int, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM items`)
	var n int
	return n, row.Scan(&n)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
