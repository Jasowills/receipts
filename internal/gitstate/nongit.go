package gitstate

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/receipts/receipts/internal/model"
)

// WarnNonGitProject is the receipt warning for projects outside Git.
// The audit pipeline (Task 31) attaches it when FindRoot reports ErrNotGit.
const WarnNonGitProject = "NON_GIT_PROJECT"

// IsRepo reports whether dir sits inside a Git working tree.
func IsRepo(ctx context.Context, dir string) bool {
	_, err := FindRoot(ctx, dir)
	return err == nil
}

// NonGitRepo builds the model repo record for non Git projects:
// no diff claims possible, baseline unknown. Callers must attach
// WarnNonGitProject and skip diff dependent checks. Run may still
// capture commands; reverify stays unavailable without a snapshot.
func NonGitRepo(workdir string) model.Repo {
	root := workdir
	if abs, err := filepath.Abs(workdir); err == nil {
		root = abs
	}
	yes := true
	return model.Repo{
		Root:               root,
		BaselineConfidence: model.BaselineUnknown,
		NonGit:             &yes,
	}
}

// AsRepo converts a collected snapshot plus baseline into the model
// repo record for Git projects.
func AsRepo(root string, snap Snapshot, b Baseline) model.Repo {
	dirtyAfter := snap.Dirty
	return model.Repo{
		Root:               root,
		HeadBefore:         b.HeadBefore,
		HeadAfter:          b.HeadAfter,
		BranchBefore:       b.BranchBefore,
		BranchAfter:        b.BranchAfter,
		BaselineConfidence: b.Confidence,
		DirtyBefore:        b.DirtyBefore,
		DirtyAfter:         &dirtyAfter,
	}
}

// IsNotGit reports whether err is or wraps ErrNotGit.
func IsNotGit(err error) bool { return errors.Is(err, ErrNotGit) }
