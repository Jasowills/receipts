// Package registry wires the concrete transcript adapters in priority
// order. It lives in a subpackage because package adapters cannot import
// its own subpackages in Go; the generic Auto engine plus Attempt types
// stay in the parent package.
package registry

import (
	"context"
	"fmt"

	"github.com/receipts/receipts/internal/adapters"
	"github.com/receipts/receipts/internal/adapters/claude"
	"github.com/receipts/receipts/internal/adapters/opencode"
)

// Default returns adapters in priority order: opencode, then claude.
// Codex joins here after its Phase G format verification.
func Default() []adapters.Adapter {
	return []adapters.Adapter{opencode.Adapter{}, claude.Adapter{}}
}

// ByName returns one adapter by name.
func ByName(name string) (adapters.Adapter, error) {
	for _, a := range Default() {
		if a.Name() == name {
			return a, nil
		}
	}
	return nil, fmt.Errorf("adapters: unknown agent %q (expected opencode|claude)", name)
}

// Auto runs auto detection over the default adapters.
func Auto(ctx context.Context, workdir string) (adapters.AutoResult, error) {
	return adapters.Auto(ctx, workdir, Default())
}
