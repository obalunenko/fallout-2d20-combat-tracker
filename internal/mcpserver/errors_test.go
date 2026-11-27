package mcpserver

import (
	"errors"
	"fmt"
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestAsToolErrorClassifiesByKindNotWording(t *testing.T) {
	var logged []string
	logf := func(format string, args ...any) { logged = append(logged, fmt.Sprintf(format, args...)) }

	for _, tc := range []struct {
		name    string
		err     error
		code    string
		message string
	}{
		{"marked invalid input", domain.InvalidInput(errors.New("strength must be at least 1")), codeInvalidInput, "strength must be at least 1"},
		{"marked not found", domain.NotFound(errors.New(`combatant "x" not found`)), codeNotFound, `combatant "x" not found`},
		{"wrapped not found sentinel", fmt.Errorf("activate: %w", domain.ErrEncounterNotFound), codeNotFound, "encounter not found"},
		{"no campaign", domain.ErrCampaignNotInitialized, codeNoCampaign, ""},
		{"busy", domain.ErrTrackerBusy, codeBusy, ""},
		// Unmarked errors are internal even when their wording looks like
		// validation: nothing is guessed from the text any more.
		{"unmarked validation-like text", errors.New("value must be at least 1"), codeInternal, ""},
		{"unmarked not-found-like text", errors.New("row not found in dictionary"), codeInternal, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logged = nil
			te := asToolError(tc.err, logf)
			require.Equal(t, tc.code, te.Code)
			if tc.message != "" {
				require.Equal(t, tc.message, te.Message, "the message passes through verbatim")
			}
			require.Equal(t, tc.code == codeInternal, len(logged) == 1, "only internal errors are logged")
		})
	}
}
