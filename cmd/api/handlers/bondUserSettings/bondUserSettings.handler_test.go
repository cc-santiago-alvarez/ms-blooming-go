package bondUserSettings_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
)

// Las rutas de administración responden 403 ERR_FORBIDDEN a una sesión sin adminBlooming, o sin sesión, sin
// llegar a la aplicación; las de /me las atiende cualquier usuario. CA-09.8
func TestBondUserSettingsRequireAdminBlooming(t *testing.T) {
	routes := []struct {
		method string
		target string
	}{
		{method: http.MethodGet, target: settingsPath},
		{method: http.MethodGet, target: settingPath},
		{method: http.MethodDelete, target: settingPath},
	}
	sessions := []struct {
		name    string
		session *session_domain.UserSession
	}{
		{name: "sin adminBlooming", session: investor},
		{name: "sin sesión", session: nil},
	}
	for _, s := range sessions {
		for _, route := range routes {
			t.Run(s.name+" "+route.method+" "+route.target, func(t *testing.T) {
				app := &fakeApp{setting: sample()}

				rec := serve(newTestAPI(app, s.session), route.method, route.target, "")

				assertError(t, rec, http.StatusForbidden, "ERR_FORBIDDEN")
				if len(app.calls) != 0 {
					t.Fatalf("la aplicación recibió %v sin adminBlooming", app.calls)
				}
			})
		}
	}

	mine := []struct {
		method   string
		target   string
		body     string
		wantCall string
	}{
		{method: http.MethodGet, target: minePath, wantCall: "GetMine"},
		{method: http.MethodPatch, target: minePath + "/selected-account", body: `{"selectedAccountId":"` + accountID + `"}`, wantCall: "SetSelectedAccount"},
		{method: http.MethodPatch, target: minePath + "/selected-bond-bound", body: `{"selectedBondBoundId":null}`, wantCall: "SetSelectedBondBound"},
		{method: http.MethodPatch, target: minePath + "/language", body: `{"language":"en"}`, wantCall: "SetLanguage"},
	}
	for _, route := range mine {
		t.Run("inversionista "+route.method+" "+route.target, func(t *testing.T) {
			app := &fakeApp{setting: sample(), detail: &bondUserSetting_domain.BondUserSettingDetail{BondUserSetting: *sample()}}

			rec := serve(newTestAPI(app, investor), route.method, route.target, route.body)

			if rec.Code != http.StatusOK {
				t.Fatalf("status %d, se esperaba 200 (body %s)", rec.Code, rec.Body)
			}
			if !slices.Equal(app.calls, []string{route.wantCall}) {
				t.Fatalf("la aplicación recibió %v, se esperaba [%s]", app.calls, route.wantCall)
			}
		})
	}
}
