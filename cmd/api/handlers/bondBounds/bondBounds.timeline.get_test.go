package bondBounds_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// timelinePath es la ruta de la línea de tiempo.
const timelinePath = boundsPath + "/timeline"

// GET /bond-bounds/timeline pasa el userId del query a la aplicación, vacío si no viene, y responde 200 con
// las compras; las de otro usuario sin ser admin responden 403 ERR_FORBIDDEN y un userId inválido 400
// ERR_INVALID_ID. CA-06.12
func TestTimeline(t *testing.T) {
	queries := []struct {
		name     string
		target   string
		wantUser string
	}{
		{name: "sin userId", target: timelinePath, wantUser: ""},
		{name: "con userId", target: timelinePath + "?userId=" + userID, wantUser: userID},
	}
	for _, tt := range queries {
		t.Run(tt.name, func(t *testing.T) {
			app := &fakeApp{timeline: []bondBound_ports.TimelineItem{
				{ID: boundID, Text: "Bono Plus 12M (BP12-1)", Start: "2026-12-31 15:00", End: "2026-12-31 15:00"},
			}}

			rec := serve(newTestAPI(app, investorIn(-5)), http.MethodGet, tt.target, "")

			var body []map[string]any
			decodeJSON(t, rec, http.StatusOK, &body)
			if len(body) != 1 || body[0]["_id"] != boundID || body[0]["text"] != "Bono Plus 12M (BP12-1)" ||
				body[0]["start"] != "2026-12-31 15:00" || body[0]["end"] != "2026-12-31 15:00" {
				t.Fatalf("body %s, se esperaban las compras que devolvió la aplicación", rec.Body)
			}
			if !slices.Equal(app.calls, []string{"Timeline"}) || app.userID != tt.wantUser {
				t.Errorf("la aplicación recibió %v con el userId %q, se esperaba Timeline con %q", app.calls, app.userID, tt.wantUser)
			}
		})
	}

	rejected := []struct {
		name   string
		err    error
		status int
	}{
		{name: "userId ajeno sin ser admin", err: shared_domain.ErrForbidden, status: http.StatusForbidden},
		{name: "userId inválido", err: shared_domain.ErrInvalidID, status: http.StatusBadRequest},
	}
	for _, tt := range rejected {
		t.Run(tt.name, func(t *testing.T) {
			app := &fakeApp{err: tt.err}

			rec := serve(newTestAPI(app, investor), http.MethodGet, timelinePath+"?userId=otro", "")

			assertError(t, rec, tt.status, tt.err.Error())
		})
	}
}
