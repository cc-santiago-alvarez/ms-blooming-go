package bondObjects_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
)

// MostPurchased responde los productos más comprados en la agencia de la sesión que ella puede ver.
func (handler *BondObjectsHandler) MostPurchased(w http.ResponseWriter, r *http.Request) {
	objects, err := handler.bondObjectApplication.MostPurchased(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, objects)
}
