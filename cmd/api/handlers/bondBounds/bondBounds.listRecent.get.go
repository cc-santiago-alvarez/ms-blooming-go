package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
)

// ListRecent responde los bonos iniciados o madurados más recientes del usuario de la sesión.
func (handler *BondBoundsHandler) ListRecent(w http.ResponseWriter, r *http.Request) {
	bounds, err := handler.bondBoundApplication.ListRecent(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, bounds)
}
