package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/receipts/receipts/internal/adapters"
)

// Adapter reads Claude Code JSONL transcripts read only.
// Sessions live at <ProjectsRoot>/<slug>/<uuid>.jsonl where the slug
// encodes the workdir with slashes replaced by dashes.
type Adapter struct {
	// ProjectsRoot overrides discovery. Empty means ~/.claude/projects.
	ProjectsRoot string
}

// Name returns the adapter name.
func (Adapter) Name() string { return "claude" }

func (a Adapter) root() string {
	if a.ProjectsRoot != "" {
		return a.ProjectsRoot
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

// Detect reports whether the projects root exists.
func (a Adapter) Detect(_ context.Context, _ string) (bool, error) {
	fi, err := os.Stat(a.root())
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return fi.IsDir(), nil
}

type candidate struct {
	path    string
	modTime time.Time
}

// LatestSession returns the most recently modified session for workdir.
// It prefers the slug directory match, then falls back to a bounded scan
// of recent files for a line level cwd match.
func (a Adapter) LatestSession(ctx context.Context, workdir string) (adapters.SessionRef, error) {
	abs, err := filepath.Abs(workdir)
	if err != nil {
		return adapters.SessionRef{}, err
	}
	slugged := filepath.Join(a.root(), slugForDir(abs))
	if ref, err := latestInDir(slugged); err == nil {
		return ref, nil
	} else if ctx.Err() != nil {
		return adapters.SessionRef{}, ctx.Err()
	}
	// Fallback: scan project dirs for a cwd match, newest first.
	ref, serr := latestByCwd(a.root(), abs)
	if serr != nil {
		return adapters.SessionRef{}, fmt.Errorf("no session recorded for %s", abs)
	}
	return ref, nil
}

func latestInDir(dir string) (adapters.SessionRef, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return adapters.SessionRef{}, err
	}
	var cands []candidate
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		cands = append(cands, candidate{path: filepath.Join(dir, e.Name()), modTime: fi.ModTime()})
	}
	if len(cands) == 0 {
		return adapters.SessionRef{}, fmt.Errorf("no sessions in %s", dir)
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].modTime.After(cands[j].modTime) })
	best := cands[0]
	return adapters.SessionRef{
		Agent:      "claude",
		ID:         strings.TrimSuffix(filepath.Base(best.path), ".jsonl"),
		Path:       best.path,
		ModifiedAt: best.modTime,
	}, nil
}

// latestByCwd scans at most 50 recently modified files per project dir
// looking for a line level cwd match, bounded to avoid huge scans.
func latestByCwd(root, workdir string) (adapters.SessionRef, error) {
	projDirs, err := os.ReadDir(root)
	if err != nil {
		return adapters.SessionRef{}, err
	}
	var cands []candidate
	for _, pd := range projDirs {
		if !pd.IsDir() {
			continue
		}
		dir := filepath.Join(root, pd.Name())
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		var files []candidate
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
				continue
			}
			fi, err := e.Info()
			if err != nil {
				continue
			}
			files = append(files, candidate{path: filepath.Join(dir, e.Name()), modTime: fi.ModTime()})
		}
		sort.Slice(files, func(i, j int) bool { return files[i].modTime.After(files[j].modTime) })
		if len(files) > 50 {
			files = files[:50]
		}
		for _, f := range files {
			if fileHasCwd(f.path, workdir) {
				cands = append(cands, f)
				break
			}
		}
	}
	if len(cands) == 0 {
		return adapters.SessionRef{}, fmt.Errorf("no cwd match")
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].modTime.After(cands[j].modTime) })
	best := cands[0]
	return adapters.SessionRef{
		Agent:      "claude",
		ID:         strings.TrimSuffix(filepath.Base(best.path), ".jsonl"),
		Path:       best.path,
		ModifiedAt: best.modTime,
	}, nil
}

// fileHasCwd scans at most 500 lines for a cwd match.
func fileHasCwd(path, workdir string) bool {
	if err := adapters.CheckFileSize(path); err != nil {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	checked := 0
	for _, raw := range bytes.Split(data, []byte("\n")) {
		if checked >= 500 {
			break
		}
		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) == 0 {
			continue
		}
		checked++
		var l line
		if json.Unmarshal(trimmed, &l) != nil {
			continue
		}
		if l.Cwd == workdir {
			return true
		}
	}
	return false
}

// Messages reads and normalizes one session file.
func (a Adapter) Messages(ctx context.Context, session adapters.SessionRef) ([]adapters.Message, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	fi, err := os.Stat(session.Path)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	_ = fi
	if err := adapters.CheckFileSize(session.Path); err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	data, err := os.ReadFile(session.Path)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	return parseJSONL(data)
}
