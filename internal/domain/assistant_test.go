package domain

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAssistantPort(t *testing.T) {
	for _, tc := range []struct {
		port    int
		wantErr bool
	}{
		{1023, true},
		{1024, false},
		{DefaultAssistantPort, false},
		{65535, false},
		{65536, true},
	} {
		t.Run(fmt.Sprint(tc.port), func(t *testing.T) {
			err := ValidateAssistantPort(tc.port)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAssistantConnectionSettingsValidate(t *testing.T) {
	valid := AssistantConnectionSettings{Port: DefaultAssistantPort, AccessSecret: "s3cret"}
	require.NoError(t, valid.Validate())

	blank := valid
	blank.AccessSecret = "  "
	require.Error(t, blank.Validate())

	badPort := valid
	badPort.Port = 80
	require.Error(t, badPort.Validate())
}

func TestAccessSecretIsRedacted(t *testing.T) {
	s := AssistantConnectionSettings{Port: DefaultAssistantPort, AccessSecret: "top-secret-value"}
	out := fmt.Sprintf("%v %+v %#v %s", s, s, s, s.AccessSecret)
	assert.NotContains(t, out, "top-secret-value")
}

func TestAssistantEnumsMatchSchema(t *testing.T) {
	// The values must match the CHECK lists in migration 00044.
	assert.Equal(t, []AssistantOutcome{"done", "rejected", "declined", "timed_out", "refused"},
		[]AssistantOutcome{OutcomeDone, OutcomeRejected, OutcomeDeclined, OutcomeTimedOut, OutcomeRefused})
	assert.Equal(t, []AssistantEntityKind{"campaign", "player_character", "encounter", "combatant", "monster_template"},
		[]AssistantEntityKind{EntityCampaign, EntityPlayerCharacter, EntityEncounter, EntityCombatant, EntityMonsterTemplate})
	assert.Equal(t, SourceDesktop, OperationSource("").Normalize())
	assert.Equal(t, SourceAssistant, SourceAssistant.Normalize())
}
