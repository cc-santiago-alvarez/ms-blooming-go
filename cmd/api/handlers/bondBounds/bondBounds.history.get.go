package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
)

// History responde el historial del bono del path en la agencia de la sesión, del más reciente al más antiguo y
// listo para mostrar, si es del usuario de la sesión o la sesión es admin.
func (handler *BondBoundsHandler) History(w http.ResponseWriter, r *http.Request) {
	id, ok := handlers_shared.PathID(w, r, "id")
	if !ok {
		return
	}

	entries, err := handler.bondBoundApplication.HistoryOf(r.Context(), id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, entries)
}
