package app

import (
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

// saveRaiderTemplate stores a raider template with torso DR and returns it.
func saveRaiderTemplate(t *testing.T, svc *Service) domain.Combatant {
	t.Helper()
	saved, err := svc.SaveMonsterTemplates(t.Context(), []domain.Combatant{{
		Name: "Raider", Level: 2, XP: 20, Initiative: 7, HP: 9, MaxHP: 9, Defense: 1,
		ResistPhysicalTorso: 1, ResistEnergyHead: 2, ImmunePoison: true,
	}})
	require.NoError(t, err)
	require.Len(t, saved, 1)
	return saved[0]
}

// partyMember returns the seeded campaign character as the desktop editor
// loads it ("Load Party From DB").
func partyMember(t *testing.T, svc *Service) domain.Combatant {
	t.Helper()
	members, err := svc.ListPartyMembers(t.Context())
	require.NoError(t, err)
	require.Len(t, members, 1)
	return members[0]
}

// rowCombatants copies template count times the way collectCombatantsFromRows
// (internal/ui/fyneui/input_row_collectors.go) builds NPC rows loaded from the
// monster library: no ID, side npc, the row's stats and resistance profile.
func rowCombatants(template domain.Combatant, count int) []domain.Combatant {
	out := make([]domain.Combatant, 0, count)
	for range count {
		c := domain.Combatant{
			Name: template.Name, Side: domain.SideNPC, Level: template.Level, XP: template.XP,
			Initiative: template.Initiative, HP: template.HP, MaxHP: template.MaxHP,
			Defense: template.Defense, TorsoOnly: template.TorsoOnly,
		}
		c.SetResistanceProfile(template.ResistanceProfile())
		out = append(out, c)
	}
	return out
}

// withoutIDs returns combatants with their generated IDs cleared, so two
// rosters built separately can be compared.
func withoutIDs(combatants []domain.Combatant) []domain.Combatant {
	out := append([]domain.Combatant(nil), combatants...)
	for i := range out {
		out[i].ID = ""
	}
	return out
}

func combatantByID(t *testing.T, enc *domain.Encounter, id string) domain.Combatant {
	t.Helper()
	for _, c := range enc.Combatants {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("combatant %s not in encounter %s", id, enc.ID)
	return domain.Combatant{}
}

func TestEncounterDraftExpandsQuantityLikeTheDesktopEditor(t *testing.T) {
	svc := newSQLiteService(t)
	raider := saveRaiderTemplate(t, svc)
	member := partyMember(t, svc)

	enc, err := svc.ExecuteCreateEncounterFromDraft(t.Context(), EncounterDraft{
		Name:              " Ambush ",
		PartyCharacterIDs: []string{"char-1"},
		Monsters:          []DraftMonster{{TemplateID: raider.ID, Quantity: 3}},
	})
	require.NoError(t, err)
	require.Equal(t, "Ambush", enc.Name)
	require.Len(t, enc.Combatants, 4)

	ids := map[string]bool{}
	for _, c := range enc.Combatants {
		require.NotEmpty(t, c.ID)
		ids[c.ID] = true
		if c.Side == domain.SideNPC {
			require.NotEqual(t, raider.ID, c.ID, "a template ID is not a combatant ID")
			require.Equal(t, raider.ResistanceProfile(), c.ResistanceProfile())
		}
	}
	require.Len(t, ids, 4, "every copy has its own generated ID")

	desktop, err := svc.ExecuteCreateEncounter(t.Context(), CreateEncounterCommand{
		Name:       "Ambush",
		Combatants: append([]domain.Combatant{member}, rowCombatants(raider, 3)...),
	})
	require.NoError(t, err)
	require.Equal(t, withoutIDs(desktop.Combatants), withoutIDs(enc.Combatants))
}

func TestEncounterDraftInitiativeOverride(t *testing.T) {
	svc := newSQLiteService(t)
	raider := saveRaiderTemplate(t, svc)
	initiative := 15

	enc, err := svc.ExecuteCreateEncounterFromDraft(t.Context(), EncounterDraft{
		Name: "Quick Raiders",
		Monsters: []DraftMonster{
			{TemplateID: raider.ID, Quantity: 2, Initiative: &initiative},
			{Monster: &domain.Combatant{Name: "Mirelurk", Level: 4, XP: 40, Initiative: 3, HP: 15, MaxHP: 15, TorsoOnly: true}, Quantity: 1},
		},
	})
	require.NoError(t, err)
	require.Len(t, enc.Combatants, 3)
	require.Equal(t, []int{15, 15, 3}, []int{enc.Combatants[0].Initiative, enc.Combatants[1].Initiative, enc.Combatants[2].Initiative})
	require.Equal(t, "Mirelurk", enc.Combatants[2].Name)
	require.True(t, enc.Combatants[2].TorsoOnly)
	require.Equal(t, domain.SideNPC, enc.Combatants[2].Side, "inline monsters are NPCs")
}

func TestPreviewEncounterDifficultyMatchesExpansionAndStoresNothing(t *testing.T) {
	svc := newSQLiteService(t)
	raider := saveRaiderTemplate(t, svc)
	member := partyMember(t, svc)
	mirelurk := domain.Combatant{Name: "Mirelurk", Level: 4, XP: 45, Initiative: 3, HP: 15, MaxHP: 15}
	draft := EncounterDraft{
		PartyCharacterIDs: []string{"char-1"},
		Monsters:          []DraftMonster{{TemplateID: raider.ID, Quantity: 3}, {Monster: &mirelurk, Quantity: 2}},
	}

	var events int
	unsubscribe := svc.SubscribeChanges(func(ChangeEvent) { events++ })
	t.Cleanup(unsubscribe)

	got, err := svc.PreviewEncounterDifficulty(WithSource(t.Context(), domain.SourceAssistant), draft)
	require.NoError(t, err)
	mirelurk.Side = domain.SideNPC
	want := domain.EvaluateEncounterDifficulty(append(append([]domain.Combatant{member}, rowCombatants(raider, 3)...), mirelurk, mirelurk))
	require.Equal(t, want, got)
	require.Equal(t, 150, got.TotalMonsterXP, "each copy counts")
	require.Equal(t, domain.EncounterDifficultyDeadly, got.Label)

	summaries, err := svc.ListEncounters(t.Context())
	require.NoError(t, err)
	require.Empty(t, summaries, "a preview stores nothing")
	require.Zero(t, events, "a preview publishes no change")

	draft.Name = "Saved"
	enc, err := svc.ExecuteCreateEncounterFromDraft(t.Context(), draft)
	require.NoError(t, err)
	require.Equal(t, got, domain.EvaluateEncounterDifficulty(enc.Combatants), "the saved roster has the previewed difficulty")
}

func TestUpdateEncounterFromDraftKeepsStoredState(t *testing.T) {
	svc := newSQLiteService(t)
	raider := saveRaiderTemplate(t, svc)
	ctx := WithSource(t.Context(), domain.SourceAssistant)
	enc, err := svc.ExecuteCreateEncounterFromDraft(ctx, EncounterDraft{
		Name:              "Ambush",
		PartyCharacterIDs: []string{"char-1"},
		Monsters:          []DraftMonster{{TemplateID: raider.ID, Quantity: 3}},
	})
	require.NoError(t, err)
	var raiders []string
	var partyID string
	for _, c := range enc.Combatants {
		if c.Side == domain.SideNPC {
			raiders = append(raiders, c.ID)
		} else {
			partyID = c.ID
		}
	}
	_, _, err = svc.ApplyDamage(t.Context(), raiders[0], domain.DamagePhysical, domain.BodyTorso, 20)
	require.NoError(t, err)
	_, _, err = svc.ApplyDamage(t.Context(), raiders[1], domain.DamagePhysical, domain.BodyTorso, 5)
	require.NoError(t, err)
	stored, err := svc.GetEncounterByID(t.Context(), enc.ID)
	require.NoError(t, err)
	revision := svc.Revision(encounterKey(enc.ID))
	var events []ChangeEvent
	unsubscribe := svc.SubscribeChanges(func(ev ChangeEvent) { events = append(events, ev) })
	t.Cleanup(unsubscribe)

	updated, err := svc.ExecuteUpdateEncounterFromDraft(ctx, enc.ID, EncounterDraft{
		KeepCombatantIDs: []string{raiders[1], partyID, raiders[0]},
		Monsters:         []DraftMonster{{Monster: &domain.Combatant{Name: "Mirelurk", Level: 4, XP: 40, Initiative: 3, HP: 15, MaxHP: 15}, Quantity: 1}},
	})
	require.NoError(t, err)
	require.Equal(t, "Ambush", updated.Name, "an empty name keeps the stored one")
	require.Len(t, updated.Combatants, 4, "the dropped raider is removed and the mirelurk added")
	for _, id := range []string{raiders[0], raiders[1], partyID} {
		want, got := combatantByID(t, stored, id), combatantByID(t, updated, id)
		require.Equal(t, want.HP, got.HP, id)
		require.Equal(t, want.Defeated, got.Defeated, id)
	}
	require.True(t, combatantByID(t, updated, raiders[0]).Defeated)
	require.Equal(t, 5, combatantByID(t, updated, raiders[1]).HP)

	require.Greater(t, svc.Revision(encounterKey(enc.ID)), revision, "the assistant update bumps the revision")
	require.Len(t, events, 1)
	require.Equal(t, domain.SourceAssistant, events[0].Source)
	require.Equal(t, enc.ID, events[0].EncounterID)

	renamed, err := svc.ExecuteUpdateEncounterFromDraft(t.Context(), enc.ID, EncounterDraft{Name: "Night Ambush", KeepCombatantIDs: []string{partyID}})
	require.NoError(t, err)
	require.Equal(t, "Night Ambush", renamed.Name)
	require.Len(t, renamed.Combatants, 1)
}

func TestEncounterDraftErrors(t *testing.T) {
	svc := newSQLiteService(t)
	raider := saveRaiderTemplate(t, svc)
	inline := &domain.Combatant{Name: "Mirelurk", Level: 4, XP: 40, Initiative: 3, HP: 15, MaxHP: 15}
	existing, err := svc.ExecuteCreateEncounterFromDraft(t.Context(), EncounterDraft{Name: "Existing", Monsters: []DraftMonster{{TemplateID: raider.ID, Quantity: 1}}})
	require.NoError(t, err)

	// The MCP adapter maps "not found" to not_found and validation wording
	// ("must", "required", "cannot", "duplicate") to invalid_input.
	for _, tc := range []struct {
		name     string
		draft    EncounterDraft
		mentions string
	}{
		{"unknown template", EncounterDraft{Monsters: []DraftMonster{{TemplateID: "missing", Quantity: 1}}}, `monster template "missing" not found`},
		{"unknown party character", EncounterDraft{PartyCharacterIDs: []string{"missing"}}, `party character "missing" not found`},
		{"zero quantity", EncounterDraft{Monsters: []DraftMonster{{TemplateID: raider.ID}}}, "quantity must be between 1 and 50"},
		{"quantity above 50", EncounterDraft{Monsters: []DraftMonster{{TemplateID: raider.ID, Quantity: 51}}}, "quantity must be between 1 and 50"},
		{"template and monster", EncounterDraft{Monsters: []DraftMonster{{TemplateID: raider.ID, Monster: inline, Quantity: 1}}}, "exactly one of a template ID or an inline monster is required"},
		{"neither template nor monster", EncounterDraft{Monsters: []DraftMonster{{Quantity: 1}}}, "exactly one of a template ID or an inline monster is required"},
		{"duplicate party character", EncounterDraft{PartyCharacterIDs: []string{"char-1", "char-1"}}, "duplicate party character"},
		{"kept combatant on create", EncounterDraft{KeepCombatantIDs: []string{"c-1"}}, `combatant "c-1" not found`},
		{"empty roster", EncounterDraft{}, "cannot create encounter without combatants"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.draft.Name = "Draft"
			_, err := svc.ExecuteCreateEncounterFromDraft(t.Context(), tc.draft)
			require.ErrorContains(t, err, tc.mentions)
			if tc.name != "empty roster" {
				_, err = svc.PreviewEncounterDifficulty(t.Context(), tc.draft)
				require.ErrorContains(t, err, tc.mentions, "the preview expands the same way")
			}
		})
	}
	_, err = svc.ExecuteCreateEncounterFromDraft(t.Context(), EncounterDraft{Monsters: []DraftMonster{{TemplateID: raider.ID, Quantity: 1}}})
	require.ErrorContains(t, err, "encounter name is required")

	_, err = svc.ExecuteUpdateEncounterFromDraft(t.Context(), existing.ID, EncounterDraft{KeepCombatantIDs: []string{"missing"}})
	require.ErrorContains(t, err, `combatant "missing" not found`)
	_, err = svc.ExecuteUpdateEncounterFromDraft(t.Context(), existing.ID, EncounterDraft{})
	require.ErrorContains(t, err, "cannot update encounter without combatants")
	_, err = svc.ExecuteUpdateEncounterFromDraft(t.Context(), "missing", EncounterDraft{Monsters: []DraftMonster{{TemplateID: raider.ID, Quantity: 1}}})
	require.ErrorIs(t, err, domain.ErrEncounterNotFound)
	_, err = svc.ExecuteUpdateEncounterFromDraft(t.Context(), existing.ID, EncounterDraft{
		KeepCombatantIDs:  []string{existing.Combatants[0].ID},
		PartyCharacterIDs: []string{"char-1"},
		Monsters:          []DraftMonster{{TemplateID: raider.ID, Quantity: 0}},
	})
	require.ErrorContains(t, err, "quantity must be between 1 and 50")

	summaries, err := svc.ListEncounters(t.Context())
	require.NoError(t, err)
	require.Len(t, summaries, 1, "failed drafts store nothing")
	stored, err := svc.GetEncounterByID(t.Context(), existing.ID)
	require.NoError(t, err)
	require.Equal(t, existing.Combatants, stored.Combatants, "failed updates change nothing")
}
