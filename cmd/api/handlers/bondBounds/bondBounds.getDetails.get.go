package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
)

// GetDetails responde el detalle del bono del path, si es del usuario de la sesión o la sesión es admin, y
// está en el alcance de agencia de la sesión.
func (handler *BondBoundsHandler) GetDetails(w http.ResponseWriter, r *http.Request) {
	id, ok := handlers_shared.PathID(w, r, "id")
	if !ok {
		return
	}

	detail, err := handler.bondBoundApplication.GetDetails(r.Context(), id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, detail)
}
