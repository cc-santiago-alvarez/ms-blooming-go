package bondObjects_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
)

// GET /bond-objects/{id}/chart no exige adminBlooming: un inversionista recibe la gráfica, y un producto
// que no ve responde 404 ERR_BOND_OBJECT_NOT_FOUND. CA-03.7
func TestChartByID(t *testing.T) {
	t.Run("inversionista", func(t *testing.T) {
		app := &fakeApp{chart: sampleChart(false)}

		rec := serve(newTestAPI(app, investor), http.MethodGet, objectPath+"/chart", "")

		var body bondObject_domain.BondObjectChart
		decodeJSON(t, rec, http.StatusOK, &body)
		if !slices.Equal(app.calls, []string{"ChartByID"}) || app.id != objectID {
			t.Fatalf("la aplicación recibió %v con el id %q, se esperaba ChartByID con %q", app.calls, app.id, objectID)
		}
		if len(body.Guides) != 1 {
			t.Errorf("body %s, se esperaba la gráfica que devolvió la aplicación", rec.Body)
		}
	})

	t.Run("producto que no ve", func(t *testing.T) {
		app := &fakeApp{err: bondObject_domain.ErrBondObjectNotFound}

		rec := serve(newTestAPI(app, investor), http.MethodGet, objectPath+"/chart?simulation=true", "")

		assertError(t, rec, http.StatusNotFound, "ERR_BOND_OBJECT_NOT_FOUND")
	})
}

// GET /bond-objects/{id}/chart?simulation=true con un id legacy de 17 caracteres llega a la aplicación con
// ese id y simulation true, y responde con data; un UUID sin el query llega con simulation false. CA-03.9
func TestChartByIDLegacyID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		query      string
		simulation bool
	}{
		{name: "legacy con simulation", id: legacyObjectID, query: "?simulation=true", simulation: true},
		{name: "UUID sin simulation", id: objectID, simulation: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &fakeApp{chart: sampleChart(tt.simulation)}

			rec := serve(newTestAPI(app, investor), http.MethodGet, objectsPath+"/"+tt.id+"/chart"+tt.query, "")

			var body bondObject_domain.BondObjectChart
			decodeJSON(t, rec, http.StatusOK, &body)
			if app.id != tt.id || app.simulation != tt.simulation {
				t.Fatalf("la aplicación recibió el id %q y simulation %v, se esperaban %q y %v",
					app.id, app.simulation, tt.id, tt.simulation)
			}
			if tt.simulation && len(body.Data) != 2 {
				t.Errorf("body %s, se esperaba data con la simulación", rec.Body)
			}
		})
	}
}
