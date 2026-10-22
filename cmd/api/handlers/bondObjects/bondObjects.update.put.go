package bondObjects_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
)

// Update reemplaza los campos del producto del path con el body y responde el documento actualizado.
func (handler *BondObjectsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := handlers_shared.PathID(w, r, "id")
	if !ok {
		return
	}
	var params bondObject_domain.UpdateParams
	if !handlers_shared.DecodeBody(w, r, &params) {
		return
	}

	o, err := handler.bondObjectApplication.Update(r.Context(), id, params)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, o)
}
