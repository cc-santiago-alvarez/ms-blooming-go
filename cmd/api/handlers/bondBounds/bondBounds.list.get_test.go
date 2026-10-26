package bondBounds_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// adminPage es la página 2 de 3 del listado de administración, con un bono.
func adminPage() shared_domain.Paginated[bondBound_ports.BondBoundAdminItem] {
	return shared_domain.Paginated[bondBound_ports.BondBoundAdminItem]{
		Items: []bondBound_ports.BondBoundAdminItem{{
			ID:              boundID,
			Key:             "BP12-58",
			Status:          bondBound_domain.StatusInitiated.Ref(),
			Amount:          1000,
			Profile:         &bondBound_ports.ProfileName{FirstName: "Juan", LastName: "Pérez"},
			BondType:        bondBound_ports.BondTypeRef{Name: "Plus"},
			BondObject:      bondBound_ports.BondObjectRef{Name: "Bono Plus 12M"},
			Month:           10,
			Year:            2026,
			DaysPassed:      30,
			AcumulateAmount: 94.52,
		}},
		Total:      25,
		Page:       2,
		Limit:      10,
		TotalPages: 3,
	}
}

// GET /bond-bounds arma los filtros y la página con el query y responde 200 con la página que devuelve la
// aplicación; sin sesión admin responde 403 ERR_FORBIDDEN sin llegar a la aplicación. CA-06.1
func TestList(t *testing.T) {
	t.Run("admin", func(t *testing.T) {
		app := &fakeApp{adminPage: adminPage()}

		rec := serve(newTestAPI(app, admin), http.MethodGet,
			boundsPath+"?page=2&limit=10&statusId=1&userId="+userID+"&search=juan&filter=aMonth&month=10&year=2026", "")

		var body struct {
			Items      []map[string]any `json:"items"`
			Total      int              `json:"total"`
			Page       int              `json:"page"`
			Limit      int              `json:"limit"`
			TotalPages int              `json:"totalPages"`
		}
		decodeJSON(t, rec, http.StatusOK, &body)
		if body.Total != 25 || body.Page != 2 || body.Limit != 10 || body.TotalPages != 3 || len(body.Items) != 1 {
			t.Fatalf("body %s, se esperaba la página de la aplicación", rec.Body)
		}
		if body.Items[0]["_id"] != boundID || body.Items[0]["acumulateAmount"] != 94.52 {
			t.Errorf("ítem %v, se esperaba el de la aplicación", body.Items[0])
		}

		want := bondBound_domain.ListFilters{
			StatusID: bondBound_domain.StatusInitiated,
			UserID:   userID,
			Search:   "juan",
			Date:     bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterAMonth, Month: 10, Year: 2026},
		}
		if !slices.Equal(app.calls, []string{"List"}) || app.listFilters != want {
			t.Errorf("la aplicación recibió %v con %+v, se esperaba List con %+v", app.calls, app.listFilters, want)
		}
		if wantPage := (shared_domain.Pagination{Page: 2, Limit: 10}); app.page != wantPage {
			t.Errorf("página %+v, se esperaba %+v", app.page, wantPage)
		}
	})

	t.Run("no admin", func(t *testing.T) {
		app := &fakeApp{adminPage: adminPage()}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundsPath, "")

		assertForbiddenAdmin(t, rec, app)
	})
}

// El filtro de fecha se lee del query tal cual llega; un mes o un año que no es entero queda en 0 para que
// lo rechace el caso de uso. CA-06.3
func TestListDateFilterQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  bondBound_domain.DateFilter
	}{
		{name: "sin filtro", query: "", want: bondBound_domain.DateFilter{}},
		{name: "aDate", query: "?filter=aDate&aDate=2026-10-13", want: bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterADate, ADate: "2026-10-13"}},
		{
			name:  "betweenDates",
			query: "?filter=betweenDates&startDate=2026-10-14&endDate=2026-10-13",
			want:  bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterBetweenDates, StartDate: "2026-10-14", EndDate: "2026-10-13"},
		},
		{name: "aYear", query: "?filter=aYear&year=2026", want: bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterAYear, Year: 2026}},
		{name: "mes que no es entero", query: "?filter=aMonth&month=oct&year=2026", want: bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterAMonth, Year: 2026}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &fakeApp{adminPage: adminPage()}

			rec := serve(newTestAPI(app, admin), http.MethodGet, boundsPath+tt.query, "")

			if rec.Code != http.StatusOK {
				t.Fatalf("status %d, se esperaba 200 (body %s)", rec.Code, rec.Body)
			}
			if app.listFilters.Date != tt.want {
				t.Errorf("filtro de fecha %+v, se esperaba %+v", app.listFilters.Date, tt.want)
			}
		})
	}
}

// Un filtro rechazado por el caso de uso responde 400 con su código: la búsqueda de más de 100 caracteres,
// el filtro de fecha inválido, el estado desconocido y el userId inválido. CA-06.2, CA-06.3
func TestListInvalidFilters(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "búsqueda larga", err: bondBound_domain.ErrInvalidSearch},
		{name: "filtro de fecha inválido", err: bondBound_domain.ErrInvalidDateFilter},
		{name: "estado desconocido", err: bondBound_domain.ErrInvalidStatus},
		{name: "userId inválido", err: shared_domain.ErrInvalidID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &fakeApp{err: tt.err}

			rec := serve(newTestAPI(app, admin), http.MethodGet, boundsPath, "")

			assertError(t, rec, http.StatusBadRequest, tt.err.Error())
		})
	}
}
