package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "receipts.toml")
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

func TestDefaultsNoFile(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("missing file should yield defaults: %v", err)
	}
	if cfg.Agent != "auto" || cfg.Strictness != "conservative" {
		t.Errorf("defaults = %+v", cfg)
	}
	if !cfg.Checks.TestFilesChanged || !cfg.Checks.SecretPatterns {
		t.Errorf("default checks should all enable: %+v", cfg.Checks)
	}
	if cfg.LLM.Enabled {
		t.Errorf("LLM must default disabled")
	}
}

func TestValidFullConfig(t *testing.T) {
	p := writeConfig(t, `
agent = "claude"
strictness = "strict"

[test]
command = "go test ./..."
timeout_seconds = 300

[scope]
allowed_paths = ["src/**"]
forbidden_paths = [".github/**"]

[checks]
test_files_changed = false
skips_added = false

[store]
directory = ".receipts"

[llm]
enabled = true
provider = "local"
model = "test-model"
endpoint = "http://127.0.0.1:11434"
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Agent != "claude" || cfg.Strictness != "strict" {
		t.Errorf("top keys = %+v", cfg)
	}
	if cfg.Test.Command != "go test ./..." || cfg.Test.TimeoutSeconds != 300 {
		t.Errorf("test = %+v", cfg.Test)
	}
	if len(cfg.Scope.Allowed) != 1 || cfg.Scope.Allowed[0] != "src/**" {
		t.Errorf("scope = %+v", cfg.Scope)
	}
	if cfg.Checks.TestFilesChanged || cfg.Checks.SkipsAdded {
		t.Errorf("toggles not applied: %+v", cfg.Checks)
	}
	if !cfg.Checks.AssertionsRemoved {
		t.Errorf("unset toggles default true: %+v", cfg.Checks)
	}
	if !cfg.LLM.Enabled || cfg.LLM.Model != "test-model" {
		t.Errorf("llm = %+v", cfg.LLM)
	}
}

func TestUnknownKeyErrors(t *testing.T) {
	p := writeConfig(t, "bogus_key = 1\n")
	if _, err := Load(p); err == nil {
		t.Errorf("expected unknown key error")
	}
	p = writeConfig(t, "[test]\nnope = 1\n")
	if _, err := Load(p); err == nil {
		t.Errorf("expected unknown section key error")
	}
	p = writeConfig(t, "[mystery]\nx = 1\n")
	if _, err := Load(p); err == nil {
		t.Errorf("expected unknown section error")
	}
}

func TestBadValuesError(t *testing.T) {
	for _, body := range []string{
		"agent = \"hal\"\n",
		"strictness = \"lax\"\n",
		"[test]\ntimeout_seconds = 0\n",
		"[test]\ntimeout_seconds = 99999\n",
		"[test]\ntimeout_seconds = \"soon\"\n",
		"[scope]\nallowed_paths = [\"/abs/path\"]\n",
		"[scope]\nforbidden_paths = [\"../escape\"]\n",
		"[store]\nretain_output_tail_bytes = 999999\n",
		"[llm]\nenabled = true\nendpoint = \"\"\n",
		"[llm]\nenabled = true\nendpoint = \"not a url\"\n",
	} {
		if _, err := Load(writeConfig(t, body)); err == nil {
			t.Errorf("expected error for %q, got nil", body)
		}
	}
}

func TestMalformedTOML(t *testing.T) {
	if _, err := Load(writeConfig(t, "[test\ncommand = \n")); err == nil {
		t.Errorf("expected parse error")
	}
}

func TestSchemaFile(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "schemas", "receipts-config.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("schema must be JSON: %v", err)
	}
	props, ok := v["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema needs properties")
	}
	for _, k := range []string{"agent", "strictness", "test", "scope", "checks", "store", "llm"} {
		if _, ok := props[k]; !ok {
			t.Errorf("schema missing %q", k)
		}
	}
}
