package integrity

import (
	"path"
	"strings"

	"github.com/receipts/receipts/internal/model"
)

// configBuckets maps well known CI, build, and lint configuration paths.
// Dependency manifests are owned by the dependency check and excluded here
// to avoid double counting.
func configBucket(p string) string {
	lower := strings.ToLower(p)
	base := path.Base(lower)
	if classifyDep(p) != notDep {
		return ""
	}
	if strings.HasPrefix(lower, ".github/workflows/") || base == ".gitlab-ci.yml" || base == "jenkinsfile" {
		return "CI_CONFIG_CHANGED"
	}
	if strings.HasPrefix(base, "dockerfile") || strings.HasPrefix(base, "docker-compose") ||
		base == "makefile" || base == "gnumakefile" || base == "justfile" || base == "just-file" {
		return "BUILD_CONFIG_CHANGED"
	}
	for _, name := range []string{
		".eslintrc", "eslint.config", ".prettierrc", "prettier.config",
		"biome.json", ".flake8", "setup.cfg", "tox.ini", ".stylelintrc",
		".pylintrc", "tsconfig", "jsconfig", "webpack.config", "rollup.config",
		"vite.config", "jest.config", "vitest.config", "playwright.config",
		"babel.config", ".babelrc",
	} {
		if base == name || strings.HasPrefix(base, name+".") || strings.HasPrefix(base, name+"-") {
			if strings.HasPrefix(name, "tsconfig") || strings.HasPrefix(name, "jsconfig") ||
				strings.HasSuffix(name, "webpack.config") || strings.HasSuffix(name, "rollup.config") ||
				strings.HasSuffix(name, "vite.config") || strings.HasSuffix(name, "jest.config") ||
				strings.HasSuffix(name, "vitest.config") || strings.HasSuffix(name, "playwright.config") ||
				strings.HasSuffix(name, "babel.config") {
				return "BUILD_CONFIG_CHANGED"
			}
			return "LINTER_CONFIG_CHANGED"
		}
	}
	return ""
}

// checkBuildConfig flags CI, build, and linter configuration changes.
func checkBuildConfig(files []FileContent) []model.Flag {
	var flags []model.Flag
	for _, f := range files {
		if !f.HasBefore && !f.HasAfter {
			continue
		}
		if f.HasBefore && f.HasAfter && f.Before == f.After {
			continue
		}
		code := configBucket(f.Path)
		if code == "" {
			continue
		}
		var title string
		switch code {
		case "CI_CONFIG_CHANGED":
			title = "CI configuration changed"
		case "BUILD_CONFIG_CHANGED":
			title = "Build configuration changed"
		default:
			title = "Linter configuration changed"
		}
		flags = append(flags, newFlag(code, model.SeverityWarning, title, "config changed: "+f.Path, []string{f.Path}))
	}
	return flags
}
