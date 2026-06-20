# Feature Specification: [FEATURE NAME]

**Feature Branch**: `[###-feature-name]`
**Created**: [DATE]
**Status**: Draft
**Input**: User description: "$ARGUMENTS"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - [Brief Title] (Priority: P1)

[Describe the user journey in plain language. Focus on what the GM/player needs, not the implementation.]

**Why this priority**: [Explain why this is the smallest valuable slice.]

**Independent Test**: [Describe how this can be tested independently through service/domain behavior and, when relevant, Fyne UI flow.]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [expected outcome]
2. **Given** [initial state], **When** [action], **Then** [expected outcome]

---

### User Story 2 - [Brief Title] (Priority: P2)

[Describe the next independently valuable journey.]

**Why this priority**: [Explain the value.]

**Independent Test**: [Describe standalone verification.]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [expected outcome]

---

### User Story 3 - [Brief Title] (Priority: P3)

[Describe a later slice.]

**Why this priority**: [Explain the value.]

**Independent Test**: [Describe standalone verification.]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [expected outcome]

### Edge Cases

- [Validation or boundary condition, e.g. no active campaign, empty encounter, HP at zero, AP above cap]
- [Persistence or migration edge case, e.g. existing campaign character linked into an encounter]
- [UI input edge case, e.g. invalid number/date/body location]

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST [specific user-visible capability]
- **FR-002**: System MUST [specific domain/application behavior]
- **FR-003**: System MUST [specific persistence behavior, if data is involved]
- **FR-004**: System MUST [specific validation or failure behavior]
- **FR-005**: System MUST [specific UI behavior, if UI is involved]

### Fallout 2d20 Rules Impact *(include when relevant)*

- **Rules Affected**: [initiative, rounds, AP, GM Threat, HP, body locations, damage types, resistance, immunity, difficulty]
- **Rules Not Affected**: [explicitly state nearby mechanics that must remain unchanged]

### Data & Persistence Impact *(include when relevant)*

- **Entities/Tables**: [campaigns, players, player_characters, encounters, combatants, stat_profiles, resistance tables, monster_templates, encounter_logs]
- **Migration Required**: [yes/no; if yes, name the intended Goose migration]
- **sqlc Impact**: [queries/schema/dbgen updates required or not]
- **DB Docs Impact**: [docs/db regeneration required or not]

### UI Impact *(include when relevant)*

- **Screens/Dialogs**: [STAT/CAMP/DATA tabs, campaign editor, encounter editor, action dialogs, list dialogs]
- **Input Validation**: [fields and expected error behavior]
- **Refresh Behavior**: [what state must update after action]

### Key Entities *(include if feature involves data)*

- **[Entity]**: [What it represents and key relationships, not database implementation details unless necessary]

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: [Outcome that can be verified through tests or a concise manual workflow]
- **SC-002**: [Outcome for persistence, UI refresh, or generated artifacts where relevant]
- **SC-003**: [Performance/reliability/outcome metric appropriate for a local desktop app]

## Assumptions

- [Assumption about local desktop/offline behavior]
- [Assumption about active campaign or active encounter preconditions]
- [Assumption about storage/migration compatibility]

## Brownfield Notes *(for migrated or extension specs)*

- **Status**: [new|migrated]
- **Source Evidence**: [tests, source files, migrations, docs used to derive this spec]
- **Known Gaps**: [missing tests, documentation, UI behavior, or ambiguity discovered during migration]
