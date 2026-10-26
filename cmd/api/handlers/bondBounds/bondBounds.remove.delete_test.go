package bondBounds_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// DELETE /bond-bounds/{id} pasa el id del path a la aplicación y responde 200 true; sin sesión admin responde
// 403 ERR_FORBIDDEN sin llegar a la aplicación, un bono inexistente o de otra agencia 404
// ERR_BOND_BOUND_NOT_FOUND y un id inválido 400 ERR_INVALID_ID. CA-06.13
func TestRemove(t *testing.T) {
	t.Run("admin", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, admin), http.MethodDelete, boundPath, "")

		var body bool
		decodeJSON(t, rec, http.StatusOK, &body)
		if !body {
			t.Errorf("body %s, se esperaba true", rec.Body)
		}
		if !slices.Equal(app.calls, []string{"Remove"}) || app.id != boundID {
			t.Errorf("la aplicación recibió %v con el id %q, se esperaba Remove con %q", app.calls, app.id, boundID)
		}
	})

	t.Run("no admin", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, investor), http.MethodDelete, boundPath, "")

		assertForbiddenAdmin(t, rec, app)
	})

	t.Run("inexistente o de otra agencia", func(t *testing.T) {
		app := &fakeApp{err: bondBound_domain.ErrBondBoundNotFound}

		rec := serve(newTestAPI(app, admin), http.MethodDelete, boundPath, "")

		assertError(t, rec, http.StatusNotFound, "ERR_BOND_BOUND_NOT_FOUND")
	})

	t.Run("id inválido", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, admin), http.MethodDelete, boundsPath+"/abc", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_ID")
		if len(app.calls) != 0 {
			t.Fatalf("la aplicación recibió %v con un id inválido", app.calls)
		}
	})
}
