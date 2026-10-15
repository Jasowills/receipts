# Receipts

Your agent says it's done. Here's the receipt.

Receipts checks a coding agent's completion claims against evidence
collected independently of that agent. Run `receipts audit` after a
session: it reads the agent transcript, captures Git state, extracts
claims, runs integrity checks, and prints which claims are verified,
contradicted, unverified, or unverifiable. Every finding cites evidence.

```
RECEIPT RC-DE07  demo  tier 0
Baseline confidence: HIGH
Created: 2026-10-14T12:00:00Z
Duration: 1000ms

CLAIMS
  C-001 "All tests pass"  VERIFIED
    npm test recorded run passed exit 0 tier 0 (E-003)
    Evidence: E-003
  C-002 "The build passed"  CONTRADICTED
    npm run build recorded run failed exit 1 tier 0 (E-004)
    Evidence: E-004
  C-003 "Lint is clean"  UNVERIFIED
    no matching command evidence in audit mode; execution cannot be reconstructed from diff and transcript alone

Summary: 1 verified, 1 contradicted, 1 unverified, 0 unverifiable, 4 flags
```

The example above is the deterministic offline demo (`receipts demo`).
No network, no API key, and it never touches your repository.

## Install

Requirements: Go 1.27 or later, Git, and (for OpenCode transcripts)
the `sqlite3` command line tool.

Build from source:

```text
git clone https://github.com/Jasowills/receipts.git
cd receipts
go build -o receipts ./cmd/receipts
```

Pinned binaries, `go install`, Homebrew, and npm distribution land
before public release. The Go module path is still a placeholder until
the owner confirms it.

## Quick start

Audit the current project after an agent session:

```text
receipts audit
```

Show a stored receipt as Markdown:

```text
receipts show --md
receipts show RC-4F2A --format json
```

Run the offline demo (four scenarios, temp directory only):

```text
receipts demo
receipts demo --scenario scope-drift
receipts demo --keep
```

Write a starter config and gitignore entry:

```text
receipts init
```

Wrap a live agent session with command capture (tier 1):

```text
receipts run -- opencode
```

Re-evaluate a receipt, optionally re-running tests in an isolated
worktree without touching the checkout:

```text
receipts check
receipts check RC-4F2A --reverify --command "go test ./..."
```

Print a PR comment body from a stored receipt (no network; the Action
posts it):

```text
receipts ci
```

`audit`, `run`, `check`, `show`, `init`, `demo`, and `ci` are all
implemented. A `receipts.toml` file in the project applies scope and
check policy with visible receipt notes; see the configuration example
below.

## Commands

```text
receipts audit [--workdir PATH] [--agent auto|opencode|claude] [--format terminal|md|json|html] [--out PATH]
receipts run [--workdir PATH] [--agent ...] [--no-shims] [--quiet] [--] <agent command> [args...]
receipts check [ID] [--reverify] [--command CMD] [--force] [--workdir PATH] [--format ...] [--out PATH]
receipts show [ID] [--format terminal|md|json|html|svg] [--out PATH] [--raw]
receipts demo [--scenario default|duplicate-claims|test-weakening|scope-drift] [--format ...] [--out PATH] [--keep]
receipts init [--workdir PATH] [--force]
receipts ci [ID] [--format comment|md|json] [--out PATH] [--base-url URL]
```

Exit codes: 0 means the tool ran (never that every claim passed);
2 is usage or config error; 3 is a collection failure; 4 is a store
or schema failure; 5 is an unsafe re-verification; 130 is interrupt.

## Verdicts

- VERIFIED: direct supporting evidence exists (for example a matching
  command that exited 0, or a diff that matches the stated paths).
  This verifies the recorded run or diff, not total correctness.
- CONTRADICTED: direct conflicting evidence exists (a failing command,
  or a reliable diff outside the claimed set). Every contradiction
  cites evidence.
- UNVERIFIED: no supporting or conflicting evidence. Absence of a
  command record is never proof the command never ran, so audit mode
  reports UNVERIFIED rather than CONTRADICTED.
- UNVERIFIABLE: no objective check exists (generic quality claims and
  issue fixed claims without a configured invariant).

Strictness settings change which checks run, never what a verdict
means.

## Integrity flags

Flags are observations, not verdicts. A flag never automatically
contradicts a claim.

- TEST_FILE_MODIFIED / TEST_FILE_DELETED: test files changed or
  removed. Deletions are high severity.
- ASSERTION_TOKEN_COUNT_REDUCED: assertion-like token counts fell in
  changed test files. Wording reports counts only.
- SKIP_ADDED: new skip markers (skip, xfail, ignore, disabled).
- OUT_OF_SCOPE_CHANGE / FORBIDDEN_PATH_CHANGED: paths outside the
  allowed scope or inside forbidden paths. No scope configured means
  no scope flags.
- DEPENDENCY_ADDED / DEPENDENCY_REMOVED / DEPENDENCY_VERSION_CHANGED:
  manifest dependency deltas. LOCKFILE_CHANGED covers lockfiles and
  unparsed manifests.
- BUILD_CONFIG_CHANGED / CI_CONFIG_CHANGED / LINTER_CONFIG_CHANGED:
  build, CI, and linter configuration edits.
- SECRET_LIKE_VALUE_ADDED: token shaped additions on added lines.
  Raw values are never stored. High severity for private keys and
  known token prefixes.
- BASELINE_UNCERTAIN: the before state is not fully known.
- CONFLICTING_AGENT_CLAIMS: the transcript states incompatible file
  lists.

## Fidelity tiers and limitations

- Tier 0 (audit): reads transcripts and Git state after the fact.
  Pre-session uncommitted changes cannot be reconstructed, so the
  baseline confidence is low and file claims stay UNVERIFIED unless a
  reliable baseline exists. Absence of a command record proves nothing.
- Tier 1 (live run): wraps the agent with PATH shims that record
  commands, with a captured pre-run baseline. Shims can be bypassed
  (absolute paths, custom runtimes), so receipts always disclose
  capture limits instead of claiming completeness.
- Tier 2 (sandboxed execution, reserved): future work.

Receipts is not a security boundary against a malicious agent or a
compromised machine. See SECURITY.md.

## Supported agents

| Agent      | Status   | Transcript source                              |
|------------|----------|-----------------------------------------------|
| OpenCode   | Verified | `~/.local/share/opencode/opencode.db` (SQLite, read only) |
| Claude Code| Verified | `~/.claude/projects/<slug>/<session>.jsonl`  |
| Codex      | Planned  | Format verification required first            |

Both formats were verified against current local installs and
authoritative references; details live in DECISIONS.md. Transcripts
are read only input and never executed.

## Privacy and local-first behavior

- Core workflows are offline. No network calls, no telemetry, no
  accounts, no hosted services.
- Receipts store claim text plus a hash of the source excerpt, never
  full transcripts.
- Command output tails are bounded (8 KiB per stream) and redacted
  before storage; hashes cover the raw bytes. Environment variables
  are never stored.
- Secret scanning is local regex on added lines only. Matches store
  pattern name, path, line, and hash, never the value.
- Optional LLM extraction is opt in only, local endpoint first, and
  can only propose claims that still validate against exact transcript
  messages. It cannot create evidence or set verdicts.

## Configuration example

`receipts init` writes this starter file. All keys are optional.

```toml
agent = "auto" # auto | opencode | claude | codex
strictness = "conservative" # conservative | balanced | strict

[test]
command = "npm test"
timeout_seconds = 900

[scope]
allowed_paths = ["src/**", "tests/**"]
forbidden_paths = [".github/**", "package.json", "package-lock.json"]

[checks]
test_files_changed = true
assertions_removed = true
skips_added = true
out_of_scope_changes = true
dependencies_changed = true
build_config_changed = true
secret_patterns = true

[store]
directory = ".receipts"
retain_output_tail_bytes = 8192

[llm]
enabled = false
provider = "local"
model = ""
endpoint = "http://127.0.0.1:11434"
```

TOML loading validates strictly: unknown keys error to catch typos,
and invalid values fail with field specific errors and a nonzero exit.
Policy choices appear as visible receipt notes.

## CI and GitHub Action

A composite action lives in `action/`. It installs a pinned release
(checksum verified), runs the audit, and updates one PR comment found
by the `<!-- receipts-report -->` marker. Point `version` at a real
release and replace OWNER with the repository owner.

```yaml
permissions:
  contents: read
  pull-requests: write # only for comment posting

steps:
  - uses: actions/checkout@v4
  - uses: ./action # or OWNER/receipts/action@<sha>
    with:
      version: 0.1.0
      fail-on: never # never|contradictions|findings
```

Fork pull requests without a write token get the report as an
artifact with an explanation instead of a comment.

## Troubleshooting

- `adapter issue: ...` in warnings means no transcript was found.
  The receipt still carries Git checks with zero claims. Check the
  agent name (`--agent`), the project directory (`--workdir`), and
  that the agent actually ran there.
- `opencode: sqlite3 CLI not found` means install sqlite3 to read
  OpenCode sessions. Git checks are unaffected.
- `store: lock busy` means another Receipts process holds
  `.receipts/.lock`. Remove it only when no Receipts process runs.
- `NOT a git repository` behavior: outside Git you still get a valid
  receipt with `NON_GIT_PROJECT` and no diff checks.
- `check: reverify: refusing` means the checkout is dirty or has no
  HEAD. Pass `--force` to test the committed state anyway; results
  are labeled committed state only and never touch your checkout.

## Development

```text
go build ./...            # compile everything
go test ./...             # full suite (also runs golden and dataset gates)
go test -race ./...       # race detector
gofmt -l .                # must print nothing
go vet ./...              # must pass
go test -run xxx -bench ShimOverhead ./internal/collectors/shell/  # shim benchmark
```

Layout follows the spec: `cmd/receipts` is a thin entry point,
`internal/` holds `cli`, `config`, `model`, `gitstate`, `collectors`
(including `shell` shims), `adapters` (plus `registry`), `claims`,
`verdict`, `integrity`, `render`, `store`, `reverify`, and `demo`.
`schemas/` publishes the receipt and config JSON schemas, `action/`
the GitHub Action, `testdata/` the golden files and labeled dataset.
`IMPLEMENTATION_PLAN.md` is the build log; `DECISIONS.md` records
every verification source and open question.

## Contributing and security

Bug reports and small focused changes are welcome. Keep changes
reviewable: one task, tests, updated plan. Never commit private
transcripts, secrets, or user content in fixtures. Security notes and
limitation disclosures live in SECURITY.md.
