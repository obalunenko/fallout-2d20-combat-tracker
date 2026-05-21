package fyneui

import (
	"context"
	"fmt"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsvc "github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

func TestMainViewRefresherShowsNoCampaign(t *testing.T) {
	state := &uiState{
		enc:                 testEncounter(),
		activeCampaign:      &domain.Campaign{ID: "camp-old", Name: "Old", StartDate: testCampaignDate()},
		selectedIndex:       1,
		expandedCombatantID: "pc-1",
	}
	repo := &refresherRepo{activeCampaignErr: domain.ErrCampaignNotInitialized}
	refresher, screen, errorCount := newTestMainViewRefresher(t, state, repo)

	refresher.Refresh()

	assert.Zero(t, *errorCount)
	assert.Nil(t, state.activeCampaign)
	assert.Nil(t, state.enc)
	assert.Empty(t, state.expandedCombatantID)
	assert.Equal(t, "Campaign: -", screen.campaignStatusLabel.Text)
	assert.Equal(t, "No active campaign", screen.campRosterOutput.Text)
	assert.Equal(t, "No active campaign", screen.partyLibraryOutput.Text)
	assert.Equal(t, "[BOOT] Pip-Boy combat tracker initialized", screen.logOutput.Text)
	assert.True(t, screen.noCampaignView.Visible())
	assert.False(t, screen.noEncounterView.Visible())
	assert.False(t, screen.tabsView.Visible())
	assert.Zero(t, repo.listEncounterLogsCalls)
}

func TestMainViewRefresherShowsNoEncounterAfterLoadingCampaignData(t *testing.T) {
	state := &uiState{enc: testEncounter(), expandedCombatantID: "npc-1"}
	repo := &refresherRepo{
		activeCampaign: &domain.Campaign{ID: "camp-1", Name: "Vault 13", StartDate: testCampaignDate()},
		encounterErr:   domain.ErrEncounterNotInitialized,
		players: []domain.NewCampaignPlayer{
			{PlayerName: "June", Character: domain.Combatant{Name: "Vault Dweller", Level: 2, HP: 8, MaxHP: 8}},
		},
		partyMembers: []domain.Combatant{
			{Name: "Vault Dweller", Level: 2, HP: 8, MaxHP: 8},
		},
	}
	refresher, screen, errorCount := newTestMainViewRefresher(t, state, repo)

	refresher.Refresh()

	assert.Zero(t, *errorCount)
	require.NotNil(t, state.activeCampaign)
	assert.Equal(t, "camp-1", state.activeCampaign.ID)
	assert.Nil(t, state.enc)
	assert.Empty(t, state.expandedCombatantID)
	assert.Equal(t, "Campaign: Vault 13 (2026-01-01)", screen.campaignStatusLabel.Text)
	assert.Contains(t, screen.campRosterOutput.Text, "June -> Vault Dweller")
	assert.Contains(t, screen.partyLibraryOutput.Text, "Vault Dweller")
	assert.Equal(t, "[BOOT] Pip-Boy combat tracker initialized", screen.logOutput.Text)
	assert.True(t, screen.noEncounterView.Visible())
	assert.False(t, screen.noCampaignView.Visible())
	assert.False(t, screen.tabsView.Visible())
	assert.Zero(t, repo.listEncounterLogsCalls)
}

func TestMainViewRefresherShowsActiveEncounterAndLogs(t *testing.T) {
	encounter := testEncounter()
	encounter.Round = 2
	encounter.Resources = domain.Resources{PartyAP: 3, GMThreat: 4}
	state := &uiState{selectedIndex: 99, expandedCombatantID: "missing"}
	repo := &refresherRepo{
		activeCampaign: &domain.Campaign{ID: "camp-1", Name: "Vault 13", StartDate: testCampaignDate()},
		encounter:      encounter,
		players: []domain.NewCampaignPlayer{
			{PlayerName: "June", Character: domain.Combatant{Name: "Vault Dweller", Level: 2, HP: 8, MaxHP: 8}},
		},
		partyMembers: []domain.Combatant{
			{Name: "Vault Dweller", Level: 2, HP: 8, MaxHP: 8},
		},
		logs: []domain.EncounterLog{
			{Round: 2, Message: "Turn advanced", CreatedAt: time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)},
		},
	}
	refresher, screen, errorCount := newTestMainViewRefresher(t, state, repo)

	refresher.Refresh()

	assert.Zero(t, *errorCount)
	require.NotNil(t, state.enc)
	assert.Equal(t, "enc-1", state.enc.ID)
	assert.Equal(t, 0, state.selectedIndex)
	assert.Empty(t, state.expandedCombatantID)
	assert.Equal(t, "Round: 2", screen.roundLabel.Text)
	assert.Equal(t, "Party AP: 3", screen.partyAPLabel.Text)
	assert.Equal(t, "GM Threat: 4", screen.threatLabel.Text)
	assert.Contains(t, screen.selectedLabel.Text, "Name: Alpha")
	assert.Contains(t, screen.campSnapshotLabel.Text, "Encounter: Test Encounter")
	assert.Contains(t, screen.logOutput.Text, "[R2] Turn advanced")
	assert.True(t, screen.tabsView.Visible())
	assert.False(t, screen.noEncounterView.Visible())
	assert.False(t, screen.noCampaignView.Visible())
	assert.Equal(t, 1, repo.listEncounterLogsCalls)
}

func newTestMainViewRefresher(
	t *testing.T,
	state *uiState,
	repo *refresherRepo,
) (*mainViewRefresher, *mainScreen, *int) {
	t.Helper()
	test.NewTempApp(t)

	labels := newMainScreenLabels()
	encounterOrder := newEncounterOrderView(
		&state.enc,
		&state.selectedIndex,
		&state.expandedCombatantID,
		labels.selectedLabel,
		func(int) {},
		func(int) {},
	)
	screen := newTestMainScreen(labels)
	svc := appsvc.NewServiceWithLogfAndTimeout(repo, func(string, ...any) {}, 0)
	errorCount := 0
	refresher := newMainViewRefresher(context.Background(), svc, state, screen, encounterOrder, func(error) {
		errorCount++
	})
	return refresher, screen, &errorCount
}

type refresherRepo struct {
	activeCampaign    *domain.Campaign
	activeCampaignErr error
	encounter         *domain.Encounter
	encounterErr      error
	players           []domain.NewCampaignPlayer
	partyMembers      []domain.Combatant
	logs              []domain.EncounterLog

	listEncounterLogsCalls int
}

func (r *refresherRepo) Get(context.Context) (*domain.Encounter, error) {
	if r.encounterErr != nil {
		return nil, r.encounterErr
	}
	if r.encounter == nil {
		return nil, domain.ErrEncounterNotInitialized
	}
	return r.encounter, nil
}

func (r *refresherRepo) Save(context.Context, *domain.Encounter) error { return nil }

func (r *refresherRepo) List(context.Context) ([]domain.EncounterSummary, error) { return nil, nil }

func (r *refresherRepo) GetEncounterByID(context.Context, string) (*domain.Encounter, error) {
	return nil, domain.ErrEncounterNotFound
}

func (r *refresherRepo) UpdateEncounter(context.Context, string, string, []domain.Combatant) (*domain.Encounter, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *refresherRepo) ListPartyMembers(context.Context) ([]domain.Combatant, error) {
	return r.partyMembers, nil
}

func (r *refresherRepo) CreateCampaign(context.Context, string, string, string, []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *refresherRepo) UpdateCampaign(context.Context, string, string, string, []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *refresherRepo) GetActiveCampaign(context.Context) (*domain.Campaign, error) {
	if r.activeCampaignErr != nil {
		return nil, r.activeCampaignErr
	}
	if r.activeCampaign == nil {
		return nil, domain.ErrCampaignNotInitialized
	}
	return r.activeCampaign, nil
}

func (r *refresherRepo) ListCampaigns(context.Context) ([]domain.Campaign, error) { return nil, nil }

func (r *refresherRepo) ListCampaignPlayers(context.Context, string) ([]domain.NewCampaignPlayer, error) {
	return r.players, nil
}

func (r *refresherRepo) ActivateCampaign(context.Context, string) error { return nil }

func (r *refresherRepo) Activate(context.Context, string) error { return nil }

func (r *refresherRepo) SoftDelete(context.Context, string) error { return nil }

func (r *refresherRepo) AppendEncounterLog(context.Context, string, int, string) error { return nil }

func (r *refresherRepo) ListEncounterLogs(context.Context, string) ([]domain.EncounterLog, error) {
	r.listEncounterLogsCalls++
	return r.logs, nil
}
