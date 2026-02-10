//go:build !darwin

package clipboard

import (
	"context"
	"errors"
)

var ErrUnsupported = errors.New("clipboard watcher not implemented for this OS yet")

type UnsupportedWatcher struct{}

func NewUnsupportedWatcher() *UnsupportedWatcher { return &UnsupportedWatcher{} }

func (w *UnsupportedWatcher) Watch(ctx context.Context) (<-chan struct{}, error) {
	_ = ctx
	return nil, ErrUnsupported
}

func (w *UnsupportedWatcher) ReadText() (string, error) {
	return "", ErrUnsupported
}
