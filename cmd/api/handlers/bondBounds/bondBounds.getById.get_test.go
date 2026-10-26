package bondBounds_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// GET /bond-bounds/{id} pasa el id del path a la aplicación y responde 200 con el bono; un bono ajeno
// responde 403 ERR_FORBIDDEN, uno inexistente 404 ERR_BOND_BOUND_NOT_FOUND, y un id inválido 400
// ERR_INVALID_ID sin llegar a la aplicación. CA-06.10
func TestGetByID(t *testing.T) {
	ids := []struct {
		name string
		id   string
	}{
		{name: "dueño", id: boundID},
		{name: "id legacy", id: legacyBoundID},
	}
	for _, tt := range ids {
		t.Run(tt.name, func(t *testing.T) {
			app := &fakeApp{bound: &bondBound_domain.BondBound{ID: tt.id, Key: "BP12-1", UserID: userID, StatusID: bondBound_domain.StatusInitiated}}

			rec := serve(newTestAPI(app, investor), http.MethodGet, boundsPath+"/"+tt.id, "")

			var body map[string]any
			decodeJSON(t, rec, http.StatusOK, &body)
			if body["_id"] != tt.id || body["key"] != "BP12-1" || body["userId"] != userID {
				t.Fatalf("body %s, se esperaba el bono que devolvió la aplicación", rec.Body)
			}
			if !slices.Equal(app.calls, []string{"GetByID"}) || app.id != tt.id {
				t.Errorf("la aplicación recibió %v con el id %q, se esperaba GetByID con %q", app.calls, app.id, tt.id)
			}
		})
	}

	rejected := []struct {
		name   string
		err    error
		status int
	}{
		{name: "bono ajeno", err: shared_domain.ErrForbidden, status: http.StatusForbidden},
		{name: "inexistente", err: bondBound_domain.ErrBondBoundNotFound, status: http.StatusNotFound},
	}
	for _, tt := range rejected {
		t.Run(tt.name, func(t *testing.T) {
			app := &fakeApp{err: tt.err}

			rec := serve(newTestAPI(app, investor), http.MethodGet, boundPath, "")

			assertError(t, rec, tt.status, tt.err.Error())
		})
	}

	t.Run("id inválido", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundsPath+"/abc", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_ID")
		if len(app.calls) != 0 {
			t.Fatalf("la aplicación recibió %v con un id inválido", app.calls)
		}
	})
}
