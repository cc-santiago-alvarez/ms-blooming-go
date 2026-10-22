package bondObjects_handlers_test

import (
	"net/http"
	"testing"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
)

// DELETE /bond-objects/{id} responde 200 true; una segunda baja, ya borrado, responde 404
// ERR_BOND_OBJECT_NOT_FOUND. CA-02.9
func TestRemove(t *testing.T) {
	t.Run("baja", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, admin), http.MethodDelete, objectPath, "")

		var body bool
		decodeJSON(t, rec, http.StatusOK, &body)
		if !body {
			t.Fatalf("body %s, se esperaba true", rec.Body)
		}
		if app.id != objectID {
			t.Fatalf("la aplicación recibió el id %q, se esperaba %q", app.id, objectID)
		}
	})

	t.Run("segunda baja", func(t *testing.T) {
		app := &fakeApp{err: bondObject_domain.ErrBondObjectNotFound}

		rec := serve(newTestAPI(app, admin), http.MethodDelete, objectPath, "")

		assertError(t, rec, http.StatusNotFound, "ERR_BOND_OBJECT_NOT_FOUND")
	})
}
