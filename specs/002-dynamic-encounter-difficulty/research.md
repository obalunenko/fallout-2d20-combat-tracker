# Research: Dynamic Encounter Difficulty Calculator

## Decision: Keep Difficulty Calculation In Domain Code

**Rationale**: The constitution requires Fallout combat rules, including difficulty evaluation, to live in `internal/domain` before UI wiring. The user requested "client side" behavior to avoid saving on every quantity change; in this local desktop app that means recalculating from in-memory draft state without a SQLite write.

**Alternatives considered**:

- UI-only formula: rejected because it would make the Fyne layer the source of a combat rule and duplicate behavior used by saved encounter summaries.
- Application-service calculator: rejected for the initial phase because no orchestration, repository access, or side effect is needed.

## Decision: Replace The Existing Ratio-Based Labels

**Rationale**: The current baseline uses an XP-ratio model with Easy and Normal labels. The feature spec requires Trivial, Simple, Average, Hard, and Deadly based on encounter level minus average PC level. A single domain evaluator should produce the requested labels everywhere difficulty is displayed.

**Alternatives considered**:

- Add a second UI-only preview calculator while keeping saved summaries ratio-based: rejected because users would see different labels before and after saving the same encounter.
- Keep old labels as aliases: rejected because the spec explicitly says Easy and Normal must not appear in this calculator.

## Decision: Use Floating Division For XP Baseline, Then Floor Encounter Level

**Rationale**: The requested formula says to divide total monster XP by player count, subtract 10, divide by 10, and round down. Floating division preserves fractional baselines before the final floor and avoids hidden truncation at the first division step.

**Alternatives considered**:

- Integer division for XP baseline: rejected because it would add an unstated rounding step before the requested floor operation.

## Decision: Round Average PC Level Up To An Integer

**Rationale**: The feature explicitly requires average selected-player level to round up to the nearest whole number. The difficulty difference should compare integer encounter level against this rounded integer average PC level.

**Alternatives considered**:

- Display decimal average PC level and use it in the difference: rejected because the spec requires a whole-number rounded-up value before comparison.

## Decision: Treat Invalid Draft Input As Unavailable Preview State

**Rationale**: The spec requires invalid numeric draft fields to avoid misleading labels. The current preview collector can default invalid level or XP text to safe numbers; implementation should distinguish draft preview collection from save collection and surface Unknown/unavailable until required values are valid.

**Alternatives considered**:

- Continue defaulting invalid values to level 1 or XP 0: rejected because the displayed label could look valid while the draft contains invalid values.
- Reuse full save validation for preview: rejected because preview should remain responsive and tolerant while the GM is mid-editing unrelated required fields such as initiative or HP.

## Decision: Do Not Persist Difficulty Preview State

**Rationale**: The initial phase is reactive preview only. Difficulty can be derived from saved combatants whenever needed, so no new stored fields are necessary.

**Alternatives considered**:

- Store calculated difficulty on encounters: rejected because it creates derived state that can drift from combatants and would require migration/sqlc/docs work without user value for this phase.
