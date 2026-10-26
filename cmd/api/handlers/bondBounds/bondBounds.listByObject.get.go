package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
)

// ListByObject responde los bonos del producto del path con el perfil público de cada dueño.
func (handler *BondBoundsHandler) ListByObject(w http.ResponseWriter, r *http.Request) {
	id, ok := handlers_shared.PathID(w, r, "id")
	if !ok {
		return
	}

	bounds, err := handler.bondBoundApplication.ListByObject(r.Context(), id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, bounds)
}
