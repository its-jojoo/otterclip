package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
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
	fmt.Println("Commands: add <text> | paste | list | pins | query <text> | count | pin <n> | unpin <n> | del <n> | pause | resume | help | quit")
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

		case "help":
			fmt.Println("Commands: add <text> | paste | list | pins | query <text> | count | pin <n> | unpin <n> | del <n> | pause | resume | help | quit")

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
			printItems(items)

		case "pins":
			items, err := store.ListRecent(ctx, 200) // scan more, then filter
			if err != nil {
				fmt.Println("error:", err)
				continue
			}
			pinned := make([]core.Item, 0, 50)
			for _, it := range items {
				if it.Pinned {
					pinned = append(pinned, it)
					if len(pinned) >= 50 {
						break
					}
				}
			}
			printItems(pinned)

		case "query", "q":
			if arg == "" {
				fmt.Println("usage: query <text>")
				continue
			}
			results, err := queryItems(ctx, store, arg, 80, 20) // scan 80, show top 20
			if err != nil {
				fmt.Println("error:", err)
				continue
			}
			if len(results) == 0 {
				fmt.Println("(no matches)")
				continue
			}
			printItems(results)

		case "count":
			n, err := store.Count(ctx)
			if err != nil {
				fmt.Println("error:", err)
				continue
			}
			fmt.Println(n)

		case "pin":
			n, ok := parseIndex(arg)
			if !ok {
				fmt.Println("usage: pin <n>")
				continue
			}
			if err := setPinnedByIndex(ctx, store, n, true); err != nil {
				fmt.Println("error:", err)
			}

		case "unpin":
			n, ok := parseIndex(arg)
			if !ok {
				fmt.Println("usage: unpin <n>")
				continue
			}
			if err := setPinnedByIndex(ctx, store, n, false); err != nil {
				fmt.Println("error:", err)
			}

		case "del":
			n, ok := parseIndex(arg)
			if !ok {
				fmt.Println("usage: del <n>")
				continue
			}
			if err := deleteByIndex(ctx, store, n); err != nil {
				fmt.Println("error:", err)
			}

		default:
			fmt.Println("unknown command:", cmd)
			fmt.Println("Commands: add <text> | paste | list | pins | query <text> | count | pin <n> | unpin <n> | del <n> | pause | resume | help | quit")
		}
	}

	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "stdin error:", err)
	}
}

func printItems(items []core.Item) {
	if len(items) == 0 {
		fmt.Println("(empty)")
		return
	}
	for i, it := range items {
		pin := " "
		if it.Pinned {
			pin = "★"
		}
		fmt.Printf("%2d %s [%s] %s\n", i+1, pin, it.Type, preview(it.Content, 80))
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

func parseIndex(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	if n <= 0 {
		return 0, false
	}
	return n, true
}

type pinStore interface {
	ListRecent(ctx context.Context, limit int) ([]core.Item, error)
	SetPinned(ctx context.Context, id string, pinned bool) error
	Delete(ctx context.Context, id string) error
}

func setPinnedByIndex(ctx context.Context, st pinStore, n int, pinned bool) error {
	items, err := st.ListRecent(ctx, 50)
	if err != nil {
		return err
	}
	if n > len(items) {
		return fmt.Errorf("index out of range (have %d)", len(items))
	}
	it := items[n-1]
	return st.SetPinned(ctx, it.ID, pinned)
}

func deleteByIndex(ctx context.Context, st pinStore, n int) error {
	items, err := st.ListRecent(ctx, 50)
	if err != nil {
		return err
	}
	if n > len(items) {
		return fmt.Errorf("index out of range (have %d)", len(items))
	}
	it := items[n-1]
	if it.Pinned {
		return fmt.Errorf("refusing to delete pinned item (unpin first)")
	}
	return st.Delete(ctx, it.ID)
}

func queryItems(ctx context.Context, st interface {
	ListRecent(ctx context.Context, limit int) ([]core.Item, error)
}, q string, scanLimit int, outLimit int) ([]core.Item, error) {
	items, err := st.ListRecent(ctx, scanLimit)
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
	now := time.Now()

	for _, it := range items {
		s := strings.ToLower(it.Content)

		matchScore := scoreMatch(s, q)
		if matchScore == 0 {
			continue
		}

		score := matchScore

		// pinned boost
		if it.Pinned {
			score += 5000
		}

		// recency boost (newer = higher)
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

	if outLimit <= 0 || outLimit > len(scoredItems) {
		outLimit = len(scoredItems)
	}

	out := make([]core.Item, 0, outLimit)
	for i := 0; i < outLimit; i++ {
		out = append(out, scoredItems[i].it)
	}
	return out, nil
}

func scoreMatch(s, q string) int {
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
