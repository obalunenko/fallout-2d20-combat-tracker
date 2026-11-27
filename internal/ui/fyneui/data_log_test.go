package fyneui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/obalunenko/fallout/internal/domain"
)

func TestFormatLogLineMarksAssistantEntries(t *testing.T) {
	at := time.Date(2026, 11, 27, 20, 15, 3, 0, time.Local)
	ts := formatTimestamp(at)

	require.Equal(t, "["+ts+"] [R2] [AI] Damage -> Raider", formatLogLine(domain.EncounterLog{
		Round: 2, Message: "Damage -> Raider", CreatedAt: at, Source: domain.SourceAssistant,
	}))
	require.Equal(t, "["+ts+"] [R2] Turn advanced -> Nora", formatLogLine(domain.EncounterLog{
		Round: 2, Message: "Turn advanced -> Nora", CreatedAt: at,
	}), "desktop entries are unchanged")
}
