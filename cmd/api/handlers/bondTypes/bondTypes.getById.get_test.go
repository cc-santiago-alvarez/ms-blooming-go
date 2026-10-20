package bondTypes_handlers_test

import (
	"net/http"
	"testing"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
)

// GET /bond-types/{id} de un tipo borrado o inexistente responde 404 ERR_BOND_TYPE_NOT_FOUND; un id
// inválido es 400 sin llegar a la aplicación. CA-01.4
func TestGetByIDNotFound(t *testing.T) {
	t.Run("no encontrado", func(t *testing.T) {
		app := &fakeApp{err: bondType_domain.ErrBondTypeNotFound}

		rec := serve(newTestAPI(app, admin), http.MethodGet, typePath, "")

		assertError(t, rec, http.StatusNotFound, "ERR_BOND_TYPE_NOT_FOUND")
		if app.id != typeID {
			t.Fatalf("la aplicación recibió el id %q, se esperaba %q", app.id, typeID)
		}
	})

	t.Run("id inválido", func(t *testing.T) {
		app := &fakeApp{bondType: sample()}

		rec := serve(newTestAPI(app, admin), http.MethodGet, typesPath+"/abc", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_ID")
		if len(app.calls) != 0 {
			t.Fatalf("la aplicación recibió %v con un id inválido", app.calls)
		}
	})

	t.Run("existente", func(t *testing.T) {
		app := &fakeApp{bondType: sample()}

		rec := serve(newTestAPI(app, admin), http.MethodGet, typePath, "")

		var body map[string]any
		decodeJSON(t, rec, http.StatusOK, &body)
		if body["_id"] != typeID || body["name"] != "Bono Plus" {
			t.Fatalf("body %s, se esperaba el tipo %s", rec.Body, typeID)
		}
	})
}
