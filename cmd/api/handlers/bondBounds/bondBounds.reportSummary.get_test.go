package bondBounds_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// summaryFileName es el nombre del reporte resumen que devuelve la aplicación.
const summaryFileName = "reporte-summary-purchased-bonds_2026-10-26_160000.xlsx"

// GET /bond-bounds/reports/purchased-summary sin parámetros responde 200 con el archivo de la aplicación como
// descarga de Excel, y de los parámetros solo pasa el statusId. Sin sesión admin responde 403 ERR_FORBIDDEN sin
// llegar a la aplicación. CA-07.3, CA-07.7
func TestReportSummary(t *testing.T) {
	t.Run("sin parámetros", func(t *testing.T) {
		app := &fakeApp{reportFile: reportFile(summaryFileName)}

		rec := serve(newTestAPI(app, admin), http.MethodGet, reportSummaryPath, "")

		assertReportResponse(t, rec, app.reportFile)
		if !slices.Equal(app.calls, []string{"ReportSummary"}) || app.reportFilters != (bondBound_domain.ReportFilters{}) {
			t.Errorf("la aplicación recibió %v con %+v, se esperaba ReportSummary sin filtros", app.calls, app.reportFilters)
		}
	})

	t.Run("solo pasa el statusId", func(t *testing.T) {
		app := &fakeApp{reportFile: reportFile(summaryFileName)}

		rec := serve(newTestAPI(app, admin), http.MethodGet,
			reportSummaryPath+"?statusId=0&userId="+userID+"&search=plus&filter=aYear&year=2026", "")

		assertReportResponse(t, rec, app.reportFile)
		if want := (bondBound_domain.ReportFilters{StatusID: bondBound_domain.ReportAllStatuses}); app.reportFilters != want {
			t.Errorf("filtros %+v, se esperaba %+v", app.reportFilters, want)
		}
	})

	t.Run("no admin", func(t *testing.T) {
		app := &fakeApp{reportFile: reportFile(summaryFileName)}

		rec := serve(newTestAPI(app, investor), http.MethodGet, reportSummaryPath, "")

		assertForbiddenAdmin(t, rec, app)
	})
}

// Si ms-exporter falla, responde 502 ERR_EXPORT_FAILED en JSON, sin detalles internos. CA-07.4
func TestReportSummaryExportFailed(t *testing.T) {
	app := &fakeApp{err: shared_domain.ErrExportFailed}

	rec := serve(newTestAPI(app, admin), http.MethodGet, reportSummaryPath, "")

	assertExportFailed(t, rec)
}
