package bondTypes_handlers_test

import (
	"net/http"
	"testing"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
)

// DELETE /bond-types/{id} de un tipo con bondObjects no borrados responde 409
// ERR_BOND_TYPE_HAS_BOND_OBJECTS; si la baja procede responde 200 true. CA-01.7
func TestRemove(t *testing.T) {
	t.Run("con bondObjects no borrados", func(t *testing.T) {
		app := &fakeApp{err: bondType_domain.ErrBondTypeHasBondObjects}

		rec := serve(newTestAPI(app, admin), http.MethodDelete, typePath, "")

		assertError(t, rec, http.StatusConflict, "ERR_BOND_TYPE_HAS_BOND_OBJECTS")
	})

	t.Run("baja", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, admin), http.MethodDelete, typePath, "")

		var body bool
		decodeJSON(t, rec, http.StatusOK, &body)
		if !body {
			t.Fatalf("body %s, se esperaba true", rec.Body)
		}
		if app.id != typeID {
			t.Fatalf("la aplicación recibió el id %q, se esperaba %q", app.id, typeID)
		}
	})
}
