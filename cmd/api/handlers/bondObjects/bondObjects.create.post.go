package bondObjects_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
)

// Create crea el producto del body y responde 201 con el documento.
func (handler *BondObjectsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var params bondObject_domain.CreateParams
	if !handlers_shared.DecodeBody(w, r, &params) {
		return
	}

	o, err := handler.bondObjectApplication.Create(r.Context(), params)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusCreated, o)
}
