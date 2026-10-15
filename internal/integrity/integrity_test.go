package integrity

import (
	"strings"
	"testing"

	"github.com/receipts/receipts/internal/model"
)

func lowInput() Input {
	return Input{BaselineConfidence: model.BaselineHigh, Checks: DefaultChecks()}
}

func flagCodes(flags []model.Flag) map[string]int {
	m := map[string]int{}
	for _, f := range flags {
		m[f.Code]++
	}
	return m
}

func TestTestFileModifiedAndDeleted(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: "src/auth.test.ts", Before: "expect(a).toBe(1)\n", After: "expect(a).toBe(1)\nexpect(b).toBe(2)\n", HasBefore: true, HasAfter: true},
		{Path: "src/old_test.go", Before: "x\n", After: "", HasBefore: true, HasAfter: false},
		{Path: "src/auth.ts", Before: "a\n", After: "b\n", HasBefore: true, HasAfter: true},
	}
	flags := Run(in)
	codes := flagCodes(flags)
	if codes["TEST_FILE_MODIFIED"] != 1 {
		t.Errorf("modified = %d, want 1 (auth.test.ts only): %v", codes["TEST_FILE_MODIFIED"], codes)
	}
	if codes["TEST_FILE_DELETED"] != 1 {
		t.Errorf("deleted = %d, want 1: %v", codes["TEST_FILE_DELETED"], codes)
	}
	for _, f := range flags {
		if f.Code == "TEST_FILE_DELETED" && f.Severity != model.SeverityHigh {
			t.Errorf("deleted severity = %q, want high", f.Severity)
		}
	}
}

func TestAddedTestFileNotFlagged(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: "tests/new_test.py", Before: "", After: "assert x\n", HasBefore: false, HasAfter: true},
	}
	for _, f := range Run(in) {
		if f.Code == "TEST_FILE_MODIFIED" || f.Code == "TEST_FILE_DELETED" {
			t.Errorf("added test file should not flag, got %s", f.Code)
		}
	}
}

func TestTestFixtureWarnsNotHigh(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: "tests/fixtures/data.json", Before: "{}\n", After: "{\"a\":1}\n", HasBefore: true, HasAfter: true},
	}
	flags := Run(in)
	if len(flags) == 0 || flags[0].Code != "TEST_FILE_MODIFIED" {
		t.Fatalf("fixture change should warn, got %+v", flags)
	}
	if flags[0].Severity != model.SeverityWarning {
		t.Errorf("fixture severity = %q, want warning", flags[0].Severity)
	}
}

func TestAssertionReduction(t *testing.T) {
	in := lowInput()
	before := "expect(a).toBe(1)\nexpect(b).toBe(2)\nassert x\nt.Errorf(\"bad\")\n"
	after := "expect(a).toBe(1)\n"
	in.Files = []FileContent{
		{Path: "src/a.test.js", Before: before, After: after, HasBefore: true, HasAfter: true},
	}
	flags := Run(in)
	found := false
	for _, f := range flags {
		if f.Code == "ASSERTION_TOKEN_COUNT_REDUCED" {
			found = true
			if !strings.Contains(f.Details, "4 -> 1") {
				t.Errorf("details should show 4 -> 1, got %q", f.Details)
			}
			if strings.Contains(f.Details, "3 assertions were removed") {
				t.Errorf("must not claim assertions removed, got %q", f.Details)
			}
		}
	}
	if !found {
		t.Errorf("expected ASSERTION_TOKEN_COUNT_REDUCED, got %+v", flags)
	}
}

func TestAssertionRenameNoFlag(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: "src/a.test.js", Before: "expect(a).toBe(1)\n", After: "assert(a)\n", HasBefore: true, HasAfter: true},
	}
	for _, f := range Run(in) {
		if f.Code == "ASSERTION_TOKEN_COUNT_REDUCED" {
			t.Errorf("renamed assertion with same count should not flag")
		}
	}
}

func TestAssertionCommentsAndStringsIgnored(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: "src/a.test.js", Before: "expect(a)\n// expect(b)\n", After: "expect(a)\n// expect(b)\nvar s = \"expect(c)\"\n", HasBefore: true, HasAfter: true},
	}
	for _, f := range Run(in) {
		if f.Code == "ASSERTION_TOKEN_COUNT_REDUCED" {
			t.Errorf("comment/string tokens must be ignored, got flag")
		}
	}
}

func TestSkipAddedFromAddedLines(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: "src/a.test.js", Before: "it('x',()=>{})\n", After: "it.skip('x',()=>{})\n", HasBefore: true, HasAfter: true},
	}
	in.Added = map[string][]AddedLine{"src/a.test.js": {{No: 1, Text: "it.skip('x',()=>{})"}}}
	flags := Run(in)
	if flagCodes(flags)["SKIP_ADDED"] != 1 {
		t.Errorf("expected SKIP_ADDED, got %+v", flagCodes(flags))
	}
}

func TestSkipWordWithoutMarkerNoFlag(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: "src/a.test.js", Before: "x\n", After: "x\n// do not skip this\n", HasBefore: true, HasAfter: true},
	}
	in.Added = map[string][]AddedLine{"src/a.test.js": {{No: 2, Text: "// do not skip this"}}}
	for _, f := range Run(in) {
		if f.Code == "SKIP_ADDED" {
			t.Errorf("plain word skip must not flag")
		}
	}
}

func TestScopeAllowedAndForbidden(t *testing.T) {
	in := lowInput()
	in.Scope = Scope{Allowed: []string{"src/**"}, Forbidden: []string{".github/**"}}
	in.Files = []FileContent{
		{Path: "src/a.ts", Before: "a", After: "b", HasBefore: true, HasAfter: true},
		{Path: "docs/x.md", Before: "a", After: "b", HasBefore: true, HasAfter: true},
		{Path: ".github/workflows/ci.yml", Before: "a", After: "b", HasBefore: true, HasAfter: true},
	}
	codes := flagCodes(Run(in))
	if codes["OUT_OF_SCOPE_CHANGE"] != 1 {
		t.Errorf("out of scope = %d, want 1 (docs only): %v", codes["OUT_OF_SCOPE_CHANGE"], codes)
	}
	if codes["FORBIDDEN_PATH_CHANGED"] != 1 {
		t.Errorf("forbidden = %d, want 1: %v", codes["FORBIDDEN_PATH_CHANGED"], codes)
	}
}

func TestScopeEmptyConfigNoFlags(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: "anything/at/all.ts", Before: "a", After: "b", HasBefore: true, HasAfter: true},
	}
	for _, f := range Run(in) {
		if f.Code == "OUT_OF_SCOPE_CHANGE" || f.Code == "FORBIDDEN_PATH_CHANGED" {
			t.Errorf("empty scope config must not imply scope, got %s", f.Code)
		}
	}
}

func TestDependencyChanges(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: "package.json",
			Before:    `{"dependencies":{"express":"4.17.1","leftpad":"1.0.0"}}`,
			After:     `{"dependencies":{"express":"4.18.0","lodash":"4.17.21"}}`,
			HasBefore: true, HasAfter: true},
	}
	codes := flagCodes(Run(in))
	if codes["DEPENDENCY_ADDED"] != 1 || codes["DEPENDENCY_REMOVED"] != 1 || codes["DEPENDENCY_VERSION_CHANGED"] != 1 {
		t.Errorf("want added+removed+version flags, got %v", codes)
	}
}

func TestGoModChange(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: "go.mod",
			Before:    "module x\n\nrequire (\n\tgithub.com/a/b v1.0.0\n)\n",
			After:     "module x\n\nrequire (\n\tgithub.com/a/b v1.0.0\n\tgithub.com/c/d v2.0.0\n)\n",
			HasBefore: true, HasAfter: true},
	}
	if flagCodes(Run(in))["DEPENDENCY_ADDED"] != 1 {
		t.Errorf("go.mod addition should flag DEPENDENCY_ADDED")
	}
}

func TestLockfileChangedInfo(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: "package-lock.json", Before: "{\"a\":1}", After: "{\"a\":2}", HasBefore: true, HasAfter: true},
	}
	flags := Run(in)
	if flagCodes(flags)["LOCKFILE_CHANGED"] != 1 {
		t.Errorf("lock change should flag LOCKFILE_CHANGED, got %+v", flagCodes(flags))
	}
	if flags[0].Severity != model.SeverityInfo {
		t.Errorf("lock severity = %q, want info", flags[0].Severity)
	}
}

func TestConfigChanges(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: ".github/workflows/ci.yml", Before: "a", After: "b", HasBefore: true, HasAfter: true},
		{Path: "Dockerfile", Before: "a", After: "b", HasBefore: true, HasAfter: true},
		{Path: ".eslintrc.json", Before: "a", After: "b", HasBefore: true, HasAfter: true},
		{Path: "src/app.ts", Before: "a", After: "b", HasBefore: true, HasAfter: true},
		{Path: "go.mod", Before: "module x\n", After: "module x\nrequire y v1\n", HasBefore: true, HasAfter: true},
	}
	codes := flagCodes(Run(in))
	if codes["CI_CONFIG_CHANGED"] != 1 || codes["BUILD_CONFIG_CHANGED"] != 1 || codes["LINTER_CONFIG_CHANGED"] != 1 {
		t.Errorf("want CI+BUILD+LINTER flags, got %v", codes)
	}
}

func TestSecretTokenFlaggedAndRedacted(t *testing.T) {
	secret := "AKIAIOSFODNN7EXAMPLE"
	in := lowInput()
	in.Files = []FileContent{
		{Path: "src/app.ts", Before: "x\n", After: "key = \"" + secret + "\"\n", HasBefore: true, HasAfter: true},
	}
	in.Added = map[string][]AddedLine{"src/app.ts": {{No: 2, Text: "key = \"" + secret + "\""}}}
	flags := Run(in)
	if flagCodes(flags)["SECRET_LIKE_VALUE_ADDED"] == 0 {
		t.Fatalf("expected SECRET_LIKE_VALUE_ADDED, got %+v", flagCodes(flags))
	}
	for _, f := range flags {
		if f.Code != "SECRET_LIKE_VALUE_ADDED" {
			continue
		}
		if strings.Contains(f.Details, secret) || strings.Contains(f.Title, secret) {
			t.Errorf("raw secret must never appear in flag")
		}
		if f.Severity != model.SeverityHigh {
			t.Errorf("token prefix severity = %q, want high", f.Severity)
		}
	}
}

func TestSecretPrivateKeyHigh(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: "deploy/key.pem", Before: "", After: "-----BEGIN RSA PRIVATE KEY-----\nfakebody\n", HasBefore: false, HasAfter: true},
	}
	in.Added = map[string][]AddedLine{"deploy/key.pem": {{No: 1, Text: "-----BEGIN RSA PRIVATE KEY-----"}}}
	flags := Run(in)
	if flagCodes(flags)["SECRET_LIKE_VALUE_ADDED"] == 0 {
		t.Errorf("private key header should flag")
	}
}

func TestSecretPlaceholderNoFlag(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: "src/app.ts", Before: "x\n", After: "password = \"changeme123\"\n", HasBefore: true, HasAfter: true},
	}
	in.Added = map[string][]AddedLine{"src/app.ts": {{No: 2, Text: "password = \"changeme123\""}}}
	for _, f := range Run(in) {
		if f.Code == "SECRET_LIKE_VALUE_ADDED" {
			t.Errorf("placeholder credentials must not flag")
		}
	}
}

func TestSecretLockfileSkipped(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: "pnpm-lock.yaml", Before: "a\n", After: "a\nintegrity: sha512-ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcd==\n", HasBefore: true, HasAfter: true},
	}
	in.Added = map[string][]AddedLine{"pnpm-lock.yaml": {{No: 3, Text: "integrity: sha512-ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcd=="}}}
	for _, f := range Run(in) {
		if f.Code == "SECRET_LIKE_VALUE_ADDED" {
			t.Errorf("lockfile hashes must not flag as secrets")
		}
	}
}

func TestBaselineUncertainFlag(t *testing.T) {
	in := lowInput()
	in.BaselineConfidence = model.BaselineLow
	in.Files = []FileContent{
		{Path: "src/a.ts", Before: "a", After: "b", HasBefore: true, HasAfter: true},
	}
	if flagCodes(Run(in))["BASELINE_UNCERTAIN"] != 1 {
		t.Errorf("low confidence should emit BASELINE_UNCERTAIN")
	}
	in.BaselineConfidence = model.BaselineHigh
	for _, f := range Run(in) {
		if f.Code == "BASELINE_UNCERTAIN" {
			t.Errorf("high confidence must not emit BASELINE_UNCERTAIN")
		}
	}
}

func TestChecksToggleable(t *testing.T) {
	in := lowInput()
	in.Checks.Secrets = false
	in.Checks.TestFiles = false
	secret := "AKIAIOSFODNN7EXAMPLE"
	in.Files = []FileContent{
		{Path: "src/a.test.ts", Before: "x\n", After: "key=\"" + secret + "\"\n", HasBefore: true, HasAfter: true},
	}
	in.Added = map[string][]AddedLine{"src/a.test.ts": {{No: 2, Text: "key=\"" + secret + "\""}}}
	for _, f := range Run(in) {
		if f.Code == "SECRET_LIKE_VALUE_ADDED" || f.Code == "TEST_FILE_MODIFIED" {
			t.Errorf("disabled check %s must not run", f.Code)
		}
	}
}

func TestFlagIDsStableAndSorted(t *testing.T) {
	in := lowInput()
	in.Files = []FileContent{
		{Path: "b.test.js", Before: "expect(x)\n", After: "x\n", HasBefore: true, HasAfter: true},
		{Path: "a.test.js", Before: "expect(x)\n", After: "x\n", HasBefore: true, HasAfter: true},
	}
	a := Run(in)
	b := Run(in)
	if len(a) == 0 {
		t.Fatalf("expected flags")
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			t.Fatalf("flag IDs not deterministic")
		}
		want := "F-" + pad3(i+1)
		if a[i].ID != want {
			t.Errorf("flag %d ID = %q, want %q", i, a[i].ID, want)
		}
	}
}

func pad3(n int) string {
	if n < 10 {
		return "00" + string(rune('0'+n))
	}
	if n < 100 {
		return "0" + string(rune('0'+n/10)) + string(rune('0'+n%10))
	}
	return "999"
}
