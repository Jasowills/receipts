# Receipt RC-4F2A

- Agent: opencode
- Session: session-example
- Tier: 0
- Baseline confidence: low
- Created: 2026-10-14T12:00:00Z
- Duration: 840000ms

## Claims (2)

### C-001 TESTS\_PASS UNVERIFIED

> all tests pass

- Source: m-1 (assistant)
- Reason: No command evidence exists in audit mode.
- Evidence: none

### C-002 FILES\_CHANGED VERIFIED

> only touched src/auth

- Source: m-2 (assistant)
- Reason: Observed changed paths match the stated list.
- Evidence: E-001

## Flags (1)

### F-001 TEST\_ASSERTIONS\_REDUCED high

Assertion tokens reduced

tests/auth.test.ts: 8 -> 5 assertion tokens

- Paths: tests/auth.test.ts
- Evidence: E-002

## Evidence (2)

- E-001 diff 2026-10-14T12:00:00Z

- E-002 diff 2026-10-14T12:00:00Z

## Warnings (1)

- adapter issue: no transcript detected

## Summary

- Verified: 1
- Contradicted: 0
- Unverified: 1
- Unverifiable: 0
- Flags: 1
