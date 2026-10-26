package bondBounds_handlers_test

import (
	"net/http"
	"reflect"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// minePath es la ruta de los bonos del usuario de la sesión.
const minePath = boundsPath + "/me"

// minePage es la página 2 de 3 de los bonos del usuario, con un bono creciendo.
func minePage() shared_domain.Paginated[bondBound_ports.BondBoundMineItem] {
	return shared_domain.Paginated[bondBound_ports.BondBoundMineItem]{
		Items: []bondBound_ports.BondBoundMineItem{{
			ID:                   boundID,
			Key:                  "BP12-1",
			StatusID:             bondBound_domain.StatusInitiated,
			BondObject:           &bondBound_ports.BondObjectRef{Name: "Bono Plus 12M"},
			Growing:              true,
			InitialDateFormatted: "1 Jan 2026",
		}},
		Total:      12,
		Page:       2,
		Limit:      5,
		TotalPages: 3,
	}
}

// GET /bond-bounds/me pasa los statusId del query, uno o varios, y la página a la aplicación, y responde 200
// con la página; "me" no se toma como el id de un bono. CA-06.8
func TestListMine(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		statuses []bondBound_domain.BondBoundStatus
		page     shared_domain.Pagination
	}{
		{
			name:     "varios estados",
			query:    "?statusId=1&statusId=2&page=2&limit=5",
			statuses: []bondBound_domain.BondBoundStatus{bondBound_domain.StatusInitiated, bondBound_domain.StatusFinalized},
			page:     shared_domain.Pagination{Page: 2, Limit: 5},
		},
		{name: "sin estados ni página", query: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &fakeApp{minePage: minePage()}

			rec := serve(newTestAPI(app, investor), http.MethodGet, minePath+tt.query, "")

			var body struct {
				Items      []map[string]any `json:"items"`
				Total      int              `json:"total"`
				TotalPages int              `json:"totalPages"`
			}
			decodeJSON(t, rec, http.StatusOK, &body)
			if body.Total != 12 || body.TotalPages != 3 || len(body.Items) != 1 {
				t.Fatalf("body %s, se esperaba la página de la aplicación", rec.Body)
			}
			item := body.Items[0]
			if want := map[string]any{"name": "Bono Plus 12M"}; !reflect.DeepEqual(item["bondObject"], want) ||
				item["growing"] != true || item["grew"] != false || item["initialDateFormatted"] != "1 Jan 2026" {
				t.Errorf("ítem %v, se esperaba el de la aplicación con bondObject, growing, grew e initialDateFormatted", item)
			}
			if !slices.Equal(app.calls, []string{"ListMine"}) || !slices.Equal(app.mineFilters.StatusIDs, tt.statuses) || app.page != tt.page {
				t.Errorf("la aplicación recibió %v con %v y %+v, se esperaba ListMine con %v y %+v",
					app.calls, app.mineFilters.StatusIDs, app.page, tt.statuses, tt.page)
			}
		})
	}

	t.Run("estado desconocido", func(t *testing.T) {
		app := &fakeApp{err: bondBound_domain.ErrInvalidStatus}

		rec := serve(newTestAPI(app, investor), http.MethodGet, minePath+"?statusId=9", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_STATUS")
	})
}
