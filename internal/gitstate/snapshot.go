package gitstate

import (
	"context"
	"errors"
	"sort"
	"strings"
)

// FileState is one status entry. X is the staged code and Y the
// unstaged code from porcelain v1. Untracked marks ?? entries.
// RenamedFrom holds the source path for renames; full rename
// detection and diff hashes land in Task 17.
type FileState struct {
	Path        string
	X           byte
	Y           byte
	Untracked   bool
	RenamedFrom string
}

// Snapshot is the collected working tree state.
type Snapshot struct {
	Root   string
	Head   string
	Branch string
	Dirty  bool
	Files  []FileState
}

// Collect gathers root, HEAD, branch, and porcelain status for dir.
// It returns ErrNotGit outside a repository. A repository with no
// commits yields an empty Head rather than an error so new projects
// still report status; baseline confidence handling lands in Task 16.
// Branch errors are tolerated as empty for the same reason.
func Collect(ctx context.Context, dir string) (Snapshot, error) {
	root, err := FindRoot(ctx, dir)
	if err != nil {
		return Snapshot{}, err
	}
	var snap Snapshot
	snap.Root = root
	if head, herr := Head(ctx, root); herr == nil {
		snap.Head = head
	} else if errors.Is(herr, ErrNotGit) {
		return Snapshot{}, herr
	}
	if branch, berr := Branch(ctx, root); berr == nil {
		snap.Branch = branch
	}
	files, err := status(ctx, root)
	if err != nil {
		return Snapshot{}, err
	}
	snap.Files = files
	snap.Dirty = len(files) > 0
	return snap, nil
}

// status runs porcelain v1 -z with full untracked file expansion and parses
// NUL separated entries. -z keeps paths with spaces and unicode intact.
// --untracked-files=all lists individual untracked files instead of bare
// directory entries so integrity checks see real paths. Rename entries
// consume two fields (new path then source path).
func status(ctx context.Context, dir string) ([]FileState, error) {
	out, err := runGit(ctx, dir, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	fields := strings.Split(out, "\x00")
	var files []FileState
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if f == "" {
			continue
		}
		if len(f) < 4 {
			continue
		}
		x, y := f[0], f[1]
		path := f[3:]
		fs := FileState{Path: path, X: x, Y: y}
		if x == '?' && y == '?' {
			fs.Untracked = true
		}
		if x == 'R' || x == 'C' {
			if i+1 < len(fields) && fields[i+1] != "" {
				fs.RenamedFrom = fields[i+1]
				i++
			}
		}
		files = append(files, fs)
	}
	sort.Slice(files, func(a, b int) bool { return files[a].Path < files[b].Path })
	return files, nil
}
