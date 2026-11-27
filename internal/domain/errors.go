package domain

import (
	"errors"
	"fmt"
)

// ErrInvalidInput marks errors caused by invalid input from the GM or the
// assistant: a rule or validation failure that the caller can fix. Test with
// errors.Is; the error message is the specific one, not this text.
var ErrInvalidInput = errors.New("invalid input")

// ErrNotFound marks errors caused by an identifier that does not exist. Test
// with errors.Is.
var ErrNotFound = errors.New("not found")

var ErrEncounterNotInitialized = errors.New("encounter not initialized")
var ErrEncounterNotFound = fmt.Errorf("encounter %w", ErrNotFound)
var ErrCampaignNotInitialized = errors.New("campaign not initialized")
var ErrCampaignNotFound = fmt.Errorf("campaign %w", ErrNotFound)

// InvalidInput marks err as caused by invalid input, keeping its message.
// It returns nil for nil, and err unchanged when err is already marked
// invalid input or not found.
func InvalidInput(err error) error { return classify(err, ErrInvalidInput) }

// NotFound marks err as caused by an unknown identifier, keeping its message.
// It returns nil for nil, and err unchanged when err is already marked
// invalid input or not found.
func NotFound(err error) error { return classify(err, ErrNotFound) }

func classify(err, kind error) error {
	if err == nil || errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrNotFound) {
		return err
	}
	return &classifiedError{err: err, kind: kind}
}

// classifiedError keeps err's message and adds kind to its error chain.
type classifiedError struct {
	err  error
	kind error
}

func (e *classifiedError) Error() string   { return e.err.Error() }
func (e *classifiedError) Unwrap() []error { return []error{e.err, e.kind} }

// ErrTrackerBusy reports that an operation could not acquire the tracker's
// operation gate before its deadline. The caller may retry.
var ErrTrackerBusy = errors.New("tracker busy, try again")

// ErrStaleEdit reports that an editor tried to save an entity that the
// assistant changed after the editor opened.
var ErrStaleEdit = errors.New("entity was changed after the editor opened")

// ErrEntityDeleted reports that an editor tried to save an entity that the
// assistant deleted after the editor opened.
var ErrEntityDeleted = errors.New("entity was deleted")
