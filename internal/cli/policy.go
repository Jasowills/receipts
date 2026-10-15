package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/receipts/receipts/internal/adapters"
	"github.com/receipts/receipts/internal/claims"
	"github.com/receipts/receipts/internal/config"
	"github.com/receipts/receipts/internal/integrity"
	"github.com/receipts/receipts/internal/model"
)

// loadPolicy reads receipts.toml from workdir. Missing files yield
// defaults. Invalid files fail with the field specific error.
func loadPolicy(workdir string) (config.Config, error) {
	return config.Load(config.ConfigPath(workdir))
}

// policyNotes renders visible policy notes for non default configuration.
// Config cannot change verdict meanings; it only scopes checks.
func policyNotes(cfg config.Config) []string {
	var notes []string
	if len(cfg.Scope.Allowed) > 0 || len(cfg.Scope.Forbidden) > 0 {
		notes = append(notes, fmt.Sprintf("policy: scope allowed=%s forbidden=%s",
			strings.Join(cfg.Scope.Allowed, ","), strings.Join(cfg.Scope.Forbidden, ",")))
	}
	if disabled := cfg.DisabledCheckNames(); len(disabled) > 0 {
		notes = append(notes, "policy: disabled checks: "+strings.Join(disabled, ", "))
	}
	if cfg.Strictness != "conservative" {
		notes = append(notes, "policy: strictness "+cfg.Strictness)
	}
	return notes
}

// policyScope maps config scope onto integrity scope.
func policyScope(cfg config.Config) integrity.Scope {
	return integrity.Scope{Allowed: cfg.Scope.Allowed, Forbidden: cfg.Scope.Forbidden}
}

// policyChecks maps config toggles onto integrity checks.
func policyChecks(cfg config.Config) integrity.Checks {
	tf, as, sk, sc, dp, bc, sec := cfg.ToIntegrityChecks()
	return integrity.Checks{
		TestFiles: tf, Assertions: as, Skips: sk, OutOfScope: sc,
		Dependencies: dp, BuildConfig: bc, Secrets: sec,
	}
}

// extractWithPolicy runs deterministic extraction plus optional LLM
// proposals merged in. LLM failures degrade to a warning while the
// deterministic claims stand.
func extractWithPolicy(ctx context.Context, msgs []adapters.Message, cfg config.Config) ([]model.Claim, []model.ExtractionWarning) {
	ex := claims.DeterministicExtractor{}
	got, warnings, err := ex.Extract(ctx, msgs)
	if err != nil {
		return nil, []model.ExtractionWarning{{Code: "EXTRACTION", Message: err.Error()}}
	}
	if !cfg.LLM.Enabled {
		return got, warnings
	}
	llm := claims.LLMExtractor{Config: claims.LLMConfig{
		Enabled: true, Provider: cfg.LLM.Provider, Model: cfg.LLM.Model, Endpoint: cfg.LLM.Endpoint,
	}}
	proposed, lwarnings, lerr := llm.Extract(ctx, msgs)
	warnings = append(warnings, lwarnings...)
	if lerr != nil {
		warnings = append(warnings, model.ExtractionWarning{Code: "LLM_EXTRACTION", Message: lerr.Error()})
		return got, warnings
	}
	merged, mwarnings := claims.MergeAdditional(got, proposed)
	return merged, append(warnings, mwarnings...)
}
