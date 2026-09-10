# go-hygiene-skills

A small, growing collection of [Claude Code Skills](https://docs.claude.com/en/docs/claude-code/skills)
for Go production-code hygiene. Each skill is a self-contained reference
guide an agent (or a human) can follow to find and fix a specific class of
problem in a Go codebase.

## Skills

| Skill | Use when |
|---|---|
| [`tracing-hygiene`](skills/tracing-hygiene/SKILL.md) | Auditing or fixing Go OpenTelemetry instrumentation — spans created per loop iteration (trace-size explosion), or spans that are never marked as errored. |

More skills will land here over time (this is the "first beachhead," not
the whole scope) — logging hygiene, metrics hygiene, and similar production
code review skills are natural next additions.

## Using a skill from this repo

### Option A: copy it in directly (works today, no auth needed)

This repo is private, so the marketplace flow below requires git credentials
for a private remote. The zero-friction path is to just copy the skill
directory you want:

```sh
git clone https://github.com/zalegrala/go-hygiene-skills.git
cp -r go-hygiene-skills/skills/tracing-hygiene ~/.claude/skills/tracing-hygiene
# or, to scope it to one project instead of your whole user config:
cp -r go-hygiene-skills/skills/tracing-hygiene <your-project>/.claude/skills/tracing-hygiene
```

Claude Code auto-discovers `SKILL.md` files placed under `~/.claude/skills/`
(all projects) or `<project>/.claude/skills/` (that project only, if
committed). No plugin prefix is needed for skills installed this way — it's
invoked as `/tracing-hygiene` or triggers automatically when the description
matches.

### Option B: install as a plugin (if you want the marketplace flow)

```sh
/plugin marketplace add zalegrala/go-hygiene-skills
/plugin install go-hygiene-skills@go-hygiene-skills
```

For a private repo, this requires git credentials configured for
`github.com/zalegrala/go-hygiene-skills`, e.g.:

```sh
git config --global url."https://x-access-token:YOUR_GITHUB_TOKEN@github.com/zalegrala/go-hygiene-skills".insteadOf \
  "https://github.com/zalegrala/go-hygiene-skills"
```

Skills installed this way are namespaced: `go-hygiene-skills:tracing-hygiene`.

## Adding a new skill to this repo

1. Create `skills/<skill-name>/SKILL.md` with YAML frontmatter (`name`,
   `description`) plus a `references/` subdirectory for anything too heavy
   to keep inline.
2. Follow the TDD-for-documentation process: run a baseline scenario without
   the skill, note what goes wrong, write the skill to address exactly
   that, then re-run to confirm it's fixed.
3. Add a row to the table above.
