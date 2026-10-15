package adapters

import (
	"context"
	"fmt"
	"strings"
)

// Attempt records one adapter tried during auto detection: whether its
// storage was detected, which session won, or why it was skipped.
// The audit pipeline converts attempts into adapter_status evidence.
type Attempt struct {
	Adapter   string
	Detected  bool
	SessionID string
	Reason    string
}

// AutoResult is the outcome of Auto: the winning adapter and session
// plus every attempt in priority order.
type AutoResult struct {
	Adapter  Adapter
	Ref      SessionRef
	Attempts []Attempt
}

// Auto tries candidates in slice order (the priority order) and returns
// the first adapter with a session for workdir. Every attempt is
// recorded with its reason so receipts can show what was tried.
func Auto(ctx context.Context, workdir string, candidates []Adapter) (AutoResult, error) {
	var attempts []Attempt
	if len(candidates) == 0 {
		return AutoResult{Attempts: attempts}, fmt.Errorf("adapters: no candidates for %s", workdir)
	}
	for _, c := range candidates {
		if err := ctx.Err(); err != nil {
			return AutoResult{Attempts: attempts}, fmt.Errorf("adapters: %w", err)
		}
		detected, derr := c.Detect(ctx, workdir)
		if derr != nil {
			attempts = append(attempts, Attempt{Adapter: c.Name(), Reason: derr.Error()})
			continue
		}
		if !detected {
			attempts = append(attempts, Attempt{Adapter: c.Name(), Reason: "storage not detected"})
			continue
		}
		ref, lerr := c.LatestSession(ctx, workdir)
		if lerr != nil {
			attempts = append(attempts, Attempt{Adapter: c.Name(), Detected: true, Reason: lerr.Error()})
			continue
		}
		attempts = append(attempts, Attempt{Adapter: c.Name(), Detected: true, SessionID: ref.ID})
		return AutoResult{Adapter: c, Ref: ref, Attempts: attempts}, nil
	}
	var parts []string
	for _, a := range attempts {
		parts = append(parts, a.Adapter+": "+a.Reason)
	}
	return AutoResult{Attempts: attempts}, fmt.Errorf("adapters: no session for %s (%s)", workdir, strings.Join(parts, "; "))
}
