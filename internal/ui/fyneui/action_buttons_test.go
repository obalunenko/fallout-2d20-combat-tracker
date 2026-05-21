package fyneui

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRefreshingActionButtonRefreshesOnlyAfterSuccessfulAction(t *testing.T) {
	actionErr := errors.New("action failed")
	tests := []struct {
		name          string
		runErr        error
		wantRefreshes int
	}{
		{name: "success", wantRefreshes: 1},
		{name: "error", runErr: actionErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collapses := 0
			handled := 0
			refreshes := 0
			runs := 0

			btn := newRefreshingActionButton(
				"Run",
				func() { collapses++ },
				func(error) { handled++ },
				func() { refreshes++ },
				func() error {
					runs++
					return tt.runErr
				},
			)

			btn.OnTapped()

			assert.Equal(t, 1, collapses)
			assert.Equal(t, 1, handled)
			assert.Equal(t, 1, runs)
			assert.Equal(t, tt.wantRefreshes, refreshes)
		})
	}
}
