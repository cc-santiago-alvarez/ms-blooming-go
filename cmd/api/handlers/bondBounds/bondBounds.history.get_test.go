package bondBounds_handlers_test

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// boundHistory es el historial de un bono vendido listo para mostrar: la venta y la compra.
func boundHistory() []bondHistory_domain.HistoryEntry {
	soldAt := time.Date(2026, 10, 12, 15, 0, 0, 0, time.UTC)
	entry := func(id, text string, at time.Time) bondHistory_domain.HistoryEntry {
		return bondHistory_domain.HistoryEntry{
			ID:               id,
			RelatedID:        boundID,
			TypeID:           bondHistory_domain.HistoryBondBound,
			Type:             bondHistory_domain.HistoryTypeRef{ID: bondHistory_domain.HistoryBondBound, Name: "Bono Enlazado"},
			AgencyID:         agencyA,
			Text:             text,
			User:             bondHistory_domain.HistoryUser{FirstName: "Juan", LastName: "Pérez"},
			Icon:             "mdi-briefcase",
			Color:            "#000000",
			BackgroundColor:  "#e5e5e5",
			CreatedAt:        at,
			CreatedAtFromNow: "a day ago",
		}
	}
	return []bondHistory_domain.HistoryEntry{
		entry("h2", "Estado cambiado de Iniciado a Vendido", soldAt),
		entry("h1", "Bono Vendido", soldAt.AddDate(0, 0, -50)),
	}
}

// GET /bond-bounds/{id}/history pasa el id del path a la aplicación y responde 200 con el historial del bono;
// un bono ajeno responde 403 ERR_FORBIDDEN, uno inexistente 404 ERR_BOND_BOUND_NOT_FOUND y un id inválido 400
// ERR_INVALID_ID sin llegar a la aplicación. CA-08.4
func TestHistory(t *testing.T) {
	t.Run("dueño", func(t *testing.T) {
		app := &fakeApp{history: boundHistory()}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundPath+"/history", "")

		var body []map[string]any
		decodeJSON(t, rec, http.StatusOK, &body)
		if len(body) != 2 || body[0]["_id"] != "h2" || body[1]["_id"] != "h1" {
			t.Fatalf("body %s, se esperaban h2 y h1 en ese orden", rec.Body)
		}
		if want := map[string]any{"_id": "2", "name": "Bono Enlazado"}; !reflect.DeepEqual(body[0]["type"], want) {
			t.Errorf("type %v, se esperaba %v", body[0]["type"], want)
		}
		if body[0]["text"] != "Estado cambiado de Iniciado a Vendido" {
			t.Errorf("text %v, se esperaba el de la aplicación", body[0]["text"])
		}
		if !slices.Equal(app.calls, []string{"HistoryOf"}) || app.id != boundID {
			t.Errorf("la aplicación recibió %v con el id %q, se esperaba HistoryOf con %q", app.calls, app.id, boundID)
		}
	})

	rejected := []struct {
		name   string
		err    error
		status int
	}{
		{name: "bono ajeno", err: shared_domain.ErrForbidden, status: http.StatusForbidden},
		{name: "inexistente", err: bondBound_domain.ErrBondBoundNotFound, status: http.StatusNotFound},
	}
	for _, tt := range rejected {
		t.Run(tt.name, func(t *testing.T) {
			app := &fakeApp{err: tt.err}

			rec := serve(newTestAPI(app, investor), http.MethodGet, boundPath+"/history", "")

			assertError(t, rec, tt.status, tt.err.Error())
		})
	}

	t.Run("id inválido", func(t *testing.T) {
		app := &fakeApp{history: boundHistory()}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundsPath+"/abc/history", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_ID")
		if len(app.calls) != 0 {
			t.Fatalf("la aplicación recibió %v con un id inválido", app.calls)
		}
	})
}
