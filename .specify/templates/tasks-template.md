---
description: "Task list template for Fallout 2d20 Combat Tracker feature implementation"
---

# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: `spec.md` and `plan.md`
**Tests**: Include focused Go tests for every changed behavior unless the spec explicitly documents why manual verification is sufficient.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and has no dependency on another open task.
- **[Story]**: Maps to a user story from `spec.md`.
- Include exact file paths.
- For migrated brownfield specs, completed work is marked `[x]` and gaps remain unchecked.

## Phase 1: Domain and Application Rules

**Purpose**: Keep Fallout mechanics and use-case behavior out of UI-specific code.

- [ ] T001 [US1] Add/update domain behavior in `internal/domain/`
- [ ] T002 [P] [US1] Add/update domain tests in `internal/domain/*_test.go`
- [ ] T003 [US1] Add/update application service command handling in `internal/app/`
- [ ] T004 [P] [US1] Add/update service tests in `internal/app/*_test.go`

## Phase 2: Persistence and Generated Data Access

**Purpose**: Keep SQLite state, migrations, and generated access aligned.

- [ ] T005 [US1] Add Goose migration in `internal/store/sqlite/migrations/` if schema changes
- [ ] T006 [US1] Update `internal/store/sqlite/sqlc/schema.sql` and `internal/store/sqlite/sqlc/query.sql` if queries change
- [ ] T007 [US1] Regenerate `internal/store/sqlite/dbgen/` with `make sqlc-generate` if sqlc files change
- [ ] T008 [P] [US1] Add/update repository and schema tests in `internal/store/sqlite/*_test.go`
- [ ] T009 [P] [US1] Regenerate `docs/db` with `make db-doc-generate` if schema changes

## Phase 3: Fyne UI Workflow

**Purpose**: Wire service behavior into the desktop app without duplicating domain rules.

- [ ] T010 [US1] Update Fyne dialog/view/presenter code in `internal/ui/fyneui/`
- [ ] T011 [P] [US1] Add/update UI collection, formatting, presenter, or controller tests in `internal/ui/fyneui/*_test.go`
- [ ] T012 [US1] Verify refresh behavior after state-changing actions

## Phase 4: Documentation and Spec Artifacts

- [ ] T013 [P] Update README or docs when user-visible commands, storage, or workflows change
- [ ] T014 [P] Update this feature's `spec.md`/`plan.md` when implementation reveals a corrected requirement
- [ ] T015 Record any follow-up gaps as unchecked tasks or a follow-up spec

## Phase 5: Verification

- [ ] T016 Run focused package tests for touched areas
- [ ] T017 Run `go test ./...`
- [ ] T018 Run `go vet ./...` when code changes affect compiled packages
- [ ] T019 Run `make lint` before completion when available
- [ ] T020 Run `make build` before completion for user-facing app changes

## Dependencies & Execution Order

- Domain/application behavior should land before UI wiring.
- Persistence migrations and sqlc generation must land before repository tests that depend on them.
- UI tests should use existing presenters/collectors/controllers where possible instead of launching the full app.
- Schema changes require `make sqlc-generate`, repository tests, and DB docs regeneration.

## Parallel Opportunities

- Domain tests and UI formatting tests can often run in parallel after requirements are clear.
- Repository tests can run in parallel with UI tests once service contracts are stable.
- Documentation and spec updates can run in parallel with final verification.
