package bondObjects_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
)

// ListGroupedByType responde los productos agrupados por tipo; los agencyId del query solo filtran si la
// agencia de la sesión es la principal.
func (handler *BondObjectsHandler) ListGroupedByType(w http.ResponseWriter, r *http.Request) {
	groups, err := handler.bondObjectApplication.ListGroupedByType(r.Context(), r.URL.Query()["agencyId"])
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, groups)
}
