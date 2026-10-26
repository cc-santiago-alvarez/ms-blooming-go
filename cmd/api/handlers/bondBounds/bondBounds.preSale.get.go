package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
)

// PreSale responde la cotización de la venta anticipada o el canje del bono del path, si es del usuario de la
// sesión o la sesión es admin.
func (handler *BondBoundsHandler) PreSale(w http.ResponseWriter, r *http.Request) {
	id, ok := handlers_shared.PathID(w, r, "id")
	if !ok {
		return
	}

	projection, err := handler.bondBoundApplication.PreSale(r.Context(), id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, projection)
}
