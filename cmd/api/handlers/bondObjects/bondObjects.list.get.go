package bondObjects_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
)

// List responde los productos no borrados, filtrados por los agencyId del query.
func (handler *BondObjectsHandler) List(w http.ResponseWriter, r *http.Request) {
	objects, err := handler.bondObjectApplication.List(r.Context(), r.URL.Query()["agencyId"])
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, objects)
}
