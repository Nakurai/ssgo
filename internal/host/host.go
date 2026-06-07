package host

import (
	"context"
	"fmt"
)

// Host is a hosting-provider adapter.
type Host interface {
	Codename() string
	Setup(ctx context.Context, root string, force bool) error
	Deploy(ctx context.Context, root string) error
}

var registry = map[string]Host{}

func register(h Host) {
	registry[h.Codename()] = h
}

// Get returns the adapter for the given codename and whether it was found.
// An empty codename always returns (nil, false).
func Get(codename string) (Host, bool) {
	if codename == "" {
		return nil, false
	}
	h, ok := registry[codename]
	return h, ok
}

// Supported returns all registered codenames.
func Supported() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	return out
}

// ErrUnknown is returned when a codename does not match any registered adapter.
func ErrUnknown(codename string) error {
	return fmt.Errorf("unknown host provider %q; supported: %v", codename, Supported())
}
