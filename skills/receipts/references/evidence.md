# Evidence: what counts and what does not

Receipts derives every verdict from verifiable evidence plus fixed
rules. The skill helps collect that evidence. It grants none.

## What counts

- Command records: executable, arguments, working directory,
  timestamps, exit code, output hash, bounded output tail. Tier 1
  wraps capture PATH lookups; absolute paths and custom runtimes
  bypass it, and the receipt always discloses that limit.
- Git state: repository root, before and after commits, branch,
  changed paths, rename detection, diff hashes, manifest deltas.
- Re-verification: the configured test command replayed in an
  isolated temporary worktree at the final commit. Labeled
  committed state only. The original checkout is never touched.
- Transcript metadata: adapter name, session id, message counts.
  Used to find claims, never to prove them.

## What does not count

- The agent's own assertions in chat, no matter how detailed or
  confident. A pasted "ok, all passed" block is text, not a run.
- Tool output quoted from memory. Only recorded runs with exit codes
  verify test, build, and lint claims.
- A test run for a different claim. Test output never verifies lint;
  a build never verifies tests.
- Diff claims on an unreliable baseline. Audit mode without a
  snapshot reports file scope as UNVERIFIED, not contradicted.

## Tiers

- Tier 0 (audit): after the fact transcript plus Git reads.
  Pre-session uncommitted changes cannot be reconstructed.
- Tier 1 (live run): wrapped execution with a captured pre-run
  baseline, so file scope can contradict on a reliable diff.
- Higher tiers are reserved. The receipt always labels its tier.

## Handling gaps

When evidence is missing, say UNVERIFIED and name what would close
the gap. When capture may be incomplete, the receipt says so; repeat
that disclosure instead of asserting nothing ran.
