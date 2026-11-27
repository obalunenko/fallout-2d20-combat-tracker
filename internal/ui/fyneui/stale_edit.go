package fyneui

import (
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/obalunenko/fallout/internal/domain"
)

const busyMessage = "The tracker is busy, try again."

// staleEdit is returned by an editor's submit when the assistant changed the
// entity after the editor opened (FR-022). The editor offers Reload, which
// discards the edits and reopens with current data, and Overwrite, which saves
// the edits over the current state.
type staleEdit struct {
	entity    string
	overwrite func() error
	reload    func()
}

func (e *staleEdit) Error() string {
	return fmt.Sprintf("this %s was changed by the assistant after you opened it", e.entity)
}

// editorCallbacks are what handleEditorSaveError needs from an open editor.
type editorCallbacks struct {
	window  fyne.Window
	entity  string
	saved   func() // refresh and close after a successful overwrite
	close   func() // close without saving
	showErr func(string)
	// ask and notify show the choice and the notice; nil uses Fyne dialogs.
	ask    func(title, message, confirmText, dismissText string, answer func(confirmed bool))
	notify func(title, message string, onClosed func())
}

func (cb editorCallbacks) askChoice(title, message, confirmText, dismissText string, answer func(bool)) {
	if cb.ask != nil {
		cb.ask(title, message, confirmText, dismissText, answer)
		return
	}
	d := dialog.NewConfirm(title, message, answer, cb.window)
	d.SetConfirmText(confirmText)
	d.SetDismissText(dismissText)
	d.Show()
}

func (cb editorCallbacks) showNotice(title, message string, onClosed func()) {
	if cb.notify != nil {
		cb.notify(title, message, onClosed)
		return
	}
	d := dialog.NewInformation(title, message, cb.window)
	d.SetOnClosed(onClosed)
	d.Show()
}

// handleEditorSaveError handles the stale-edit, deleted and busy outcomes of
// a save. It reports whether it handled err; when it did, nothing was saved
// unless the GM later chooses Overwrite.
func handleEditorSaveError(err error, cb editorCallbacks) bool {
	var stale *staleEdit
	switch {
	case errors.As(err, &stale):
		cb.askChoice("Changed by the assistant",
			fmt.Sprintf("This %s was changed by the assistant after you opened it.\nReload discards your changes; Overwrite saves your values over the assistant's.", stale.entity),
			"Overwrite", "Reload",
			func(overwrite bool) {
				if !overwrite {
					cb.close()
					stale.reload()
					return
				}
				if err := stale.overwrite(); err != nil {
					if !handleEditorSaveError(err, cb) {
						cb.showErr(err.Error())
					}
					return
				}
				cb.saved()
			})
		return true
	case errors.Is(err, domain.ErrEntityDeleted):
		cb.showNotice("Deleted by the assistant",
			fmt.Sprintf("This %s was deleted by the assistant. The editor will close without saving.", cb.entity), cb.close)
		return true
	case errors.Is(err, domain.ErrTrackerBusy):
		cb.showErr(busyMessage)
		return true
	}
	return false
}
