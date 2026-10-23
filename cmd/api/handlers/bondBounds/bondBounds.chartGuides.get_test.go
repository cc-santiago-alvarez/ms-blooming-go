package bondBounds_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// GET /bond-bounds/{id}/chart-guides responde 200 con las guías, y la de retiro sale sin toDate ni
// fillColor; un bono ajeno responde 403 ERR_FORBIDDEN, uno inexistente 404 ERR_BOND_BOUND_NOT_FOUND, y un
// id inválido 400 ERR_INVALID_ID sin llegar a la aplicación. CA-03.8
func TestChartGuides(t *testing.T) {
	t.Run("dueño", func(t *testing.T) {
		app := &fakeApp{chart: sampleChart()}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundPath+"/chart-guides", "")

		var body struct {
			Guides []map[string]any `json:"guides"`
		}
		decodeJSON(t, rec, http.StatusOK, &body)
		if !slices.Equal(app.calls, []string{"ChartGuides"}) || app.id != boundID {
			t.Fatalf("la aplicación recibió %v con el id %q, se esperaba ChartGuides con %q", app.calls, app.id, boundID)
		}
		if len(body.Guides) != 2 || body.Guides[0]["date"] != "2026-01-31" || body.Guides[1]["date"] != "2026-01-11" {
			t.Fatalf("body %s, se esperaban las guías que devolvió la aplicación", rec.Body)
		}
		for _, key := range []string{"toDate", "fillColor"} {
			if _, ok := body.Guides[1][key]; ok {
				t.Errorf("la guía de retiro tiene %q: %v", key, body.Guides[1])
			}
		}
	})

	t.Run("id legacy", func(t *testing.T) {
		app := &fakeApp{chart: sampleChart()}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundsPath+"/"+legacyBoundID+"/chart-guides", "")

		var body bondBound_domain.BondBoundChart
		decodeJSON(t, rec, http.StatusOK, &body)
		if app.id != legacyBoundID || len(body.Guides) != 2 {
			t.Fatalf("la aplicación recibió el id %q y el body es %s, se esperaba %q y las 2 guías",
				app.id, rec.Body, legacyBoundID)
		}
	})

	t.Run("bono ajeno", func(t *testing.T) {
		app := &fakeApp{err: shared_domain.ErrForbidden}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundPath+"/chart-guides", "")

		assertError(t, rec, http.StatusForbidden, "ERR_FORBIDDEN")
	})

	t.Run("inexistente", func(t *testing.T) {
		app := &fakeApp{err: bondBound_domain.ErrBondBoundNotFound}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundPath+"/chart-guides", "")

		assertError(t, rec, http.StatusNotFound, "ERR_BOND_BOUND_NOT_FOUND")
	})

	t.Run("id inválido", func(t *testing.T) {
		app := &fakeApp{chart: sampleChart()}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundsPath+"/abc/chart-guides", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_ID")
		if len(app.calls) != 0 {
			t.Fatalf("la aplicación recibió %v con un id inválido", app.calls)
		}
	})
}
