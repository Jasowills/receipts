package integrity

import (
	"path"
	"strings"

	"github.com/receipts/receipts/internal/model"
)

// isTestPath classifies paths with known test naming conventions.
// Matching is case insensitive. Fixtures and snapshots under test
// directories still match; they flag at warning severity by design.
func isTestPath(p string) bool {
	lower := strings.ToLower(p)
	base := path.Base(lower)
	if strings.Contains(lower, "/test/") || strings.Contains(lower, "/tests/") || strings.Contains(lower, "/__tests__/") ||
		strings.HasPrefix(lower, "test/") || strings.HasPrefix(lower, "tests/") || strings.HasPrefix(lower, "__tests__/") {
		return true
	}
	if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, "_test.rs") || strings.HasSuffix(base, "_test.py") {
		return true
	}
	if strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") {
		return true
	}
	if strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
		return true
	}
	return false
}

// checkTestFiles flags modified and deleted test files. Added test files
// are not flagged; additions stay visible through diff evidence.
func checkTestFiles(files []FileContent) []model.Flag {
	var flags []model.Flag
	for _, f := range files {
		if !isTestPath(f.Path) {
			continue
		}
		switch {
		case f.HasBefore && !f.HasAfter:
			flags = append(flags, newFlag("TEST_FILE_DELETED", model.SeverityHigh,
				"Test file deleted", "test file removed: "+f.Path, []string{f.Path}))
		case f.HasBefore && f.HasAfter && f.Before != f.After:
			flags = append(flags, newFlag("TEST_FILE_MODIFIED", model.SeverityWarning,
				"Test file modified", "test file changed: "+f.Path, []string{f.Path}))
		}
	}
	return flags
}
