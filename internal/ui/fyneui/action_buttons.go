package fyneui

import "fyne.io/fyne/v2/widget"

func newRefreshingActionButton(
	label string,
	collapse func(),
	handleErr func(error),
	afterSuccess func(),
	run func() error,
) *widget.Button {
	return widget.NewButton(label, func() {
		if collapse != nil {
			collapse()
		}
		err := run()
		handleErr(err)
		if err != nil {
			return
		}
		if afterSuccess != nil {
			afterSuccess()
		}
	})
}
