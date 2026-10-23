package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// BulkPurchase compra la cantidad de bonos del body del producto del body con la cuenta del body y responde
// 201 con los bonos, el total y el saldo que le queda a la cuenta.
func (handler *BondBoundsHandler) BulkPurchase(w http.ResponseWriter, r *http.Request) {
	var params bondBound_domain.BulkPurchaseParams
	if !handlers_shared.DecodeBody(w, r, &params) {
		return
	}

	result, err := handler.bondBoundApplication.BulkPurchase(r.Context(), params)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusCreated, result)
}
