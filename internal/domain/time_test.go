package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCampaignStartDate(t *testing.T) {
	actual, err := ParseCampaignStartDate(" 2026-05-22 ")
	require.NoError(t, err)

	assert.Equal(t, time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC), actual)
}

func TestParseCampaignStartDateRejectsInvalidDate(t *testing.T) {
	_, err := ParseCampaignStartDate("2026/05/22")

	require.Error(t, err)
}
