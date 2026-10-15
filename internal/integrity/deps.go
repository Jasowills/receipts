package integrity

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/receipts/receipts/internal/model"
)

// depKind classifies a changed path for dependency analysis.
type depKind int

const (
	notDep depKind = iota
	depManifestJSON
	depGoMod
	depRequirements
	depCargo
	depPyproject
	depLock
)

func classifyDep(p string) depKind {
	lower := strings.ToLower(p)
	base := lower
	if i := strings.LastIndexByte(lower, '/'); i >= 0 {
		base = lower[i+1:]
	}
	switch base {
	case "package.json":
		return depManifestJSON
	case "go.mod":
		return depGoMod
	case "cargo.toml":
		return depCargo
	case "pyproject.toml":
		return depPyproject
	}
	if strings.HasPrefix(base, "requirements") && strings.HasSuffix(base, ".txt") {
		return depRequirements
	}
	switch base {
	case "package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "pnpm-lock.yaml",
		"go.sum", "cargo.lock", "pipfile", "pipfile.lock", "poetry.lock",
		"gemfile", "gemfile.lock", "composer.json", "composer.lock", "directory.packages.props":
		return depLock
	}
	if strings.HasSuffix(base, ".csproj") {
		return depLock
	}
	return notDep
}

// ManifestDepMaps returns before and after dependency maps for known
// manifests and lockfiles. It returns nil, nil when the path is not a
// dependency file. Lockfiles yield empty maps while still counting as
// manifests for change joining.
func ManifestDepMaps(path, before, after string, hasBefore, hasAfter bool) (map[string]string, map[string]string) {
	var b, a map[string]string
	switch classifyDep(path) {
	case depManifestJSON:
		if hasBefore {
			b = parsePackageJSON(before)
		}
		if hasAfter {
			a = parsePackageJSON(after)
		}
	case depGoMod:
		if hasBefore {
			b = parseGoMod(before)
		}
		if hasAfter {
			a = parseGoMod(after)
		}
	case depRequirements:
		if hasBefore {
			b = parseRequirements(before)
		}
		if hasAfter {
			a = parseRequirements(after)
		}
	case depCargo:
		if hasBefore {
			b, _ = parseCargo(before)
		}
		if hasAfter {
			a, _ = parseCargo(after)
		}
	case depPyproject:
		if hasBefore {
			b, _ = parsePyproject(before)
		}
		if hasAfter {
			a, _ = parsePyproject(after)
		}
	case depLock:
		b = map[string]string{}
		a = map[string]string{}
	default:
		return nil, nil
	}
	if b == nil {
		b = map[string]string{}
	}
	if a == nil {
		a = map[string]string{}
	}
	return b, a
}
func checkDependencies(files []FileContent) []model.Flag {
	var flags []model.Flag
	for _, f := range files {
		if f.Binary || !f.HasBefore || !f.HasAfter {
			if !(!f.HasBefore && f.HasAfter && classifyDep(f.Path) == depLock) {
				continue
			}
			// Added lockfile still counts as a lockfile change below.
		}
		switch classifyDep(f.Path) {
		case depManifestJSON:
			flags = append(flags, diffDepMaps(f.Path, parsePackageJSON(f.Before), parsePackageJSON(f.After))...)
		case depGoMod:
			flags = append(flags, diffDepMaps(f.Path, parseGoMod(f.Before), parseGoMod(f.After))...)
		case depRequirements:
			flags = append(flags, diffDepMaps(f.Path, parseRequirements(f.Before), parseRequirements(f.After))...)
		case depCargo:
			before, bok := parseCargo(f.Before)
			after, aok := parseCargo(f.After)
			if !bok && !aok {
				flags = append(flags, newFlag("LOCKFILE_CHANGED", model.SeverityInfo,
					"Manifest changed", f.Path+" changed (no parseable dependency delta)", []string{f.Path}))
				continue
			}
			flags = append(flags, diffDepMaps(f.Path, before, after)...)
		case depPyproject:
			before, bok := parsePyproject(f.Before)
			after, aok := parsePyproject(f.After)
			if !bok && !aok {
				flags = append(flags, newFlag("LOCKFILE_CHANGED", model.SeverityInfo,
					"Manifest changed", f.Path+" changed (no parseable dependency delta)", []string{f.Path}))
				continue
			}
			flags = append(flags, diffDepMaps(f.Path, before, after)...)
		case depLock:
			if f.Before != f.After {
				flags = append(flags, newFlag("LOCKFILE_CHANGED", model.SeverityInfo,
					"Lockfile changed", "lockfile or unparsed manifest changed: "+f.Path, []string{f.Path}))
			}
		}
	}
	return flags
}

func diffDepMaps(path string, before, after map[string]string) []model.Flag {
	var flags []model.Flag
	var added, removed, changed []string
	for k, av := range after {
		bv, ok := before[k]
		if !ok {
			added = append(added, k)
		} else if bv != av {
			changed = append(changed, k+" "+bv+" -> "+av)
		}
	}
	for k := range before {
		if _, ok := after[k]; !ok {
			removed = append(removed, k)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(changed)
	if len(added) > 0 {
		flags = append(flags, newFlag("DEPENDENCY_ADDED", model.SeverityInfo,
			"Dependency added", path+": added "+strings.Join(added, ", "), []string{path}))
	}
	if len(removed) > 0 {
		flags = append(flags, newFlag("DEPENDENCY_REMOVED", model.SeverityWarning,
			"Dependency removed", path+": removed "+strings.Join(removed, ", "), []string{path}))
	}
	if len(changed) > 0 {
		flags = append(flags, newFlag("DEPENDENCY_VERSION_CHANGED", model.SeverityInfo,
			"Dependency version changed", path+": "+strings.Join(changed, ", "), []string{path}))
	}
	return flags
}

func parsePackageJSON(s string) map[string]string {
	var v struct {
		Deps     map[string]string `json:"dependencies"`
		DevDeps  map[string]string `json:"devDependencies"`
		PeerDeps map[string]string `json:"peerDependencies"`
		OptDeps  map[string]string `json:"optionalDependencies"`
	}
	out := map[string]string{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return out
	}
	for _, m := range []map[string]string{v.Deps, v.DevDeps, v.PeerDeps, v.OptDeps} {
		for k, ver := range m {
			out[k] = ver
		}
	}
	return out
}

func parseGoMod(s string) map[string]string {
	out := map[string]string{}
	inBlock := false
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "require (") {
			inBlock = true
			continue
		}
		if inBlock && line == ")" {
			inBlock = false
			continue
		}
		var fields []string
		if inBlock {
			fields = strings.Fields(line)
		} else if rest, ok := strings.CutPrefix(line, "require "); ok {
			fields = strings.Fields(rest)
		} else {
			continue
		}
		if len(fields) >= 2 {
			out[fields[0]] = fields[1]
		}
	}
	return out
}

func parseRequirements(s string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		split := len(line)
		for i, r := range line {
			if r == '=' || r == '>' || r == '<' || r == '!' || r == '~' || r == ' ' || r == '\t' || r == '[' || r == ';' {
				split = i
				break
			}
		}
		name := strings.ToLower(strings.TrimSpace(line[:split]))
		if name == "" {
			continue
		}
		out[name] = strings.TrimSpace(line[split:])
	}
	return out
}

func parseCargo(s string) (map[string]string, bool) {
	out := map[string]string{}
	parsed := false
	section := ""
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			section = strings.ToLower(strings.Trim(t, "[]"))
			continue
		}
		if section != "dependencies" && section != "dev-dependencies" && section != "build-dependencies" {
			continue
		}
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		eq := strings.IndexByte(t, '=')
		if eq < 0 {
			continue
		}
		name := strings.TrimSpace(t[:eq])
		if name == "" {
			continue
		}
		rest := t[eq+1:]
		ver := ""
		if i := strings.Index(rest, "version"); i >= 0 {
			rest = rest[i+len("version"):]
			if q1 := strings.IndexByte(rest, '"'); q1 >= 0 {
				if q2 := strings.IndexByte(rest[q1+1:], '"'); q2 >= 0 {
					ver = rest[q1+1 : q1+1+q2]
				}
			}
		} else if q1 := strings.IndexByte(rest, '"'); q1 >= 0 {
			if q2 := strings.IndexByte(rest[q1+1:], '"'); q2 >= 0 {
				ver = rest[q1+1 : q1+1+q2]
			}
		}
		out[name] = ver
		parsed = true
	}
	return out, parsed
}

func parsePyproject(s string) (map[string]string, bool) {
	out := map[string]string{}
	parsed := false
	inDeps := false
	section := ""
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			section = strings.ToLower(strings.Trim(t, "[]"))
			inDeps = false
			continue
		}
		if section != "project" {
			continue
		}
		if strings.HasPrefix(t, "dependencies") && strings.Contains(t, "[") {
			inDeps = true
			if strings.Contains(t, "]") {
				inDeps = false
			}
			// fall through to parse inline entries on the same line
		}
		if !inDeps {
			continue
		}
		parsed = true
		start := 0
		for {
			q1 := strings.IndexByte(t[start:], '"')
			if q1 < 0 {
				break
			}
			q1 += start
			q2 := strings.IndexByte(t[q1+1:], '"')
			if q2 < 0 {
				break
			}
			req := t[q1+1 : q1+1+q2]
			name := req
			for i, r := range req {
				if r == '=' || r == '>' || r == '<' || r == '!' || r == '~' || r == ' ' || r == '\t' || r == '[' || r == ';' {
					name = req[:i]
					break
				}
			}
			name = strings.ToLower(strings.TrimSpace(name))
			if name != "" {
				out[name] = strings.TrimSpace(req[len(name):])
			}
			start = q1 + 1 + q2 + 1
		}
		if strings.Contains(t, "]") {
			inDeps = false
		}
	}
	return out, parsed
}
