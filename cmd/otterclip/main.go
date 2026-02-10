package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/its-jojoo/otterclip/internal/adapter/storage/sqlite"
	"github.com/its-jojoo/otterclip/internal/core"
	"github.com/its-jojoo/otterclip/internal/usecase/capture"
)

func main() {
	var (
		dbPath       = flag.String("db", "./otterclip.dev.db", "sqlite db path")
		maxItems     = flag.Int("max-items", 5000, "max clipboard history items")
		ignoreCSV    = flag.String("ignore", "password=,token=,apikey=,secret=,authorization: bearer", "comma-separated ignore patterns (substring match)")
		useRegex     = flag.Bool("ignore-regex", false, "treat ignore patterns as regex")
		dedupeConsec = flag.Bool("dedupe-consecutive", true, "dedupe consecutive items")

		watch    = flag.Bool("watch", false, "watch system clipboard and capture automatically (darwin only for now)")
		interval = flag.Duration("interval", 350*time.Millisecond, "clipboard polling interval (darwin)")
	)
	flag.Parse()

	patterns := splitCSV(*ignoreCSV)

	pf, err := core.NewPrivacyFilter(patterns, *useRegex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid ignore patterns: %v\n", err)
		os.Exit(1)
	}

	store, err := sqlite.Open(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db open error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	svc := capture.New(store, pf, capture.Config{
		MaxItems:          *maxItems,
		DedupeConsecutive: *dedupeConsec,
	})

	// Cancelable context (Ctrl+C friendly)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if *watch {
		fmt.Println("OtterClip (watch mode)")
		fmt.Println("DB:", *dbPath)
		runWatchMode(ctx, svc, *interval) // implemented via build tags
		return
	}

	fmt.Println("OtterClip (dev mode)")
	fmt.Println("DB:", *dbPath)
	fmt.Println("Commands: add <text> | paste | list | count | pause | resume | quit")
	fmt.Println("Tip: 'paste' lets you type/paste a full line, then hit Enter.")
	fmt.Println("Tip: run with --watch to capture the real clipboard (macOS only for now).")

	paused := false
	sc := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !sc.Scan() {
			break
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		cmd, arg := splitCmd(line)

		switch cmd {
		case "quit", "exit":
			return

		case "pause":
			paused = true
			fmt.Println("capture paused")

		case "resume":
			paused = false
			fmt.Println("capture resumed")

		case "add":
			if paused {
				fmt.Println("paused: not capturing")
				continue
			}
			if arg == "" {
				fmt.Println("usage: add <text>")
				continue
			}
			saveOne(ctx, svc, arg)

		case "paste":
			if paused {
				fmt.Println("paused: not capturing")
				continue
			}
			fmt.Print("(paste) ")
			if !sc.Scan() {
				return
			}
			txt := sc.Text()
			saveOne(ctx, svc, txt)

		case "list":
			items, err := store.ListRecent(ctx, 20)
			if err != nil {
				fmt.Println("error:", err)
				continue
			}
			if len(items) == 0 {
				fmt.Println("(empty)")
				continue
			}
			for i, it := range items {
				pin := " "
				if it.Pinned {
					pin = "★"
				}
				fmt.Printf("%2d %s [%s] %s\n", i+1, pin, it.Type, preview(it.Content, 80))
			}

		case "count":
			n, err := store.Count(ctx)
			if err != nil {
				fmt.Println("error:", err)
				continue
			}
			fmt.Println(n)

		default:
			fmt.Println("unknown command:", cmd)
			fmt.Println("Commands: add <text> | paste | list | count | pause | resume | quit")
		}
	}

	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "stdin error:", err)
	}
}

func saveOne(ctx context.Context, svc *capture.Service, raw string) {
	_, saved, err := svc.ProcessText(ctx, raw)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	if !saved {
		fmt.Println("(ignored)")
		return
	}
	fmt.Println("saved")
}

func splitCmd(s string) (cmd, arg string) {
	parts := strings.Fields(s)
	cmd = strings.ToLower(parts[0])
	if len(parts) > 1 {
		arg = strings.TrimSpace(s[len(parts[0]):])
	}
	return cmd, arg
}

func splitCSV(s string) []string {
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		r = strings.TrimSpace(r)
		if r != "" {
			out = append(out, r)
		}
	}
	return out
}

func preview(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
