package bondUserSettings_handlers_test

import (
	"net/http"
	"testing"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
)

// DELETE /bond-user-settings/{id} responde 200 true con el id del path; uno inexistente o fuera del alcance
// de la sesión responde 404 ERR_BOND_USER_SETTING_NOT_FOUND. CA-09.9
func TestRemove(t *testing.T) {
	t.Run("baja", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, admin), http.MethodDelete, settingPath, "")

		var body bool
		decodeJSON(t, rec, http.StatusOK, &body)
		if !body {
			t.Fatalf("body %s, se esperaba true", rec.Body)
		}
		if app.id != settingID {
			t.Fatalf("la aplicación recibió el id %q, se esperaba %q", app.id, settingID)
		}
	})

	t.Run("inexistente", func(t *testing.T) {
		app := &fakeApp{err: bondUserSetting_domain.ErrBondUserSettingNotFound}

		rec := serve(newTestAPI(app, admin), http.MethodDelete, settingPath, "")

		assertError(t, rec, http.StatusNotFound, "ERR_BOND_USER_SETTING_NOT_FOUND")
	})
}
