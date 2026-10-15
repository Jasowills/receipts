package gitstate

import (
	"github.com/receipts/receipts/internal/model"
)

// Baseline is the resolved before/after repository state for a receipt.
// HeadBefore is empty unless a run start snapshot or a verified session
// commit supports it; audit mode never invents a before state.
// DirtyBefore is unknown unless a run start snapshot observed it,
// because a commit alone cannot preserve working tree state.
type Baseline struct {
	HeadBefore   string
	HeadAfter    string
	BranchBefore string
	BranchAfter  string
	DirtyBefore  model.DirtyState
	DirtyAfter   bool
	Confidence   model.BaselineConfidence
	Warnings     []string
}

// ResolveBaseline applies the spec Section 11 resolution order:
// run start snapshot first (high), then a verified session commit
// (medium), otherwise current state only (low) or nothing (unknown).
// A run start from a different repository root is ignored.
func ResolveBaseline(current Snapshot, runStart *Snapshot, sessionCommit string) Baseline {
	var b Baseline
	b.HeadAfter = current.Head
	b.BranchAfter = current.Branch
	b.DirtyAfter = current.Dirty

	if runStart != nil && runStart.Root != "" && runStart.Root == current.Root {
		b.HeadBefore = runStart.Head
		b.BranchBefore = runStart.Branch
		if runStart.Dirty {
			b.DirtyBefore = model.DirtyTrue
		} else {
			b.DirtyBefore = model.DirtyFalse
		}
		b.Confidence = model.BaselineHigh
		return b
	}
	if sessionCommit != "" {
		b.HeadBefore = sessionCommit
		b.DirtyBefore = model.DirtyUnknown
		b.Confidence = model.BaselineMedium
		b.Warnings = []string{"session commit linked but pre-session working tree remains unknown"}
		return b
	}
	b.DirtyBefore = model.DirtyUnknown
	if current.Head == "" {
		b.Confidence = model.BaselineUnknown
		b.Warnings = []string{"no commits: no baseline available"}
		return b
	}
	b.Confidence = model.BaselineLow
	b.Warnings = []string{"audit mode: pre-session working tree unknown; git diff HEAD is not the exact agent diff"}
	return b
}
