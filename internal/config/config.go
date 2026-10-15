package config

// Config is the normalized receipts.toml contract. All fields have
// defaults; only receipts.toml keys in this shape are accepted.
type Config struct {
	Agent      string
	Strictness string
	Test       TestConfig
	Scope      ScopeConfig
	Checks     ChecksConfig
	Store      StoreConfig
	LLM        LLMConfig
}

// TestConfig carries the re-verification command contract.
type TestConfig struct {
	Command        string
	TimeoutSeconds int
	Shell          bool
}

// ScopeConfig carries path policy. Empty means no user scope.
type ScopeConfig struct {
	Allowed   []string
	Forbidden []string
}

// ChecksConfig toggles optional integrity checks. True enables.
type ChecksConfig struct {
	TestFilesChanged    bool
	AssertionsRemoved   bool
	SkipsAdded          bool
	OutOfScopeChanges   bool
	DependenciesChanged bool
	BuildConfigChanged  bool
	SecretPatterns      bool
}

// StoreConfig carries store paths and bounds.
type StoreConfig struct {
	Directory             string
	RetainOutputTailBytes int
}

// LLMConfig carries optional extraction settings. Disabled by default.
type LLMConfig struct {
	Enabled  bool
	Provider string
	Model    string
	Endpoint string
}

// Defaults returns the conservative configuration used with no file.
func Defaults() Config {
	return Config{
		Agent:      "auto",
		Strictness: "conservative",
		Test:       TestConfig{TimeoutSeconds: 900},
		Checks: ChecksConfig{
			TestFilesChanged: true, AssertionsRemoved: true, SkipsAdded: true,
			OutOfScopeChanges: true, DependenciesChanged: true,
			BuildConfigChanged: true, SecretPatterns: true,
		},
		Store: StoreConfig{Directory: ".receipts", RetainOutputTailBytes: 8192},
		LLM:   LLMConfig{Provider: "local", Endpoint: "http://127.0.0.1:11434"},
	}
}

// ToIntegrityChecks maps config toggles onto integrity checks.
func (c Config) ToIntegrityChecks() (testFiles, assertions, skips, scope, deps, buildcfg, secrets bool) {
	k := c.Checks
	return k.TestFilesChanged, k.AssertionsRemoved, k.SkipsAdded, k.OutOfScopeChanges,
		k.DependenciesChanged, k.BuildConfigChanged, k.SecretPatterns
}

// DisabledCheckNames lists toggled off checks for policy notes.
func (c Config) DisabledCheckNames() []string {
	var out []string
	k := c.Checks
	if !k.TestFilesChanged {
		out = append(out, "test_files_changed")
	}
	if !k.AssertionsRemoved {
		out = append(out, "assertions_removed")
	}
	if !k.SkipsAdded {
		out = append(out, "skips_added")
	}
	if !k.OutOfScopeChanges {
		out = append(out, "out_of_scope_changes")
	}
	if !k.DependenciesChanged {
		out = append(out, "dependencies_changed")
	}
	if !k.BuildConfigChanged {
		out = append(out, "build_config_changed")
	}
	if !k.SecretPatterns {
		out = append(out, "secret_patterns")
	}
	return out
}
