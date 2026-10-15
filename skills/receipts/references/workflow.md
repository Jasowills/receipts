# Workflow: from requirements to honest reports

Follow these six steps for every piece of work. Skipping a step is
how false completions happen.

## 1. Before claiming completion: name requirements and evidence

Write down each requirement and what would prove it. A requirement
without a named proof is not ready to claim. Examples:

- "Tests pass" needs a recorded test run with exit code 0.
- "Only `src/auth.ts` changed" needs a reliable diff showing exactly
  that path.
- "No new dependencies" needs manifest diffs with no additions.

## 2. Run the checks

Invoke the real verifiers: the test command, the build, the linter,
or the configured check. Intent to run them is not running them.
"I will run tests" creates no claim and no evidence.

## 3. Capture evidence

Record actual command results and artifacts with Receipts:

- After the fact: `receipts audit` reads the transcript and Git state.
- Live: `receipts run -- <agent command>` wraps execution and records
  commands through PATH shims.

Keep raw outputs bounded and redacted. Never store secrets, full
transcripts, or environment variables in reports.

## 4. Evaluate every claim

Map each requirement to the verdict model in `verdicts.md`:

- PASS means VERIFIED. Report it.
- FAIL means CONTRADICTED. Fix or disclose, then re-check.
- BLOCKED or INCONCLUSIVE means UNVERIFIED or UNVERIFIABLE.
  Say what is missing, then go get it or state the limit.

## 5. Handle failures

Fix the work where appropriate, rerun the exact checks, and capture
a fresh receipt. The old receipt stays in history; never edit or
delete evidence to make a run pass. Weakening tests, adding skips,
or deleting failing files to get green is itself flagged.

## 6. Report honestly

State each requirement with its verdict and receipt ID. Never claim
a requirement passed when it was not checked or the result is
inconclusive. A short honest report beats a confident false one.
