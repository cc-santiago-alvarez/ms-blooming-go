package bondUserSettings_handlers_test

import (
	"net/http"
	"testing"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
)

// GET /bond-user-settings/{id} responde 200 con el setting; uno inexistente o fuera del alcance de la sesión
// responde 404 ERR_BOND_USER_SETTING_NOT_FOUND, y un id inválido 400 ERR_INVALID_ID sin llegar a la
// aplicación. CA-09.9
func TestGetByID(t *testing.T) {
	t.Run("existente", func(t *testing.T) {
		app := &fakeApp{setting: sample()}

		rec := serve(newTestAPI(app, admin), http.MethodGet, settingPath, "")

		var got struct {
			ID string `json:"_id"`
		}
		decodeJSON(t, rec, http.StatusOK, &got)
		if got.ID != settingID || app.id != settingID {
			t.Fatalf("body %s y la aplicación recibió %q, se esperaba el setting %s", rec.Body, app.id, settingID)
		}
	})

	t.Run("inexistente", func(t *testing.T) {
		app := &fakeApp{err: bondUserSetting_domain.ErrBondUserSettingNotFound}

		rec := serve(newTestAPI(app, admin), http.MethodGet, settingPath, "")

		assertError(t, rec, http.StatusNotFound, "ERR_BOND_USER_SETTING_NOT_FOUND")
	})

	t.Run("id inválido", func(t *testing.T) {
		app := &fakeApp{setting: sample()}

		rec := serve(newTestAPI(app, admin), http.MethodGet, settingsPath+"/abc", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_ID")
		if len(app.calls) != 0 {
			t.Fatalf("la aplicación recibió %v con un id inválido", app.calls)
		}
	})
}
