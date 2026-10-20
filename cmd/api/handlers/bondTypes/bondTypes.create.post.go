package bondTypes_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
)

// Create crea el tipo del body y responde 201 con el documento.
func (handler *BondTypesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var params bondType_domain.CreateParams
	if !handlers_shared.DecodeBody(w, r, &params) {
		return
	}

	t, err := handler.bondTypeApplication.Create(r.Context(), params)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusCreated, t)
}
