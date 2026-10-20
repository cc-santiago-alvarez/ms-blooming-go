package bondTypes_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
)

// ListActive responde los tipos activos de la agencia de la sesión con su cantidad de bonos.
func (handler *BondTypesHandler) ListActive(w http.ResponseWriter, r *http.Request) {
	types, err := handler.bondTypeApplication.ListActive(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, types)
}
