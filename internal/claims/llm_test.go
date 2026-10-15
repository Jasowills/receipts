package claims

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/receipts/receipts/internal/adapters"
)

func llmServer(t *testing.T, body string, seen *int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seen++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestLLMDisabledByDefault(t *testing.T) {
	ex := LLMExtractor{}
	_, _, err := ex.Extract(t.Context(), []adapters.Message{assistantMsg("m-1", "All tests pass")})
	if err == nil {
		t.Fatalf("expected opt-in error, got nil")
	}
}

func TestLLMGoodProposal(t *testing.T) {
	seen := 0
	url := llmServer(t, `{"claims":[{"type":"TESTS_PASS","message_id":"m-1","quote":"All tests pass"}]}`, &seen)
	ex := LLMExtractor{Config: LLMConfig{Enabled: true, Model: "local-test", Endpoint: url}}
	claims, _, err := ex.Extract(t.Context(), []adapters.Message{assistantMsg("m-1", "All tests pass. Lint is clean.")})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(claims) != 1 || claims[0].Type != "TESTS_PASS" {
		t.Fatalf("claims = %+v", claims)
	}
	if claims[0].Source.MessageID != "m-1" || claims[0].Source.ExcerptHash == "" {
		t.Errorf("source not mapped: %+v", claims[0].Source)
	}
	if seen != 1 {
		t.Errorf("provider calls = %d, want 1", seen)
	}
}

func TestLLMDropsBadProposals(t *testing.T) {
	seen := 0
	url := llmServer(t, `{"claims":[
			{"type":"NOPE","message_id":"m-1","quote":"All tests pass"},
			{"type":"TESTS_PASS","message_id":"m-404","quote":"All tests pass"},
			{"type":"TESTS_PASS","message_id":"m-1","quote":"something else entirely"},
			{"type":"TESTS_PASS","message_id":"m-1","quote":"All tests pass","evidence":"forged"}
		]}`, &seen)
	ex := LLMExtractor{Config: LLMConfig{Enabled: true, Model: "m", Endpoint: url}}
	claims, _, err := ex.Extract(t.Context(), []adapters.Message{assistantMsg("m-1", "All tests pass")})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(claims) != 0 {
		t.Errorf("all bad proposals should drop, got %+v", claims)
	}
}

func TestLLMCacheAvoidsRecall(t *testing.T) {
	seen := 0
	url := llmServer(t, `{"claims":[]}`, &seen)
	ex := LLMExtractor{Config: LLMConfig{Enabled: true, Model: "m", Endpoint: url}}
	msgs := []adapters.Message{assistantMsg("m-1", "All tests pass")}
	if _, _, err := ex.Extract(t.Context(), msgs); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, _, err := ex.Extract(t.Context(), msgs); err != nil {
		t.Fatalf("second: %v", err)
	}
	if seen != 1 {
		t.Errorf("cache should prevent second call, calls = %d", seen)
	}
}

type stubTransport struct{ body string }

func (s stubTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
	}, nil
}

func TestLLMPrivacyWarningRemote(t *testing.T) {
	ex := LLMExtractor{
		Config:     LLMConfig{Enabled: true, Model: "m", Endpoint: "https://provider.example.com/v1"},
		HTTPClient: &http.Client{Transport: stubTransport{body: `{"claims":[]}`}},
	}
	_, warnings, err := ex.Extract(t.Context(), []adapters.Message{assistantMsg("m-1", "hi")})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "provider.example.com") {
			found = true
		}
	}
	if !found {
		t.Errorf("remote endpoint must warn, got %+v", warnings)
	}
	local := LLMExtractor{
		Config:     LLMConfig{Enabled: true, Model: "m", Endpoint: "http://127.0.0.1:11434"},
		HTTPClient: &http.Client{Transport: stubTransport{body: `{"claims":[]}`}},
	}
	_, warnings, err = local.Extract(t.Context(), []adapters.Message{assistantMsg("m-1", "hi")})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	for _, w := range warnings {
		if strings.Contains(w.Message, "127.0.0.1") {
			t.Errorf("local endpoint must not warn, got %+v", warnings)
		}
	}
}
