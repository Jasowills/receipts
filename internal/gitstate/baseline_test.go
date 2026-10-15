package gitstate

import (
	"testing"

	"github.com/receipts/receipts/internal/model"
)

func snapWith(root, head, branch string, dirty bool) Snapshot {
	return Snapshot{Root: root, Head: head, Branch: branch, Dirty: dirty}
}

func TestBaselineRunStartHigh(t *testing.T) {
	current := snapWith("/repo", "aaa", "main", true)
	start := snapWith("/repo", "bbb", "main", false)
	b := ResolveBaseline(current, &start, "")
	if b.Confidence != model.BaselineHigh {
		t.Errorf("confidence = %q, want high", b.Confidence)
	}
	if b.HeadBefore != "bbb" || b.HeadAfter != "aaa" {
		t.Errorf("before/after = %q/%q, want bbb/aaa", b.HeadBefore, b.HeadAfter)
	}
	if b.DirtyBefore != model.DirtyFalse {
		t.Errorf("dirty before = %q, want false", b.DirtyBefore)
	}
	if !b.DirtyAfter {
		t.Errorf("dirty after = false, want true")
	}
	if b.BranchBefore != "main" || b.BranchAfter != "main" {
		t.Errorf("branches = %q/%q, want main/main", b.BranchBefore, b.BranchAfter)
	}
}

func TestBaselineRunStartDirtyTrue(t *testing.T) {
	current := snapWith("/repo", "aaa", "main", false)
	start := snapWith("/repo", "aaa", "main", true)
	b := ResolveBaseline(current, &start, "")
	if b.Confidence != model.BaselineHigh {
		t.Errorf("confidence = %q, want high", b.Confidence)
	}
	if b.DirtyBefore != model.DirtyTrue {
		t.Errorf("dirty before = %q, want true", b.DirtyBefore)
	}
}

func TestBaselineRunStartRootMismatchFallsThrough(t *testing.T) {
	current := snapWith("/repo", "aaa", "main", true)
	start := snapWith("/other", "bbb", "main", false)
	b := ResolveBaseline(current, &start, "")
	if b.Confidence != model.BaselineLow {
		t.Errorf("confidence = %q, want low (mismatched start ignored)", b.Confidence)
	}
	if b.HeadBefore != "" {
		t.Errorf("HeadBefore = %q, want empty (never invent)", b.HeadBefore)
	}
}

func TestBaselineSessionCommitMedium(t *testing.T) {
	current := snapWith("/repo", "aaa", "main", true)
	b := ResolveBaseline(current, nil, "ccc")
	if b.Confidence != model.BaselineMedium {
		t.Errorf("confidence = %q, want medium", b.Confidence)
	}
	if b.HeadBefore != "ccc" {
		t.Errorf("HeadBefore = %q, want ccc", b.HeadBefore)
	}
	if b.DirtyBefore != model.DirtyUnknown {
		t.Errorf("dirty before = %q, want unknown (commit cannot preserve worktree)", b.DirtyBefore)
	}
}

func TestBaselineAuditOnlyLow(t *testing.T) {
	current := snapWith("/repo", "aaa", "main", true)
	b := ResolveBaseline(current, nil, "")
	if b.Confidence != model.BaselineLow {
		t.Errorf("confidence = %q, want low", b.Confidence)
	}
	if b.HeadBefore != "" {
		t.Errorf("HeadBefore = %q, want empty (never invent before state)", b.HeadBefore)
	}
	if b.HeadAfter != "aaa" {
		t.Errorf("HeadAfter = %q, want aaa", b.HeadAfter)
	}
	if b.DirtyBefore != model.DirtyUnknown {
		t.Errorf("dirty before = %q, want unknown", b.DirtyBefore)
	}
	if len(b.Warnings) == 0 {
		t.Errorf("expected baseline uncertainty warning")
	}
}

func TestBaselineNoCommitsUnknown(t *testing.T) {
	current := snapWith("/repo", "", "", true)
	b := ResolveBaseline(current, nil, "")
	if b.Confidence != model.BaselineUnknown {
		t.Errorf("confidence = %q, want unknown", b.Confidence)
	}
	if b.HeadBefore != "" || b.HeadAfter != "" {
		t.Errorf("heads = %q/%q, want empty/empty", b.HeadBefore, b.HeadAfter)
	}
}

func TestBaselinePrefersRunStartOverSessionCommit(t *testing.T) {
	current := snapWith("/repo", "aaa", "main", true)
	start := snapWith("/repo", "bbb", "dev", false)
	b := ResolveBaseline(current, &start, "ccc")
	if b.Confidence != model.BaselineHigh || b.HeadBefore != "bbb" {
		t.Errorf("run start must win: confidence %q before %q", b.Confidence, b.HeadBefore)
	}
}
