package gitstate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ChangedPath is one worktree change versus HEAD or an untracked file.
// Status is the name-status letter: A, M, D, R, C, T, U, or ? for
// untracked. PatchHash fingerprints the change: for tracked files it is
// hex SHA256 of git diff --no-color --no-ext-diff --no-textconv -U0 output;
// for untracked files it is hex SHA256 of the worktree file content.
// The two forms are both deterministic fingerprints, not unified diffs.
type ChangedPath struct {
	Path        string
	Status      byte
	RenamedFrom string
	Binary      bool
	PatchHash   string
}

// Changed returns sorted worktree changes: tracked diffs versus HEAD
// merged with untracked files from status. Outside a repository it
// returns ErrNotGit.
func Changed(ctx context.Context, dir string) ([]ChangedPath, error) {
	root, err := FindRoot(ctx, dir)
	if err != nil {
		return nil, err
	}
	nameOut, err := runGit(ctx, root, "diff", "--name-status", "-z", "-M", "HEAD")
	if err != nil {
		// No commits yet: diff HEAD fails; fall back to status only so
		// untracked files still report. Tracked changes need a HEAD.
		if strings.Contains(err.Error(), "bad default revision") || strings.Contains(err.Error(), "unknown revision") {
			return untrackedOnly(ctx, root)
		}
		return nil, err
	}
	binaryByPath, _ := numstatBinary(ctx, root)
	changes := parseNameStatusZ(nameOut)
	for i := range changes {
		norm, nerr := NormalizePath(root, changes[i].Path)
		if nerr != nil {
			return nil, nerr
		}
		changes[i].Path = norm
		if changes[i].RenamedFrom != "" {
			rn, rerr := NormalizePath(root, changes[i].RenamedFrom)
			if rerr != nil {
				return nil, rerr
			}
			changes[i].RenamedFrom = rn
		}
		changes[i].Binary = binaryByPath[changes[i].Path]
		patch, perr := runGit(ctx, root, "diff", "--no-color", "--no-ext-diff", "--no-textconv", "-U0", "HEAD", "--", changes[i].Path)
		if perr != nil {
			return nil, perr
		}
		changes[i].PatchHash = hexHash([]byte(patch))
	}
	untracked, err := untrackedOnly(ctx, root)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, c := range changes {
		seen[c.Path] = true
	}
	for _, u := range untracked {
		if !seen[u.Path] {
			changes = append(changes, u)
		}
	}
	sort.Slice(changes, func(a, b int) bool { return changes[a].Path < changes[b].Path })
	return changes, nil
}

// untrackedOnly lists untracked files with content hashes.
func untrackedOnly(ctx context.Context, root string) ([]ChangedPath, error) {
	snapFiles, err := status(ctx, root)
	if err != nil {
		return nil, err
	}
	var out []ChangedPath
	for _, f := range snapFiles {
		if !f.Untracked {
			continue
		}
		norm, nerr := NormalizePath(root, f.Path)
		if nerr != nil {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(root, norm))
		if rerr != nil {
			continue
		}
		out = append(out, ChangedPath{Path: norm, Status: '?', PatchHash: hexHash(data)})
	}
	return out, nil
}

// numstatBinary returns paths marked binary (- -) by numstat.
func numstatBinary(ctx context.Context, root string) (map[string]bool, error) {
	out, err := runGit(ctx, root, "diff", "--numstat", "-z", "HEAD")
	if err != nil {
		return nil, err
	}
	marks := map[string]bool{}
	if out == "" {
		return marks, nil
	}
	for _, field := range strings.Split(out, "\x00") {
		if field == "" {
			continue
		}
		// numstat lines are "added\tdeleted\tpath" possibly with the
		// path prefixed after tabs; -z keeps the whole line in one field.
		parts := strings.SplitN(field, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		if parts[0] == "-" && parts[1] == "-" {
			marks[parts[2]] = true
		}
	}
	return marks, nil
}

func parseNameStatusZ(out string) []ChangedPath {
	if out == "" {
		return nil
	}
	// diff --name-status -z emits NUL separated fields where each record
	// starts with a bare status field (M, A, D, R100, ...) followed by one
	// path field, or two for renames and copies in old then new order.
	// This differs from status porcelain -z, which joins status and path
	// in one field.
	fields := strings.Split(out, "\x00")
	var changes []ChangedPath
	i := 0
	for i < len(fields) {
		st := fields[i]
		i++
		if st == "" {
			continue
		}
		if !isNameStatusField(st) {
			continue
		}
		if i >= len(fields) {
			break
		}
		p := fields[i]
		i++
		if p == "" {
			continue
		}
		c := ChangedPath{Path: p, Status: st[0]}
		if st[0] == 'R' || st[0] == 'C' {
			if i < len(fields) && fields[i] != "" {
				c.RenamedFrom = c.Path
				c.Path = fields[i]
				i++
			}
		}
		changes = append(changes, c)
	}
	return changes
}

func isNameStatusField(f string) bool {
	if len(f) < 1 || len(f) > 4 {
		return false
	}
	if f[0] < 'A' || f[0] > 'Z' {
		return false
	}
	for _, r := range f[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func hexHash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// DiffHash returns a stable overall fingerprint of sorted changes.
func DiffHash(changes []ChangedPath) string {
	cp := append([]ChangedPath{}, changes...)
	sort.Slice(cp, func(a, b int) bool { return cp[a].Path < cp[b].Path })
	var b strings.Builder
	for _, c := range cp {
		b.WriteByte(c.Status)
		b.WriteByte(' ')
		b.WriteString(c.Path)
		b.WriteByte(' ')
		b.WriteString(c.PatchHash)
		b.WriteByte('\n')
	}
	return hexHash([]byte(b.String()))
}
