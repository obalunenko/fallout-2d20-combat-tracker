package sqlite

import (
	"errors"
	"testing"

	"github.com/obalunenko/fallout/internal/store/sqlite/dbgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInTxRollsBackOnCallbackError(t *testing.T) {
	store := newTestStore(t)

	err := store.runInTx(t.Context(), func(qtx *dbgen.Queries) error {
		if err := qtx.InsertCampaign(t.Context(), dbgen.InsertCampaignParams{
			ID:        "rollback-campaign",
			Name:      "Rollback Campaign",
			StartDate: testCampaignStartDate(t),
		}); err != nil {
			return err
		}
		return errors.New("forced rollback")
	})

	require.Error(t, err)
	assert.Equal(t, int64(0), queryInt64(t, store.db, `SELECT COUNT(*) FROM campaigns WHERE id = ?`, "rollback-campaign"))
}
