package fyneui

import (
	"context"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	appsvc "github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

type listRepo struct {
	*refresherRepo
	summaries []domain.EncounterSummary
}

func (r *listRepo) List(context.Context) ([]domain.EncounterSummary, error) { return r.summaries, nil }

func labelTexts(obj fyne.CanvasObject) []string {
	var out []string
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		switch v := o.(type) {
		case *widget.Label:
			out = append(out, v.Text)
		case *fyne.Container:
			for _, child := range v.Objects {
				walk(child)
			}
		case fyne.Widget:
			for _, child := range test.WidgetRenderer(v).Objects() {
				walk(child)
			}
		}
	}
	walk(obj)
	return out
}

func overlayText(w fyne.Window) string {
	var all []string
	for _, o := range w.Canvas().Overlays().List() {
		all = append(all, labelTexts(o)...)
	}
	return strings.Join(all, "\n")
}

func TestEncounterListReloadKeepsOrClearsSelection(t *testing.T) {
	test.NewTempApp(t)
	w := test.NewTempWindow(t, widget.NewLabel("main"))
	w.Resize(fyne.NewSize(1100, 700))
	repo := &listRepo{refresherRepo: &refresherRepo{}, summaries: []domain.EncounterSummary{
		{ID: "a", Name: "Alpha"}, {ID: "b", Name: "Bravo"},
	}}
	svc := appsvc.NewServiceWithLogfAndTimeout(repo, func(string, ...any) {}, 0)

	var reload func()
	showEncounterListDialogWindow(t.Context(), w, svc, nil, func(err error) { t.Fatal(err) }, func(r func()) func() {
		reload = r
		return func() {}
	})
	require.NotNil(t, reload, "the open list registers for live reload")
	require.Contains(t, overlayText(w), "Name: Alpha")

	// A new encounter appears before the selected one: the selection stays.
	repo.summaries = []domain.EncounterSummary{{ID: "c", Name: "Charlie"}, {ID: "a", Name: "Alpha"}, {ID: "b", Name: "Bravo"}}
	reload()
	require.Contains(t, overlayText(w), "Name: Alpha")

	// The selected encounter is deleted by the assistant.
	repo.summaries = []domain.EncounterSummary{{ID: "c", Name: "Charlie"}, {ID: "b", Name: "Bravo"}}
	reload()
	require.Contains(t, overlayText(w), "deleted by the assistant")
}
