package verdict

import (
	"strings"

	"github.com/receipts/receipts/internal/model"
)

// argvOf extracts the argument vector from command shaped evidence data,
// accepting the JSON decoded []any form and the in memory []string form.
func argvOf(e model.Evidence) []string {
	raw, ok := e.Data["argv"]
	if !ok {
		if cmd, ok := e.Data["command"].(string); ok && cmd != "" {
			return strings.Fields(cmd)
		}
		return nil
	}
	if ss, ok := raw.([]string); ok {
		return append([]string{}, ss...)
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		s, ok := item.(string)
		if !ok {
			return nil
		}
		out = append(out, s)
	}
	return out
}

func exitOf(e model.Evidence) (int, bool) {
	raw, ok := e.Data["exit_code"]
	if !ok {
		return 0, false
	}
	switch n := raw.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

func baseOf(argv0 string) string {
	if i := strings.LastIndexByte(argv0, '/'); i >= 0 {
		argv0 = argv0[i+1:]
	}
	return argv0
}

// classifyArgv reports whether argv runs a test, build, or lint command.
// Anything else yields "". Matching stays narrow so a build never
// verifies a test claim and test output never verifies lint.
func classifyArgv(argv []string) string {
	if len(argv) == 0 {
		return ""
	}
	base := baseOf(argv[0])
	rest := argv[1:]
	has := func(words ...string) bool {
		for _, a := range rest {
			for _, w := range words {
				if a == w {
					return true
				}
			}
		}
		return false
	}
	joined := strings.Join(rest, " ")
	_ = joined
	switch base {
	case "npm", "pnpm", "yarn", "bun":
		if len(rest) > 0 && rest[0] == "test" {
			return "test"
		}
		if len(rest) > 1 && rest[0] == "run" {
			switch rest[1] {
			case "test":
				return "test"
			case "build":
				return "build"
			case "lint":
				return "lint"
			}
		}
		return ""
	case "go":
		if has("test") {
			return "test"
		}
		if has("build") {
			return "build"
		}
		if has("vet") {
			return "lint"
		}
		return ""
	case "cargo":
		if has("test") {
			return "test"
		}
		if has("build") {
			return "build"
		}
		if has("clippy") {
			return "lint"
		}
		return ""
	case "pytest", "vitest", "jest", "playwright":
		return "test"
	case "python", "python3":
		if has("pytest") {
			return "test"
		}
		return ""
	case "tsc":
		return "build"
	case "eslint", "flake8", "pylint", "ruff", "golangci-lint":
		return "lint"
	case "make", "just":
		for _, a := range rest {
			if strings.Contains(a, "test") {
				return "test"
			}
		}
		for _, a := range rest {
			if a == "lint" {
				return "lint"
			}
		}
		for _, a := range rest {
			if a == "build" || a == "all" || a == "dist" {
				return "build"
			}
		}
		return ""
	case "dotnet":
		if has("test") {
			return "test"
		}
		if has("build") {
			return "build"
		}
		return ""
	default:
		return ""
	}
}

// matchKind reports whether evidence kind carries runnable results.
func matchKind(kind model.EvidenceKind) bool {
	return kind == model.EvidenceCommand || kind == model.EvidenceReverify
}

// commandWord joins argv for reason text.
func commandWord(argv []string) string {
	if len(argv) == 0 {
		return "command"
	}
	return strings.Join(argv, " ")
}
