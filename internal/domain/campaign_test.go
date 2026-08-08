package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSpecialValuesAreCompleteAndValid(t *testing.T) {
	values := DefaultSpecialValues()
	require.NoError(t, values.Validate())
	for _, attribute := range SpecialAttributes() {
		assert.Equal(t, 1, values.Value(attribute))
	}
}

func TestSpecialValuesValidateEveryPositiveAttribute(t *testing.T) {
	values := DefaultSpecialValues()
	require.NoError(t, values.Set(SpecialLuck, 99))
	require.NoError(t, values.Validate())

	require.NoError(t, values.Set(SpecialPerception, 0))
	err := values.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "perception")
}

func TestCampaignPlayerNotesPreserveWhitespace(t *testing.T) {
	player := NewCampaignPlayer{Notes: "  first line\nsecond line  "}
	assert.Equal(t, "  first line\nsecond line  ", player.Notes)
}
