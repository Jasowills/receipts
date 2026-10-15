package integrity

import (
	"path"
	"strings"

	"github.com/receipts/receipts/internal/gitstate"
	"github.com/receipts/receipts/internal/model"
)

// matchGlob reports whether repository relative path p matches glob
// pattern. Supported syntax: * (any run inside one segment), **
// (any run across segments, including zero), ? (one non slash rune).
// Patterns use slash separators on all platforms.
func matchGlob(pattern, p string) bool {
	pattern = strings.TrimPrefix(pattern, "/")
	if !strings.Contains(pattern, "**") {
		ok, err := path.Match(pattern, p)
		return err == nil && ok
	}
	return matchSegments(strings.Split(pattern, "/"), strings.Split(p, "/"))
}

func matchSegments(pat, parts []string) bool {
	if len(pat) == 0 {
		return len(parts) == 0
	}
	if pat[0] == "**" {
		for i := 0; i <= len(parts); i++ {
			if matchSegments(pat[1:], parts[i:]) {
				return true
			}
		}
		return false
	}
	if len(parts) == 0 {
		return false
	}
	ok, err := path.Match(pat[0], parts[0])
	if err != nil || !ok {
		return false
	}
	return matchSegments(pat[1:], parts[1:])
}

// checkScope flags paths outside allowed globs and paths under forbidden
// globs. Forbidden always wins. With no configured scope it raises
// nothing so output never implies a user scope exists.
func checkScope(files []FileContent, scope Scope) []model.Flag {
	if len(scope.Allowed) == 0 && len(scope.Forbidden) == 0 {
		return nil
	}
	var flags []model.Flag
	for _, f := range files {
		norm, err := gitstate.NormalizePath(".", f.Path)
		checkPath := f.Path
		if err != nil {
			flags = append(flags, newFlag("OUT_OF_SCOPE_CHANGE", model.SeverityWarning,
				"Unparseable changed path", "path rejected during normalization: "+f.Path, []string{f.Path}))
			continue
		}
		checkPath = norm
		forbidden := false
		for _, g := range scope.Forbidden {
			if matchGlob(g, checkPath) {
				forbidden = true
				break
			}
		}
		if forbidden {
			flags = append(flags, newFlag("FORBIDDEN_PATH_CHANGED", model.SeverityHigh,
				"Forbidden path changed", "changed path matches forbidden scope: "+checkPath, []string{checkPath}))
			continue
		}
		if len(scope.Allowed) > 0 {
			allowed := false
			for _, g := range scope.Allowed {
				if matchGlob(g, checkPath) {
					allowed = true
					break
				}
			}
			if !allowed {
				flags = append(flags, newFlag("OUT_OF_SCOPE_CHANGE", model.SeverityWarning,
					"Out of scope change", "changed path outside allowed scope: "+checkPath, []string{checkPath}))
			}
		}
	}
	return flags
}
