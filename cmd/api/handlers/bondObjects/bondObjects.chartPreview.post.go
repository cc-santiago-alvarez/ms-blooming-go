package bondObjects_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
)

// ChartPreview responde las guías de los términos del body y, si los pide, su simulación diaria.
func (handler *BondObjectsHandler) ChartPreview(w http.ResponseWriter, r *http.Request) {
	var params bondObject_domain.ChartPreviewParams
	if !handlers_shared.DecodeBody(w, r, &params) {
		return
	}

	chart, err := handler.bondObjectApplication.ChartPreview(r.Context(), params)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, chart)
}
