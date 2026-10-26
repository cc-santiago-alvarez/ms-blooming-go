package bondBounds_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// purchasedFileName es el nombre del reporte de comprados que devuelve la aplicación.
const purchasedFileName = "reporte-purchased-bonds_2026-10-26_160000.xlsx"

// GET /bond-bounds/reports/purchased arma los filtros con el query y responde 200 con el archivo de la
// aplicación como descarga de Excel; sin sesión admin responde 403 ERR_FORBIDDEN sin llegar a la aplicación.
// CA-07.1, CA-07.7
func TestReportPurchased(t *testing.T) {
	t.Run("admin", func(t *testing.T) {
		app := &fakeApp{reportFile: reportFile(purchasedFileName)}

		rec := serve(newTestAPI(app, admin), http.MethodGet,
			reportPurchasedPath+"?statusId=0&userId="+userID+"&search=plus&filter=aMonth&month=10&year=2026", "")

		assertReportResponse(t, rec, app.reportFile)
		want := bondBound_domain.ReportFilters{
			StatusID: bondBound_domain.ReportAllStatuses,
			UserID:   userID,
			Search:   "plus",
			Date:     bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterAMonth, Month: 10, Year: 2026},
		}
		if !slices.Equal(app.calls, []string{"ReportPurchased"}) || app.reportFilters != want {
			t.Errorf("la aplicación recibió %v con %+v, se esperaba ReportPurchased con %+v", app.calls, app.reportFilters, want)
		}
	})

	t.Run("no admin", func(t *testing.T) {
		app := &fakeApp{reportFile: reportFile(purchasedFileName)}

		rec := serve(newTestAPI(app, investor), http.MethodGet, reportPurchasedPath, "")

		assertForbiddenAdmin(t, rec, app)
	})
}

// Si ms-exporter falla, responde 502 ERR_EXPORT_FAILED en JSON, sin detalles internos. CA-07.4
func TestReportPurchasedExportFailed(t *testing.T) {
	app := &fakeApp{err: shared_domain.ErrExportFailed}

	rec := serve(newTestAPI(app, admin), http.MethodGet, reportPurchasedPath, "")

	assertExportFailed(t, rec)
}

// Un filtro rechazado por el caso de uso responde 400 con su código. CA-07.1
func TestReportPurchasedInvalidFilters(t *testing.T) {
	for _, err := range []error{
		bondBound_domain.ErrInvalidStatus,
		bondBound_domain.ErrInvalidSearch,
		bondBound_domain.ErrInvalidDateFilter,
		shared_domain.ErrInvalidID,
	} {
		t.Run(err.Error(), func(t *testing.T) {
			app := &fakeApp{err: err}

			rec := serve(newTestAPI(app, admin), http.MethodGet, reportPurchasedPath, "")

			assertError(t, rec, http.StatusBadRequest, err.Error())
		})
	}
}
