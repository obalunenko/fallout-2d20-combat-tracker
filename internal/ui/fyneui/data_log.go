package fyneui

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2/widget"

	appsvc "github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

func refreshEncounterDataLog(
	ctx context.Context,
	svc *appsvc.Service,
	enc *domain.Encounter,
	output *widget.Entry,
	handleErr func(error),
) {
	if enc == nil {
		output.SetText("[BOOT] Pip-Boy combat tracker initialized")
		return
	}

	logs, err := svc.ListEncounterLogs(ctx, enc.ID)
	if err != nil {
		handleErr(err)
		return
	}

	if len(logs) == 0 {
		output.SetText("No operations yet")
		return
	}

	lines := make([]string, 0, len(logs))
	for _, logEntry := range logs {
		lines = append(lines, fmt.Sprintf("[%s] [R%d] %s",
			formatTimestamp(logEntry.CreatedAt), logEntry.Round, logEntry.Message))
	}
	output.SetText(strings.Join(lines, "\n"))
}
