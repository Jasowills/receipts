# Receipts Decisions and Verification Log

Date started: 2026-10-14
Spec: `RECEIPTS_OPENCODE_IMPLEMENTATION_SPEC.md` version 1.0

This file records decisions, open verification items, and sources.
Transcript paths and formats are especially likely to change, so they must be verified against current official docs or real local fixtures before use.
No em dashes are used in this file per global spec constraint.

## D-001: Planning and implementation workflow

Decision: follow spec Section 17 phased plan, one task at a time, with Matt Pocock skill workflow layered on top.

Sources and evidence:

- Skill tool invocations on 2026-10-14: `ask-matt`, `implement`, `to-tickets`, `tdd` all loaded successfully.
- `ask-matt` routing confirms: spec exists, so use `to-tickets` style breakdown then `implement` per ticket with `tdd` inside.
- `implement` rules adopted: TDD at agreed seams, regular type checks, single file tests often, full suite at end, code review before commit.
- `to-tickets` limitation: no issue tracker is configured (no git remote, no `docs/agents/issue-tracker.md`). `setup-matt-pocock-skills` not run yet because spec Section 20 requires plan files first in a non git directory. Plan tracking stays in `IMPLEMENTATION_PLAN.md` until a tracker is chosen. No skill behavior invented.
- `tdd` seams agreed for Phase A: CLI help and version output, model JSON round trip, schema validation. See IMPLEMENTATION_PLAN.md.

Status: active

## D-002: Repository state and git init

Finding at start: `/Users/jasonamadi/receipts` was not a git repository. It contained only the spec file.

Update 2026-10-15: at the owner request, the tree is now git initialized (main branch), committed (one initial commit with the full task 1 to 59 tree), and pushed to https://github.com/Jasowills/receipts. Remote description set via gh. Working tree clean at push time.

Status: done

## D-003: Go module path

Spec Section 3: use `github.com/<owner>/receipts` until owner and repo name are confirmed. Do not assume GitHub name is available.

Decision for Task 4: use placeholder `github.com/receipts/receipts` in `go.mod` unless owner provides a different path before Task 4 starts. This keeps imports stable and matches spec layout `cmd/receipts`, `internal/...`.

Rationale: a valid module path is required for `go mod init`. The placeholder is clearly marked temporary and will be renamed only after name availability checks in Phase G Task 56.

Status: provisional, needs owner confirmation

## D-004: Go version and toolchain

Finding 2026-10-14:

- `go version go1.27.1 darwin/amd64`
- `GOROOT /usr/local/Cellar/go/1.27.1/libexec`
- `GOPATH /Users/jasonamadi/go`
- `gofmt` present, `go vet` present via Go toolchain, `make` present, `golangci-lint` not found.

Decision: set `go` directive to 1.27 or current stable supported by toolchain at Task 4 time, document in `go.mod` and README. CI matrix will test macOS latest and Ubuntu latest per spec Section 14.

Status: recorded, revisit if CI requires older Go for compatibility

## D-005: Minimal CLI scope for Task 4

Decision: Task 4 builds only `--help` and version output plus command stubs. No audit logic, no transcript reading, no Git calls, no network.

Rationale: spec Section 20 step 6 says start with minimal compiling Go CLI after documenting plan and blockers. Full CLI reference in Section 6 comes in later phases.

Exit code policy (from spec Section 6): document early that exit 0 means tool succeeded, not that all claims verified. Full `--fail-on` policy comes in v0.4. Task 4 help text must not promise verdict behavior.

Status: agreed for Task 4

## D-006: Schema validation without extra dependencies

Decision for Task 7: implement `ValidateReceipt` in `internal/model/validate.go` with stdlib only (regexp plus enum switches) that mirrors `schemas/receipt.schema.json`. No JSON schema engine dependency added. go.mod still has zero requires.

Rationale: spec product goal 7 requires core small enough to audit as a single Go binary. A full Draft 2020-12 engine would add audit surface for Phase A. The published schema file remains the contract; the Go validator enforces the same required fields, patterns (^RC-, ^C-, ^E-, ^F-), enums, evidence ref resolution, duplicate and unique checks. Canonical nil to [] normalization is deferred to Phase B Task 9 store serializer. Revisit only if schema complexity outgrows manual checks.

Status: active for Phase A, revisit if needed in Phase B

## D-007: Render formats and CLI color scope for Tasks 12-14

Decision: show supports terminal, md (plus markdown alias), and json (via store canonical). Dedicated render/json.go, html.go, comment.go, svg.go deferred to Phase F/G per spec layout. Show terminal defaults Color false so piped and file output stay plain; renderer color API is implemented and tested, CLI TTY auto detection deferred to avoid a new x/term dependency. --raw prints a sensitive content warning to stderr; secret redaction for command tails lands with Phase F capture.

Status: active for Phase B

## D-008: Adapter read access without new dependencies

Decision: OpenCode adapter queries its SQLite store through the sqlite3 CLI in -json mode over a read only URI, reusing the controlled runner pattern from gitstate. No Go SQLite driver added; go.mod still has zero requires. Missing CLI produces an actionable error; Git evidence still flows. Claude adapter reads JSONL files directly with stdlib only.

Rationale: spec product goal 7 keeps the core small and auditable. A pure Go SQLite driver is heavy native free but large; the CLI is preinstalled on macOS and GitHub Linux runners and matches existing community practice. WAL single writer with short reads plus a 5 try retry keeps busy errors rare.

Limitations recorded: read only URI sees committed state only; in flight OpenCode writes stay invisible until commit. Channel specific database names (opencode-<channel>.db) are not probed; OPENCODE_DB override is honored. Windows paths use documented locations only, unverified locally.

Status: active for Phase D

## D-009: Registry placement versus spec layout

Decision: the generic Auto engine and Attempt types live in package adapters, while concrete wiring (opencode then claude) lives in subpackage registry. Spec Section 3 sketches registry.go inside the adapters package, but Go forbids a package importing its own subpackages, so the layout is adjusted without behavior change. Codex joins the registry after Phase G verification.

Status: active

## D-010: Dirty and untracked patch reconstruction deferred

Decision for Task 45: re-verification covers committed state only, labeled `committed_state_only` on every reverify evidence. Reconstructing dirty and untracked worktree state into a temp worktree cannot be proven safe in general (symlinks, partial stages, ignored files, concurrent edits), so it is explicitly deferred rather than half built. If a receipt lacks a committed final state, `check --reverify` refuses with an explanation.

Status: deferred, revisit only with a proven safe design plus safety tests

## D-011: Shim overhead versus the 15ms target

Decision: keep per event fsync for evidence durability. Measured on MacBook Pro 2019 i7-9750H darwin/amd64 with go1.27.1: shim median ~29.5ms, p95 ~34ms, direct exec median ~10ms, overhead median ~19.5ms. p95 meets the 40ms target; median misses 15ms. Profile attributes ~20ms per event append to fsync dominated file I/O, not matching logic. No unsubstantiated claims in README. Future work: a collector daemon with batched fsync.

Status: recorded

## D-012: LLM extractor shape

Decision: HTTP provider interface with a strict proposal contract (known type, existing assistant message, verbatim quote, exact fields), shared dedup and source mapping, UNVERIFIED verdicts only, in memory cache, no auth fields in v1 (local endpoints are the practical target). Remote endpoints always warn with the destination. Config wiring lands with Task 53; until then the extractor is unreachable from the CLI and off by default.

Status: active

## D-013: PNG export deferred

Decision for Task 55: SVG is the required portable badge; PNG is deferred. Go has no dependable stdlib text rasterization, and adding a font plus imaging dependency fails the auditability bar for a badge. Revisit only with a justified portable encoder.

Status: deferred

## Verification items (ASSUMPTION TO VERIFY)

### V-001: OpenCode transcript location and format

Spec refs: Sections 4.1, 17 Task 21.

Status: VERIFIED 2026-10-14 against three independent sources. No paths assumed from memory.

Evidence:

- Live local store: `~/.local/share/opencode/opencode.db` (SQLite, WAL mode with -shm/-wal present) plus `storage/`, `snapshot/`, `tool-output/`, `repos/`. Tables verified via immutable read only `.schema`: session(id, project_id, directory, title, time_created/updated integers, ...), message(id, session_id, time_created, data JSON), part(id, message_id, session_id, time_created, data JSON), project(id, worktree, ...), project_directory(project_id, directory).
- Live row shapes (keys only, values never recorded): message roles user and assistant; part types text, step-start, reasoning, tool, step-finish, patch, compaction, file. message.data keys role/time/agent/model; text part keys type/text; tool part keys type/tool/callID/state(status/input/output/metadata/title/time); times are epoch millisecond integers.
- Official docs (opencode.ai troubleshooting): database lives at `~/.local/share/opencode/opencode.db`, overridable via OPENCODE_DB.
- Open source schema (sst/opencode message-v2.ts): ToolPart with callID/tool/state(status pending/running/completed/error, input record, output, title), TextPart, ReasoningPart, step parts. Matches live shapes.
- Data dir resolution: `${XDG_DATA_HOME:-~/.local/share}/opencode`, OPENCODE_DB override, channel specific names. Project association via session.directory plus project_directory fallback.

Action: implemented in Task 22 against these shapes. Fixtures are synthetic rows in the verified query shapes; no private prompts or user content committed.

### V-002: Claude Code transcript location, format, project association

Spec refs: Sections 4.1, 17 Task 23.

Status: VERIFIED 2026-10-14 against local files plus community references.

Evidence:

- Live local sessions: `~/.claude/projects/-Users-jasonamadi/<uuid>.jsonl` (2 files). Slug rule confirmed: cwd `/Users/jasonamadi` maps to `-Users-jasonamadi` (slashes to dashes, leading dash).
- Live line shapes (keys and block type names only, values never recorded): user lines with message.content as str or list, assistant lines with message.content lists of text blocks, system lines with subtype, plus meta lines (mode, permission-mode, file-history-snapshot, last-prompt, cost-state, attachment). Timestamps are RFC 3339 (`2026-08-31T21:39:38.950Z`). Every line carries cwd, uuid, parentUuid, sessionId.
- Community references for tool shapes absent from the small local sample: tool_use blocks carry id/name/input and tool_result blocks carry tool_use_id/content as str or array (claude-dev.tools JSONL reference, opendray claude_jsonl.go, claude-code-transcripts crate). Parser handles both content forms and pairs results by id.
- No official format document found; treated as locally plus community verified, not vendor documented. Parser ignores unknown line and block types so schema drift cannot crash collection.

Action: implemented in Task 24. Fixture is a synthetic session.jsonl in the verified shapes; no private content committed.

### V-003: Shell invocation and PATH inheritance

Spec refs: Sections 7.3, 17 Task 42.

Status: PARTIALLY VERIFIED 2026-10-14.

Evidence:

- VERIFIED (probe test): wrapped children inherit the shim PATH with the run bin directory leading, and wrapped names resolve to shims (`TestRunPathPropagation` asserts both from a probe file).
- UNVERIFIED: per agent shell hooks. No experiment can show how OpenCode, Claude Code, or Codex configure child shells without running those agents against live backends, which needs network access and credentials outside this offline build. Until verified per agent, capture is documented as shim only on every tier 1 receipt, and bypasses stay disclosed rather than assumed absent.

Action: keep shim only disclosure; revisit with a live agent fixture when credentials and network are available.

### V-004: Codex CLI transcript format

Spec refs: Sections 4.1, 17 Task 51.

Status: PARTIALLY VERIFIED 2026-10-14 from code and community sources; no local install exists to confirm against (`codex` binary absent, `~/.codex/` holds only hooks.json).

Evidence:

- OpenAI codex-rs rollout recorder: `$CODEX_HOME/sessions` else `~/.codex/sessions`, sharded `YYYY/MM/DD/`, files `rollout-*.jsonl`, one JSON object per line with timestamp, type, payload.
- Community references agree the first line is `session_meta` with `payload.cwd` carrying the workdir, since paths encode dates, not projects. Project association needs a tree scan, unlike Claude slugs.
- The format is not formally documented and drifts between releases. No message payload shapes were committed to fixtures.

Action: adapter stays disabled per the v0.3 schedule. Fixtures plus parser land with the adapter only after a local install confirms current shapes. No code added in this pass.

### V-005: Module owner and repo name

See D-003. Unconfirmed.

Status: UNVERIFIED

### V-006: Receipts name availability

Spec refs: Sections 2 v0.4 acceptance, 17 Task 56, 18 Q8.

Status: CHECKED 2026-10-14 via registry and RDAP reads. The `receipts` name is taken on npm and the .dev domain. Do not claim availability. Do not rename internal code until the owner decides.

Results (method: live registry/RDAP lookups on 2026-10-14):

- npm `receipts`: TAKEN since 2015 (receipts/npm-receipts, Zenedith).
- npm `receipts-cli`: TAKEN since 2026-06 (Shaheer Shoaib, agent verification gates, actively maintained).
- npm `getreceipts`: TAKEN since 2026-06 (Ruxiu0409, Claude Code claim checker).
- npm `agent-receipts`: TAKEN since 2026-05 (agent-viscro, MCP audit trails).
- npm `rcpt-cli`: FREE (registry 404). Only fallback from the spec list still free on npm.
- GitHub `receipts/receipts`: no such repo (API 404). The `receipts` org account itself exists.
- Homebrew `receipts`: no formula (formulae.sh 404).
- receipts.dev: REGISTERED (Namecheap since 2023-12-09, Cloudflare nameservers, expires 2026-12-09).

Notes: three live agent verification tools already crowd receipts-like npm names, so confusion risk persists even for `rcpt-cli`. Homebrew availability ultimately needs an owner plus repo. Domain alternatives were not checked beyond receipts.dev. Recommendation for the owner: pick a fresh name (only `rcpt-cli` survives from the fallback list) before any public release, tag, or tap.

### V-007: PNG export

Spec refs: Sections 2 v0.4, 17 Task 55.

SVG required and deterministic with no scripts or external resources. PNG deferred unless dependable portable encoder justified. Record decision when reached.

Status: DEFERRED to Phase G

### V-008: Shim performance target

Spec refs: Section 7.5.

Target median under 15 ms, p95 under 40 ms per invocation on dev laptop, excluding command runtime. Must benchmark with hardware and Go version recorded. No unsubstantiated performance claims in README.

Status: DEFERRED to Phase F Task 48

## Privacy and safety notes (from spec Sections 13, 19)

- Store only claim text and excerpt hash by default, not whole transcript.
- Command tails bounded and redacted. No env vars stored.
- Secrets detector never writes raw matched values.
- Optional LLM extraction opt in only, with disclosure. Not in v0.1.
- Hash chain is tamper evident only if history not rewritten. Not a signature, not a security boundary.
- Collection and parse failures must appear in receipt warnings. No silent success.
- Absence of evidence is UNVERIFIED, not CONTRADICTED.
- Only agent messages create agent claims. Tool output alone is evidence, not a claim.
- No automatic repair, no unsafe cleanup, no reset or clean of user checkout.

## Change log

- 2026-10-14: file created. D-001 to D-005 recorded. V-001 to V-008 marked UNVERIFIED or deferred. Repo found non git, Go 1.27.1 recorded. Skill invocations recorded.
- 2026-10-14: D-006 recorded (stdlib only schema validation). Phase A Tasks 4 to 7 done.
- 2026-10-14: D-007 recorded (render scope, CLI color default). Phase B Tasks 8 to 14 done.
- 2026-10-14: Phase C Tasks 15 to 20 done (git runner, baseline, paths/diff, nongit, integrity). Transcript formats V-001/V-002 still UNVERIFIED; no adapter code written.
- 2026-10-14: V-001/V-002 VERIFIED (local immutable reads plus official and open source references). D-008 recorded (sqlite3 CLI read only, zero deps). Phase D Tasks 21 to 24 done.
- 2026-10-14: D-009 recorded (registry subpackage). Phase D Tasks 25 to 28 done (auto engine, safety bounds, claim patterns, dedup and conflicts).
- 2026-10-14: Phase E Tasks 29 to 36 done (verdict table, audit pipeline, init, demo goldens, labeled gates 1.000/1.000, README, readiness). Release blocked on Linux CI, license, name checks.
- 2026-10-14: V-003 partially verified (PATH probe test; agent hooks still unverified). D-010 recorded (dirty patch deferred). D-011 recorded (shim overhead 19.5ms vs 15ms target). Phase F Tasks 37 to 48 done.
- 2026-10-14: V-004 partially verified from code and community sources (no local install; adapter deferred to v0.3). D-012 recorded (LLM opt-in shape). Phase G Tasks 49 to 52 done (ci command, action, codex investigation, LLM extractor).
- 2026-10-15: V-006 checked (receipts name taken on npm and .dev; only rcpt-cli free). D-013 recorded (PNG deferred). Phase G Tasks 53 to 56 done (TOML config, policies, SVG badge, name checks).
- 2026-10-15: Phase G Tasks 57 to 59 done (release config validated on four arch builds, npm wrapper proven provisionally, acceptance evaluated). Build complete; release blocked on name, license, repo.
