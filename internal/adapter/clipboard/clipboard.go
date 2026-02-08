package clipboard

import "context"

// Watch emits a signal when clipboard *may* have changed.
// Implementations can poll or subscribe to OS events.
type Watcher interface {
	Watch(ctx context.Context) (<-chan struct{}, error)
	ReadText() (string, error)
}
