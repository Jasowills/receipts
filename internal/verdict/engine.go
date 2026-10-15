package verdict

import (
	"fmt"
	"sort"
	"strings"

	"github.com/receipts/receipts/internal/model"
)

// Engine evaluates claims against stored evidence. It is deterministic
// and pure: it attaches verdicts, evidence references, and reasons but
// never creates evidence.
type Engine interface {
	Evaluate(receipt *model.Receipt) ([]model.Claim, []model.VerdictWarning, error)
}

// DeterministicEngine implements the Section 9 verdict table. Absence
// of a command record is never proof a command never ran: missing
// evidence yields UNVERIFIED, never CONTRADICTED.
type DeterministicEngine struct{}

// Evaluate rewrites claim verdicts in place on the receipt and returns
// the updated claims. Re-evaluation is idempotent.
func (DeterministicEngine) Evaluate(receipt *model.Receipt) ([]model.Claim, []model.VerdictWarning, error) {
	if receipt == nil {
		return nil, nil, fmt.Errorf("verdict: nil receipt")
	}
	for i := range receipt.Claims {
		receipt.Claims[i].Verdict, receipt.Claims[i].EvidenceRefs, receipt.Claims[i].Reason =
			decide(receipt, &receipt.Claims[i])
	}
	receipt.Summary = Summarize(receipt.Claims, receipt.Flags)
	return receipt.Claims, nil, nil
}

func decide(r *model.Receipt, c *model.Claim) (model.Verdict, []string, string) {
	switch c.Type {
	case model.ClaimTestsPass:
		return decideRun(r, c, "test")
	case model.ClaimBuildOK:
		return decideRun(r, c, "build")
	case model.ClaimLintClean:
		return decideRun(r, c, "lint")
	case model.ClaimFilesChanged:
		return decideFilesChanged(r, c)
	case model.ClaimNoOtherChanges:
		return decideNoOtherChanges(r, c)
	case model.ClaimDepAdded:
		return decideDepAdded(r, c)
	case model.ClaimIssueFixed, model.ClaimOther:
		return model.VerdictUnverifiable, []string{}, unverifiableReason(c.Type)
	default:
		return model.VerdictUnverifiable, []string{}, "no objective rule for this claim type"
	}
}

func unverifiableReason(t model.ClaimType) string {
	if t == model.ClaimIssueFixed {
		return "passing tests alone do not prove the issue is fixed; no configured invariant maps to this issue"
	}
	return "no supported objective check exists for this claim"
}

// decideRun joins test, build, and lint claims to runnable evidence.
// Structured test reports only join test claims: a failing test suite
// says nothing about the build or lint. A failing report contradicts a
// pass claim even without a command record; a passing one verifies it.
func decideRun(r *model.Receipt, c *model.Claim, want string) (model.Verdict, []string, string) {
	if want == "test" {
		for _, e := range r.Evidence {
			if e.Kind != model.EvidenceTestResult {
				continue
			}
			passed, ok := e.Data["passed"].(bool)
			if !ok {
				continue
			}
			if !passed {
				return model.VerdictContradicted, []string{e.ID},
					fmt.Sprintf("structured test report records failures (%s)", e.ID)
			}
			return model.VerdictVerified, []string{e.ID},
				fmt.Sprintf("structured test report records success (%s)", e.ID)
		}
	}
	for _, e := range r.Evidence {
		if !matchKind(e.Kind) {
			continue
		}
		argv := argvOf(e)
		if classifyArgv(argv) != want {
			continue
		}
		exit, ok := exitOf(e)
		if !ok {
			continue
		}
		rev := e.Kind == model.EvidenceReverify
		v, how := verdictOfRun(exit, rev)
		reason := fmt.Sprintf("%s %s exit %d tier %d (%s)", commandWord(argv), how, exit, r.Tier, e.ID)
		return v, []string{e.ID}, reason
	}
	if r.Tier == 0 {
		return model.VerdictUnverified, []string{},
			"no matching command evidence in audit mode; execution cannot be reconstructed from diff and transcript alone"
	}
	return model.VerdictUnverified, []string{},
		"no observed matching command; capture has known gaps so absence proves nothing"
}

// diffs returns diff evidences in ID order for determinism.
func diffs(r *model.Receipt) []model.Evidence {
	var out []model.Evidence
	for _, e := range r.Evidence {
		if e.Kind == model.EvidenceDiff {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID < out[b].ID })
	return out
}

func reliableBaseline(r *model.Receipt) bool {
	return r.Repo.BaselineConfidence == model.BaselineHigh
}

func decideFilesChanged(r *model.Receipt, c *model.Claim) (model.Verdict, []string, string) {
	ds := diffs(r)
	if len(ds) == 0 || !reliableBaseline(r) {
		return model.VerdictUnverified, []string{},
			"baseline is not reliable enough to join claimed paths to observed paths"
	}
	claimed := claimedPaths(c.Text)
	actual := changedPaths(ds[0])
	if sameSet(claimed, actual) {
		return model.VerdictVerified, []string{ds[0].ID},
			fmt.Sprintf("observed changed paths match the stated list (%s)", ds[0].ID)
	}
	return model.VerdictContradicted, []string{ds[0].ID},
		fmt.Sprintf("observed diff contains paths outside the claimed set or misses claimed paths (%s)", ds[0].ID)
}

// fileClaimPaths unions claimed paths across file scope claims.
func fileClaimPaths(r *model.Receipt) []string {
	set := map[string]bool{}
	for i := range r.Claims {
		if r.Claims[i].Type != model.ClaimFilesChanged {
			continue
		}
		for _, p := range claimedPaths(r.Claims[i].Text) {
			set[p] = true
		}
	}
	var out []string
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func decideNoOtherChanges(r *model.Receipt, c *model.Claim) (model.Verdict, []string, string) {
	ds := diffs(r)
	if len(ds) == 0 || !reliableBaseline(r) {
		return model.VerdictUnverified, []string{},
			"baseline is not reliable enough to rule out other changes"
	}
	if deniesDependencies(c.Text) {
		for _, e := range ds {
			if len(strList(e.Data, "manifests_changed")) > 0 {
				return model.VerdictContradicted, []string{e.ID},
					fmt.Sprintf("manifest and lockfile changes contradict the no change claim (%s)", e.ID)
			}
		}
		return model.VerdictUnverified, []string{},
			"no manifest change found; installation may be transient or elsewhere"
	}
	actual := changedPaths(ds[0])
	allowed := fileClaimPaths(r)
	if len(actual) == 0 {
		return model.VerdictVerified, []string{ds[0].ID},
			fmt.Sprintf("reliable diff shows no changes outside an empty tree (%s)", ds[0].ID)
	}
	if len(allowed) > 0 && subsetOf(actual, allowed) {
		return model.VerdictVerified, []string{ds[0].ID},
			fmt.Sprintf("reliable diff has no paths outside the claimed set (%s)", ds[0].ID)
	}
	if len(allowed) == 0 && len(actual) > 0 {
		return model.VerdictUnverified, []string{},
			"no file scope claim defines the allowed set, so extras cannot be judged"
	}
	return model.VerdictContradicted, []string{ds[0].ID},
		fmt.Sprintf("reliable diff shows extra changes: %s (%s)", strings.Join(difference(actual, allowed), ", "), ds[0].ID)
}

func difference(a, b []string) []string {
	in := map[string]bool{}
	for _, s := range b {
		in[s] = true
	}
	var out []string
	for _, s := range a {
		if !in[s] {
			out = append(out, s)
		}
	}
	return out
}

func decideDepAdded(r *model.Receipt, c *model.Claim) (model.Verdict, []string, string) {
	name := claimedPackage(c.Text)
	for _, e := range diffs(r) {
		if !reliableBaseline(r) {
			continue
		}
		for _, dep := range strList(e.Data, "added_dependencies") {
			if name != "" && dep == name {
				return model.VerdictVerified, []string{e.ID},
					fmt.Sprintf("manifest diff adds dependency %s (%s)", name, e.ID)
			}
		}
	}
	if deniesDependencies(c.Text) {
		for _, e := range diffs(r) {
			if !reliableBaseline(r) {
				continue
			}
			if len(strList(e.Data, "manifests_changed")) > 0 {
				return model.VerdictContradicted, []string{e.ID},
					fmt.Sprintf("manifest and lockfile changes contradict the no change claim (%s)", e.ID)
			}
		}
	}
	if name == "" {
		return model.VerdictUnverifiable, []string{}, "no dependency name recovered from claim"
	}
	return model.VerdictUnverified, []string{},
		"no manifest change found for the named package; installation may be transient or elsewhere"
}

// Summarize counts claims by verdict and flags separately.
func Summarize(claims []model.Claim, flags []model.Flag) model.Summary {
	var s model.Summary
	for _, c := range claims {
		switch c.Verdict {
		case model.VerdictVerified:
			s.Verified++
		case model.VerdictContradicted:
			s.Contradicted++
		case model.VerdictUnverified:
			s.Unverified++
		default:
			s.Unverifiable++
		}
	}
	s.Flags = len(flags)
	return s
}

// CheckInvariants verifies summary counts, ID uniqueness, verdict enums,
// and that every evidence reference resolves to stored evidence.
func CheckInvariants(r *model.Receipt) error {
	known := map[string]bool{}
	for _, e := range r.Evidence {
		if known[e.ID] {
			return fmt.Errorf("verdict: duplicate evidence id %s", e.ID)
		}
		known[e.ID] = true
	}
	seenClaims := map[string]bool{}
	for _, c := range r.Claims {
		if seenClaims[c.ID] {
			return fmt.Errorf("verdict: duplicate claim id %s", c.ID)
		}
		seenClaims[c.ID] = true
		switch c.Verdict {
		case model.VerdictVerified, model.VerdictContradicted, model.VerdictUnverified, model.VerdictUnverifiable:
		default:
			return fmt.Errorf("verdict: claim %s has unknown verdict %q", c.ID, c.Verdict)
		}
		for _, ref := range c.EvidenceRefs {
			if !known[ref] {
				return fmt.Errorf("verdict: claim %s references unknown evidence %s", c.ID, ref)
			}
		}
	}
	seenFlags := map[string]bool{}
	for _, f := range r.Flags {
		if seenFlags[f.ID] {
			return fmt.Errorf("verdict: duplicate flag id %s", f.ID)
		}
		seenFlags[f.ID] = true
		for _, ref := range f.EvidenceRefs {
			if !known[ref] {
				return fmt.Errorf("verdict: flag %s references unknown evidence %s", f.ID, ref)
			}
		}
	}
	want := Summarize(r.Claims, r.Flags)
	if r.Summary != want {
		return fmt.Errorf("verdict: summary %+v does not match counts %+v", r.Summary, want)
	}
	return nil
}
