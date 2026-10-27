package bondBounds_handlers_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// sampleFullChart son las guías de sampleChart con tres puntos de la evolución: el día de compra, hoy con
// los valores del momento y el vencimiento con la ganancia final.
func sampleFullChart() bondBound_domain.BondBoundChart {
	chart := sampleChart()
	v := func(f float64) *float64 { return &f }
	chart.Data = []shared_domain.BoundChartPoint{
		{Date: "2026-01-01", Inversion: v(1000), Ganancia: v(1000), Perdida: v(1000)},
		{Date: "2026-03-12", Inversion: v(1015.51), Perdida: v(1003.88), AtMoment: v(1015.51),
			PerdidaAtMoment: v(1003.88), BulletClass: "lastBullet"},
		{Date: "2027-01-01", Ganancia: v(1150), GananciaFinal: v(1150)},
	}
	return chart
}

// GET /bond-bounds/{id}/chart responde 200 con las guías y la evolución, también a un admin y con un id
// legacy; un bono ajeno responde 403 ERR_FORBIDDEN, uno inexistente o borrado 404 ERR_BOND_BOUND_NOT_FOUND,
// y un id inválido 400 ERR_INVALID_ID sin llegar a la aplicación. CA-03.16
func TestChart(t *testing.T) {
	t.Run("dueño", func(t *testing.T) {
		app := &fakeApp{chart: sampleFullChart()}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundPath+"/chart", "")

		var body struct {
			Guides json.RawMessage  `json:"guides"`
			Data   []map[string]any `json:"data"`
		}
		decodeJSON(t, rec, http.StatusOK, &body)
		if !slices.Equal(app.calls, []string{"Chart"}) || app.id != boundID {
			t.Fatalf("la aplicación recibió %v con el id %q, se esperaba Chart con %q", app.calls, app.id, boundID)
		}
		if len(body.Data) != 3 {
			t.Fatalf("body %s, se esperaban los 3 puntos que devolvió la aplicación", rec.Body)
		}
		for i, p := range body.Data {
			for key, v := range p {
				if _, ok := v.(float64); !ok && key != "date" && key != "bulletClass" {
					t.Errorf("día %d, %s = %#v, se esperaba un número", i, key, v)
				}
			}
		}
		if today := body.Data[1]; today["atMoment"] != 1015.51 || today["bulletClass"] != "lastBullet" {
			t.Errorf("el punto de hoy es %v, se esperaba atMoment 1015.51 y lastBullet", today)
		}

		guides := serve(newTestAPI(app, investor), http.MethodGet, boundPath+"/chart-guides", "")
		var guidesBody struct {
			Guides json.RawMessage `json:"guides"`
		}
		decodeJSON(t, guides, http.StatusOK, &guidesBody)
		if string(body.Guides) != string(guidesBody.Guides) {
			t.Errorf("guías %s, /chart-guides responde %s", body.Guides, guidesBody.Guides)
		}
	})

	t.Run("admin que no es dueño", func(t *testing.T) {
		app := &fakeApp{chart: sampleFullChart()}

		rec := serve(newTestAPI(app, admin), http.MethodGet, boundPath+"/chart", "")

		var body bondBound_domain.BondBoundChart
		decodeJSON(t, rec, http.StatusOK, &body)
		if !slices.Equal(app.calls, []string{"Chart"}) || len(body.Guides) != 2 || len(body.Data) != 3 {
			t.Fatalf("la aplicación recibió %v y el body es %s, se esperaba Chart con las guías y la evolución",
				app.calls, rec.Body)
		}
	})

	t.Run("id legacy", func(t *testing.T) {
		app := &fakeApp{chart: sampleFullChart()}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundsPath+"/"+legacyBoundID+"/chart", "")

		var body bondBound_domain.BondBoundChart
		decodeJSON(t, rec, http.StatusOK, &body)
		if app.id != legacyBoundID || len(body.Guides) != 2 || len(body.Data) != 3 {
			t.Fatalf("la aplicación recibió el id %q y el body es %s, se esperaba %q con las guías y la evolución",
				app.id, rec.Body, legacyBoundID)
		}
	})

	t.Run("bono ajeno", func(t *testing.T) {
		app := &fakeApp{err: shared_domain.ErrForbidden}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundPath+"/chart", "")

		assertError(t, rec, http.StatusForbidden, "ERR_FORBIDDEN")
	})

	t.Run("inexistente o borrado", func(t *testing.T) {
		app := &fakeApp{err: bondBound_domain.ErrBondBoundNotFound}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundPath+"/chart", "")

		assertError(t, rec, http.StatusNotFound, "ERR_BOND_BOUND_NOT_FOUND")
	})

	t.Run("id inválido", func(t *testing.T) {
		app := &fakeApp{chart: sampleFullChart()}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundsPath+"/abc/chart", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_ID")
		if len(app.calls) != 0 {
			t.Fatalf("la aplicación recibió %v con un id inválido", app.calls)
		}
	})
}
