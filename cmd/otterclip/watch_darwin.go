//go:build darwin

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/its-jojoo/otterclip/internal/adapter/clipboard"
	"github.com/its-jojoo/otterclip/internal/usecase/capture"
)

func runWatchMode(ctx context.Context, svc *capture.Service, interval time.Duration) {
	w := clipboard.NewDarwinWatcher(interval)

	events, err := w.Watch(ctx)
	if err != nil {
		fmt.Println("watch error:", err)
		return
	}

	fmt.Println("watching clipboard... (Ctrl+C to exit)")
	for range events {
		txt, err := w.ReadText()
		if err != nil {
			continue
		}
		_, saved, err := svc.ProcessText(ctx, txt)
		if err != nil {
			fmt.Println("capture error:", err)
			continue
		}
		if saved {
			fmt.Println("captured:", preview(txt, 60))
		}
	}
}
