package bondBounds_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// GET /bond-bounds/kpis/count-by-status no exige admin: arma los filtros con el query y responde 200 con el
// total, la cantidad y el estado de la aplicación. Si la aplicación rechaza el usuario pedido responde 403, y un
// filtro inválido, 400. CA-07.6
func TestCountByStatus(t *testing.T) {
	t.Run("no admin", func(t *testing.T) {
		app := &fakeApp{statusCount: bondBound_ports.StatusCount{Total: 125000, Count: 120, Status: "Iniciado"}}

		rec := serve(newTestAPI(app, investor), http.MethodGet,
			countByStatusPath+"?statusId=1&userId="+userID+"&search=plus&filter=betweenDates&startDate=2026-01-25&endDate=2026-02-05", "")

		var body map[string]any
		decodeJSON(t, rec, http.StatusOK, &body)
		if len(body) != 3 || body["total"] != 125000.0 || body["count"] != 120.0 || body["status"] != "Iniciado" {
			t.Errorf("body %s, se esperaba {total: 125000, count: 120, status: Iniciado}", rec.Body)
		}
		want := bondBound_domain.KpiFilters{
			StatusID: bondBound_domain.StatusInitiated,
			UserID:   userID,
			Search:   "plus",
			Date:     bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterBetweenDates, StartDate: "2026-01-25", EndDate: "2026-02-05"},
		}
		if !slices.Equal(app.calls, []string{"CountByStatus"}) || app.kpiFilters != want {
			t.Errorf("la aplicación recibió %v con %+v, se esperaba CountByStatus con %+v", app.calls, app.kpiFilters, want)
		}
	})

	t.Run("userId ajeno", func(t *testing.T) {
		app := &fakeApp{err: shared_domain.ErrForbidden}

		rec := serve(newTestAPI(app, investor), http.MethodGet, countByStatusPath+"?userId=user2AAAAAAAAAAAA", "")

		assertError(t, rec, http.StatusForbidden, "ERR_FORBIDDEN")
	})

	t.Run("filtro de fecha inválido", func(t *testing.T) {
		app := &fakeApp{err: bondBound_domain.ErrInvalidDateFilter}

		rec := serve(newTestAPI(app, investor), http.MethodGet, countByStatusPath+"?filter=aWeek", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_DATE_FILTER")
	})
}
