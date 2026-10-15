package reverify

import (
	"context"
	"fmt"
	"strings"

	"github.com/receipts/receipts/internal/gitstate"
)

// CheckSafe refuses re-verification on unsafe or ambiguous repository
// states unless force is passed. Force never deletes user data; it only
// permits testing the committed state while the checkout is dirty, and
// the receipt still labels the result committed state only.
// Tool owned .receipts/ noise never counts as dirt.
func CheckSafe(ctx context.Context, repoRoot, head string, force bool) error {
	if repoRoot == "" || head == "" {
		return fmt.Errorf("reverify: refusing: no committed HEAD to check out")
	}
	if _, err := gitstate.FindRoot(ctx, repoRoot); err != nil {
		return fmt.Errorf("reverify: refusing: %w", err)
	}
	changes, err := gitstate.Changed(ctx, repoRoot)
	if err != nil {
		return fmt.Errorf("reverify: refusing: %w", err)
	}
	dirty := false
	for _, c := range changes {
		if c.Path == ".receipts" || strings.HasPrefix(c.Path, ".receipts/") {
			continue
		}
		dirty = true
		break
	}
	if dirty && !force {
		return fmt.Errorf("reverify: refusing: checkout has uncommitted changes (pass --force to test the committed state anyway)")
	}
	return nil
}
