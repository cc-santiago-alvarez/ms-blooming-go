package bondObjects_handlers_test

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
)

// historyEntries es el historial de un producto listo para mostrar: la última edición, de un usuario, y el
// alta, del sistema.
func historyEntries() []bondHistory_domain.HistoryEntry {
	createdAt := time.Date(2026, 10, 12, 15, 0, 0, 0, time.UTC)
	entry := func(id, text string, user bondHistory_domain.HistoryUser, at time.Time) bondHistory_domain.HistoryEntry {
		return bondHistory_domain.HistoryEntry{
			ID:               id,
			RelatedID:        objectID,
			TypeID:           bondHistory_domain.HistoryBond,
			Type:             bondHistory_domain.HistoryTypeRef{ID: bondHistory_domain.HistoryBond, Name: "Bono"},
			AgencyID:         agencyA,
			Text:             text,
			User:             user,
			Icon:             "mdi-briefcase",
			Color:            "#000000",
			BackgroundColor:  "#e5e5e5",
			CreatedAt:        at,
			CreatedAtFromNow: "a day ago",
		}
	}
	return []bondHistory_domain.HistoryEntry{
		entry("h2", "Bono Actualizado", bondHistory_domain.HistoryUser{FirstName: "Laura", LastName: "Gómez"}, createdAt),
		entry("h1", "Bono Añadido", bondHistory_domain.HistoryUser{System: true}, createdAt.Add(-time.Hour)),
	}
}

// GET /bond-objects/{id}/history responde 200, a un admin, con el historial del producto del path en el orden
// de la aplicación y sin params ni userId; un no admin recibe 403 ERR_FORBIDDEN y un id inválido 400
// ERR_INVALID_ID, sin llegar al historial. CA-08.3
func TestHistory(t *testing.T) {
	t.Run("admin", func(t *testing.T) {
		history := &fakeHistory{entries: historyEntries()}

		rec := serve(newTestAPIWithHistory(&fakeApp{}, history, admin), http.MethodGet, objectPath+"/history", "")

		var body []map[string]any
		decodeJSON(t, rec, http.StatusOK, &body)
		if len(body) != 2 || body[0]["_id"] != "h2" || body[1]["_id"] != "h1" {
			t.Fatalf("body %s, se esperaban h2 y h1 en ese orden", rec.Body)
		}
		if want := map[string]any{"_id": "1", "name": "Bono"}; !reflect.DeepEqual(body[0]["type"], want) {
			t.Errorf("type %v, se esperaba %v", body[0]["type"], want)
		}
		if want := map[string]any{"name": "System"}; !reflect.DeepEqual(body[1]["user"], want) {
			t.Errorf("user del alta %v, se esperaba %v", body[1]["user"], want)
		}
		for _, e := range body {
			for _, key := range []string{"params", "userId"} {
				if _, ok := e[key]; ok {
					t.Errorf("el registro %v tiene %q", e["_id"], key)
				}
			}
		}
		if !slices.Equal(history.calls, []string{"ListByRelated"}) || history.typeID != bondHistory_domain.HistoryBond ||
			history.relatedID != objectID {
			t.Errorf("el historial recibió %v con %q y %q, se esperaba ListByRelated con \"1\" y %q",
				history.calls, history.typeID, history.relatedID, objectID)
		}
	})

	t.Run("id legacy", func(t *testing.T) {
		history := &fakeHistory{entries: historyEntries()}

		rec := serve(newTestAPIWithHistory(&fakeApp{}, history, admin), http.MethodGet, objectsPath+"/"+legacyObjectID+"/history", "")

		var body []map[string]any
		decodeJSON(t, rec, http.StatusOK, &body)
		if history.relatedID != legacyObjectID || len(body) != 2 {
			t.Errorf("el historial recibió %q y el body es %s, se esperaba %q y los 2 registros", history.relatedID, rec.Body, legacyObjectID)
		}
	})

	t.Run("no admin", func(t *testing.T) {
		history := &fakeHistory{entries: historyEntries()}

		rec := serve(newTestAPIWithHistory(&fakeApp{}, history, investor), http.MethodGet, objectPath+"/history", "")

		assertError(t, rec, http.StatusForbidden, "ERR_FORBIDDEN")
		if len(history.calls) != 0 {
			t.Errorf("el historial recibió %v sin adminBlooming", history.calls)
		}
	})

	t.Run("id inválido", func(t *testing.T) {
		history := &fakeHistory{entries: historyEntries()}

		rec := serve(newTestAPIWithHistory(&fakeApp{}, history, admin), http.MethodGet, objectsPath+"/abc/history", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_ID")
		if len(history.calls) != 0 {
			t.Errorf("el historial recibió %v con un id inválido", history.calls)
		}
	})
}
