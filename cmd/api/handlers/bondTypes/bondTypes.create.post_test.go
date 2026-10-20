package bondTypes_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
)

// Un POST válido responde 201 con el documento creado: _id, isRemove false y createdAt en UTC. Sin name
// responde 400 ERR_BOND_TYPE_NAME_REQUIRED, y un body ilegible no llega a la aplicación. CA-01.5
func TestCreate(t *testing.T) {
	t.Run("alta válida", func(t *testing.T) {
		app := &fakeApp{bondType: sample()}

		rec := serve(newTestAPI(app, admin), http.MethodPost, typesPath, validBody)

		var body map[string]any
		decodeJSON(t, rec, http.StatusCreated, &body)
		want := map[string]any{"_id": typeID, "isRemove": false, "createdAt": "2026-01-10T15:00:00Z"}
		for k, w := range want {
			if body[k] != w {
				t.Errorf("%s = %v, se esperaba %v", k, body[k], w)
			}
		}
		p := app.params
		if p.Name != "Bono Plus" || p.Description != "Renta fija 12 meses" || !p.Active ||
			!slices.Equal(p.AgencyID, []string{agencyA, agencyB}) {
			t.Errorf("la aplicación recibió %+v, se esperaba el body del request", p)
		}
	})

	t.Run("sin name", func(t *testing.T) {
		app := &fakeApp{err: bondType_domain.ErrBondTypeNameRequired}

		rec := serve(newTestAPI(app, admin), http.MethodPost, typesPath, `{"agencyId":["`+agencyA+`"]}`)

		assertError(t, rec, http.StatusBadRequest, "ERR_BOND_TYPE_NAME_REQUIRED")
	})

	unreadable := []struct {
		name string
		body string
		code string
	}{
		{name: "sin body", body: "", code: "BODY_REQUIRED"},
		{name: "JSON inválido", body: `{"name":`, code: "INVALID_REQUEST_BODY"},
		{name: "agencyId como string", body: `{"name":"Bono Plus","agencyId":"` + agencyA + `"}`, code: "INVALID_REQUEST_BODY"},
	}
	for _, tt := range unreadable {
		t.Run(tt.name, func(t *testing.T) {
			app := &fakeApp{bondType: sample()}

			rec := serve(newTestAPI(app, admin), http.MethodPost, typesPath, tt.body)

			assertError(t, rec, http.StatusBadRequest, tt.code)
			if len(app.calls) != 0 {
				t.Fatalf("la aplicación recibió %v con un body ilegible", app.calls)
			}
		})
	}
}
