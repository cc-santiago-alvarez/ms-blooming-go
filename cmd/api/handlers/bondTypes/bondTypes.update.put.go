package bondTypes_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
)

// Update reemplaza los campos del tipo del path con el body y responde el documento actualizado.
func (handler *BondTypesHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := handlers_shared.PathID(w, r, "id")
	if !ok {
		return
	}
	var params bondType_domain.UpdateParams
	if !handlers_shared.DecodeBody(w, r, &params) {
		return
	}

	t, err := handler.bondTypeApplication.Update(r.Context(), id, params)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, t)
}
