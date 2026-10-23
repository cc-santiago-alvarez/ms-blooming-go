package bondUserSettings_handlers_test

import (
	"net/http"
	"testing"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// GET /bond-user-settings pasa page y limit del query a la aplicación, en 0 si faltan o no son números, y
// responde la página. CA-09.8
func TestList(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  shared_domain.Pagination
	}{
		{name: "page y limit", query: "?page=2&limit=5", want: shared_domain.Pagination{Page: 2, Limit: 5}},
		{name: "no numéricos", query: "?page=dos&limit=x", want: shared_domain.Pagination{}},
		{name: "sin query", query: "", want: shared_domain.Pagination{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := shared_domain.NewPaginated([]*bondUserSetting_domain.BondUserSetting{sample()}, 6, shared_domain.Pagination{Page: 2, Limit: 5})
			app := &fakeApp{page: page}

			rec := serve(newTestAPI(app, admin), http.MethodGet, settingsPath+tt.query, "")

			var got struct {
				Items []struct {
					ID string `json:"_id"`
				} `json:"items"`
				Total      int64 `json:"total"`
				Page       int   `json:"page"`
				Limit      int   `json:"limit"`
				TotalPages int   `json:"totalPages"`
			}
			decodeJSON(t, rec, http.StatusOK, &got)
			if app.pagination != tt.want {
				t.Fatalf("la aplicación recibió %+v, se esperaba %+v", app.pagination, tt.want)
			}
			if len(got.Items) != 1 || got.Items[0].ID != settingID || got.Total != 6 || got.Page != 2 || got.Limit != 5 || got.TotalPages != 2 {
				t.Errorf("body %s, se esperaba la página 2 de 2 con el setting %s", rec.Body, settingID)
			}
		})
	}
}
