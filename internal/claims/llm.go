package claims

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/receipts/receipts/internal/adapters"
	"github.com/receipts/receipts/internal/model"
)

// PromptVersion versions the extraction prompt so cached results stay
// comparable across prompt changes.
const PromptVersion = "llm-extract-v1"

// LLMConfig enables optional LLM claim proposals. Everything defaults
// to off. Local endpoints are preferred; remote endpoints always warn.
type LLMConfig struct {
	Enabled  bool
	Provider string
	Model    string
	Endpoint string
}

// LLMExtractor proposes claims through an HTTP endpoint and validates
// every proposal against exact transcript messages. It cannot create
// evidence, alter command records, or set verdicts: validated proposals
// flow through the same source and dedup path with UNVERIFIED verdicts.
type LLMExtractor struct {
	Config LLMConfig
	// HTTPClient overrides the default client (tests stub it).
	HTTPClient *http.Client
	cache      map[string][]model.Claim
}

// Name returns the extractor name.
func (LLMExtractor) Name() string { return "llm" }

type llmProposal struct {
	Type      string `json:"type"`
	MessageID string `json:"message_id"`
	Quote     string `json:"quote"`
}

var validProposalTypes = map[string]model.ClaimType{
	"TESTS_PASS": model.ClaimTestsPass, "BUILD_OK": model.ClaimBuildOK,
	"LINT_CLEAN": model.ClaimLintClean, "FILES_CHANGED": model.ClaimFilesChanged,
	"NO_OTHER_CHANGES": model.ClaimNoOtherChanges, "DEPENDENCY_ADDED": model.ClaimDepAdded,
	"ISSUE_FIXED": model.ClaimIssueFixed, "OTHER": model.ClaimOther,
}

// Extract proposes claims via the configured endpoint with strict output
// validation. Disabled by default; enabling a remote endpoint attaches a
// privacy warning naming where transcript content goes.
func (e *LLMExtractor) Extract(ctx context.Context, messages []adapters.Message) ([]model.Claim, []model.ExtractionWarning, error) {
	if !e.Config.Enabled {
		return nil, nil, fmt.Errorf("claims: llm extraction disabled (explicit opt-in required)")
	}
	if e.Config.Endpoint == "" || e.Config.Model == "" {
		return nil, nil, fmt.Errorf("claims: llm extraction needs model and endpoint")
	}
	var warnings []model.ExtractionWarning
	if msg := privacyWarning(e.Config.Endpoint); msg != "" {
		warnings = append(warnings, model.ExtractionWarning{Code: "LLM_PRIVACY", Message: msg})
	}
	key := cacheKey(e.Config, messages)
	if e.cache != nil {
		if hit, ok := e.cache[key]; ok {
			return hit, warnings, nil
		}
	}
	proposals, err := e.query(ctx, messages)
	if err != nil {
		return nil, warnings, err
	}
	claims := e.validate(messages, proposals)
	if e.cache == nil {
		e.cache = map[string][]model.Claim{}
	}
	e.cache[key] = claims
	return claims, warnings, nil
}

func cacheKey(cfg LLMConfig, messages []adapters.Message) string {
	h := sha256.New()
	h.Write([]byte(cfg.Model + "\x00" + PromptVersion + "\x00" + cfg.Endpoint + "\x00"))
	for _, m := range messages {
		h.Write([]byte(m.ID + "\x00" + string(m.Role) + "\x00" + m.Text + "\x00"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// privacyWarning names remote transcript destinations. Loopback and
// localhost endpoints are local and silent.
func privacyWarning(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Sprintf("transcript would be sent to unparseable endpoint %q", endpoint)
	}
	host := strings.ToLower(u.Hostname())
	if host == "127.0.0.1" || host == "::1" || host == "localhost" {
		return ""
	}
	return fmt.Sprintf("transcript content will be sent to %s", endpoint)
}

func (e *LLMExtractor) client() *http.Client {
	if e.HTTPClient != nil {
		return e.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (e *LLMExtractor) query(ctx context.Context, messages []adapters.Message) ([]llmProposal, error) {
	type wireMsg struct {
		ID   string `json:"id"`
		Role string `json:"role"`
		Text string `json:"text"`
	}
	in := struct {
		Model         string    `json:"model"`
		PromptVersion string    `json:"prompt_version"`
		Messages      []wireMsg `json:"messages"`
	}{Model: e.Config.Model, PromptVersion: PromptVersion}
	for _, m := range messages {
		in.Messages = append(in.Messages, wireMsg{ID: m.ID, Role: string(m.Role), Text: m.Text})
	}
	body, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.Config.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("claims: llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("claims: llm request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("claims: llm endpoint status %d", resp.StatusCode)
	}
	var raw struct {
		Claims []json.RawMessage `json:"claims"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("claims: llm decode: %w", err)
	}
	var out []llmProposal
	for i, item := range raw.Claims {
		var keys map[string]json.RawMessage
		if err := json.Unmarshal(item, &keys); err != nil {
			continue
		}
		allowed := map[string]bool{"type": true, "message_id": true, "quote": true}
		strict := true
		for k := range keys {
			if !allowed[k] {
				strict = false
			}
		}
		if !strict {
			continue
		}
		var p llmProposal
		if err := json.Unmarshal(item, &p); err != nil {
			continue
		}
		if p.Type == "" || p.MessageID == "" || p.Quote == "" {
			continue
		}
		_ = i
		out = append(out, p)
	}
	return out, nil
}

// validate maps proposals to claims only when the type is known, the
// message exists with the assistant role, and the quote appears verbatim
// in the message text. Everything else is discarded.
func (e *LLMExtractor) validate(messages []adapters.Message, proposals []llmProposal) []model.Claim {
	byID := map[string]adapters.Message{}
	for _, m := range messages {
		byID[m.ID] = m
	}
	seen := map[string]bool{}
	var out []model.Claim
	for _, p := range proposals {
		typ, ok := validProposalTypes[p.Type]
		if !ok {
			continue
		}
		m, ok := byID[p.MessageID]
		if !ok || m.Role != adapters.RoleAssistant {
			continue
		}
		if !strings.Contains(m.Text, p.Quote) {
			continue
		}
		key := string(typ) + "\x00" + normalizeMeaning(p.Quote)
		if seen[key] {
			continue
		}
		seen[key] = true
		text := p.Quote
		if len(text) > ExcerptLimit {
			text = text[:ExcerptLimit]
		}
		claim := model.Claim{
			Type: typ, Text: text, Verdict: model.VerdictUnverified,
			EvidenceRefs: []string{},
		}
		claim.Source.MessageID = m.ID
		claim.Source.Role = string(m.Role)
		if !m.Timestamp.IsZero() {
			tcopy := m.Timestamp.UTC()
			claim.Source.Timestamp = &tcopy
		}
		claim.Source.ExcerptHash = excerptHash(text)
		out = append(out, claim)
	}
	if out == nil {
		return []model.Claim{}
	}
	for i := range out {
		out[i].ID = fmt.Sprintf("C-%03d", i+1)
	}
	return out
}
