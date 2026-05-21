package fyneui

import (
	"context"
	"errors"

	appsvc "github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

type mainViewRefresher struct {
	ctx context.Context
	svc *appsvc.Service

	state     *uiState
	presenter *mainViewPresenter
	handleErr func(error)
}

func newMainViewRefresher(
	ctx context.Context,
	svc *appsvc.Service,
	state *uiState,
	screen *mainScreen,
	encounterOrder *encounterOrderView,
	handleErr func(error),
) *mainViewRefresher {
	return &mainViewRefresher{
		ctx:       ctx,
		svc:       svc,
		state:     state,
		presenter: newMainViewPresenter(state, screen, encounterOrder),
		handleErr: handleErr,
	}
}

func (r *mainViewRefresher) Refresh() {
	var err error
	r.state.activeCampaign, err = r.svc.GetActiveCampaign(r.ctx)
	if err != nil {
		if errors.Is(err, domain.ErrCampaignNotInitialized) {
			r.presenter.showNoCampaign()
			r.refreshEncounterDataLog()
			return
		}
		r.handleErr(err)
		return
	}
	activeCampaign := r.state.activeCampaign
	r.presenter.showActiveCampaign(activeCampaign)

	r.refreshCampaignRoster(activeCampaign.ID)
	r.refreshPartyLibrary()

	r.state.enc, err = r.svc.GetEncounter(r.ctx)
	if err != nil {
		if errors.Is(err, domain.ErrEncounterNotInitialized) {
			r.presenter.showNoEncounter()
			r.refreshEncounterDataLog()
			return
		}
		r.handleErr(err)
		return
	}

	r.presenter.showActiveEncounter()
	r.refreshEncounterDataLog()
}

func (r *mainViewRefresher) refreshCampaignRoster(campaignID string) {
	players, err := r.svc.ListCampaignPlayers(r.ctx, campaignID)
	if err != nil {
		r.handleErr(err)
		r.presenter.showCampaignRosterError()
		return
	}
	r.presenter.showCampaignRoster(players)
}

func (r *mainViewRefresher) refreshPartyLibrary() {
	partyMembers, err := r.svc.ListPartyMembers(r.ctx)
	if err != nil {
		r.handleErr(err)
		r.presenter.showPartyLibraryError()
		return
	}
	r.presenter.showPartyLibrary(partyMembers)
}

func (r *mainViewRefresher) refreshEncounterDataLog() {
	refreshEncounterDataLog(r.ctx, r.svc, r.state.enc, r.presenter.logOutput(), r.handleErr)
}
