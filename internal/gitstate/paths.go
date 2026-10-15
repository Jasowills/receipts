package gitstate

import (
	"errors"
	"path"
	"path/filepath"
	"strings"
)

// NormalizePath returns the repository relative slash separated form of p.
// Root is the repository top level. Relative inputs are cleaned with slash
// semantics; absolute inputs must resolve inside root. Case, spaces, and
// unicode are preserved. Backslashes are ordinary filename characters on
// unix and are never treated as separators here. Traversal outside the
// repository root is rejected so randomized inputs cannot escape.
func NormalizePath(root, p string) (string, error) {
	if p == "" {
		return "", errors.New("gitstate: empty path")
	}
	if filepath.IsAbs(p) {
		cleanRoot := filepath.Clean(root)
		cleanP := filepath.Clean(p)
		rel, err := filepath.Rel(cleanRoot, cleanP)
		if err != nil {
			return "", errors.New("gitstate: path outside repository")
		}
		if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", errors.New("gitstate: path outside repository")
		}
		return filepath.ToSlash(rel), nil
	}
	cleaned := path.Clean(p)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return "", errors.New("gitstate: path outside repository")
	}
	return cleaned, nil
}
