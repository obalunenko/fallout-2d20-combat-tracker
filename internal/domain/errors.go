package domain

import "errors"

var ErrEncounterNotInitialized = errors.New("encounter not initialized")
var ErrEncounterNotFound = errors.New("encounter not found")
