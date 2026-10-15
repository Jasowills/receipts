package demo

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/receipts/receipts/internal/adapters"
	"github.com/receipts/receipts/internal/claims"
	"github.com/receipts/receipts/internal/collectors"
	"github.com/receipts/receipts/internal/gitstate"
	"github.com/receipts/receipts/internal/integrity"
	"github.com/receipts/receipts/internal/model"
	"github.com/receipts/receipts/internal/store"
	"github.com/receipts/receipts/internal/verdict"
)

// ValidScenarios lists demo scenarios.
func ValidScenarios() []string {
	return []string{"default", "duplicate-claims", "test-weakening", "scope-drift"}
}

// Run builds a deterministic synthetic receipt in a temp Git repository.
// Nothing outside the temp directory is touched. The caller runs cleanup
// unless keep leaves the directory for inspection.
func Run(scenario string, keep bool) (model.Receipt, string, func(), error) {
	noop := func() {}
	if !validScenario(scenario) {
		return model.Receipt{}, "", noop, fmt.Errorf("demo: unknown scenario %q (expected default|duplicate-claims|test-weakening|scope-drift)", scenario)
	}
	ctx := context.Background()
	dir, cleanup, err := fakeRepo(keep)
	if err != nil {
		return model.Receipt{}, "", noop, err
	}
	fail := func(err error) (model.Receipt, string, func(), error) {
		cleanup()
		return model.Receipt{}, "", noop, err
	}
	pre, err := gitstate.Collect(ctx, dir)
	if err != nil {
		return fail(err)
	}
	if err := mutate(scenario, dir); err != nil {
		return fail(err)
	}
	post, err := gitstate.Collect(ctx, dir)
	if err != nil {
		return fail(err)
	}
	base := gitstate.ResolveBaseline(post, &pre, "")
	repo := gitstate.AsRepo(dir, post, base)
	changes, err := gitstate.Changed(ctx, dir)
	if err != nil {
		return fail(err)
	}
	changedPaths := make([]string, 0, len(changes))
	for _, c := range changes {
		changedPaths = append(changedPaths, c.Path)
	}
	files, added, warnings := collectors.ChangedFiles(ctx, dir, post.Head, changes)
	flags := integrity.Run(integrity.Input{
		Files: files, Added: added, Checks: integrity.DefaultChecks(), BaselineConfidence: base.Confidence,
	})

	rec := model.Receipt{
		SchemaVersion: model.SchemaVersion,
		CreatedAt:     demoTime,
		Agent:         model.Agent{Name: "demo"},
		Tier:          model.Tier0,
		DurationMS:    1000,
		Repo:          repo,
		Claims:        []model.Claim{},
		Flags:         flags,
		Evidence: []model.Evidence{
			{ID: "E-001", Kind: model.EvidenceGitSnapshot, Timestamp: demoTime, Data: map[string]any{
				"root": dir, "head_before": base.HeadBefore, "head_after": base.HeadAfter,
				"baseline_confidence": string(base.Confidence),
				"dirty_before":        string(base.DirtyBefore), "dirty_after": base.DirtyAfter,
			}},
		},
		Warnings: append([]string{"demo receipt: synthetic evidence for illustration only"}, base.Warnings...),
		Summary:  model.Summary{},
		Chain:    model.Chain{Algorithm: model.ChainAlgorithmSHA256},
	}
	rec.Warnings = append(rec.Warnings, warnings...)
	diffData := map[string]any{"changed_paths": changedPaths, "diff_hash": gitstate.DiffHash(changes)}
	if depAdded, manifests := collectors.DepDeltas(files); len(manifests) > 0 {
		diffData["manifests_changed"] = manifests
		if len(depAdded) > 0 {
			diffData["added_dependencies"] = depAdded
		}
	}
	rec.Evidence = append(rec.Evidence, model.Evidence{ID: "E-002", Kind: model.EvidenceDiff, Timestamp: demoTime, Data: diffData})
	for i := range rec.Flags {
		rec.Flags[i].EvidenceRefs = append(rec.Flags[i].EvidenceRefs, "E-002")
	}
	for _, cmd := range scenarioCommands(scenario) {
		rec.Evidence = append(rec.Evidence, model.Evidence{
			ID: fmt.Sprintf("E-%03d", len(rec.Evidence)+1), Kind: model.EvidenceCommand, Timestamp: demoTime,
			Data: map[string]any{"argv": cmd.argv, "exit_code": float64(cmd.exit)},
		})
	}
	var msgs []adapters.Message
	for _, m := range messages(scenario) {
		msgs = append(msgs, adapters.Message{ID: m.ID, Role: adapters.RoleAssistant, Timestamp: demoTime, Text: m.Text})
	}
	ex := claims.DeterministicExtractor{}
	got, ewarns, err := ex.Extract(ctx, msgs)
	if err != nil {
		return fail(err)
	}
	rec.Claims = got
	for _, w := range ewarns {
		if w.Code == model.ExtractionConflictAgents {
			rec.Flags = append(rec.Flags, model.Flag{Code: w.Code, Severity: model.SeverityWarning, Title: "Conflicting agent claims", Details: w.Message, EvidenceRefs: []string{}})
		} else {
			rec.Warnings = append(rec.Warnings, w.Code+": "+w.Message)
		}
	}
	eng := verdict.DeterministicEngine{}
	if _, _, err := eng.Evaluate(&rec); err != nil {
		return fail(err)
	}
	integrity.FillFlagIDs(rec.Flags)
	src := store.FixedSource([]uint16{0xDE07})
	id, err := store.GenerateID(func(string) bool { return false }, src.Next)
	if err != nil {
		return fail(err)
	}
	rec.ID = id
	if err := verdict.CheckInvariants(&rec); err != nil {
		return fail(err)
	}
	if err := model.ValidateReceipt(rec); err != nil {
		return fail(err)
	}
	st := store.New(filepath.Join(dir, ".receipts"))
	if err := st.Save(ctx, rec); err != nil {
		return fail(err)
	}
	stored, err := st.Get(ctx, id)
	if err != nil {
		return fail(err)
	}
	// Keep the temp store out of the diff evidence: it was created after.
	return stored, dir, cleanup, nil
}

type demoCommand struct {
	argv []string
	exit int
}

// scenarioCommands returns synthetic command evidence. The fake agent
// never executes project commands; these stand in for recorded runs so
// the demo shows verified and contradicted verdicts alongside
// unverified ones.
func scenarioCommands(scenario string) []demoCommand {
	switch scenario {
	case "duplicate-claims", "test-weakening":
		return []demoCommand{{argv: []string{"npm", "test"}, exit: 0}}
	case "scope-drift":
		return nil
	default:
		return []demoCommand{
			{argv: []string{"npm", "test"}, exit: 0},
			{argv: []string{"npm", "run", "build"}, exit: 1},
		}
	}
}
