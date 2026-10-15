package claims

import (
	"testing"

	"github.com/receipts/receipts/internal/adapters"
	"github.com/receipts/receipts/internal/model"
)

func assistantMsg(id, text string) adapters.Message {
	return adapters.Message{ID: id, Role: adapters.RoleAssistant, Text: text}
}

func extractTypes(t *testing.T, msgs []adapters.Message) []model.Claim {
	t.Helper()
	ex := DeterministicExtractor{}
	claims, _, err := ex.Extract(t.Context(), msgs)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	return claims
}

func wantType(t *testing.T, text string, want model.ClaimType) {
	t.Helper()
	claims := extractTypes(t, []adapters.Message{assistantMsg("m-1", text)})
	if len(claims) != 1 {
		t.Fatalf("%q: got %d claims, want 1 (%s)", text, len(claims), want)
	}
	if claims[0].Type != want {
		t.Errorf("%q: type = %s, want %s", text, claims[0].Type, want)
	}
}

func wantNone(t *testing.T, text string) {
	t.Helper()
	claims := extractTypes(t, []adapters.Message{assistantMsg("m-1", text)})
	if len(claims) != 0 {
		t.Errorf("%q: got %d claims, want none (%+v)", text, len(claims), claims)
	}
}

func TestTestsPassPositives(t *testing.T) {
	for _, s := range []string{
		"All tests pass",
		"all tests are passing",
		"The test suite passes",
		"Tests passed",
		"npm test passed",
		"go test ./... passed",
		"I ran the tests and they passed",
		"Tests pass locally",
	} {
		wantType(t, s, model.ClaimTestsPass)
	}
}

func TestTestsPassNegatives(t *testing.T) {
	for _, s := range []string{
		"Tests should pass",
		"I did not run tests",
		"Tests may pass",
		"If tests pass, then deploy",
		"Tests did not pass",
		"I will run tests",
		"No tests passed",
		"> All tests pass",
		"```\nAll tests pass\n```",
		"Run `npm test` to verify",
		"Do all tests pass?",
		"Did the build succeed?",
	} {
		wantNone(t, s)
	}
}

func TestBuildPositivesAndNegatives(t *testing.T) {
	wantType(t, "The build succeeds", model.ClaimBuildOK)
	wantType(t, "production build completed successfully", model.ClaimBuildOK)
	wantType(t, "npm run build passed", model.ClaimBuildOK)
	wantNone(t, "The build might succeed")
	wantNone(t, "Build did not succeed")
}

func TestLintPositivesAndNegatives(t *testing.T) {
	wantType(t, "lint passed", model.ClaimLintClean)
	wantType(t, "Lint is clean", model.ClaimLintClean)
	wantType(t, "no lint errors", model.ClaimLintClean)
	wantType(t, "eslint passed", model.ClaimLintClean)
	wantNone(t, "lint should be clean")
}

func TestFilesChanged(t *testing.T) {
	claims := extractTypes(t, []adapters.Message{assistantMsg("m-1", "I only changed src/auth.ts and src/auth.test.ts")})
	if len(claims) != 1 || claims[0].Type != model.ClaimFilesChanged {
		t.Fatalf("got %+v", claims)
	}
	wantNone(t, "I changed some files somewhere")
	wantNone(t, "I only changed ../../../etc/passwd")
}

func TestNoOtherChanges(t *testing.T) {
	wantType(t, "No other files were changed", model.ClaimNoOtherChanges)
	wantType(t, "I made no unrelated changes", model.ClaimNoOtherChanges)
	wantType(t, "Nothing else was modified", model.ClaimNoOtherChanges)
}

func TestDependencyAdded(t *testing.T) {
	wantType(t, "Added lodash as a dependency", model.ClaimDepAdded)
	wantType(t, "Updated package.json to include lodash", model.ClaimDepAdded)
	wantNone(t, "Installed the tool globally")
}

func TestIssueFixed(t *testing.T) {
	wantType(t, "Fixed the bug", model.ClaimIssueFixed)
	wantType(t, "The issue is resolved", model.ClaimIssueFixed)
}

func TestQualifiedTestsBecomeOther(t *testing.T) {
	wantType(t, "All tests pass except integration tests", model.ClaimOther)
}

func TestOnlyAssistantEligible(t *testing.T) {
	ex := DeterministicExtractor{}
	msgs := []adapters.Message{
		{ID: "u-1", Role: adapters.RoleUser, Text: "All tests pass"},
		{ID: "t-1", Role: adapters.RoleTool, Text: "All tests pass"},
		{ID: "s-1", Role: adapters.RoleSystem, Text: "All tests pass"},
	}
	claims, _, err := ex.Extract(t.Context(), msgs)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(claims) != 0 {
		t.Errorf("only assistant messages create claims, got %+v", claims)
	}
}
