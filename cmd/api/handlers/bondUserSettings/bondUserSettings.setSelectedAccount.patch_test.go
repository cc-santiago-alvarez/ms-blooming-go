package bondUserSettings_handlers_test

import (
	"net/http"
	"testing"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// PATCH /me/selected-account responde 200 con el setting del usuario de la sesión y pasa a la aplicación
// solo selectedAccountId: un userId en el body se descarta. Una cuenta ajena o inexistente responde 404
// ERR_ACCOUNT_NOT_FOUND. CA-09.5
func TestSetSelectedAccount(t *testing.T) {
	t.Run("userId ajeno en el body", func(t *testing.T) {
		app := &fakeApp{setting: sample()}
		body := `{"selectedAccountId":"` + accountID + `","userId":"` + user2ID + `"}`

		rec := serve(newTestAPI(app, investor), http.MethodPatch, minePath+"/selected-account", body)

		var got struct {
			ID                string `json:"_id"`
			UserID            string `json:"userId"`
			SelectedAccountID string `json:"selectedAccountId"`
		}
		decodeJSON(t, rec, http.StatusOK, &got)
		if got.ID != settingID || got.UserID != userID || got.SelectedAccountID != accountID {
			t.Errorf("body %s, se esperaba el setting %s de %s con la cuenta %s", rec.Body, settingID, userID, accountID)
		}
		if want := (bondUserSetting_domain.SetSelectedAccountParams{SelectedAccountID: accountID}); app.account != want {
			t.Fatalf("la aplicación recibió %+v, se esperaba %+v", app.account, want)
		}
	})

	t.Run("cuenta ajena", func(t *testing.T) {
		app := &fakeApp{err: shared_domain.ErrAccountNotFound}

		rec := serve(newTestAPI(app, investor), http.MethodPatch, minePath+"/selected-account", `{"selectedAccountId":"account3AAAAAAAAA"}`)

		assertError(t, rec, http.StatusNotFound, "ERR_ACCOUNT_NOT_FOUND")
	})
}
