package bondBounds_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// userBoundsPath es la ruta de los bonos de userID en el producto objectID.
const userBoundsPath = "/api/v1/users/" + userID + "/bond-bounds?bondObjectId=" + objectID

// GET /users/{userId}/bond-bounds pasa el usuario del path y el producto del query a la aplicación y responde
// 200 con sus bonos; sin sesión admin responde 403 ERR_FORBIDDEN sin llegar a la aplicación. Sin
// bondObjectId responde 400, y un userId inválido 400 ERR_INVALID_ID sin llegar a la aplicación. CA-06.7
func TestListByUserAndObject(t *testing.T) {
	t.Run("admin", func(t *testing.T) {
		app := &fakeApp{bounds: []*bondBound_domain.BondBound{
			{ID: boundID, UserID: userID, BondObjectID: objectID, StatusID: bondBound_domain.StatusInitiated},
			{ID: legacyBoundID, UserID: userID, BondObjectID: objectID, StatusID: bondBound_domain.StatusTook},
		}}

		rec := serve(newTestAPI(app, admin), http.MethodGet, userBoundsPath, "")

		var body []map[string]any
		decodeJSON(t, rec, http.StatusOK, &body)
		if len(body) != 2 || body[0]["_id"] != boundID || body[1]["statusId"] != "6" {
			t.Fatalf("body %s, se esperaban los bonos que devolvió la aplicación", rec.Body)
		}
		if !slices.Equal(app.calls, []string{"ListByUserAndObject"}) || app.userID != userID || app.bondObjectID != objectID {
			t.Errorf("la aplicación recibió %v con %q y %q, se esperaba ListByUserAndObject con %q y %q",
				app.calls, app.userID, app.bondObjectID, userID, objectID)
		}
	})

	t.Run("no admin", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, investor), http.MethodGet, userBoundsPath, "")

		assertForbiddenAdmin(t, rec, app)
	})

	t.Run("sin sesión", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, nil), http.MethodGet, userBoundsPath, "")

		assertForbiddenAdmin(t, rec, app)
	})

	t.Run("sin bondObjectId", func(t *testing.T) {
		app := &fakeApp{err: bondBound_domain.ErrBondObjectIDRequired}

		rec := serve(newTestAPI(app, admin), http.MethodGet, "/api/v1/users/"+userID+"/bond-bounds", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_BOND_OBJECT_ID_REQUIRED")
		if app.bondObjectID != "" {
			t.Errorf("la aplicación recibió bondObjectId %q, se esperaba vacío", app.bondObjectID)
		}
	})

	t.Run("userId inválido", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, admin), http.MethodGet, "/api/v1/users/abc/bond-bounds?bondObjectId="+objectID, "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_ID")
		if len(app.calls) != 0 {
			t.Fatalf("la aplicación recibió %v con un id inválido", app.calls)
		}
	})
}
