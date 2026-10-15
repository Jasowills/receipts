---
name: receipts
description: Verify completion claims against independent evidence before reporting done. Use after editing code, running tests, or changing dependencies.
---

# Receipts

Your agent says it's done. Here is the receipt.

Receipts checks completion claims against evidence collected
independently of the agent. The CLI decides every verdict from
evidence and fixed rules. This skill only teaches when to invoke it,
what counts as evidence, and how to respond. It never replaces a
check with an assertion.

## Core loop

1. Requirements: list what must be proven before claiming done.
   See `references/workflow.md`.
2. Checks: run the tests, builds, and linters that prove each item.
3. Evidence: capture receipts with `receipts audit` or `receipts run`.
   See `references/evidence.md`.
4. Verdicts: read VERIFIED, CONTRADICTED, UNVERIFIED, UNVERIFIABLE
   and respond to each. See `references/verdicts.md`.
5. Failures: fix the work or disclose the conflict, then re-check.
   Never report a failed or unchecked requirement as passed.

## Commands

```text
receipts audit                      # check the current project now
receipts run -- <agent command>     # wrap live work with command capture
receipts check                      # re-evaluate the latest receipt
receipts show [ID]                  # display a stored receipt
receipts demo                       # offline example, touches nothing
```

## Install

Copy `skills/receipts/` from the Receipts repository into one
discovery path. Details in `references/install.md`:

- OpenCode project: `.opencode/skills/receipts/`
- OpenCode global: `~/.config/opencode/skills/receipts/`
- Claude Code project: `.claude/skills/receipts/`
- Claude Code global: `~/.claude/skills/receipts/`
- Either harness also reads `.agents/skills/receipts/`

## Rules that are never optional

- A claim without evidence is UNVERIFIED, never a pass.
- A contradicted claim is never reported as done.
- Pasted output in chat is not evidence. Only recorded runs count.
- Full transcripts stay out of reports. Quote the receipt ID.
