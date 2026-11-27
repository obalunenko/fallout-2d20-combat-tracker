# Project Instructions

## Read First

Read the constitution and the feature you are changing before changing code:

- `.specify/memory/constitution.md`: project governance; it overrides generic Spec Kit guidance.
- `specs/001-existing-combat-tracker-baseline/`: migrated baseline of the behavior that existed before Spec Kit.
- `specs/<NNN-feature-name>/spec.md`, `plan.md`, and `tasks.md` of the feature you are working on.

The section between the `SPECKIT` markers is managed by the Spec Kit agent-context extension, which rewrites it after specify and plan runs. Keep hand-written instructions outside the markers.

<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan
at specs/004-mcp-server/plan.md
<!-- SPECKIT END -->

## Claude Code configuration

- Project skills live in `.claude/skills/<name>/SKILL.md`. Spec Kit generates its skills there through the `claude` integration.
- Spec Kit and its extensions are pinned in the `Makefile`. `make speckit-install` installs the pinned Specify CLI into `tools/bin` and regenerates the skills; use `tools/bin/specify` for manual Spec Kit commands.
- Invoke Spec Kit skills as `/speckit-<name>`, for example `/speckit-plan` or `/speckit-bugfix-report`. Extension skills are rendered with dotted `/speckit.<ext>.<name>` ids; `make speckit-install` rewrites them automatically, and after any other Spec Kit regeneration run `make speckit-skills-normalize`.
- Do not edit the generated `speckit-*` skills by hand. Change the source in `.specify/` or `.speckit/` and run `make speckit-install`.

## Spec Kit workflows

New work:

```text
/speckit-specify -> /speckit-clarify -> /speckit-plan -> /speckit-tasks -> /speckit-analyze -> /speckit-implement
```

Existing-code adoption and drift checks: `/speckit-brownfield-scan`, `/speckit-brownfield-bootstrap`, `/speckit-brownfield-validate`, `/speckit-brownfield-migrate`.

Bugs in specified behavior:

```text
/speckit-bugfix-report -> /speckit-bugfix-patch -> /speckit-bugfix-verify -> /speckit-implement
```

- New features get the next `specs/NNN-feature-name` directory from the local feature-numbering extension's `before_specify` hook; use the directory it returns.
- SpecKit Companion records lifecycle progress in each feature's `.spec-context.json`; `/speckit-companion-status` and `/speckit-companion-resume` read it.

## Go style and quality

- Use `.claude/skills/go-code-quality-review/SKILL.md` for Go code creation, modification, simplification, and review. It applies [Google's Go Style Guide](https://google.github.io/styleguide/go/): the core guide is canonical, style decisions are normative, and best practices are advisory. These instructions, the constitution, and the configured linters still apply; where the guide leaves a choice, follow established package conventions.
- Use `.claude/skills/code-simplification-review/SKILL.md` on working, changed code before commit or merge to remove duplication, redundant state, wasted work, and band-aid fixes without changing behavior. It is not a correctness review.

## Project Boundaries

- `internal/domain`: Fallout 2d20 rules and pure domain behavior. Do not import application, store, UI, or Fyne packages here.
- `internal/app`: use-case services, commands, repository contracts, operation-log side effects, and context timeouts. This layer may depend on `internal/domain`, but not on Fyne or concrete SQLite details.
- `internal/store/sqlite`: SQLite repository implementation, migrations, sqlc schema/query files, generated db access, and schema tests. Do not push SQLite-specific concerns into `internal/domain`.
- `internal/ui/fyneui`: Fyne UI composition, dialogs, presenters, collectors, refresh behavior, formatting, and UI tests. UI should call `internal/app` services instead of reimplementing rules.
- `internal/mcpserver`: MCP adapter: tools, resources, skills, request admission, and the stdio bridge. It may depend on `internal/app` and `internal/domain`, but not on `internal/store` or `internal/ui`. It talks to the Fyne UI only through ports in `internal/app`.
- `cmd/fallout-tracker`: application assembly only.
- `docs/db`: generated database documentation; regenerate when schema changes.

## Commands

Use the narrowest command that verifies the change, then run broader gates when behavior crosses layers.

```bash
go test ./...
go vet ./...
make lint
make build
make sqlc-generate
make db-doc-generate
make goreleaser-check
make speckit-update-test
```

## Spec Discipline

- New user-visible behavior starts with a spec under `specs/`.
- Brownfield specs must include `status: migrated` and cite source/test evidence.
- Persistence specs must name migration, sqlc, dbgen, and docs impact.
- UI specs must name affected Fyne screens/dialogs and refresh behavior.
- Keep checked-in Spec Kit assets versioned together: `.claude/skills`, `.specify`, `.speckit`, and the active `specs/` feature directory should move as one set when commands regenerate them.
- Do not store secrets, tokens, connector credentials, local caches, or machine-specific state in `.claude/`, `.speckit/`, or `.specify/`; this project uses those directories for Spec Kit skills, templates, extension metadata, and project guidance only.
