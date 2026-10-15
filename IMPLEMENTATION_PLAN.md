# Receipts Implementation Plan

Status: Build complete through Task 59. Release blocked on owner decisions (name, license, repo).

## Skill workflow in use (Phase E addition)

- Seams agreed for Phase E: verdict table rows as unit cases, audit exit codes in temp repos with hermetic HOME, demo goldens with fixed IDs and timestamps, dataset gates as failing tests.

## Skill workflow in use (Phase C addition)

- Seams agreed for Phase C: gitstate public functions (FindRoot, Head, Branch, Collect, ResolveBaseline, NormalizePath, Changed, DiffHash, IsRepo, NonGitRepo) against temp Git repositories with fixed identity. No outer repo state used. Integrity checks operate on changed paths plus before/after contents with fixtures.
Spec: `RECEIPTS_OPENCODE_IMPLEMENTATION_SPEC.md` version 1.0
Date: 2026-10-14
Repo state at start: single file `RECEIPTS_OPENCODE_IMPLEMENTATION_SPEC.md`, no `.git`, no Go code

This file is the working tracker for the phased build in spec Section 17.
It records status, changed files, exact test commands and results, and unresolved questions per task.
One task is implemented at a time. A task is complete only when its acceptance checks pass.

## Skill workflow in use

Per spec Sections 17, 20 and user instruction, the installed Matt Pocock skills were invoked:

- `ask-matt` loaded for routing. Main flow confirmed: spec exists, so next is `to-tickets` style breakdown, then `implement` per ticket with `tdd` inside.
- `implement` loaded. Rules in effect: implement from spec or tickets, use `tdd` at pre-agreed seams where possible, run type checks regularly, run single test files regularly, run full suite at end, then code review, then commit.
- `to-tickets` loaded. Rule: tracer bullet vertical slices with blocking edges, published to configured tracker. Limitation: no tracker is configured in this repo (no git remote, no `docs/agents/issue-tracker.md`, `setup-matt-pocock-skills` has not been run). So this file plus spec Section 17 serves as the plan tracker for now. No tickets were published to an external tracker.
- `tdd` loaded. Rules in effect: red before green, one slice at a time, tests at pre-agreed public seams only, no implementation coupled tests.

Seams agreed for Phase A:

- CLI seam: `receipts --help`, `receipts --version`, `receipts <command> --help` via `cmd/receipts/main.go` stdout and exit code.
- Model seam: public Go types in `internal/model` plus JSON round trip.
- Schema seam: `schemas/receipt.schema.json` validates golden receipts, invalid receipts fail.

`setup-matt-pocock-skills` was not run because it is prompt driven setup for issue tracker and domain docs, and the spec explicitly requires `IMPLEMENTATION_PLAN.md` and `DECISIONS.md` first in a non git directory. This is recorded as a follow up, not a blocker for Phase A Tasks 4 to 7.

## Global constraints (from spec)

- No em dashes in user facing project documentation.
- Local first, offline first. No network calls, telemetry, hosted services, or LLM requirement in core workflows.
- A verdict must never be stronger than its evidence. Prefer `UNVERIFIED`.
- Receipts is not a security boundary against a malicious agent. Label fidelity tier on every receipt and document bypasses.
- Never put transcript contents, command output, secrets, or env vars in telemetry. There is no telemetry in v1.
- Do not silently alter test commands, git state, user files, or project configuration.
- Never execute arbitrary project commands except on `receipts run` or explicit re-verification request.
- Preserve existing files and uncommitted changes. Only spec file existed at start and it was not modified.

## Phase A: repository and contracts (spec Section 17)

### Task 1: Inspect repository, toolchain, existing changes

Status: done

Findings:

- Directory `/Users/jasonamadi/receipts` contains only `RECEIPTS_OPENCODE_IMPLEMENTATION_SPEC.md` (77 KB, 1354 lines, read fully).
- `git status` reports `fatal: not a git repository`. No git history to preserve. No uncommitted changes to overwrite. Decision: do not run `git init` automatically. See `DECISIONS.md` D-002.
- Go toolchain: `go1.27.1 darwin/amd64`, `GOROOT /usr/local/Cellar/go/1.27.1/libexec`, `GOPATH /Users/jasonamadi/go`.
- Tools: `git /usr/bin/git` present, `gofmt` present, `make /usr/bin/make` present, `golangci-lint` not found.
- OS: Darwin 25.5.0 x86_64 (MacBook Pro Intel). Linux CI still required per spec but not run locally.
- Existing work preserved: no files modified or deleted. Only new files added after this point.

Changed files: none
Tests: none (no Go code yet)
Validation: `git status`, `go version`, `go env`, `which` checks, directory listing. All recorded above.

### Task 2: Use installed Matt Pocock skill

Status: done

- Invoked via Skill tool: `ask-matt`, `implement`, `to-tickets`, `tdd`.
- Result: successfully invoked. Workflow documented in this file under Skill workflow in use.
- Limitation: no issue tracker configured, so `to-tickets` publishing step deferred. Plan tracking stays in this file per spec Section 20. No skill behavior was invented.
- Recorded in `DECISIONS.md` D-001.

Changed files: none
Tests: none

### Task 3: Create IMPLEMENTATION_PLAN.md and DECISIONS.md

Status: done

Acceptance:

- [x] This plan mirrors spec Section 17 Phase A to G
- [x] DECISIONS.md lists open verification items including transcript format assumptions
- [x] No existing files overwritten
- [x] Both files written and reviewed for em dash ban

Changed files:

- `IMPLEMENTATION_PLAN.md` (new)
- `DECISIONS.md` (new)

Tests: none (docs only). Validation is file presence plus grep for em dash character.
Validation 2026-10-14: `grep -n em dash` on both new docs reports no match. PASS.

### Task 4: Go module and minimal CLI

Status: done 2026-10-14

Scope from spec:

- Init Go module. Module path decision in DECISIONS.md D-003. Used placeholder `github.com/receipts/receipts` until owner confirmed. No public publish yet.
- Minimal `cmd/receipts/main.go` plus `internal/cli/root.go` supporting `--help` on root and every command stub, plus version output.
- Commands stub as help only for now. Full behavior comes in later phases.
- No network, no telemetry, no transcript reading yet.

Acceptance:

- [x] `go build ./...` succeeds
- [x] `go run ./cmd/receipts --help` exits 0 and lists audit, show, demo, init (plus later commands as not yet implemented or help only, documented clearly)
- [x] `go run ./cmd/receipts --version` prints version and exits 0
- [x] `gofmt -l .` reports no files
- [x] `go vet ./...` passes
- [x] `go test ./...` passes (6 CLI tests pass)

Changed files:

- `go.mod` (new, module github.com/receipts/receipts, go 1.27.1)
- `cmd/receipts/main.go` (new, thin wrapper around cli.Run)
- `internal/cli/root.go` (new, stdlib only parser, help and version, unknown command returns 2)
- `internal/cli/root_test.go` (new, 6 tests for help, version, per command help, unknown command, em dash ban)

TDD record:

- Red: wrote `internal/cli/root_test.go` first, ran `go test ./internal/cli -v` before implementation. Result: FAIL build failed, undefined Run and Version. This confirmed the test would catch missing implementation.
- Green: implemented `internal/cli/root.go` and `cmd/receipts/main.go`, reran tests. Result: PASS all 6 tests.

Exact validation commands and results 2026-10-14 (workdir /Users/jasonamadi/receipts):

- `go mod init github.com/receipts/receipts` -> created go.mod, module github.com/receipts/receipts, go 1.27.1. PASS.
- `go test ./internal/cli -v` before impl -> FAIL build failed undefined Run, Version. Expected red. PASS as red step.
- `gofmt -l .` -> empty output (no files). PASS.
- `go vet ./...` -> empty output, exit 0. PASS.
- `go test ./... -v` -> cmd/receipts no test files, internal/cli 6 PASS (TestRootHelpListsCommands, TestRootNoArgsPrintsHelp, TestVersionFlag, TestEveryCommandSupportsHelp, TestUnknownCommandFailsWithUsageError, TestHelpHasNoEmDash). PASS.
- `go build -o /tmp/receipts ./cmd/receipts` -> BUILD-OK. PASS.
- `/tmp/receipts --help` -> exit 0, lists audit, show, demo, init, run, check, ci. PASS.
- `/tmp/receipts --version` -> `receipts 0.0.1-dev`, exit 0. PASS.
- `/tmp/receipts audit --help` -> exit 0, Usage present. PASS.
- `/tmp/receipts does-not-exist` -> stderr unknown command, exit 2. PASS.
- `go run ./cmd/receipts --help` -> exit 0. PASS.
- `go run ./cmd/receipts --version` -> receipts 0.0.1-dev, exit 0. PASS.

Notes:

- Version is `0.0.1-dev` via `internal/cli.Version`, overridable with ldflags. Documented in code.
- Help output contains no em dash character, verified by TestHelpHasNoEmDash and grep.
- Bare stub `receipts audit` currently prints command help with Status not yet implemented and returns 0. This will change when audit is implemented. No false claim that audit ran.

### Task 5: CI workflow

Status: done 2026-10-14

Scope:

- Add `.github/workflows/ci.yml` for formatting, vet, tests, OS matrix (macOS latest, Ubuntu latest), Go versions from `go.mod` and README.
- Include race tests on Linux per spec Section 14.
- Pinned action versions where practical.

Acceptance:

- [x] Workflow file exists and is valid YAML
- [x] Local `gofmt`, `go vet ./...`, `go test ./...` pass before claiming done
- [x] No release artifacts made until checks pass

Changed files:

- `.github/workflows/ci.yml` (new)

Workflow content:

- Triggers on pull_request and push to main.
- Matrix os macos-latest and ubuntu-latest, Go version from `go.mod` via `go-version-file` (currently go 1.27.1). README does not exist yet (lands in Phase E Task 35), so `go.mod` is the single source for now.
- Pinned: `actions/checkout@v4`, `actions/setup-go@v5`.
- Steps: Check formatting with `test -z "$(gofmt -l .)"`, `go vet ./...`, `go test ./...`, plus `go test -race ./...` on ubuntu-latest only per spec.
- No release job, no artifacts.

Exact validation commands and results 2026-10-14:

- `python3 -c yaml.safe_load(open('.github/workflows/ci.yml'))` -> YAML-OK. PASS.
- `gofmt -l .` -> empty output. PASS.
- `go vet ./...` -> exit 0. PASS.
- `go test ./... -v` -> cmd no test files, internal/cli 6 PASS. PASS.
- `go test -race ./...` -> ok internal/cli 2.292s, no data races. PASS.

### Task 6: model enums and structs

Status: done 2026-10-14

Scope from spec Sections 4 and 5:

- `internal/model/` with receipt, claim, evidence, flag, enums. `model` must not import other internal packages.
- Types: Tier 0,1,2; verdicts VERIFIED, CONTRADICTED, UNVERIFIED, UNVERIFIABLE; evidence kinds git_snapshot, diff, command, test_result, reverify, transcript, adapter_status, collection_warning; baseline confidence high, medium, low, unknown; claim types TESTS_PASS, BUILD_OK, LINT_CLEAN, FILES_CHANGED, NO_OTHER_CHANGES, DEPENDENCY_ADDED, ISSUE_FIXED, OTHER; flag severity info, warning, high.
- Stable IDs: RC-XXXX, C-001, E-001, F-001 patterns.
- UTC RFC 3339 timestamps.
- Unit test enum values and JSON serialization.

Acceptance:

- [x] `go test ./internal/model -v` passes
- [x] Enum string values match spec
- [x] JSON round trip preserves fields
- [x] `model` imports no other internal packages (verified by `go list` or grep)

Changed files:

- `internal/model/enums.go` (new, tiers, verdicts, evidence kinds, confidence, claim types, severity, dirty state, schema version and chain algorithm constants)
- `internal/model/receipt.go` (new, Receipt, Agent, Repo, Summary, Chain)
- `internal/model/claim.go` (new, Claim, Source)
- `internal/model/evidence.go` (new, Evidence)
- `internal/model/flag.go` (new, Flag)
- `internal/model/model_test.go` (new, 3 tests)

TDD record:

- Red: wrote `internal/model/model_test.go` first, ran `go test ./internal/model -v` before implementation. Result: FAIL build failed undefined VerdictVerified, Tier0 and others. Expected red.
- Green: implemented 5 model files, ran tests. Result: PASS 3 tests.
- Formatting: `gofmt -w` applied to model files after initial green because `gofmt -l .` listed them. Reran checks after formatting.

Exact validation commands and results 2026-10-14:

- `go test ./internal/model -v` before impl -> FAIL build failed. PASS as red step.
- `gofmt -l .` after impl before fmt -> listed 5 model files. Fixed with `gofmt -w`. After fix `gofmt -l .` empty. PASS.
- `go vet ./...` -> exit 0. PASS.
- `go test ./internal/model -v` -> 3 PASS (TestEnumValuesMatchSpec, TestReceiptJSONRoundTrip, TestClaimAndEvidenceJSON). PASS.
- `go test ./...` -> cmd no test files, cli ok cached, model ok. PASS.
- `go test -race ./...` -> ok cli cached, ok model 2.188s. PASS.
- `go list -f Imports ./internal/model` -> [time] only. PASS, no internal imports.
- `grep -r internal model` -> NO-INTERNAL-IMPORTS-PASS.
- `grep em dash model` -> NO-EMDASH-PASS.

Notes:

- Repo.DirtyBefore uses string enum true/false/unknown per schema Section 5.3. Repo.DirtyAfter and NonGit use *bool with omitempty so unset stays absent, matching optional schema fields.
- Timestamps use time.Time for required fields and *time.Time for optional source timestamp. JSON uses RFC 3339 via Go time marshaling, UTC in tests.
- Required arrays (claims, flags, evidence, warnings) use non omitempty tags. Tests use non nil empty slices. Nil handling to be hardened in store canonical JSON task if needed.
- No collectors implemented. Model contract stable for Task 7 schema work.

### Task 7: receipt schema and validation test

Status: done 2026-10-14

Scope from spec Section 5.3:

- Created `schemas/receipt.schema.json` using JSON Schema Draft 2020-12, matching the minimum contract in spec.
- Schema validation test with stdlib only validator. No collectors implemented, model contract now stable.

Acceptance:

- [x] Golden valid receipt validates
- [x] Invalid receipts (bad id pattern, bad verdict enum, missing required field) fail validation
- [x] `go test ./...` passes
- [x] No collectors implemented in this task

Changed files:

- `schemas/receipt.schema.json` (new, Draft 2020-12, required top level fields, $defs for claim flag evidence, additionalProperties false, patterns and enums per spec)
- `testdata/golden/v0.1-valid.json` (new, spec Section 5.2 canonical example RC-4F2A)
- `internal/model/validate.go` (new, ValidateReceipt with stdlib regexp checks for IDs, enums, required fields, evidence ref resolution, duplicate and unique checks)
- `internal/model/schema_test.go` (new, 3 tests: schema file exists and valid JSON with draft URL, golden validates, 5 invalid cases fail)
- `DECISIONS.md` D-006 recorded (stdlib only validation, zero dependencies)

TDD record:

- Red: wrote `internal/model/schema_test.go` first, ran `go test ./internal/model -run Schema`. Result: FAIL build failed undefined ValidateReceipt. Expected red. Schema and golden files also missing at that point.
- Green: created schema JSON, golden JSON, validate.go. Reran tests. Result: PASS.

Exact validation commands and results 2026-10-14:

- `go test ./internal/model -run Schema -v` before impl -> FAIL build failed. PASS as red step.
- `gofmt -l .` -> empty. PASS.
- `go vet ./...` -> exit 0. PASS.
- `go test ./internal/model -v` -> 8 PASS total (3 model tests plus TestReceiptSchemaFileExistsAndIsValidJSON, TestGoldenReceiptValidates, TestInvalidReceiptsFailValidation with 5 subcases). PASS.
- `go test ./...` -> cmd no test files, cli ok, model ok. PASS.
- `go test -race ./...` -> ok cli, ok model 2.101s. PASS.
- `python3 json.load schemas/receipt.schema.json` -> SCHEMA-JSON-OK. PASS.
- `python3 json.load testdata/golden/v0.1-valid.json` -> GOLDEN-JSON-OK, required top level list matches spec. PASS.
- `ls internal/` -> cli, model only. NO-COLLECTORS-PASS.
- `grep em dash` on plan, decisions, schema, model -> NO-EMDASH-PASS.

Notes:

- Validator mirrors schema required, patterns (^RC-[A-F0-9]{4}$, ^C-, ^E-, ^F-), enums, evidence ref resolution, duplicate IDs, unique refs. Nil slices treated as empty; canonical nil to [] normalization deferred to Phase B Task 9 store serializer.
- Zero new Go dependencies. go.mod still has no requires. This keeps core small and auditable per spec product goal 7. See D-006.

## Phase B: store and renderers (spec Section 17 Tasks 8 to 14)

### Task 8: receipt ID generator

Status: done 2026-10-14

Scope:

- RC-XXXX display IDs with collision retry and injectable source for deterministic tests.
- IDs are display identifiers only, not security tokens.

Acceptance:

- [x] Format matches ^RC-[A-F0-9]{4}$
- [x] Collision retries to next free ID
- [x] Exhaustion returns ErrIDCollision
- [x] Deterministic FixedSource for tests
- [x] `go test ./...` and race pass, stdlib only

Changed files:

- `internal/store/id.go` (new, GenerateID, GenerateIDWithRetries, DefaultRandomUint16 via crypto rand, FixedSource)
- `internal/store/id_test.go` (new, 4 tests)

TDD record:

- Red: wrote `id_test.go` first, ran `go test ./internal/store -v`. Result: FAIL build failed undefined FixedSource, GenerateID, GenerateIDWithRetries, ErrIDCollision. Expected red.
- Green: implemented `id.go`. Result: PASS 4 tests.

Exact validation commands and results 2026-10-14:

- `go test ./internal/store -v` before impl -> FAIL build failed. PASS as red step.
- `gofmt -l .` -> empty. PASS.
- `go vet ./...` -> exit 0. PASS.
- `go test ./internal/store -v` -> 4 PASS (Format, RetriesOnCollision, FailsWhenExhausted, ZeroValue). PASS.
- `go test ./...` -> cmd no test files, cli ok, model ok cached, store ok. PASS.
- `go test -race ./...` -> ok cli, model, store 1.899s. PASS.
- `go list -f Imports ./internal/store` -> [crypto/rand encoding/binary errors fmt] stdlib only. PASS.

Notes:

- DefaultMaxIDAttempts 100. 16 bit space (65536 IDs). Collision retry via exists func, wired to store existence check in Task 10.
- FixedSource cycles preset values, empty source returns error. Zero value RC-0000 covered.

### Task 9: canonical JSON serializer and schema validation

Status: done 2026-10-14

Scope:

- Deterministic canonical JSON for receipts with stable field and slice order.
- Validation before marshal and on unmarshal via model contract, stdlib only.

Acceptance:

- [x] Same receipt marshals byte identical twice
- [x] Nil slices normalize to [] for schema arrays
- [x] Invalid receipts fail marshal, bad JSON fails unmarshal
- [x] Evidence slice order does not affect output (sorted by ID)
- [x] `go test ./...` and race pass

Changed files:

- `internal/store/canonical.go` (new, MarshalCanonical, UnmarshalCanonical, normalize with UTC times, sorted claims/evidence/flags)
- `internal/store/canonical_test.go` (new, 5 tests)

TDD record:

- Red: wrote `canonical_test.go` first, ran `go test -run Canonical/Marshal`. Result: FAIL build failed undefined MarshalCanonical, UnmarshalCanonical. Expected red.
- Green: implemented `canonical.go`. Result: PASS 5 canonical tests plus existing 4 ID tests.

Exact validation commands and results 2026-10-14:

- `go test ./internal/store -run Canonical/Marshal -v` before impl -> FAIL build failed. PASS as red step.
- `gofmt -l .` -> empty. PASS.
- `go vet ./...` -> exit 0. PASS.
- `go test ./internal/store -v` -> 9 PASS (5 canonical: Deterministic, NormalizesNilSlices, RejectsInvalid, RejectsBadJSON, StableSliceOrdering; 4 ID). PASS.
- `go test ./...` -> cmd no test files, cli ok, model ok, store ok. PASS.
- `go test -race ./...` -> ok all. PASS.

Notes:

- Struct field order plus Go sorted map keys give deterministic output. Claims, evidence, flags sorted by ID. Warnings keep input order. Timestamps converted to UTC without mutating input (copies slices).
- Nil to [] normalization resolves Task 6 open item for schema arrays. EvidenceRefs also normalized.

### Task 10-11: JSONL hash chain store plus corruption tests

Status: done 2026-10-14

Scope (spec Sections 4.6 and 13):

- File store at Dir (.receipts): receipts/RC-XXXX.json, events.jsonl, tmp/.
- Append only JSONL events with sequence, timestamp, receipt_id, event_type, payload_hash, previous_hash, event_hash.
- event_hash is hex SHA256 over canonical JSON of the other six fields. First previous hash is 64 zeroes.
- Atomic receipt writes via temp file plus rename. Lock file with bounded wait (500 tries, 10ms) for append safety.
- VerifyChain checks sequence continuity, linkage, event hashes, and payload hashes. Reports first failing sequence, never rewrites history.
- Save requires ID set, fills chain linkage on write. Same content twice is idempotent (no duplicate event). Different content under existing ID fails.
- Added model.ReceiptSummary and model.ChainVerification for the Store contract.

Acceptance:

- [x] Save/Get round trip with chain filled
- [x] Latest returns most recent, List in chain order
- [x] VerifyChain passes on valid chain
- [x] Idempotent Save, duplicate ID rejection
- [x] Corrupt line, tamper, reorder, delete all fail VerifyChain with correct sequence
- [x] 10 concurrent Saves all succeed and verify
- [x] No temp files remain, no collectors beyond store, stdlib only

Changed files:

- `internal/model/receipt.go` (added ReceiptSummary, ChainVerification)
- `internal/store/chain.go` (new, Event, ZeroHash, event and payload hashing)
- `internal/store/jsonl.go` (new, appendLine, readLines)
- `internal/store/store.go` (new, Store with Save/Get/Latest/List/VerifyChain, lock file, atomic writes)
- `internal/store/store_test.go` (new, 11 tests)
- `internal/store/canonical_test.go` (added NilAndEmptySlicesMatch regression test)

TDD record:

- Red: wrote `store_test.go` first. Result: FAIL build failed undefined Store, New. Expected red.
- Green then fix: implemented chain, jsonl, store. Initial green had 4 failures from canonical empty slice bug (append to nil base returned nil, stored null arrays, payload mismatch). Diagnosed via debug test showing stored `"claims":null` vs recomputed `[]`. Fixed normalizeReceipt to append to empty non nil base. Added regression test for nil vs empty equivalence. Result: PASS all.

Exact validation commands and results 2026-10-14:

- `go test ./internal/store -v` before impl -> FAIL build failed. PASS as red step.
- `gofmt -l .` -> empty. PASS.
- `go vet ./...` -> exit 0. PASS.
- `go test ./internal/store -v` -> 21 PASS (6 canonical incl new regression, 4 ID, 11 store: RoundTrip, LatestAndList, VerifyPasses, Idempotent, DuplicateReject, CorruptLine seq 2, Tamper, Reorder, Delete, Concurrent 10, NoTempFiles). PASS.
- `go test ./...` -> cmd no test files, cli/model/store ok. PASS.
- `go test -race ./...` -> ok all. PASS.

Notes:

- Payload is hash of chain zeroed canonical receipt, avoiding circular hashing. Stored receipt carries event linkage for display.
- Save holds the lock across existence check, event sequencing, file write, and append, so concurrent same ID saves serialize to idempotent or duplicate error, never split chain.
- Get reads without lock (atomic rename files are complete). Latest/List/VerifyChain hold the lock for consistent event reads.
- Lock timeout error names stale lock path for manual recovery. No automatic stale removal in v0.1.

### Tasks 12-14: terminal and Markdown renderers plus show command

Status: done 2026-10-14

Scope (spec Sections 4.5 and 6 show):

- Terminal renderer with color suppression (Options.Color plus NO_COLOR env) and stable ID ordering.
- Markdown renderer with user text escaping and golden tests.
- Show command with latest/ID lookup, terminal/md/json formats, --workdir, --out, --raw.

Acceptance:

- [x] Terminal output deterministic, golden matched, no ANSI when disabled or NO_COLOR set
- [x] Markdown golden matched, user backticks/asterisks escaped
- [x] Show latest, by ID, md/json formats, --out file, unknown ID exit 4, missing store exit 4, bad format exit 2
- [x] Existing CLI help/version tests still pass, show --help has Usage
- [x] `go test ./...` and race pass

Changed files:

- `internal/render/renderer.go` (new, Format, Options, Renderer interface per spec 4.5)
- `internal/render/terminal.go` (new, sorted sections, verdict/severity colors, Compact omits reasons, IncludeRaw adds evidence data, flag singular/plural summary)
- `internal/render/markdown.go` (new, escaped headings, bullets, code blocks for raw evidence data)
- `internal/render/render_test.go` (new, 5 tests: 2 goldens, color suppression incl NO_COLOR, determinism, escaping)
- `internal/render/golden_gen_test.go` (new, UPDATE_GOLDENS=1 writer, skipped by default)
- `testdata/golden/show-terminal.txt` (new golden)
- `testdata/golden/show-markdown.md` (new golden)
- `internal/cli/show.go` (new, flag parsing incl = forms, format/workdir/out/raw, exit 0/2/4, raw stderr warning, --out creates parents)
- `internal/cli/show_test.go` (new, 8 tests)
- `internal/cli/root.go` (show dispatches to runShow, show help without not implemented note, root status line updated)

TDD record:

- Red render: wrote `render_test.go` first. Result: FAIL undefined TerminalRenderer, Options, MarkdownRenderer. Color/determinism/escaping tests also blocked. Expected red.
- Green render: implemented renderer/terminal/markdown, generated goldens via UPDATE_GOLDENS=1, inspected output, reran. Result: PASS (goldens match, color/escaping pass).
- Red show: wrote `show_test.go` first. Result: FAIL undefined runShow. Expected red.
- Green show: implemented show.go plus root.go wiring. Result: PASS all 8 show tests plus 6 existing CLI tests.

Exact validation commands and results 2026-10-14:

- `go test ./internal/render -v` before impl -> FAIL build failed. PASS as red step.
- `go test ./internal/cli -run TestShow` before impl -> FAIL undefined runShow. PASS as red step.
- `gofmt -l .` -> empty. PASS.
- `go vet ./...` -> exit 0. PASS.
- `go test ./internal/render -v` -> 5 PASS plus 1 SKIP (golden writer). PASS.
- `go test ./internal/cli -v` -> 14 PASS (6 existing + 8 show). PASS.
- `go test ./...` -> cmd no test files, cli/render/model/store ok. PASS.
- `go test -race ./...` -> ok all. PASS.
- `receipts show --help` -> exit 0 with Usage and flags. PASS. `receipts audit --help` -> still stub note. PASS.

Notes:

- Show json uses store.MarshalCanonical (canonical receipt JSON). Dedicated render/json.go, html.go, comment.go, svg.go deferred to Phase F/G per spec layout. See D-007.
- Show terminal defaults Color false (plain text). Renderer color API tested and available; CLI auto TTY detection deferred to avoid a new dependency. --out files never contain ANSI.
- --raw sets IncludeRaw and prints a sensitive content warning to stderr. Secret redaction for command tails lands with Phase F capture.

## Phase C: Git evidence (spec Section 17 Tasks 15 to 20)

### Task 15: Git root/HEAD/status via controlled runner

Status: done 2026-10-14

Scope:

- Git executable through a small process runner with separate stdout/stderr capture and context cancellation. No Go Git library.
- FindRoot via rev-parse --show-toplevel with ErrNotGit mapping.
- Head via rev-parse HEAD, Branch via rev-parse --abbrev-ref HEAD.
- Collect snapshot with porcelain v1 -z status, sorted files, Dirty flag, empty Head tolerated for no commit repos.

Acceptance:

- [x] FindRoot from nested subdir, ErrNotGit outside repo
- [x] HEAD is 40 hex, Branch non empty, no commit HEAD errors
- [x] Dirty status with spaced untracked path, clean repo not dirty
- [x] Canceled context errors, Snapshot outside repo is ErrNotGit
- [x] `go test ./...` and race pass, zero new dependencies

Changed files:

- `internal/gitstate/git.go` (new, runner, ErrNotGit, FindRoot, Head, Branch)
- `internal/gitstate/snapshot.go` (new, FileState, Snapshot, Collect with -z parsing and rename second field capture)
- `internal/gitstate/git_test.go` (new, 8 tests with temp repos and fixed git identity)

TDD record:

- Red: wrote `git_test.go` first. Result: FAIL undefined FindRoot, ErrNotGit, Head, Branch, Snapshot. Expected red.
- Green issues fixed: renamed Collect (type/func collision), fixed macOS /private symlink comparison via EvalSymlinks. Result: PASS 8 tests.

Exact validation commands and results 2026-10-14 (git 2.50.1 Apple Git-155):

- `go test ./internal/gitstate -v` before impl -> FAIL build failed. PASS as red step.
- `gofmt -l .` -> empty. PASS.
- `go vet ./...` -> exit 0. PASS.
- `go test ./internal/gitstate -v` -> 8 PASS (FindRoot, OutsideRepo, HeadAndBranch, HeadNoCommits, StatusDirtyAndUntracked with space path, StatusClean, ContextCancel, SnapshotOutsideRepo). PASS.
- `go test ./...` -> all packages ok. PASS.
- `go test -race ./internal/gitstate/` -> ok 4.388s. PASS.
- `go list -m all` -> module only, zero requires. NO-GIT-LIB-PASS.

Notes:

- Runner uses git -C dir so callers pass any workdir. Stderr truncated to 500 chars in errors. Rename entries consume the -z second field into RenamedFrom; full rename detection and diff hashes deferred to Task 17. Untracked dirs appear as single entries per porcelain default; Task 17 decides on --untracked-files handling. No commit repos return empty Head with status intact; confidence mapping lands in Task 16.

### Task 16: dirty baseline and confidence

Status: done 2026-10-14

Scope (spec Sections 6 audit step 4 and 11):

- Resolution order: run start snapshot (high), verified session commit (medium), current state only (low), no commits (unknown).
- Never invent a before state. DirtyBefore unknown unless a run start observed it.

Acceptance:

- [x] Run start same root yields high with mapped dirty true/false and branches
- [x] Mismatched root ignored, falls to low with empty HeadBefore
- [x] Session commit yields medium with DirtyBefore unknown
- [x] Audit only with HEAD yields low, empty HeadBefore, uncertainty warning
- [x] No commits yields unknown with empty heads
- [x] Run start wins over session commit
- [x] `go test ./...` and race pass, no new dependencies

Changed files:

- `internal/gitstate/baseline.go` (new, Baseline type, ResolveBaseline)
- `internal/gitstate/baseline_test.go` (new, 7 tests)

TDD record:

- Red: wrote `baseline_test.go` first. Result: FAIL undefined ResolveBaseline (7 sites). Expected red.
- Green: implemented `baseline.go`. Result: PASS 7 tests plus existing 8.

Exact validation commands and results 2026-10-14:

- `go test ./internal/gitstate -run TestBaseline` before impl -> FAIL build failed. PASS as red step.
- `gofmt -l .` -> empty. PASS.
- `go vet ./...` -> exit 0. PASS.
- `go test ./internal/gitstate -v` -> 15 PASS (8 Task 15 + 7 Task 16). PASS.
- `go test ./...` -> all packages ok. PASS.
- `go test -race ./internal/gitstate/` -> ok 3.516s. PASS.
- Imports: bytes context errors fmt model os/exec sort strings. Model import is one way (model imports nothing internal). No cycle.

Notes:

- Warnings strings are plain sentences without em dashes. BASELINE_UNCERTAIN flag code itself lands with integrity checks in Task 19; warnings here feed receipt warnings.
- RunStart path is forward compatible for Phase F run snapshots; session commit path for Phase D adapters. Both tested with constructed snapshots, no adapter or run code needed.

### Task 17: path normalization, renames, diff hashes

Status: done 2026-10-14

Scope:

- NormalizePath to repo relative slash form with traversal rejection.
- Rename capture in status (new plus source) and diff (old then new order).
- Status uses --untracked-files=all so untracked dirs expand to files.
- Changed merges tracked diffs vs HEAD with untracked content hashes; per file patch hashes plus binary flags; DiffHash overall fingerprint.

Acceptance:

- [x] Normalization cases, traversal rejection, case/space/unicode preserved
- [x] git mv rename captured with source, diff rename order unit locked
- [x] Untracked dir expands, no bare dir entries
- [x] Patch hashes deterministic, change on content change, spaced paths listed
- [x] Binary files marked
- [x] Full suite and race pass, zero new dependencies

Changed files:

- `internal/gitstate/paths.go` (new, NormalizePath)
- `internal/gitstate/diff.go` (new, ChangedPath, Changed, DiffHash, numstat binary merge)
- `internal/gitstate/snapshot.go` (status adds --untracked-files=all)
- `internal/gitstate/paths_test.go` (new, 8 tests)

TDD record:

- Red: wrote `paths_test.go` first. Result: FAIL undefined NormalizePath, Changed, ChangedPath. Expected red.
- Green issue: diff --name-status -z separates status and path into distinct NUL fields with rename order old then new (verified by hand against git 2.50.1: R100 old new), unlike status porcelain which joins them. Rewrote parser positionally with isNameStatusField guard. Added parse order unit test. Result: PASS all.

Exact validation commands and results 2026-10-14:

- Red run -> FAIL build failed. PASS as red step.
- Hand verification of -z formats via xxd for modify and rename cases. Confirmed.
- `gofmt -l .` -> empty. PASS.
- `go vet ./...` -> exit 0. PASS.
- `go test ./internal/gitstate -v` -> 23 PASS (15 prior + 8 new). PASS.
- `go test ./...` -> all packages ok. PASS.
- `go test -race ./internal/gitstate/` -> ok. PASS (rerun in Task 18 batch).

Notes:

- Tracked patch hash is SHA256 of git diff --no-color --no-ext-diff --no-textconv -U0 HEAD output; untracked is SHA256 of file content. Both deterministic fingerprints; semantics documented on ChangedPath. Unreadable untracked files are skipped.
- No commit repos fall back to untracked only in Changed since diff HEAD has no revision.
- Backslashes stay literal on unix; Windows separator handling deferred with full Windows support (spec non goal for v0.1-v0.4).

### Task 18: non-Git fallback

Status: done 2026-10-14

Scope (spec Section 11 non Git projects):

- IsRepo check, NonGitRepo record (NonGit true, confidence unknown, absolute root), WarnNonGitProject constant, AsRepo converter for Git snapshots, IsNotGit helper.

Acceptance:

- [x] Fresh temp dir is not a repo, subdir of repo is
- [x] NonGitRepo has NonGit true, unknown confidence, non empty root
- [x] Full suite passes

Changed files:

- `internal/gitstate/nongit.go` (new)
- `internal/gitstate/nongit_test.go` (new, 2 tests)

TDD record:

- Red: wrote `nongit_test.go` first. Result: FAIL undefined IsRepo, NonGitRepo, WarnNonGitProject. Expected red.
- Green: implemented `nongit.go`. Result: PASS.

Exact validation 2026-10-14:

- `go test ./internal/gitstate -run TestIsRepo|TestNonGit -v` -> 2 PASS.
- `go test ./internal/gitstate -v` (final) -> 25 PASS (see Task 19-20 batch).
- `go test ./...` -> all packages ok.

Notes:

- Audit pipeline in Task 31 will branch on IsNotGit: Git path uses Collect plus ResolveBaseline plus AsRepo; non Git path uses NonGitRepo plus WarnNonGitProject and skips diff checks. Documented on NonGitRepo.

- Audit pipeline in Task 31 will branch on IsNotGit: Git path uses Collect plus ResolveBaseline plus AsRepo; non Git path uses NonGitRepo plus WarnNonGitProject and skips diff checks. Documented on NonGitRepo.

### Tasks 19-20: integrity checks plus fixtures

Status: done 2026-10-14

Scope (spec Section 10, all seven checks plus codes in 10.8):

- internal/integrity package operating on changed files with before/after contents plus added lines, scope policy, check toggles, and baseline confidence. Pure functions, no filesystem or network use. Flags never auto contradict claims.
- Test files (10.1): TEST_FILE_MODIFIED warning, TEST_FILE_DELETED high. Added test files not flagged. Fixtures flag at warning by design.
- Assertions (10.2): token count decrease in changed test files flags ASSERTION_TOKEN_COUNT_REDUCED warning with N -> M wording; comments and strings stripped per extension; never claims removals.
- Skips (10.3): added line markers flag SKIP_ADDED warning with path:line, marker, and trimmed context (redacted when secret-like); count fallback without diff lines.
- Scope (10.4): OUT_OF_SCOPE_CHANGE warning outside allowed globs, FORBIDDEN_PATH_CHANGED high on forbidden match (forbidden wins), nothing without configured scope, unparseable paths flagged fail safe.
- Dependencies (10.5): package.json, go.mod, requirements, Cargo, pyproject parsed and diffed into DEPENDENCY_ADDED info, DEPENDENCY_REMOVED warning, DEPENDENCY_VERSION_CHANGED info; lockfiles and unparsed manifests flag LOCKFILE_CHANGED info.
- Config (10.6): CI_CONFIG_CHANGED, BUILD_CONFIG_CHANGED, LINTER_CONFIG_CHANGED warnings; dep manifests excluded to avoid double counting.
- Secrets (10.7): added lines only, private key and token prefixes high, assignments and JWT and high entropy warning, placeholder denylist, lockfile hashes skipped, raw values never stored (pattern name, path:line, sha256 only).
- BASELINE_UNCERTAIN info when confidence is not high. Stable F-001 IDs after sorting by code, title, paths.

Acceptance:

- [x] Every check has positive, negative, and false positive tests (21 tests)
- [x] Secrets never appear in flag output (asserted)
- [x] Flag IDs deterministic across runs
- [x] All toggles honored
- [x] `go test ./...` and race pass, zero new dependencies

Changed files:

- `internal/integrity/types.go` (new, Input, Scope, Checks, Run, ID assignment)
- `internal/integrity/testfiles.go`, `assertions.go`, `skips.go`, `scope.go`, `deps.go`, `config.go`, `secrets.go` (new)
- `internal/integrity/integrity_test.go` (new, 21 tests, inline synthetic fixtures)

TDD record:

- Red: wrote `integrity_test.go` first. Result: FAIL undefined Input, DefaultChecks, FileContent, Run across the file. Expected red.
- Green issue: scope.go import block closed with } instead of ) (typo caught by gofmt missing import path at 9:1; bisected with od/cmp to byte 145). Fixed to ). Result: PASS all 21 on first green run.

Exact validation commands and results 2026-10-14:

- Red run -> FAIL build failed undefined package. PASS as red step.
- `gofmt -l .` -> empty after fix. PASS.
- `go vet ./...` -> exit 0. PASS.
- `go test ./internal/integrity -v` -> 21 PASS. PASS.
- `go test ./...` -> all 6 packages ok. PASS.
- `go test -race ./...` -> ok all. PASS.
- `go list -m all` -> module only, zero requires. PASS.

Notes:

- Fixtures are inline synthetic values (example AWS key, fake PEM, placeholder passwords); no real secrets, no private transcripts. testdata stays limited to golden receipts and renderer goldens.
- Secret hash in details is SHA256 over the match only, never reversible to the value; weak values additionally protected by the placeholder denylist. If a future check needs hashes for dedup, reuse this format.
- Config check owns CI/build/lint paths; dependency manifests excluded there. Scope check owns allowed/forbidden policy (config file wiring lands in Phase G Tasks 53-54; Scope struct is ready).
- EvidenceRefs on flags are empty at this layer; the Phase E audit pipeline attaches diff evidence IDs.

## Phase D: adapters and claim extraction (spec Section 17 Tasks 21 to 28)

### Task 21: verify OpenCode transcript format

Status: done 2026-10-14. VERIFIED, no guessing. Evidence and sources in DECISIONS.md V-001.

- Live immutable reads: `~/.local/share/opencode/opencode.db` with session/message/part/project/project_directory tables; roles user/assistant; part types text/step-start/reasoning/tool/step-finish/patch/compaction/file; epoch ms times.
- Official docs confirm DB path plus OPENCODE_DB override. Open source message-v2.ts confirms tool part shapes. Only key names recorded, values never read.

### Task 22: OpenCode adapter

Status: done 2026-10-14.

Scope: adapter types foundation plus SQLite backed adapter with synthetic fixtures.

Acceptance:

- [x] Detect by DB presence; LatestSession by directory match incl project_directory fallback, newest time_updated wins
- [x] Messages map roles, concat text (ignored skipped), completed/error tool calls with args and results, pending/running skipped, other parts skipped
- [x] Timestamps ms to UTC, 100k message bound, 50MiB output bound, SQL escaping, missing CLI actionable error
- [x] Full suite and race pass, zero new dependencies

Changed files:

- `internal/adapters/adapter.go` (new, Role/Message/ToolCall/SessionRef/Adapter plus 50MiB/100k limits)
- `internal/adapters/opencode/parser.go` (new, pure row parsing and selection)
- `internal/adapters/opencode/adapter.go` (new, read only sqlite3 -json queries with WAL retry)
- `internal/adapters/opencode/fixtures/session_rows.json`, `message_rows.json` (new, synthetic verified shapes)
- `internal/adapters/opencode/parser_test.go`, `adapter_test.go` (new, 9 tests)

TDD record:

- Red: parser test first (undefined parseSessionRows etc.), adapter test (undefined runShow equivalent runShow? no: undefined Adapter helpers). Green fixes: added missing -json flag (output was list mode), fixed huge test to use distinct IDs. Result: PASS 9.

Exact validation 2026-10-14:

- `go test ./internal/adapters/opencode -v` -> 9 PASS. `go test ./...` PASS. Race PASS.

### Task 23: verify Claude Code transcript format

Status: done 2026-10-14. VERIFIED. Evidence and sources in DECISIONS.md V-002.

- Live files: `~/.claude/projects/-Users-jasonamadi/<uuid>.jsonl`; slug rule slashes to dashes; user content str or list, assistant text blocks, system subtype lines, meta lines; RFC 3339 timestamps; per line cwd.
- Tool shapes from community references (id/name/input, tool_use_id/content str or array) since local sample had no tool calls. Parser defensive to drift.

### Task 24: Claude adapter

Status: done 2026-10-14.

Scope: JSONL adapter with tool pairing and synthetic fixture.

Acceptance:

- [x] Slug directory preferred, bounded cwd scan fallback, newest mtime wins
- [x] user/assistant/system mapping, tool_use/result pairing by id, orphans dropped, sidechain tagged, meta skipped
- [x] 50MiB bound, 100k cap returns parsed prefix plus error (provable line boundary), corrupt lines fail
- [x] Full suite and race pass, stdlib only

Changed files:

- `internal/adapters/claude/parser.go`, `adapter.go` (new)
- `internal/adapters/claude/fixtures/session.jsonl` (new, synthetic)
- `internal/adapters/claude/parser_test.go` (new, 8 tests)

TDD record:

- Red: undefined parseJSONL, slugForDir, Adapter. Green: refactored tool id tracking to side channel (spec ToolCall has no id field). Result: PASS 8.

Exact validation 2026-10-14:

- `go test ./internal/adapters/claude -v` -> 8 PASS. `go test ./...` PASS. Race PASS.

Notes:

- Registry (Task 25), message safety boundaries (Task 26), claim patterns (Tasks 27-28) remain. No transcript content, secrets, or user data in fixtures or logs; verification recorded only shapes and paths.

### Tasks 25-28: registry, safety, claim patterns, dedup and conflicts

Status: done 2026-10-14

Scope (spec Sections 4.1, 4.3, 8):

- Task 25: generic Auto engine with Attempt records in package adapters plus concrete registry subpackage (opencode then claude; codex after Phase G verification).
- Task 26: shared CheckFileSize bound wired into Claude reads and scans; read only file and DB tests; UTC timestamp tests.
- Task 27: DeterministicExtractor with high precision patterns, negation and conditional guards, fence/quote/code span handling, qualified test claims to OTHER.
- Task 28: source refs with excerpt hashes, UNVERIFIED placeholder verdicts for the engine, identical meaning dedup (earliest wins), file scope conflict warnings; pass versus fail contradictions stay in the verdict layer via command evidence.

Acceptance:

- [x] Auto picks first success, records skips and failures, lists reasons, honors cancel
- [x] Oversize rejected pre read, read only storage reads fine, all fixture timestamps UTC
- [x] All spec 8.2 positive and negative examples behave; edge cases (negated, conditional, quoted, fenced, intent, no tests passed, qualified) excluded or typed OTHER
- [x] Only assistant messages create claims; stable C-001 IDs; pattern ids recorded off receipt
- [x] Full suite and race pass, zero new dependencies

Changed files:

- `internal/adapters/registry.go`, `registry_test.go` (new, Auto engine, 5 tests)
- `internal/adapters/registry/registry.go`, `registry_test.go` (new, defaults and ByName, 3 tests)
- `internal/adapters/safety.go`, `safety_test.go` (new, size bound, 1 test)
- `internal/adapters/claude/adapter.go` (size guard reuse), `parser_test.go` (read only, UTC tests)
- `internal/adapters/opencode/adapter_test.go`, `parser_test.go` (read only DB, UTC tests)
- `internal/model/claim.go` (ExtractionWarning plus conflict code)
- `internal/model/enums.go`, `validate.go`, `model_test.go` (ClaimNoOtherChanges identifier rename, value unchanged)
- `internal/claims/source.go`, `patterns.go`, `extractor.go` (new)
- `internal/claims/patterns_test.go` (10 tests), `extractor_test.go` (6 tests)

TDD record:

- Red: registry tests (undefined Auto), safety test (undefined CheckFileSize), claims tests (undefined extractor). Adapter read only and UTC tests written against existing helpers.
- Green fixes: smarter sentence splitter (periods split only before whitespace plus uppercase or end, keeping ./... and package.json intact), runner gap class widened to cross dots, ClaimNoOtherChanges rename propagated. Result: PASS all.

Exact validation 2026-10-14:

- `go test ./internal/adapters/ ./internal/adapters/registry/` -> 8 PASS (5 Auto + 3 registry). `go test ./internal/adapters/...` all ok.
- `go test ./internal/claims -v` -> 16 PASS (10 patterns + 6 extractor).
- `go test ./...` -> all 9 packages ok. `go test -race` on claims and adapters -> ok.
- `gofmt -l` empty, `go vet` exit 0, zero requires, no em dashes.

Notes:

- Registry lives in a subpackage because package adapters cannot import its own children in Go; generic Auto stays in the parent. See D-009.
- Dedup interpretation: identical normalized meaning merges (earliest source wins; multi source retention awaits schema support, documented on the extractor). Distinct claims keep own sources. Conflict warnings only for differing FILES_CHANGED path sets.
- Extractor verdicts are UNVERIFIED placeholders; the Task 29 engine overwrites every verdict deterministically.

## Phase E: verdicts and v0.1 CLI (spec Section 17 Tasks 29 to 36)

### Tasks 29-30: verdict engine, summary, invariants

Status: done 2026-10-14.

Scope (spec Section 9 table): DeterministicEngine joining test/build/lint claims to command, test_result, and reverify evidence; file claims to diff paths only on high baselines; dependency rules with denial contradiction; issue/other always unverifiable. Verdicts never exceed evidence. Summarize plus CheckInvariants (unique IDs, enum verdicts, resolvable refs, matching counts).

Changed files: `internal/verdict/engine.go`, `rules.go`, `evidence_match.go`, `verdict_test.go` (19 tests); `internal/model/claim.go` (VerdictWarning).

TDD record: red undefined engine/summary/invariants. Green fixes: dual []any/[]string evidence forms, word scan path recovery, denial trailing word boundary, summary test setup. Result: PASS all.

### Task 31: audit pipeline

Status: done 2026-10-14.

Scope: collectors git evidence builder (snapshot, diff with dep deltas, file contents, added lines, flags reference diff) plus cli audit wiring (adapter auto or named with Git only fallback, transcript and adapter_status evidence, extractor, conflict flags, engine, invariants, validation, store, render). Exit 0 on ran, 2 usage, 3 collection failure, 4 store/schema failure. ID assigned before validation (fixed ordering bug found in tests).

Changed files: `internal/collectors/git.go` (+ChangedFiles export, DepDeltas export), `git_test.go`; `internal/cli/audit.go`, `audit_test.go` (5 tests, hermetic HOME); root dispatch and help.

### Task 32: init command

Status: done 2026-10-14. Starter receipts.toml plus append only .gitignore handling with --force backup semantics. Refusal exit 2. 5 tests. Root wiring.

Changed files: `internal/cli/init.go`, `init_test.go`.

### Task 33: demo scenarios

Status: done 2026-10-14. Temp repo fake agent plus deterministic assembly (fixed RC-DE07, fixed timestamps, high captured baseline, synthetic command evidence). Default shows VERIFIED plus CONTRADICTED plus UNVERIFIED plus flags per acceptance. Golden terminal output. --keep support. 6 scenario tests plus CLI tests.

Changed files: `internal/demo/fake_agent.go`, `scenario.go`, tests, `testdata/golden/demo-terminal.txt`; `internal/cli/demo.go`, `demo_test.go`; root wiring.

### Task 34: labeled dataset

Status: done 2026-10-14. 200 items (50 positive, 50 negative, 40 evidence, 30 file scope, 30 ambiguous) with harness measuring gates as failing tests. Measured: precision 1.000, recall 1.000, verdict 65/65, zero unexpected contradictions. Two extractor improvements from misses: question guard, dependency denial extraction.

Changed files: `testdata/labeled-claims.jsonl`; `internal/claims/labeled_test.go`.

### Task 35: README and SECURITY

Status: done 2026-10-14. README follows the spec outline with real demo output, honest command availability, tiers, verified adapter table, privacy, config, troubleshooting. SECURITY.md states non boundary, chain limits, data handling. No em dashes.

### Task 36: v0.1 readiness

Status: done 2026-10-14 (code complete locally; release blocked, see checklist).

Readiness against v0.1 acceptance criteria:

- [x] `go test ./...` passes on macOS (verified). Linux runs in CI (workflow present, not runnable from here).
- [x] `receipts audit` in a Git repo yields a valid schema-versioned receipt with no adapter (TestAuditGitNoTranscript validates plus invariants).
- [x] Every verdict carries a reason and resolvable refs (engine sets both; invariants enforce).
- [x] Every receipt records tier 0, baseline confidence, limitations (audit sets all; warnings always present on low baselines).
- [x] Extraction and renderers deterministic on fixtures (determinism tests plus goldens).
- [x] No command modifies tracked files; audit/show/demo/init run no project commands (exec audit: git and sqlite3 reads only, demo git in temp dirs).
- [x] Demo output holds verified, contradicted, unverified claims plus flags (golden).
- [x] README states audit cannot reconstruct the exact pre-session tree without a snapshot.

Release blockers (do not publish): Linux CI run, license selection, receipt name availability checks, git init plus remote. Recorded as owner decisions.

Exact validation 2026-10-14: `gofmt -l .` empty; `go vet ./...` exit 0; `go test -count=1 ./...` all 14 packages ok; `go test -count=1 -race ./...` all ok.

## Phase F: live run and re-verification (spec Section 17 Tasks 37 to 48)

### Tasks 37-38: run context plus shim executable and event protocol

Status: done 2026-10-14. Run dirs under `.receipts/runs/<run-id>/` with `commands.jsonl`; single binary re-exec via argv0 symlinks; JSONL event protocol with lock guarded appends and corrupt line failures. 9 shim tests (red first).

### Tasks 39-40: shim set, PATH resolution, streaming output

Status: done 2026-10-14. 17 supported names; REAL_PATH resolution skipping the shim dir (recursion test with trap binary); missing real exits 127 with a recorded event; collector outage never breaks the wrapped command. Concurrent stdout/stderr tee with full output SHA-256, 8 KiB redacted tails, unbounded forwarding. Secret redaction exported from integrity with placeholder parity.

### Tasks 41-43: run command, shell verification, claim matching

Status: done 2026-10-14. `receipts run` with pre snapshots, child inheritance, signal forwarding, child exit preservation, high baselines from captured pre state, tier 1 or tier 0 with COMMAND_CAPTURE_INCOMPLETE disclosure, --quiet, receipt summary. Claim matching reuses the verdict argv classifiers; test_result evidence flows from output parsing. PATH inheritance to direct children verified by probe test. Agent shell hook behavior stays UNVERIFIED (V-003): capture is documented shim only.

### Tasks 44-46: worktree reverify plus check

Status: done 2026-10-14. Detached temp worktrees with prefix guarded idempotent cleanup; dirty/no HEAD refusals with --force override (never deletes user data); shell-less command parsing with metachar refusal; timeout (124) and interrupt (130) codes; committed_state_only evidence labels; `check` re-evaluates without running commands, reports staleness, and appends reverify evidence as new receipts. Owned .receipts/ noise excluded from diffs, safety dirt, and staleness. SIGINT during reverify verified manually: exit 130, no worktree or temp leaks, partial receipt saved with warning.

### Task 45: dirty patch reconstruction

Status: explicitly deferred 2026-10-14, see D-010. Committed state only, labeled on every reverify evidence.

### Tasks 47-48: HTML renderer plus shim benchmark

Status: done 2026-10-14. Self contained escaped HTML with golden plus markup, forbidden construct, and determinism tests; --format html on show, audit, demo, check. Benchmark: shim median ~29.5ms, p95 ~34ms, overhead median ~19.5ms vs direct exec on i7-9750H darwin/amd64 go1.27.1. p95 meets the 40ms target; median misses 15ms. Profile attributes ~20ms to fsync dominated event append; durability kept, batching noted as future work. See D-011.

Exact validation 2026-10-14: `gofmt` empty, `go vet` 0, `go test -count=1 ./...` all 16 packages ok, race on cli/reverify/collectors/render ok, zero requires, no em dashes.

## Phase G: CI, policies, and release (spec Section 17 Tasks 49 to 59)

### Tasks 49-50: ci command, comment renderer, GitHub Action

Status: done 2026-10-14.

Scope: `receipts ci` with comment/md/json formats and optional base URL link; CommentRenderer with update marker, secret redaction, no invented links; composite action with pinned release download plus checksum verification, audit, comment create-or-update by marker, artifact fallback for forks, and fail policy parsed from receipt JSON.

Changed files: `internal/render/comment.go` (+tests, golden `ci-comment.md`), `internal/cli/ci.go` (+tests, dispatch), `action/action.yml`, README CI section.

Validation: render 4 comment tests PASS plus golden; cli ci 4 tests PASS; action YAML parses with 5 expected inputs; all 5 embedded scripts pass `bash -n`. Full suite PASS.

Notes: download base URL carries an OWNER placeholder until the owner is confirmed; the action only works against published releases.

### Task 51: Codex verification

Status: investigated 2026-10-14, adapter deferred per schedule. No local install exists. Location plus envelope plus cwd association verified from the open source recorder and community references; format undocumented and drifting. No code or fixtures added; registry unchanged. See V-004.

### Task 52: LLM extractor

Status: done 2026-10-14. Opt-in only (disabled errors), local endpoint preferred, remote endpoints warn with destination, strict proposal validation (known type, existing assistant message, verbatim quote, exact claim fields), shared dedup and source path, UNVERIFIED verdicts only, in memory cache by model plus prompt plus transcript hash. 5 tests with local stub servers; no network in tests. See D-012.

Changed files: `internal/claims/llm.go`, `llm_test.go`.

## Phase G continued: Tasks 53-59

### Tasks 53-54: TOML config plus scope and check policies

Status: done 2026-10-15.

Scope: strict subset TOML parser with unknown key errors, typed validation with field errors, published config schema, policy wiring into audit/run/check with visible receipt notes, LLM merge path.

Acceptance:

- [x] Missing file yields conservative defaults; LLM stays disabled
- [x] Unknown keys/sections error; malformed TOML errors; bad enums, timeouts, globs, tails, llm endpoint error with field paths
- [x] Scope globs drive integrity flags; disabled checks stay off with notes; strictness recorded
- [x] check --reverify falls back to configured test.command
- [x] LLM enabled merges validated proposals with dedup; endpoint failure degrades to a warning
- [x] Full suite and race pass, zero new dependencies

Changed files: `internal/config/` (5 files plus schema), `internal/claims/merge.go` (+test), `internal/cli/policy.go` (+tests), collectors GitOptions, audit/run/check wiring.

### Task 55: SVG badge

Status: done 2026-10-15. Deterministic badge with brand verdict colors, fixed text lengths, no scripts/links/remote assets (xmlns excepted and asserted singular). Golden plus color, XML, and determinism tests. `--format svg` on show. PNG deferred: no dependable portable text rendering without new dependencies. See D-013.

Changed files: `internal/render/svg.go` (+tests, golden `badge.svg`), show wiring.

### Task 56: name availability

Status: done 2026-10-15, recorded in V-006. npm receipts, receipts-cli, getreceipts, agent-receipts all taken; rcpt-cli free; GitHub repo path free; Homebrew formula free; receipts.dev registered. No rename performed; owner decision required.

## Phase G final: Tasks 57-59

### Task 57: release builds and checksums

Status: done 2026-10-15. GoReleaser config for four targets with trimpath, stripped ldflags, cgo off, tar.gz archives, SHA-256 checksums, and a draft release gate behind green checks. Tag workflow plus local `scripts/release.sh`. Validated: all four YAML files parse, all scripts pass `bash -n`, all four `GOOS/GOARCH` pairs compile, the amd64 binary prints version and demo output, and a checksum round trip verifies.

Changed files: `.goreleaser.yaml`, `.github/workflows/release.yml`, `scripts/release.sh`.

### Task 58: npm wrapper feasibility

Status: validated 2026-10-15, publish deferred. Provisional `npm/` wrapper (private true, name rcpt-cli pending the owner call): pinned version to asset mapping for all four targets, SHA-256 verification with node crypto, user space install, stdio and exit forwarding. Mapping table plus node versus system checksum agreement both execute green. npm `receipts` is taken, so nothing here can ship under the product name.

Changed files: `npm/package.json`, `install.js`, `bin/run.js`, `test/mapping.js`, `test/checksum.js`.

### Task 59: final acceptance pass

Status: evaluated 2026-10-15. Code complete per the honest reading below. No release published (no repo, no remote, owner decisions open).

v0.1: PASS on macOS with one environment caveat. `go test ./...` green; audit yields valid receipts without transcripts; verdicts carry reasons and resolvable refs; tier, confidence, and limits always recorded; extraction and renderers deterministic; only git and sqlite3 reads run (demo git stays in temp dirs); demo golden holds all three verdicts plus flags; README documents the audit baseline limit. Linux runs only in CI, unrunnable from here.

v0.2: PASS except the documented patch deferral. Wrapped commands record exit codes with bounded output; child PATH inheritance proven; bypasses disclose instead of denying; unsafe reverify refuses without force and never deletes user data; temp cleanup verified live under SIGINT; reverify evidence carries command, checkout, result, and committed state only limits (patch hash N/A by D-010).

v0.3: PARTIAL. Action ships with least privilege docs, marker updates, fork artifact fallback, and no untrusted code execution. Transcript failures keep Git receipts. LLM output is schema strict and verdict free. Codex adapter stays off: format partially verified only, no local install to confirm against.

v0.4: PARTIAL. Config validation, defaults, scope toggles with visible notes, and verdict immunity all hold. SVG deterministic and clean. Name research done with a rename required. Schema compatibility rests on model level validation plus golden checks (no standalone JSON Schema engine, by the zero dependency rule).

Release checklist (all blocked on owner input unless noted):

- [ ] Owner and repository name decision (V-004, V-006). Only rcpt-cli survives from the fallback list.
- [ ] License selection. No LICENSE file exists on purpose.
- [ ] Linux CI green on a pushed repo (workflow present).
- [ ] Rename module path, schema $ids (receipts.dev), action OWNER, and npm name if the name changes.
- [ ] Tag v0.1.0 and let the release workflow build plus checksum (dry validated locally).
- [ ] Publish the npm wrapper under the decided name.
- [ ] Homebrew tap after owner confirmation.

Exact validation 2026-10-15: four arch builds compile and smoke test; checksum round trip ok; node mapping plus checksum tests execute green; `gofmt` clean; `go vet` 0; `go test -count=1 ./...` all 17 packages ok; race on config/cli/render ok; zero requires; no em dashes.

## Validation log

Each entry records exact command, result, and date. Update after every task.

- 2026-10-14 Task 1: `git status` -> fatal not a git repo (expected, no history to preserve). `go version` -> go1.27.1 darwin/amd64 pass. `go env` pass. `which git go gofmt make` pass, `golangci-lint` not found recorded.
- 2026-10-14 Task 2: Skill invocations `ask-matt`, `implement`, `to-tickets`, `tdd` -> success via Skill tool. No tests (skill load only).
- 2026-10-14 Task 3: `grep -n em dash char` on IMPLEMENTATION_PLAN.md and DECISIONS.md -> no match. PASS. `ls -la` confirms only spec existed before, no overwrite. PASS.
- 2026-10-14 Task 4: `gofmt -l .` -> empty. PASS. `go vet ./...` -> exit 0. PASS. `go test ./... -v` -> 6 CLI tests PASS. `go build -o /tmp/receipts ./cmd/receipts` -> BUILD-OK. `go run ./cmd/receipts --help` exit 0 lists commands. `go run ./cmd/receipts --version` prints 0.0.1-dev exit 0. See Task 4 section for full command list.
- 2026-10-14 Task 5: YAML parse YAML-OK. `gofmt -l .` empty PASS. `go vet ./...` exit 0 PASS. `go test ./... -v` 6 PASS. `go test -race ./...` ok PASS.
- 2026-10-14 Task 6: `go test ./internal/model -v` 3 PASS. `go test ./...` PASS. `go test -race ./...` PASS. `gofmt -l .` empty after fmt PASS. `go vet ./...` exit 0 PASS. `go list` imports [time] only PASS.
- 2026-10-14 Task 7: `go test ./internal/model -v` 8 PASS (incl 5 invalid subcases). `go test ./...` PASS. `go test -race ./...` PASS. Schema and golden JSON parse PASS. No collectors PASS.
- 2026-10-14 Task 8: `go test ./internal/store -v` 4 PASS. `go test ./...` PASS. `go test -race ./...` PASS. Stdlib only imports PASS.
- 2026-10-14 Task 9: `go test ./internal/store -v` 9 PASS (5 canonical + 4 ID). `go test ./...` PASS. `go test -race ./...` PASS.
- 2026-10-14 Tasks 10-11: `go test ./internal/store -v` 21 PASS (6 canonical, 4 ID, 11 store). `go test ./...` PASS. `go test -race ./...` PASS. Fixed empty slice null bug with regression test.
- 2026-10-14 Tasks 12-14: `go test ./internal/render -v` 5 PASS + 1 SKIP. `go test ./internal/cli -v` 14 PASS (6 existing + 8 show). `go test ./...` PASS. `go test -race ./...` PASS. Goldens terminal + markdown verified.
- 2026-10-14 Task 15: `go test ./internal/gitstate -v` 8 PASS. `go test ./...` PASS. `go test -race ./internal/gitstate/` ok. Zero new deps, no git library.
- 2026-10-14 Task 16: `go test ./internal/gitstate -v` 15 PASS (8+7). `go test ./...` PASS. `go test -race` PASS.
- 2026-10-14 Task 17: `go test ./internal/gitstate -v` 23 PASS (15+8). Hand verified -z formats via xxd. `go test ./...` PASS. Race PASS.
- 2026-10-14 Task 18: `go test ./internal/gitstate -run TestIsRepo|TestNonGit` 2 PASS. Full suite PASS in batch.
- 2026-10-14 Tasks 19-20: `go test ./internal/integrity -v` 21 PASS. `go test ./...` PASS (6 pkgs). `go test -race ./...` PASS. Zero requires. Fixed scope.go typo.
- Phase D Tasks 21-24: verification via immutable local reads plus official and open source references (no guessing); adapter types; opencode 9 PASS; claude 8 PASS. `go test ./...` PASS. `go test -race ./internal/adapters/...` PASS. Zero requires.
- Phase D Tasks 25-28: Auto 5 PASS + registry 3 PASS + safety bounds + read only/UTC; claims 16 PASS (10 patterns + 6 extractor). `go test ./...` all ok. Race ok. Zero requires.
- Phase E Tasks 29-36: verdict 19 tests PASS; collectors 2 PASS; cli audit 5 + init 5 + demo 3 + existing PASS; demo 7 PASS; labeled gates precision 1.000 recall 1.000 verdict 65/65. `gofmt` empty, `go vet` 0, `go test -count=1 ./...` all ok, race all ok.
- Phase F Tasks 37-48: shell 9 + redaction + parser 5 PASS; run 7 PASS incl subprocess capture and PATH probe; reverify 6 + safety/owned tests PASS; check 7 PASS; html 5 PASS + golden; interrupt drill exit 130 no leaks; bench recorded. Full suite + race PASS. Zero requires.
- Phase G Tasks 49-52: render comment 4 PASS + golden; cli ci 4 PASS; action YAML + 5 bash-n PASS; codex investigated, deferred; claims LLM 5 PASS. Full suite PASS. Zero requires.
- Phase G Tasks 53-56: config 6 PASS + schema; merge 2 PASS; cli policy 5 PASS; render svg 4 PASS + golden + show svg PASS; name checks via registry/RDAP reads. Full suite + race PASS. Zero requires.
- Phase G Tasks 57-59: 4 arch builds compile + smoke + checksums; node mapping/checksum execute green; action YAML + bash-n PASS; full suite (17 pkgs) + race PASS. Zero requires.

## Unresolved verification items (summary, detail in DECISIONS.md)

- V-001 OpenCode transcript location and format: unverified, must check official docs plus local fixtures before adapter work.
- V-002 Claude Code transcript location, format, project association: unverified.
- V-003 Shell invocation and PATH inheritance for OpenCode, Claude Code, Codex: unverified.
- V-004 Go module owner and repo name: unconfirmed, placeholder in use.
- V-005 Git repo init: decision deferred, do not auto init.
- V-006 Receipts name availability: CHECKED 2026-10-15 (npm receipts/-cli/getreceipts/agent-receipts taken, rcpt-cli free; GitHub path free; Homebrew free; receipts.dev registered). Rename deferred to owner.
- V-007 PNG export: deferred, see D-013. SVG required and done.

## Next single task

None remaining in the spec task list. Awaiting owner decisions: product name, license, repository remote. Suggested next work after those land: rename sweep, v0.1.0 tag, Linux CI confirmation.
