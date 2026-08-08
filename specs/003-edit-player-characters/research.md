# Research: Edit Campaign Player Characters

## Decision: Keep Character-Only Details Out Of Combatant

**Rationale**: Notes and S.P.E.C.I.A.L. belong to the campaign player character and the spec explicitly excludes duplicating them into encounter combatants. A dedicated S.P.E.C.I.A.L. value object carried by campaign player data prevents unrelated NPC/monster and combat persistence paths from acquiring unused fields.

**Alternatives considered**:

- Add notes and seven attributes to `Combatant`: rejected because every NPC, monster template, and encounter copy would inherit character-only data and persistence obligations.
- Store the fields only in UI state: rejected because values must survive restarts and be available anywhere campaign characters are listed or edited.

## Decision: Normalize S.P.E.C.I.A.L. With A Dictionary And Value Rows

**Rationale**: The constitution says new combat stat dimensions should prefer dictionary/row-based storage over wide columns. Stable attribute codes plus one value row per player character and attribute provide validation, extensibility, and consistent normalized documentation. Notes remain a player-character column because they are a single character property rather than a repeated stat dimension.

**Alternatives considered**:

- Seven columns on `player_characters`: rejected because it creates another wide stat representation contrary to the constitution.
- Seven columns on `stat_profiles`: rejected because S.P.E.C.I.A.L. is not required on encounter or monster stat profiles and would be duplicated into snapshots.
- A JSON object: rejected because individual values would lose relational constraints and query/type generation.

## Decision: Backfill And Default Every Attribute To 1

**Rationale**: The spec requires positive values and no unknown state. Migration `00043` can seed every existing player character with all seven rows at value 1, while new campaign characters use the same defaults until edited. This keeps old databases and the current create-campaign flow valid immediately after upgrade.

**Alternatives considered**:

- Allow missing or zero values: rejected because it violates the specified minimum and makes edit validation inconsistent with migrated records.
- Block until every existing character is manually completed: rejected because it would make existing campaigns unusable after migration.
- Infer values from other combat stats: rejected because there is no reliable rule connecting the current data to S.P.E.C.I.A.L.

## Decision: Extend The Existing Campaign Editor With Expandable Details

**Rationale**: The campaign editor already loads, validates, and saves player-character level, HP, Defense, and DR in one transaction. An expandable Character Details section per row can add notes and S.P.E.C.I.A.L. without crowding the roster table or creating a second save lifecycle. Existing create/edit behavior, Cancel semantics, and refresh callbacks remain the user-facing foundation.

**Alternatives considered**:

- Add all fields as permanent roster columns: rejected because seven attributes plus multiline notes would make the desktop dialog difficult to scan and use.
- Add a separate standalone character-management screen: rejected because it duplicates campaign selection, save, error, and refresh behavior for the requested scope.
- Save each field immediately: rejected because the spec requires atomic save and Cancel without persistence.

## Decision: Synchronize Active Combat In The Campaign Transaction

**Rationale**: A campaign edit and its linked active-combatant update must either both succeed or both roll back. The SQLite repository already wraps campaign updates in a transaction and defines the effective active encounter as the most recently updated non-deleted encounter. Synchronization therefore belongs in that repository transaction after player-character validation and persistence, and only when the edited campaign is the application’s active campaign.

**Alternatives considered**:

- Make a second service call after campaign save: rejected because the campaign could commit while encounter synchronization fails.
- Synchronize every encounter in the campaign: rejected because the spec preserves closed encounters as historical snapshots.
- Put transaction logic in the UI: rejected because it violates layer boundaries and is not reliably testable.

## Decision: Make Linked Encounter Reads Snapshot-Based

**Rationale**: Current encounter queries substitute the latest player-character scalar and DR profile for every linked party combatant. That behavior would make closed encounters change when campaign data changes, contrary to the feature contract. Encounter reads should use the combatant’s stored stat profile; campaign values are copied when a character first enters an encounter, active copies are updated by campaign edits, and encounter saves may continue synchronizing their explicitly edited linked party state back to the campaign.

**Alternatives considered**:

- Keep live substitution and claim closed rows are not physically rewritten: rejected because users would still observe historical encounters changing.
- Add a second historical-only read path: rejected because identical encounter IDs should not produce different combat values depending on caller.

## Decision: Preserve Notes Exactly And Preserve Existing Immunity Values

**Rationale**: Notes may contain intentional whitespace and line breaks, so the application and repository must not trim them. DR changes reuse the existing resistance profile and inputs, which retain immunity state even though expanding immunity behavior is outside the feature scope.

**Alternatives considered**:

- Trim notes like names: rejected because it loses user-authored formatting.
- Recreate DR using only numeric values: rejected because it could silently clear existing immunity settings.

## Decision: Verify Migration, Atomicity, And UI State Separately

**Rationale**: The highest risks occur at distinct boundaries: backfilling existing databases, rolling back multi-record edits, preserving closed snapshots, and retaining unsaved UI values after validation errors. Focused tests at domain, application, repository/schema, and UI helper levels provide faster diagnosis before full repository gates.

**Alternatives considered**:

- Rely only on a manual desktop smoke test: rejected because rollback and migration edge cases are difficult to reproduce reliably by hand.
- Rely only on repository tests: rejected because widget collection and refresh behavior could regress independently of persistence.
