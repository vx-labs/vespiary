package state

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vx-labs/vespiary/vespiary/api"
)

func Test_memDBStore_ApplicationProfiles(t *testing.T) {
	id := "4bffa69d-6a09-4b4c-ade8-197fa75b7d8e"
	applicationID := "491c17f0-eb84-4bdf-85ac-411fdbe928ff"
	accountID := "491c17f0-eb84-4bdf-85ac-411fdbe928ca"

	t.Run("create-get-list-delete", func(t *testing.T) {
		store := newApplicationProfilesState(newDB())

		require.NoError(t, store.Create(&api.ApplicationProfile{
			ID:            id,
			ApplicationID: applicationID,
			Name:          "my_app_profile",
			AccountID:     accountID,
		}))
		app, err := store.ByID(id)
		require.NoError(t, err)
		require.NotNil(t, app)
		require.Equal(t, app.ID, id)
		require.Equal(t, app.Name, "my_app_profile")

		app, err = store.ByAccountID(id, accountID)
		require.NoError(t, err)
		require.NotNil(t, app)
		require.Equal(t, app.ID, id)
		require.Equal(t, app.Name, "my_app_profile")

		apps, err := store.ListByAccountID(accountID)
		require.NoError(t, err)
		require.NotNil(t, app)
		require.Equal(t, 1, len(apps))
		require.Equal(t, apps[0].Name, "my_app_profile")

		err = store.Delete(id)
		require.NoError(t, err)
		_, err = store.ByID(id)
		require.Equal(t, err, ErrApplicationProfileNotFound)
		_, err = store.ByAccountID(id, accountID)
		require.Equal(t, err, ErrApplicationProfileNotFound)

	})
}
