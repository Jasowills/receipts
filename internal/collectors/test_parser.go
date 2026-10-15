package collectors

import (
	"regexp"
	"strings"
)

// ParseTestOutput reports whether captured test output shows a passing
// run. It returns nil when the output is inconclusive or the command is
// not a recognized test runner, so callers never overstate evidence.
func ParseTestOutput(argv []string, stdoutTail, stderrTail string) *bool {
	kind := ""
	if len(argv) > 0 {
		base := argv[0]
		if i := strings.LastIndexByte(base, '/'); i >= 0 {
			base = base[i+1:]
		}
		switch base {
		case "go":
			for _, a := range argv[1:] {
				if a == "test" {
					kind = "go"
				}
			}
		case "pytest", "python", "python3":
			kind = "pytest"
		case "npm", "pnpm", "yarn", "bun", "vitest", "jest":
			kind = "npm"
		case "cargo":
			for _, a := range argv[1:] {
				if a == "test" {
					kind = "cargo"
				}
			}
		}
	}
	if kind == "" {
		return nil
	}
	text := stdoutTail + "\n" + stderrTail
	yes, no := false, false
	switch kind {
	case "go", "cargo":
		no = regexp.MustCompile(`(?m)^(FAIL|--- FAIL)`).MatchString(text)
		yes = regexp.MustCompile(`(?m)^ok\s`).MatchString(text)
	case "pytest":
		no = regexp.MustCompile(`(?m)(\d+\s+failed|FAILED)`).MatchString(text)
		yes = regexp.MustCompile(`(?m)(\d+\s+passed)`).MatchString(text)
	case "npm":
		no = regexp.MustCompile(`(?m)(\d+\s+failing|failing\s+\d+|ERR!)`).MatchString(text)
		yes = regexp.MustCompile(`(?m)(\d+\s+passing|passing\s+\d+|Tests:\s+.*passed)`).MatchString(text)
	}
	switch {
	case no:
		v := false
		return &v
	case yes:
		v := true
		return &v
	default:
		return nil
	}
}
