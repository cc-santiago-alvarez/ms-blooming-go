package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// Purchase compra un bono del producto del body con la cuenta del body y responde 201 con el bono y el saldo
// que le queda a la cuenta.
func (handler *BondBoundsHandler) Purchase(w http.ResponseWriter, r *http.Request) {
	var params bondBound_domain.PurchaseParams
	if !handlers_shared.DecodeBody(w, r, &params) {
		return
	}

	result, err := handler.bondBoundApplication.Purchase(r.Context(), params)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusCreated, result)
}
