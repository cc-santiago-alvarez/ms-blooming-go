package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
)

// GetByID responde el bono del path, si es del usuario de la sesión o la sesión es admin.
func (handler *BondBoundsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := handlers_shared.PathID(w, r, "id")
	if !ok {
		return
	}

	bound, err := handler.bondBoundApplication.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, bound)
}
