package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/its-jojoo/otterclip/internal/adapter/storage/sqlite"
)

type ExportItem struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Fingerprint string `json:"fingerprint"`
	CreatedAt   string `json:"created_at"`
	LastSeenAt  string `json:"last_seen_at"`
	Pinned      bool   `json:"pinned"`
}

func main() {
	var (
		dbPath = flag.String("db", "./otterclip.dev.db", "sqlite db path")
		out    = flag.String("out", "otterclip-export.json", "output json file path")
		limit  = flag.Int("limit", 5000, "max items to export")
	)
	flag.Parse()

	st, err := sqlite.Open(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db open error: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	ctx := context.Background()
	items, err := st.ListRecent(ctx, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list error: %v\n", err)
		os.Exit(1)
	}

	export := make([]ExportItem, 0, len(items))
	for _, it := range items {
		export = append(export, ExportItem{
			ID:          it.ID,
			Type:        string(it.Type),
			Content:     it.Content,
			Fingerprint: it.Fingerprint,
			CreatedAt:   it.CreatedAt.UTC().Format(time.RFC3339Nano),
			LastSeenAt:  it.LastSeenAt.UTC().Format(time.RFC3339Nano),
			Pinned:      it.Pinned,
		})
	}

	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(export); err != nil {
		fmt.Fprintf(os.Stderr, "encode error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("exported", len(export), "items to", *out)
}
