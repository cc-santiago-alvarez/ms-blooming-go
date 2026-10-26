package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
)

// Timeline responde las compras del userId del query o, sin él, las del usuario de la sesión. Las de otro
// usuario exigen admin; lo valida el caso de uso.
func (handler *BondBoundsHandler) Timeline(w http.ResponseWriter, r *http.Request) {
	items, err := handler.bondBoundApplication.Timeline(r.Context(), r.URL.Query().Get("userId"))
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, items)
}
