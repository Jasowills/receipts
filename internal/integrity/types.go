package integrity

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/receipts/receipts/internal/gitstate"
	"github.com/receipts/receipts/internal/model"
)

// FileContent is one changed file with before/after text.
// HasBefore false means added; HasAfter false means deleted.
// Binary files carry no reliable text; content checks skip them.
type FileContent struct {
	Path      string
	Before    string
	After     string
	HasBefore bool
	HasAfter  bool
	Binary    bool
}

// AddedLine is one added diff line with its 1 based line number.
type AddedLine struct {
	No   int
	Text string
}

// Scope holds optional path policy. Empty allowed plus empty forbidden
// means no user scope exists and no scope flags are raised.
type Scope struct {
	Allowed   []string
	Forbidden []string
}

// Checks toggles each integrity check independently.
type Checks struct {
	TestFiles    bool
	Assertions   bool
	Skips        bool
	OutOfScope   bool
	Dependencies bool
	BuildConfig  bool
	Secrets      bool
}

// DefaultChecks enables every check.
func DefaultChecks() Checks {
	return Checks{TestFiles: true, Assertions: true, Skips: true, OutOfScope: true, Dependencies: true, BuildConfig: true, Secrets: true}
}

// Input is everything the checks need: changed files with contents,
// added lines keyed by normalized path, scope policy, toggles, and the
// baseline confidence used for the uncertainty flag.
type Input struct {
	Files              []FileContent
	Added              map[string][]AddedLine
	Scope              Scope
	Checks             Checks
	BaselineConfidence model.BaselineConfidence
}

// Run executes enabled checks in fixed order and returns flags sorted by
// code, title, then paths with stable F-001 IDs. It never edits files.
func Run(in Input) []model.Flag {
	files := normalizeFiles(in.Files)
	var flags []model.Flag
	if in.BaselineConfidence == "" || in.BaselineConfidence == model.BaselineLow ||
		in.BaselineConfidence == model.BaselineMedium || in.BaselineConfidence == model.BaselineUnknown {
		conf := in.BaselineConfidence
		if conf == "" {
			conf = model.BaselineUnknown
		}
		flags = append(flags, newFlag("BASELINE_UNCERTAIN", model.SeverityInfo,
			"Baseline uncertain", fmt.Sprintf("baseline confidence is %s; audit mode may miss pre-existing changes and cannot always attribute edits", conf), nil))
	}
	if in.Checks.TestFiles {
		flags = append(flags, checkTestFiles(files)...)
	}
	if in.Checks.Assertions {
		flags = append(flags, checkAssertions(files)...)
	}
	if in.Checks.Skips {
		flags = append(flags, checkSkips(files, in.Added)...)
	}
	if in.Checks.OutOfScope {
		flags = append(flags, checkScope(files, in.Scope)...)
	}
	if in.Checks.Dependencies {
		flags = append(flags, checkDependencies(files)...)
	}
	if in.Checks.BuildConfig {
		flags = append(flags, checkBuildConfig(files)...)
	}
	if in.Checks.Secrets {
		flags = append(flags, checkSecrets(files, in.Added)...)
	}
	sort.Slice(flags, func(a, b int) bool {
		if flags[a].Code != flags[b].Code {
			return flags[a].Code < flags[b].Code
		}
		if flags[a].Title != flags[b].Title {
			return flags[a].Title < flags[b].Title
		}
		return stringsJoin(flags[a].Paths) < stringsJoin(flags[b].Paths)
	})
	for i := range flags {
		flags[i].ID = fmt.Sprintf("F-%03d", i+1)
		if flags[i].EvidenceRefs == nil {
			flags[i].EvidenceRefs = []string{}
		}
	}
	return flags
}

func normalizeFiles(files []FileContent) []FileContent {
	out := make([]FileContent, 0, len(files))
	for _, f := range files {
		norm, err := gitstate.NormalizePath(".", f.Path)
		if err != nil {
			out = append(out, FileContent{Path: f.Path, Before: f.Before, After: f.After, HasBefore: f.HasBefore, HasAfter: f.HasAfter, Binary: f.Binary})
			continue
		}
		f.Path = norm
		out = append(out, f)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Path < out[b].Path })
	return out
}

func newFlag(code string, sev model.Severity, title, details string, paths []string) model.Flag {
	return model.Flag{Code: code, Severity: sev, Title: title, Details: details, Paths: paths, EvidenceRefs: []string{}}
}

var flagIDRe = regexp.MustCompile(`^F-(\d+)$`)

// FillFlagIDs assigns stable IDs to flags lacking one, continuing the
// highest existing number. Pipelines use it for flags appended after
// Run, such as conflict or capture notes.
func FillFlagIDs(flags []model.Flag) {
	max := 0
	for _, f := range flags {
		if m := flagIDRe.FindStringSubmatch(f.ID); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil && n > max {
				max = n
			}
		}
	}
	next := max + 1
	for i := range flags {
		if flags[i].ID == "" {
			flags[i].ID = fmt.Sprintf("F-%03d", next)
			next++
		}
		if flags[i].EvidenceRefs == nil {
			flags[i].EvidenceRefs = []string{}
		}
	}
}

func stringsJoin(s []string) string {
	out := ""
	for i, p := range s {
		if i > 0 {
			out += "\x00"
		}
		out += p
	}
	return out
}
