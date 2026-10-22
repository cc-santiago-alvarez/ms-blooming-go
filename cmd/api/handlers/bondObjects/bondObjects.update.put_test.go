package bondObjects_handlers_test

import (
	"net/http"
	"strings"
	"testing"
)

// Un PUT que cambia amount a 2000 e inflectionPoint a 30 llega a la aplicación con el id del path y esos
// valores, y responde 200 con el documento actualizado. CA-02.6
func TestUpdate(t *testing.T) {
	updated := sample()
	updated.Amount = 2000
	updated.EarlyCashTerm.InflectionPoint = 30
	app := &fakeApp{object: updated}
	body := strings.NewReplacer(`"amount":1000`, `"amount":2000`, `"inflectionPoint":90`, `"inflectionPoint":30`).Replace(validBody)

	rec := serve(newTestAPI(app, admin), http.MethodPut, objectPath, body)

	var got struct {
		ID            string  `json:"_id"`
		Amount        float64 `json:"amount"`
		EarlyCashTerm struct {
			InflectionPoint int `json:"inflectionPoint"`
		} `json:"earlyCashTerm"`
	}
	decodeJSON(t, rec, http.StatusOK, &got)
	if got.ID != objectID || got.Amount != 2000 || got.EarlyCashTerm.InflectionPoint != 30 {
		t.Errorf("body %s, se esperaba el producto %s actualizado", rec.Body, objectID)
	}
	if app.id != objectID {
		t.Fatalf("la aplicación recibió el id %q, se esperaba %q", app.id, objectID)
	}
	if app.params.Amount != 2000 || app.params.EarlyCashTerm.InflectionPoint != 30 {
		t.Errorf("la aplicación recibió amount %v e inflectionPoint %d, se esperaban 2000 y 30",
			app.params.Amount, app.params.EarlyCashTerm.InflectionPoint)
	}
}
