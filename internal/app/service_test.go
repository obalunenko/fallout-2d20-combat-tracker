package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceAppliesDefaultOperationTimeoutWhenDeadlineIsMissing(t *testing.T) {
	repo := &contextCaptureRepo{
		logFailingRepo: &logFailingRepo{
			encounter: &domain.Encounter{ID: "enc-1", Round: 1},
		},
	}
	svc := NewServiceWithLogfAndTimeout(repo, func(string, ...any) {}, 200*time.Millisecond)

	_, err := svc.GetEncounter(t.Context())
	require.NoError(t, err)
	require.True(t, repo.gotHasDeadline, "service should apply operation timeout when caller has no deadline")
}

func TestServiceKeepsCallerDeadlineWhenAlreadySet(t *testing.T) {
	repo := &contextCaptureRepo{
		logFailingRepo: &logFailingRepo{
			encounter: &domain.Encounter{ID: "enc-1", Round: 1},
		},
	}
	svc := NewServiceWithLogfAndTimeout(repo, func(string, ...any) {}, 5*time.Second)

	parentCtx, cancel := context.WithTimeout(t.Context(), 120*time.Millisecond)
	defer cancel()
	parentDeadline, ok := parentCtx.Deadline()
	require.True(t, ok)

	_, err := svc.GetEncounter(parentCtx)
	require.NoError(t, err)
	require.True(t, repo.gotHasDeadline)
	require.False(t, repo.gotDeadline.After(parentDeadline), "service should not extend caller deadline")
}

func newSQLiteService(t *testing.T) *Service {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "tracker.db")
	db, err := sqlite.OpenAndMigrate(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	svc := NewService(sqlite.NewEncounterStore(db))
	_, err = svc.CreateCampaign(t.Context(), "test-campaign", "Test Campaign", testCampaignStartDate(t), []domain.NewCampaignPlayer{
		{
			PlayerName: "Player 1",
			Character: domain.Combatant{
				ID:         "char-1",
				Name:       "Vault Dweller",
				Side:       domain.SideParty,
				Level:      1,
				Initiative: 9,
				HP:         7,
				Defense:    1,
			},
		},
	})
	require.NoError(t, err)
	return svc
}

func testCampaignStartDate(t *testing.T) time.Time {
	t.Helper()
	startDate, err := domain.ParseCampaignStartDate("2026-01-01")
	require.NoError(t, err)
	return startDate
}

type logFailingRepo struct {
	encounter     *domain.Encounter
	appendErr     error
	saveCalls     int
	resourceCalls int
	appendCalls   int
}

func (r *logFailingRepo) Get(_ context.Context) (*domain.Encounter, error) {
	return cloneEncounter(r.encounter), nil
}

func (r *logFailingRepo) Save(_ context.Context, encounter *domain.Encounter) error {
	r.saveCalls++
	r.encounter = cloneEncounter(encounter)
	return nil
}

func (r *logFailingRepo) List(_ context.Context) ([]domain.EncounterSummary, error) { return nil, nil }

func (r *logFailingRepo) GetEncounterByID(_ context.Context, _ string) (*domain.Encounter, error) {
	return cloneEncounter(r.encounter), nil
}

func (r *logFailingRepo) UpdateEncounter(_ context.Context, _, _ string, _ []domain.Combatant) (*domain.Encounter, error) {
	return nil, nil
}

func (r *logFailingRepo) ListPartyMembers(_ context.Context) ([]domain.Combatant, error) {
	return nil, nil
}

func (r *logFailingRepo) ListMonsterTemplates(_ context.Context) ([]domain.Combatant, error) {
	return nil, nil
}

func (r *logFailingRepo) UpsertMonsterTemplate(_ context.Context, monster domain.Combatant) (domain.Combatant, error) {
	return monster, nil
}

func (r *logFailingRepo) CreateCampaign(_ context.Context, _, _ string, _ time.Time, _ []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	return nil, nil
}

func (r *logFailingRepo) UpdateCampaign(_ context.Context, _, _ string, _ time.Time, _ []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	return nil, nil
}

func (r *logFailingRepo) GetActiveCampaign(_ context.Context) (*domain.Campaign, error) {
	return nil, nil
}

func (r *logFailingRepo) ListCampaigns(_ context.Context) ([]domain.Campaign, error) { return nil, nil }

func (r *logFailingRepo) ListCampaignPlayers(_ context.Context, _ string) ([]domain.NewCampaignPlayer, error) {
	return nil, nil
}

func (r *logFailingRepo) ActivateCampaign(_ context.Context, _ string) error { return nil }

func (r *logFailingRepo) UpdateCampaignResources(_ context.Context, _ string, resources domain.Resources) error {
	r.resourceCalls++
	if r.encounter != nil {
		r.encounter.Resources = resources
	}
	return nil
}

func (r *logFailingRepo) Activate(_ context.Context, _ string) error { return nil }

func (r *logFailingRepo) SoftDelete(_ context.Context, _ string) error { return nil }

func (r *logFailingRepo) AppendEncounterLog(_ context.Context, _ string, _ int, _ string) error {
	r.appendCalls++
	return r.appendErr
}

func (r *logFailingRepo) ListEncounterLogs(_ context.Context, _ string) ([]domain.EncounterLog, error) {
	return nil, nil
}

func cloneEncounter(src *domain.Encounter) *domain.Encounter {
	if src == nil {
		return nil
	}
	cp := *src
	cp.Combatants = append([]domain.Combatant(nil), src.Combatants...)
	return &cp
}

type contextCaptureRepo struct {
	*logFailingRepo
	gotHasDeadline bool
	gotDeadline    time.Time
}

func (r *contextCaptureRepo) Get(ctx context.Context) (*domain.Encounter, error) {
	r.gotDeadline, r.gotHasDeadline = ctx.Deadline()
	return cloneEncounter(r.encounter), nil
}
