package config

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// validate enforces field contracts with field specific errors.
func validate(cfg Config) error {
	switch cfg.Agent {
	case "auto", "opencode", "claude", "codex":
	default:
		return fmt.Errorf("agent: must be auto|opencode|claude|codex, got %q", cfg.Agent)
	}
	switch cfg.Strictness {
	case "conservative", "balanced", "strict":
	default:
		return fmt.Errorf("strictness: must be conservative|balanced|strict, got %q", cfg.Strictness)
	}
	if cfg.Test.TimeoutSeconds < 1 || cfg.Test.TimeoutSeconds > 7200 {
		return fmt.Errorf("test.timeout_seconds: must be 1 to 7200, got %d", cfg.Test.TimeoutSeconds)
	}
	for _, p := range append(append([]string{}, cfg.Scope.Allowed...), cfg.Scope.Forbidden...) {
		if err := checkGlob(p); err != nil {
			return err
		}
	}
	if cfg.Store.Directory == "" {
		return fmt.Errorf("store.directory: must not be empty")
	}
	if cfg.Store.RetainOutputTailBytes < 0 || cfg.Store.RetainOutputTailBytes > 8192 {
		return fmt.Errorf("store.retain_output_tail_bytes: must be 0 to 8192, got %d", cfg.Store.RetainOutputTailBytes)
	}
	if cfg.LLM.Enabled {
		if cfg.LLM.Endpoint == "" {
			return fmt.Errorf("llm.endpoint: required when llm.enabled is true")
		}
		u, err := url.Parse(cfg.LLM.Endpoint)
		if err != nil || u.Host == "" {
			return fmt.Errorf("llm.endpoint: must be a URL, got %q", cfg.LLM.Endpoint)
		}
		if cfg.LLM.Provider == "" {
			return fmt.Errorf("llm.provider: must not be empty when llm.enabled is true")
		}
	}
	return nil
}

// checkGlob keeps patterns repository relative with slash separators.
func checkGlob(p string) error {
	if p == "" {
		return fmt.Errorf("scope: glob must not be empty")
	}
	if strings.Contains(p, "\\") {
		return fmt.Errorf("scope: glob %q must use / separators", p)
	}
	if filepath.IsAbs(p) {
		return fmt.Errorf("scope: glob %q must be repository relative", p)
	}
	if p == ".." || strings.HasPrefix(p, "../") || strings.Contains(p, "/../") {
		return fmt.Errorf("scope: glob %q must not escape the repository", p)
	}
	return nil
}
