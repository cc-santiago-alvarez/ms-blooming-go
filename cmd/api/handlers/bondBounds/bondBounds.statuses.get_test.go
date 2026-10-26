package bondBounds_handlers_test

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
)

// GET /bond-bounds/statuses responde 200 con los seis estados en orden, como {_id, name}, sin tomar
// "statuses" como el id de un bono. CA-06.6
func TestStatuses(t *testing.T) {
	app := &fakeApp{}

	rec := serve(newTestAPI(app, investor), http.MethodGet, boundsPath+"/statuses", "")

	var body []map[string]any
	decodeJSON(t, rec, http.StatusOK, &body)
	want := []map[string]any{
		{"_id": "1", "name": "Iniciado"},
		{"_id": "2", "name": "Madurado"},
		{"_id": "3", "name": "Bloqueado"},
		{"_id": "4", "name": "Desligado"},
		{"_id": "5", "name": "Vendido"},
		{"_id": "6", "name": "Canjeado"},
	}
	if !reflect.DeepEqual(body, want) {
		t.Errorf("body %s, se esperaban los 6 estados en orden", rec.Body)
	}
	if !slices.Equal(app.calls, []string{"Statuses"}) {
		t.Errorf("la aplicación recibió %v, se esperaba Statuses", app.calls)
	}
}
