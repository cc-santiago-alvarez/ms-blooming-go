package bondBounds_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// buyersPath es la ruta de los compradores del producto.
const buyersPath = "/api/v1/bond-objects/" + objectID + "/bond-bounds"

// GET /bond-objects/{id}/bond-bounds pasa el producto del path a la aplicación y responde 200 con los bonos
// y el perfil público de cada dueño, sin password; sin sesión admin responde 403 ERR_FORBIDDEN sin llegar a
// la aplicación, y un id inválido 400 ERR_INVALID_ID. CA-06.11
func TestListByObject(t *testing.T) {
	t.Run("admin", func(t *testing.T) {
		app := &fakeApp{buyers: []bondBound_ports.BondBoundWithProfile{
			{
				BondBound: bondBound_domain.BondBound{ID: boundID, UserID: userID, BondObjectID: objectID},
				Profile:   &shared_domain.PublicProfile{ID: userID, FirstName: "Juan", LastName: "Pérez", Email: "juan@example.com"},
			},
			{BondBound: bondBound_domain.BondBound{ID: legacyBoundID, UserID: "user2AAAAAAAAAAAA", BondObjectID: objectID}},
		}}

		rec := serve(newTestAPI(app, admin), http.MethodGet, buyersPath, "")

		var body []map[string]any
		decodeJSON(t, rec, http.StatusOK, &body)
		if len(body) != 2 || body[0]["_id"] != boundID || body[1]["_id"] != legacyBoundID {
			t.Fatalf("body %s, se esperaban los bonos que devolvió la aplicación", rec.Body)
		}
		profile, ok := body[0]["profile"].(map[string]any)
		if !ok || profile["first_name"] != "Juan" || profile["email"] != "juan@example.com" {
			t.Fatalf("profile %v, se esperaba el perfil público del dueño", body[0]["profile"])
		}
		if _, ok := profile["password"]; ok {
			t.Errorf("el perfil tiene password: %s", rec.Body)
		}
		if _, ok := body[1]["profile"]; ok {
			t.Errorf("el bono sin perfil tiene profile: %s", rec.Body)
		}
		if !slices.Equal(app.calls, []string{"ListByObject"}) || app.bondObjectID != objectID {
			t.Errorf("la aplicación recibió %v con %q, se esperaba ListByObject con %q", app.calls, app.bondObjectID, objectID)
		}
	})

	t.Run("no admin", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, investor), http.MethodGet, buyersPath, "")

		assertForbiddenAdmin(t, rec, app)
	})

	t.Run("id inválido", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, admin), http.MethodGet, "/api/v1/bond-objects/abc/bond-bounds", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_ID")
		if len(app.calls) != 0 {
			t.Fatalf("la aplicación recibió %v con un id inválido", app.calls)
		}
	})
}
