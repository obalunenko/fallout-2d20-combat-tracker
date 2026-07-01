# Research: Existing Combat Tracker Baseline

## Decision: Preserve The Existing Layered Go Desktop Architecture

**Rationale**: The current dependency direction keeps Fallout 2d20 rules in `internal/domain`, use-case orchestration in `internal/app`, SQLite concerns in `internal/store/sqlite`, and Fyne UI code in `internal/ui/fyneui`. This lets domain and service behavior be tested without launching the desktop app.

**Alternatives considered**:

- Merge domain rules into UI handlers: rejected because it would make combat behavior harder to test and reuse.
- Introduce a separate backend/API service: rejected because the migrated baseline is explicitly local desktop/offline-first.

## Decision: Treat SQLite As The System Of Record

**Rationale**: The app is designed for a single GM workstation with local persistence. SQLite is already wired through `modernc.org/sqlite`, embedded Goose migrations, repository tests, and DB docs.

**Alternatives considered**:

- In-memory state only: rejected because restart persistence is a core baseline requirement.
- Network database or sync service: out of scope for the migrated baseline and conflicts with local-first assumptions.

## Decision: Keep Schema Evolution Migration-First

**Rationale**: Existing storage has a long Goose migration history, sqlc schema/query sources, generated `dbgen`, and `docs/db`. Future persistent changes should start with migrations and regenerate dependent artifacts.

**Alternatives considered**:

- Manual DB edits: rejected because they bypass reproducibility.
- Hand-edited generated sqlc code: rejected by constitution and generation workflow.

## Decision: Represent Combat Resistance As Normalized Profiles

**Rationale**: Physical, energy, and radiation resistance are location-based; poison is global-only. The existing schema and domain model encode these distinctions through dictionaries and row-based resistance profiles.

**Alternatives considered**:

- Wide columns for every future stat: rejected because the current normalization guidance prefers dictionary/row-based extensions.
- Single global resistance value for all types: rejected because body-location combat is part of the existing behavior.

## Decision: Treat Encounter Logs As Non-Critical Side Effects

**Rationale**: The application should not roll back saved combat/resource state simply because an audit log write failed. Existing service behavior records and logs side-effect failures without surfacing them as primary operation failures.

**Alternatives considered**:

- Make log writes transactional blockers: rejected because it would degrade live play reliability for non-critical feedback.
- Drop logs entirely: rejected because DATA tab visibility and audit trail are part of the current UI baseline.

## Decision: Use Requirements-Quality Checklists For Migrated Specs

**Rationale**: Brownfield specs are reverse-engineered from implementation. Checklists should expose ambiguity and missing requirement detail so future specs do not inherit vague language.

**Alternatives considered**:

- Use checklists as QA execution scripts: rejected by Spec Kit checklist guidance; quickstart covers runnable validation instead.

## Open Questions For Future Specs

- What exact user-facing documentation should explain encounter difficulty scoring?
- Should encounter logs have retention, export, or cleanup requirements?
- Should Fyne UI accessibility and keyboard navigation become explicit requirements?
- Should `.agents/` be partially ignored if future tools store local credentials there?
