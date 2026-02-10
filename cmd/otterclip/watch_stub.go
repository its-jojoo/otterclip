//go:build !darwin

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/its-jojoo/otterclip/internal/usecase/capture"
)

func runWatchMode(ctx context.Context, svc *capture.Service, interval time.Duration) {
	_ = ctx
	_ = svc
	_ = interval
	fmt.Println("watch mode is not supported on this OS yet (darwin only for now).")
}
