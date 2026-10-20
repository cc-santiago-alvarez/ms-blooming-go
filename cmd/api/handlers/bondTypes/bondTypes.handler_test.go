package bondTypes_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
)

// Las rutas de administración responden 403 ERR_FORBIDDEN a una sesión sin adminBlooming sin llegar a la
// aplicación; el listado de activos queda abierto a cualquier usuario. CA-01.2
func TestBondTypesRequireAdminBlooming(t *testing.T) {
	routes := []struct {
		method string
		target string
		body   string
	}{
		{method: http.MethodGet, target: typesPath},
		{method: http.MethodGet, target: typePath},
		{method: http.MethodPost, target: typesPath, body: validBody},
		{method: http.MethodPut, target: typePath, body: validBody},
		{method: http.MethodDelete, target: typePath},
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
				app := &fakeApp{bondType: sample()}

				rec := serve(newTestAPI(app, s.session), route.method, route.target, route.body)

				assertError(t, rec, http.StatusForbidden, "ERR_FORBIDDEN")
				if len(app.calls) != 0 {
					t.Fatalf("la aplicación recibió %v sin adminBlooming", app.calls)
				}
			})
		}
	}

	t.Run("listado de activos sin adminBlooming", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, investor), http.MethodGet, typesPath+"/active", "")

		if rec.Code != http.StatusOK {
			t.Fatalf("status %d, se esperaba 200 (body %s)", rec.Code, rec.Body)
		}
		if !slices.Equal(app.calls, []string{"ListActive"}) {
			t.Fatalf("la aplicación recibió %v, se esperaba [ListActive]", app.calls)
		}
	})
}
