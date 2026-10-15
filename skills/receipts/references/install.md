# Install: discovery paths per harness

The skill core (`SKILL.md` plus `references/`) is portable. Only the
destination directory changes per harness. Copy the whole
`skills/receipts/` directory, never just the entry file, so the
references travel with it.

## OpenCode

Verified against the official skills documentation. OpenCode
discovers `skills/<name>/SKILL.md` in these places:

- Project: `.opencode/skills/receipts/`
- Global: `~/.config/opencode/skills/receipts/`
- Claude compatible project: `.claude/skills/receipts/`
- Claude compatible global: `~/.claude/skills/receipts/`
- Agent compatible: `.agents/skills/receipts/` and
  `~/.agents/skills/receipts/`

```text
mkdir -p .opencode/skills
cp -r /path/to/receipts/skills/receipts .opencode/skills/
```

Frontmatter needs `name` (lowercase, matching the directory) and
`description` so the skill tool can advertise it.

## Claude Code

Claude Code reads the shared Agent Skills format from:

- Project: `.claude/skills/receipts/`
- Global: `~/.claude/skills/receipts/`

```text
mkdir -p .claude/skills
cp -r /path/to/receipts/skills/receipts .claude/skills/
```

The `.agents/skills/receipts/` path works for harnesses that follow
the same convention.

## Notes

- Keep one copy per scope. Project copies override global copies of
  the same name.
- The skill needs the `receipts` binary on PATH wherever the agent
  runs commands.
- Codex has no skill step yet. Its adapter stays off until transcript
  shapes are verified against a local install.
