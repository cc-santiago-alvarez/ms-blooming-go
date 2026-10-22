package bondObjects_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
)

// Latest responde los últimos productos publicados que ve la sesión.
func (handler *BondObjectsHandler) Latest(w http.ResponseWriter, r *http.Request) {
	objects, err := handler.bondObjectApplication.Latest(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, objects)
}
