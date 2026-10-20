package bondTypes_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
)

// PUT /bond-types/{id} pasa el id y los cuatro campos a la aplicación y responde 200 con el documento;
// sobre un tipo inexistente responde 404. CA-01.6
func TestUpdate(t *testing.T) {
	t.Run("tipo existente", func(t *testing.T) {
		app := &fakeApp{bondType: sample()}

		rec := serve(newTestAPI(app, admin), http.MethodPut, typePath, validBody)

		var body map[string]any
		decodeJSON(t, rec, http.StatusOK, &body)
		if body["_id"] != typeID {
			t.Errorf("_id = %v, se esperaba %s", body["_id"], typeID)
		}
		if app.id != typeID {
			t.Errorf("la aplicación recibió el id %q, se esperaba %q", app.id, typeID)
		}
		p := app.params
		if p.Name != "Bono Plus" || p.Description != "Renta fija 12 meses" || !p.Active ||
			!slices.Equal(p.AgencyID, []string{agencyA, agencyB}) {
			t.Errorf("la aplicación recibió %+v, se esperaba el body del request", p)
		}
	})

	t.Run("inexistente", func(t *testing.T) {
		app := &fakeApp{err: bondType_domain.ErrBondTypeNotFound}

		rec := serve(newTestAPI(app, admin), http.MethodPut, typePath, validBody)

		assertError(t, rec, http.StatusNotFound, "ERR_BOND_TYPE_NOT_FOUND")
	})

	t.Run("id inválido", func(t *testing.T) {
		app := &fakeApp{bondType: sample()}

		rec := serve(newTestAPI(app, admin), http.MethodPut, typesPath+"/abc", validBody)

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_ID")
		if len(app.calls) != 0 {
			t.Fatalf("la aplicación recibió %v con un id inválido", app.calls)
		}
	})
}
