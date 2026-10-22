package bondObjects_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
)

// ListByType responde los productos activos del tipo del path que ve la sesión, con su fecha de
// maduración.
func (handler *BondObjectsHandler) ListByType(w http.ResponseWriter, r *http.Request) {
	bondTypeID, ok := handlers_shared.PathID(w, r, "bondTypeId")
	if !ok {
		return
	}

	objects, err := handler.bondObjectApplication.ListByType(r.Context(), bondTypeID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, objects)
}
