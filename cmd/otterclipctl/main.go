package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
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
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "export":
		exportCmd(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println("otterclipctl")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  otterclipctl export --db <path> [--out file] [--limit N] [--pinned-only] [--type t] [--since dur]")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  otterclipctl export --db ./otterclip.dev.db --out export.json --limit 2000")
	fmt.Println("  otterclipctl export --db ./otterclip.dev.db --pinned-only --out pins.json")
	fmt.Println("  otterclipctl export --db ./otterclip.dev.db --type url --since 168h")
}

func exportCmd(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)

	var (
		dbPath     = fs.String("db", "./otterclip.dev.db", "sqlite db path")
		out        = fs.String("out", "otterclip-export.json", "output json file path")
		limit      = fs.Int("limit", 5000, "max items to export (scanned)")
		pinnedOnly = fs.Bool("pinned-only", false, "export only pinned items")
		typeFilter = fs.String("type", "", "filter by type: text|url|code|command")
		sinceStr   = fs.String("since", "", "filter by last_seen_at age, e.g. 24h, 30m, 168h")
	)

	_ = fs.Parse(args)

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

	var since time.Time
	if strings.TrimSpace(*sinceStr) != "" {
		d, err := time.ParseDuration(strings.TrimSpace(*sinceStr))
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --since duration: %v\n", err)
			os.Exit(2)
		}
		since = time.Now().Add(-d)
	}

	tf := strings.TrimSpace(strings.ToLower(*typeFilter))
	if tf != "" && tf != "text" && tf != "url" && tf != "code" && tf != "command" {
		fmt.Fprintf(os.Stderr, "invalid --type: %q (expected text|url|code|command)\n", *typeFilter)
		os.Exit(2)
	}

	export := make([]ExportItem, 0, len(items))
	for _, it := range items {
		if *pinnedOnly && !it.Pinned {
			continue
		}
		if tf != "" && strings.ToLower(string(it.Type)) != tf {
			continue
		}
		if !since.IsZero() && it.LastSeenAt.Before(since) {
			continue
		}

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
