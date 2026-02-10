//go:build darwin

package clipboard

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

type DarwinWatcher struct {
	Interval time.Duration

	last string
}

func NewDarwinWatcher(interval time.Duration) *DarwinWatcher {
	if interval <= 0 {
		interval = 350 * time.Millisecond
	}
	return &DarwinWatcher{Interval: interval}
}

func (w *DarwinWatcher) Watch(ctx context.Context) (<-chan struct{}, error) {
	ch := make(chan struct{}, 1)

	// prime initial state
	if txt, err := w.ReadText(); err == nil {
		w.last = txt
	}

	t := time.NewTicker(w.Interval)

	go func() {
		defer t.Stop()
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				txt, err := w.ReadText()
				if err != nil {
					continue
				}
				if txt != "" && txt != w.last {
					w.last = txt
					select {
					case ch <- struct{}{}:
					default:
					}
				}
			}
		}
	}()

	return ch, nil
}

func (w *DarwinWatcher) ReadText() (string, error) {
	// pbpaste reads the system clipboard (text)
	cmd := exec.Command("pbpaste")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	// keep raw as-is; normalization happens in core
	return strings.TrimRight(out.String(), "\n"), nil
}
