package bondObjects_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
)

// History responde el historial del producto del path en la agencia de la sesión, del más reciente al más
// antiguo y listo para mostrar.
func (handler *BondObjectsHandler) History(w http.ResponseWriter, r *http.Request) {
	id, ok := handlers_shared.PathID(w, r, "id")
	if !ok {
		return
	}

	entries, err := handler.bondHistoryApplication.ListByRelated(r.Context(), bondHistory_domain.HistoryBond, id)
	if err != nil {
		handlers_shared.ResponseWithError(w, r, bondHistory_domain.MapToHTTPStatus(err), err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, entries)
}
