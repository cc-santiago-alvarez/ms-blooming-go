package bondObjects_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
)

// UpdateKeys cambia la clave de los productos del body y responde cuántos encontró y cuántos cambió.
func (handler *BondObjectsHandler) UpdateKeys(w http.ResponseWriter, r *http.Request) {
	var params bondObject_domain.UpdateKeysParams
	if !handlers_shared.DecodeBody(w, r, &params) {
		return
	}

	result, err := handler.bondObjectApplication.UpdateKeys(r.Context(), params)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, result)
}
