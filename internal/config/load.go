package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Load reads path or returns Defaults when the file is absent. Present
// files must parse under the strict subset below and validate.
func Load(path string) (Config, error) {
	cfg := Defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	seen := map[string]bool{}
	section := ""
	for i, raw := range strings.Split(string(data), "\n") {
		line := stripComment(raw)
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return cfg, fmt.Errorf("config line %d: bad section %q", i+1, raw)
			}
			section = strings.TrimSpace(line[1 : len(line)-1])
			switch section {
			case "test", "scope", "checks", "store", "llm":
			default:
				return cfg, fmt.Errorf("config: unknown section [%s]", section)
			}
			continue
		}
		key, value, ok := splitKV(line)
		if !ok {
			return cfg, fmt.Errorf("config line %d: expected key = value, got %q", i+1, raw)
		}
		full := key
		if section != "" {
			full = section + "." + key
		}
		if seen[full] {
			return cfg, fmt.Errorf("config: duplicate key %q", full)
		}
		seen[full] = true
		if err := applyValue(&cfg, section, key, value); err != nil {
			return cfg, err
		}
	}
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// stripComment removes # comments outside double quoted strings.
func stripComment(line string) string {
	inStr := false
	esc := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if esc {
			esc = false
			continue
		}
		if c == '\\' && inStr {
			esc = true
			continue
		}
		if c == '"' {
			inStr = !inStr
			continue
		}
		if c == '#' && !inStr {
			return line[:i]
		}
	}
	return line
}

func splitKV(line string) (string, string, bool) {
	idx := strings.IndexByte(line, '=')
	if idx < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	if key == "" || value == "" {
		return "", "", false
	}
	return key, value, true
}

func parseString(value string) (string, error) {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return "", fmt.Errorf("expected quoted string, got %q", value)
	}
	inner := value[1 : len(value)-1]
	var b strings.Builder
	for i := 0; i < len(inner); i++ {
		if inner[i] == '\\' && i+1 < len(inner) {
			switch inner[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '"', '\\':
				b.WriteByte(inner[i+1])
			default:
				return "", fmt.Errorf("bad escape in %q", value)
			}
			i++
			continue
		}
		b.WriteByte(inner[i])
	}
	return b.String(), nil
}

func parseStringArray(value string) ([]string, error) {
	t := strings.TrimSpace(value)
	if !strings.HasPrefix(t, "[") || !strings.HasSuffix(t, "]") {
		return nil, fmt.Errorf("expected string array, got %q", value)
	}
	inner := strings.TrimSpace(t[1 : len(t)-1])
	if inner == "" {
		return []string{}, nil
	}
	var out []string
	var cur strings.Builder
	inStr := false
	esc := false
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if esc {
			cur.WriteByte(c)
			esc = false
			continue
		}
		if c == '\\' && inStr {
			esc = true
			cur.WriteByte(c)
			continue
		}
		if c == '"' {
			inStr = !inStr
			cur.WriteByte(c)
			continue
		}
		if c == ',' && !inStr {
			s, err := parseString(strings.TrimSpace(cur.String()))
			if err != nil {
				return nil, err
			}
			out = append(out, s)
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	if inStr {
		return nil, fmt.Errorf("unterminated string in %q", value)
	}
	if last := strings.TrimSpace(cur.String()); last != "" {
		s, err := parseString(last)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func applyValue(cfg *Config, section, key, value string) error {
	str := func() (string, error) { return parseString(value) }
	boolean := func() (bool, error) {
		switch value {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return false, fmt.Errorf("expected true or false, got %q", value)
		}
	}
	num := func() (int, error) {
		n, err := strconv.Atoi(value)
		if err != nil {
			return 0, fmt.Errorf("expected integer, got %q", value)
		}
		return n, nil
	}
	setStr := func(dst *string) error {
		s, err := str()
		if err != nil {
			return fmt.Errorf("%s: %w", keyPath(section, key), err)
		}
		*dst = s
		return nil
	}
	setBool := func(dst *bool) error {
		b, err := boolean()
		if err != nil {
			return fmt.Errorf("%s: %w", keyPath(section, key), err)
		}
		*dst = b
		return nil
	}
	setInt := func(dst *int) error {
		n, err := num()
		if err != nil {
			return fmt.Errorf("%s: %w", keyPath(section, key), err)
		}
		*dst = n
		return nil
	}
	switch section {
	case "":
		switch key {
		case "agent":
			return setStr(&cfg.Agent)
		case "strictness":
			return setStr(&cfg.Strictness)
		default:
			return fmt.Errorf("config: unknown key %q", key)
		}
	case "test":
		switch key {
		case "command":
			return setStr(&cfg.Test.Command)
		case "timeout_seconds":
			return setInt(&cfg.Test.TimeoutSeconds)
		case "shell":
			return setBool(&cfg.Test.Shell)
		default:
			return fmt.Errorf("config: unknown key %q", keyPath(section, key))
		}
	case "scope":
		switch key {
		case "allowed_paths", "forbidden_paths":
			arr, err := parseStringArray(value)
			if err != nil {
				return fmt.Errorf("%s: %w", keyPath(section, key), err)
			}
			if key == "allowed_paths" {
				cfg.Scope.Allowed = arr
			} else {
				cfg.Scope.Forbidden = arr
			}
			return nil
		default:
			return fmt.Errorf("config: unknown key %q", keyPath(section, key))
		}
	case "checks":
		switch key {
		case "test_files_changed":
			return setBool(&cfg.Checks.TestFilesChanged)
		case "assertions_removed":
			return setBool(&cfg.Checks.AssertionsRemoved)
		case "skips_added":
			return setBool(&cfg.Checks.SkipsAdded)
		case "out_of_scope_changes":
			return setBool(&cfg.Checks.OutOfScopeChanges)
		case "dependencies_changed":
			return setBool(&cfg.Checks.DependenciesChanged)
		case "build_config_changed":
			return setBool(&cfg.Checks.BuildConfigChanged)
		case "secret_patterns":
			return setBool(&cfg.Checks.SecretPatterns)
		default:
			return fmt.Errorf("config: unknown key %q", keyPath(section, key))
		}
	case "store":
		switch key {
		case "directory":
			return setStr(&cfg.Store.Directory)
		case "retain_output_tail_bytes":
			return setInt(&cfg.Store.RetainOutputTailBytes)
		default:
			return fmt.Errorf("config: unknown key %q", keyPath(section, key))
		}
	case "llm":
		switch key {
		case "enabled":
			return setBool(&cfg.LLM.Enabled)
		case "provider":
			return setStr(&cfg.LLM.Provider)
		case "model":
			return setStr(&cfg.LLM.Model)
		case "endpoint":
			return setStr(&cfg.LLM.Endpoint)
		default:
			return fmt.Errorf("config: unknown key %q", keyPath(section, key))
		}
	default:
		return fmt.Errorf("config: unknown section [%s]", section)
	}
}

func keyPath(section, key string) string {
	if section == "" {
		return key
	}
	return section + "." + key
}

// ConfigPath returns the receipts.toml path for a workdir.
func ConfigPath(workdir string) string {
	return filepath.Join(workdir, "receipts.toml")
}
