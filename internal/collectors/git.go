package collectors

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/receipts/receipts/internal/gitstate"
	"github.com/receipts/receipts/internal/integrity"
	"github.com/receipts/receipts/internal/model"
)

// MaxContentBytes bounds per file content reads for integrity checks.
// Larger files stay listed in diff paths but skip content analysis.
const MaxContentBytes = 1024 * 1024

// OwnedDir is tool owned storage, never agent work. It stays out of
// diff evidence and integrity inputs so receipts do not flag themselves.
const OwnedDir = ".receipts/"

// DropOwnedDir removes tool owned paths from a change list.
func DropOwnedDir(changes []gitstate.ChangedPath) []gitstate.ChangedPath {
	out := changes[:0]
	for _, c := range changes {
		if len(c.Path) >= len(OwnedDir) && c.Path[:len(OwnedDir)] == OwnedDir {
			continue
		}
		if c.Path == ".receipts" {
			continue
		}
		out = append(out, c)
	}
	return out
}

// GitEvidence is the assembled Git side of an audit: the repo record,
// ordered evidence, integrity flags, and collector warnings.
type GitEvidence struct {
	Repo     model.Repo
	Evidence []model.Evidence
	Flags    []model.Flag
	Warnings []string
}

// BuildGitEvidence collects with default checks and no scope policy.
func BuildGitEvidence(ctx context.Context, workdir string) (GitEvidence, error) {
	return BuildGitEvidenceWith(ctx, workdir, GitOptions{
		Scope:  integrity.Scope{},
		Checks: integrity.DefaultChecks(),
	})
}

// GitOptions carries scope policy and check toggles.
type GitOptions struct {
	Scope  integrity.Scope
	Checks integrity.Checks
}

// BuildGitEvidenceWith collects the current snapshot, baseline, diff,
// file contents, and integrity flags for workdir at tier 0. It runs no
// project commands and changes no tracked files.
func BuildGitEvidenceWith(ctx context.Context, workdir string, opts GitOptions) (GitEvidence, error) {
	abs, err := filepath.Abs(workdir)
	if err != nil {
		return GitEvidence{}, err
	}
	root, err := gitstate.FindRoot(ctx, abs)
	if err != nil {
		if gitstate.IsNotGit(err) {
			return GitEvidence{
				Repo:     gitstate.NonGitRepo(abs),
				Warnings: []string{gitstate.WarnNonGitProject},
			}, nil
		}
		return GitEvidence{}, err
	}
	snap, err := gitstate.Collect(ctx, root)
	if err != nil {
		return GitEvidence{}, err
	}
	base := gitstate.ResolveBaseline(snap, nil, "")
	repo := gitstate.AsRepo(root, snap, base)
	changes, err := gitstate.Changed(ctx, root)
	if err != nil {
		return GitEvidence{}, err
	}
	changes = DropOwnedDir(changes)
	changedPaths := make([]string, 0, len(changes))
	for _, c := range changes {
		changedPaths = append(changedPaths, c.Path)
	}
	files, added, warnings := ChangedFiles(ctx, root, snap.Head, changes)
	in := integrity.Input{
		Files: files, Added: added, Scope: opts.Scope, Checks: opts.Checks,
		BaselineConfidence: base.Confidence,
	}
	flags := integrity.Run(in)
	out := GitEvidence{Repo: repo, Warnings: append(base.Warnings, warnings...)}
	out.Evidence = append(out.Evidence, model.Evidence{
		ID:        "E-001",
		Kind:      model.EvidenceGitSnapshot,
		Timestamp: snapTime(),
		Data: map[string]any{
			"root": root, "head_before": base.HeadBefore, "head_after": base.HeadAfter,
			"branch_before": base.BranchBefore, "branch_after": base.BranchAfter,
			"baseline_confidence": string(base.Confidence),
			"dirty_before":        string(base.DirtyBefore), "dirty_after": base.DirtyAfter,
		},
	})
	diffData := map[string]any{
		"changed_paths": changedPaths,
		"diff_hash":     gitstate.DiffHash(changes),
	}
	depAdded, manifests := DepDeltas(files)
	if len(depAdded) > 0 {
		diffData["added_dependencies"] = depAdded
	}
	if len(manifests) > 0 {
		diffData["manifests_changed"] = manifests
	}
	out.Evidence = append(out.Evidence, model.Evidence{
		ID: "E-002", Kind: model.EvidenceDiff, Timestamp: snapTime(), Data: diffData,
	})
	for i := range flags {
		flags[i].EvidenceRefs = append(flags[i].EvidenceRefs, "E-002")
	}
	out.Flags = flags
	return out, nil
}

// DepDeltas aggregates manifest changes for verdict joining.
func DepDeltas(files []integrity.FileContent) ([]string, []string) {
	addedSet := map[string]bool{}
	var manifests []string
	for _, f := range files {
		before, after := integrity.ManifestDepMaps(f.Path, f.Before, f.After, f.HasBefore, f.HasAfter)
		if before == nil && after == nil {
			continue
		}
		manifests = append(manifests, f.Path)
		for k := range after {
			if _, ok := before[k]; !ok {
				addedSet[k] = true
			}
		}
	}
	var added []string
	for k := range addedSet {
		added = append(added, k)
	}
	sort.Strings(added)
	sort.Strings(manifests)
	return added, manifests
}

// ChangedFiles reads before and after contents plus added lines for
// changed paths, skipping oversized files with warnings.
func ChangedFiles(ctx context.Context, root, head string, changes []gitstate.ChangedPath) ([]integrity.FileContent, map[string][]integrity.AddedLine, []string) {
	var files []integrity.FileContent
	added := map[string][]integrity.AddedLine{}
	var warnings []string
	for _, c := range changes {
		if c.Status == '?' {
			data, err := readWorktreeFile(root, c.Path)
			if err != nil {
				continue
			}
			if len(data) > MaxContentBytes {
				warnings = append(warnings, "skipped large untracked file for content checks: "+c.Path)
				files = append(files, integrity.FileContent{Path: c.Path, After: "", HasBefore: false, HasAfter: true, Binary: isBinary(data)})
				continue
			}
			text := string(data)
			files = append(files, integrity.FileContent{Path: c.Path, After: text, HasBefore: false, HasAfter: true, Binary: isBinary(data)})
			added[c.Path] = numberedLines(text)
			continue
		}
		before := showHeadFile(ctx, root, head, c.Path)
		afterData, rerr := readWorktreeFile(root, c.Path)
		if rerr != nil {
			// Deleted from worktree.
			files = append(files, integrity.FileContent{Path: c.Path, Before: before, HasBefore: true, HasAfter: false, Binary: c.Binary})
			continue
		}
		if len(afterData) > MaxContentBytes {
			warnings = append(warnings, "skipped large file for content checks: "+c.Path)
			files = append(files, integrity.FileContent{Path: c.Path, Before: before, HasBefore: true, HasAfter: true, Binary: true})
			continue
		}
		after := string(afterData)
		fc := integrity.FileContent{Path: c.Path, Before: before, After: after, HasBefore: true, HasAfter: true, Binary: c.Binary || isBinary(afterData)}
		files = append(files, fc)
		if !fc.Binary {
			added[c.Path] = diffAddedLines(ctx, root, c.Path)
		}
	}
	return files, added, warnings
}

func readWorktreeFile(root, p string) ([]byte, error) {
	return os.ReadFile(filepath.Join(root, p))
}

func isBinary(data []byte) bool {
	return bytes.IndexByte(data, 0) >= 0
}

func numberedLines(text string) []integrity.AddedLine {
	lines := strings.Split(text, "\n")
	out := make([]integrity.AddedLine, 0, len(lines))
	for i, ln := range lines {
		out = append(out, integrity.AddedLine{No: i + 1, Text: ln})
	}
	return out
}

// showHeadFile returns HEAD content or "" when absent.
func showHeadFile(ctx context.Context, root, head, p string) string {
	if head == "" {
		return ""
	}
	cmd := exec.CommandContext(ctx, "git", "-C", root, "show", "HEAD:"+p)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return ""
	}
	return stdout.String()
}

var hunkRe = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// diffAddedLines parses -U0 diff output into added lines with numbers.
func diffAddedLines(ctx context.Context, root, p string) []integrity.AddedLine {
	cmd := exec.CommandContext(ctx, "git", "-C", root, "diff", "--no-color", "--no-ext-diff", "--no-textconv", "-U0", "HEAD", "--", p)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil
	}
	var out []integrity.AddedLine
	newNo := 0
	for _, line := range strings.Split(stdout.String(), "\n") {
		if m := hunkRe.FindStringSubmatch(line); m != nil {
			n, _ := strconv.Atoi(m[1])
			newNo = n
			continue
		}
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			out = append(out, integrity.AddedLine{No: newNo, Text: line[1:]})
			newNo++
		} else if strings.HasPrefix(line, " ") {
			newNo++
		}
	}
	return out
}

func snapTime() time.Time { return time.Now().UTC() }
