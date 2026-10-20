package bondTypes_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
)

// List responde los tipos no borrados, filtrados por los agencyId del query.
func (handler *BondTypesHandler) List(w http.ResponseWriter, r *http.Request) {
	types, err := handler.bondTypeApplication.List(r.Context(), r.URL.Query()["agencyId"])
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, types)
}
