# Verdicts: read them, then respond

Receipts verdicts use four words. Map your working language onto them
and never soften the result.

## The mapping

- PASS is VERIFIED. Direct supporting evidence exists: a matching
  command exited 0, a structured report shows success, or a reliable
  diff matches the stated paths. This verifies the recorded run or
  diff, not total correctness.
- FAIL is CONTRADICTED. Direct conflicting evidence exists: a
  nonzero exit, a failing report, manifest changes behind a denial,
  or a reliable diff outside the claimed set. Every contradiction
  cites its evidence.
- BLOCKED is UNVERIFIED. No supporting or conflicting evidence.
  Typical causes: no command was observed, the baseline is
  unreliable, or capture has known gaps. The fix is more evidence,
  not stronger words.
- INCONCLUSIVE is UNVERIFIABLE. No objective check exists for the
  claim, such as a generic quality judgment or an issue fix with no
  configured invariant.

Strictness settings change which checks run. They never change what
a verdict means.

## How to respond

- VERIFIED: report the requirement as met with the receipt ID.
- CONTRADICTED: stop. Fix the work or disclose the conflict with the
  evidence reference. Re-run the checks and capture a new receipt.
  Never report the requirement as met.
- UNVERIFIED: state what evidence is missing and how to get it
  ("no lint command observed; run the linter, then re-check").
  Never present it as a pass.
- UNVERIFIABLE: describe what was actually done without a verdict
  word ("changed the retry logic; no invariant covers the reported
  issue, so this stays unverifiable").

## Flags are not verdicts

Integrity flags (weakened tests, new skips, scope drift, secret
shaped additions) never flip a claim on their own. Show the flagged
diff when reporting; hiding it to stay green is itself evidence of
an unreliable run.
