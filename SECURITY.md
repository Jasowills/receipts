# Security policy and limitations

Receipts is an evidence trail, not a security boundary.

## What Receipts does not promise

- It does not defeat a deliberately adversarial agent. An agent that
  bypasses command shims, rewrites history, or forges local state can
  produce misleading receipts. Fidelity tiers label collection strength
  so readers know what was actually observed.
- It does not defend a compromised operating system. Local attackers
  with write access to `.receipts/` can rewrite history. The hash
  chain is tamper evident only while stored history is intact. It is
  not a digital signature and proves nothing on its own.
- It does not validate secrets. Secret scanning reports shapes, never
  validity. Raw matched values are never written to receipts or logs.
- Re-verification runs project test commands, which execute arbitrary
  code from the checkout. Never re-verify untrusted code in a
  privileged environment.

## Data handling

- Transcripts are untrusted data. Receipts never executes tool calls
  found in a transcript and never alters agent session storage.
- Stored by default: claim text, source message id and role, excerpt
  hash, bounded command output tails, diff paths and hashes.
- Never stored: full transcripts, environment variables, raw secret
  values, telemetry (there is none).
- `receipts show --raw` may print sensitive content. Redact before
  sharing.

## Reporting

Report security relevant bugs through the project issue tracker with
a minimal reproduction. Do not include secrets, private transcripts,
or environment details in the report.
