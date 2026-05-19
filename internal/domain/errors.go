package domain

import "errors"

var ErrEncounterNotInitialized = errors.New("encounter not initialized")
var ErrEncounterNotFound = errors.New("encounter not found")
var ErrCampaignNotInitialized = errors.New("campaign not initialized")
var ErrCampaignNotFound = errors.New("campaign not found")
