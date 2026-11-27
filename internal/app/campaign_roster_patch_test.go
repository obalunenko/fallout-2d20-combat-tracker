package app

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite"
	"github.com/stretchr/testify/require"
)

// rosterPlayers returns the stored roster of the seeded test campaign.
func rosterPlayers(t *testing.T, svc *Service) []domain.NewCampaignPlayer {
	t.Helper()
	players, err := svc.ListCampaignPlayers(t.Context(), "test-campaign")
	require.NoError(t, err)
	return players
}

func addSecondPlayer(t *testing.T, svc *Service) domain.NewCampaignPlayer {
	t.Helper()
	res, err := svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{
		CampaignID: "test-campaign",
		AddPlayer: &domain.NewCampaignPlayer{PlayerName: "Player 2", Character: domain.Combatant{
			Name: "Ranger", Level: 2, Initiative: 8, HP: 9, MaxHP: 9, Defense: 1,
		}},
	})
	require.NoError(t, err)
	require.NotNil(t, res.Player)
	return *res.Player
}

// partyEncounter creates an encounter in the active campaign with the given
// character and one NPC.
func partyEncounter(t *testing.T, svc *Service, name string, p domain.NewCampaignPlayer) *domain.Encounter {
	t.Helper()
	c := p.Character
	c.ID, c.PlayerCharacterID, c.Side = "", p.Character.ID, domain.SideParty
	enc, err := svc.CreateEncounter(t.Context(), "", name, []domain.Combatant{
		c,
		{Name: "Raider", Side: domain.SideNPC, Level: 1, XP: 10, Initiative: 5, HP: 5, MaxHP: 5},
	})
	require.NoError(t, err)
	return enc
}

func hasCharacter(t *testing.T, svc *Service, encounterID, characterID string) bool {
	t.Helper()
	enc, err := svc.GetEncounterByID(t.Context(), encounterID)
	require.NoError(t, err)
	for _, c := range enc.Combatants {
		if c.PlayerCharacterID == characterID {
			return true
		}
	}
	return false
}

func TestCampaignRosterPatchDetails(t *testing.T) {
	svc := newSQLiteService(t)
	before := rosterPlayers(t, svc)

	name := "Renamed Campaign"
	res, err := svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{CampaignID: "test-campaign", Details: &CampaignDetailsPatch{Name: &name}})
	require.NoError(t, err)
	require.Equal(t, name, res.Campaign.Name)
	require.Equal(t, testCampaignStartDate(t), res.Campaign.StartDate, "an unset start date is kept")
	require.Nil(t, res.Player)

	start, err := domain.ParseCampaignStartDate("2027-03-04")
	require.NoError(t, err)
	res, err = svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{CampaignID: "test-campaign", Details: &CampaignDetailsPatch{StartDate: &start}})
	require.NoError(t, err)
	require.Equal(t, name, res.Campaign.Name, "an unset name is kept")
	require.True(t, start.Equal(res.Campaign.StartDate))
	require.Equal(t, before, rosterPlayers(t, svc), "the roster is unchanged")
}

func TestCampaignRosterPatchAddPlayer(t *testing.T) {
	svc := newSQLiteService(t)
	added := addSecondPlayer(t, svc)
	require.NotEmpty(t, added.Character.ID)
	require.Equal(t, "Ranger", added.Character.Name)
	require.Equal(t, domain.DefaultSpecialValues(), added.Special)
	require.Len(t, rosterPlayers(t, svc), 2)

	_, err := svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{
		CampaignID: "test-campaign",
		AddPlayer:  &domain.NewCampaignPlayer{PlayerName: "  player 2 ", Character: domain.Combatant{Name: "Other", Level: 1, HP: 3, MaxHP: 3}},
	})
	require.ErrorContains(t, err, "duplicate player name")
	require.Len(t, rosterPlayers(t, svc), 2, "a duplicate is not stored")
}

func TestCampaignRosterPatchUpdateCharacter(t *testing.T) {
	svc := newSQLiteService(t)
	special := domain.SpecialValues{Strength: 6, Perception: 5, Endurance: 7, Charisma: 4, Intelligence: 8, Agility: 6, Luck: 3}

	res, err := svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{
		CampaignID: "test-campaign",
		UpdateCharacter: &CharacterPatch{CharacterID: "char-1", Edit: func(p *domain.NewCampaignPlayer) {
			p.Character.Level, p.Character.MaxHP, p.Character.HP = 4, 12, 10
			p.Character.ResistPhysicalTorso, p.Character.ImmunePoison = 2, true
			p.Special, p.Notes = special, "Line 1\nLine 2"
			p.PlayerName, p.Inactive = "Hijacked", true // ignored
		}},
	})
	require.NoError(t, err)
	require.Empty(t, res.ReplacedCharacterID)
	got := *res.Player
	require.Equal(t, "char-1", got.Character.ID, "no rename keeps the ID")
	require.Equal(t, "Player 1", got.PlayerName)
	require.False(t, got.Inactive)
	require.Equal(t, 4, got.Character.Level)
	require.Equal(t, 10, got.Character.HP)
	require.Equal(t, 12, got.Character.MaxHP)
	require.Equal(t, 2, got.Character.ResistPhysicalTorso)
	require.True(t, got.Character.ImmunePoison)
	require.Equal(t, special, got.Special)
	require.Equal(t, "Line 1\nLine 2", got.Notes)
	require.Equal(t, []domain.NewCampaignPlayer{got}, rosterPlayers(t, svc))

	res, err = svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{
		CampaignID:      "test-campaign",
		UpdateCharacter: &CharacterPatch{CharacterID: "char-1", Edit: func(p *domain.NewCampaignPlayer) { p.Character.Name = "Lone Wanderer" }},
	})
	require.NoError(t, err)
	require.Equal(t, "char-1", res.ReplacedCharacterID)
	require.NotEqual(t, "char-1", res.Player.Character.ID, "a rename creates a new character")
	require.Equal(t, "Lone Wanderer", res.Player.Character.Name)
	require.Equal(t, 4, res.Player.Character.Level, "the other fields carry over")
	players := rosterPlayers(t, svc)
	require.Len(t, players, 1)
	require.Equal(t, res.Player.Character.ID, players[0].Character.ID)

	_, err = svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{
		CampaignID:      "test-campaign",
		UpdateCharacter: &CharacterPatch{CharacterID: res.Player.Character.ID, Edit: func(p *domain.NewCampaignPlayer) { p.Character.HP = 99 }},
	})
	require.ErrorContains(t, err, "current HP cannot exceed max HP")
	require.Equal(t, players, rosterPlayers(t, svc), "an invalid edit stores nothing")
}

func TestCampaignRosterPatchAvailability(t *testing.T) {
	svc := newSQLiteService(t)
	first := rosterPlayers(t, svc)[0]
	enc1 := partyEncounter(t, svc, "First", first)
	enc2 := partyEncounter(t, svc, "Second", first)

	res, err := svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{
		CampaignID:      "test-campaign",
		SetAvailability: &CharacterAvailabilityPatch{CharacterID: "char-1", Available: false},
	})
	require.NoError(t, err)
	require.True(t, res.Player.Inactive)
	require.False(t, hasCharacter(t, svc, enc1.ID, "char-1"), "an unavailable character leaves every encounter")
	require.False(t, hasCharacter(t, svc, enc2.ID, "char-1"))

	res, err = svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{
		CampaignID:      "test-campaign",
		SetAvailability: &CharacterAvailabilityPatch{CharacterID: "char-1", Available: true},
	})
	require.NoError(t, err)
	require.False(t, res.Player.Inactive)
	require.Equal(t, "char-1", res.Player.Character.ID)
	require.False(t, hasCharacter(t, svc, enc1.ID, "char-1"), "the character does not rejoin past encounters")
}

func TestCampaignRosterPatchRemovePlayer(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{CampaignID: "test-campaign", RemovePlayer: &RemovePlayerPatch{CharacterID: "char-1"}})
	require.ErrorContains(t, err, "cannot remove the last player")
	require.Len(t, rosterPlayers(t, svc), 1, "the last player stays")
	require.False(t, rosterPlayers(t, svc)[0].Inactive)

	second := addSecondPlayer(t, svc)
	enc := partyEncounter(t, svc, "Fight", second)
	var events []ChangeEvent
	t.Cleanup(svc.SubscribeChanges(func(ev ChangeEvent) { events = append(events, ev) }))

	res, err := svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{CampaignID: "test-campaign", RemovePlayer: &RemovePlayerPatch{CharacterID: second.Character.ID}})
	require.NoError(t, err)
	require.Len(t, events, 1, "one save, one change event")
	require.Nil(t, res.Player)
	players := rosterPlayers(t, svc)
	require.Len(t, players, 1)
	require.Equal(t, "char-1", players[0].Character.ID)
	require.False(t, hasCharacter(t, svc, enc.ID, second.Character.ID), "a removed player leaves the campaign's encounters")
}

func TestCampaignRosterPatchUnknownTargets(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{CampaignID: "missing", SetAvailability: &CharacterAvailabilityPatch{CharacterID: "char-1"}})
	require.ErrorIs(t, err, domain.ErrCampaignNotFound)
	_, err = svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{CampaignID: "test-campaign", RemovePlayer: &RemovePlayerPatch{CharacterID: "missing"}})
	require.ErrorContains(t, err, "not found")
	_, err = svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{CampaignID: "test-campaign"})
	require.ErrorContains(t, err, "required")
}

func TestCampaignRosterPatchSyncsActiveEncounterCombatant(t *testing.T) {
	svc := newSQLiteService(t)
	enc := partyEncounter(t, svc, "Live", rosterPlayers(t, svc)[0])

	_, err := svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{
		CampaignID: "test-campaign",
		UpdateCharacter: &CharacterPatch{CharacterID: "char-1", Edit: func(p *domain.NewCampaignPlayer) {
			p.Character.MaxHP, p.Character.HP, p.Character.Level = 9, 4, 3
		}},
	})
	require.NoError(t, err)
	got, err := svc.GetEncounterByID(t.Context(), enc.ID)
	require.NoError(t, err)
	for _, c := range got.Combatants {
		if c.PlayerCharacterID == "char-1" {
			require.Equal(t, 4, c.HP, "the linked combatant follows the character (spec 003)")
			require.Equal(t, 9, c.MaxHP)
			require.Equal(t, 3, c.Level)
			return
		}
	}
	t.Fatal("linked combatant missing")
}

func TestCampaignRosterPatchBumpsRevisionsAndPublishes(t *testing.T) {
	svc := newSQLiteService(t)
	enc := partyEncounter(t, svc, "Live", rosterPlayers(t, svc)[0])
	var mu sync.Mutex
	var events []ChangeEvent
	unsubscribe := svc.SubscribeChanges(func(ev ChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, ev)
	})
	t.Cleanup(unsubscribe)
	campaignRev, encounterRev := svc.Revision(campaignKey("test-campaign")), svc.Revision(encounterKey(enc.ID))

	ctx := WithSource(t.Context(), domain.SourceAssistant)
	_, err := svc.ExecuteCampaignRosterPatch(ctx, CampaignRosterPatch{
		CampaignID:      "test-campaign",
		UpdateCharacter: &CharacterPatch{CharacterID: "char-1", Edit: func(p *domain.NewCampaignPlayer) { p.Notes = "changed" }},
	})
	require.NoError(t, err)
	require.Greater(t, svc.Revision(campaignKey("test-campaign")), campaignRev)
	require.Greater(t, svc.Revision(encounterKey(enc.ID)), encounterRev)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, events, 1)
	require.Equal(t, domain.SourceAssistant, events[0].Source)
	require.Equal(t, "test-campaign", events[0].CampaignID)
	require.Contains(t, events[0].Kinds, domain.EntityPlayerCharacter)
}

// pausingRepo pauses the first ListCampaignPlayers call once armed, so a
// test can act between a patch's read and its write.
type pausingRepo struct {
	EncounterRepository
	armed   atomic.Bool
	paused  chan struct{}
	release chan struct{}
	once    sync.Once
}

func (r *pausingRepo) ListCampaignPlayers(ctx context.Context, campaignID string) ([]domain.NewCampaignPlayer, error) {
	if r.armed.Load() {
		r.once.Do(func() {
			close(r.paused)
			<-r.release
		})
	}
	return r.EncounterRepository.ListCampaignPlayers(ctx, campaignID)
}

func TestCampaignRosterPatchHoldsTheWriteGate(t *testing.T) {
	db, err := sqlite.OpenAndMigrate(filepath.Join(t.TempDir(), "tracker.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &pausingRepo{EncounterRepository: sqlite.NewEncounterStore(db), paused: make(chan struct{}), release: make(chan struct{})}
	svc := NewServiceWithLogfAndTimeout(repo, t.Logf, 30*time.Second)
	_, err = svc.CreateCampaign(t.Context(), "test-campaign", "Test Campaign", testCampaignStartDate(t), []domain.NewCampaignPlayer{
		{PlayerName: "Player 1", Character: domain.Combatant{ID: "char-1", Name: "Vault Dweller", Side: domain.SideParty, Level: 1, HP: 7, MaxHP: 7}},
	})
	require.NoError(t, err)
	desktopPlayers := rosterPlayers(t, svc)

	repo.armed.Store(true)
	done := make(chan error, 1)
	go func() {
		_, err := svc.ExecuteCampaignRosterPatch(t.Context(), CampaignRosterPatch{
			CampaignID:      "test-campaign",
			UpdateCharacter: &CharacterPatch{CharacterID: "char-1", Edit: func(p *domain.NewCampaignPlayer) { p.Character.Level = 5 }},
		})
		done <- err
	}()
	<-repo.paused

	// A desktop save between the patch's read and write must wait for the
	// gate; with a short deadline it reports busy instead of interleaving.
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	desktopPlayers[0].Character.Level = 2
	_, err = svc.ExecuteUpdateCampaign(ctx, UpdateCampaignCommand{CampaignID: "test-campaign", Name: "Desktop", StartDate: testCampaignStartDate(t), Players: desktopPlayers})
	require.ErrorIs(t, err, domain.ErrTrackerBusy)

	close(repo.release)
	require.NoError(t, <-done)
	players := rosterPlayers(t, svc)
	require.Equal(t, 5, players[0].Character.Level, "the patch wrote what it read and changed")
	campaigns, err := svc.ListCampaigns(t.Context())
	require.NoError(t, err)
	require.Equal(t, "Test Campaign", campaigns[0].Name, "the desktop save did not run")
}
