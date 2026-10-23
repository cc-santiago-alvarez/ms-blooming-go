package bondBound_domain

import (
	"strings"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// PurchaseParams es el body de la compra de un bono. El usuario y la agencia salen de la sesión.
type PurchaseParams struct {
	AccountID    string `json:"accountId"`
	BondObjectID string `json:"bondObjectId"`
}

// Normalize recorta accountId y bondObjectId.
func (p *PurchaseParams) Normalize() {
	p.AccountID = strings.TrimSpace(p.AccountID)
	p.BondObjectID = strings.TrimSpace(p.BondObjectID)
}

// Validation exige accountId y bondObjectId, y que los dos sean ids válidos.
func (p PurchaseParams) Validation() error {
	switch {
	case p.AccountID == "":
		return ErrAccountIDRequired
	case p.BondObjectID == "":
		return ErrBondObjectIDRequired
	case !shared_domain.IsValidID(p.AccountID) || !shared_domain.IsValidID(p.BondObjectID):
		return shared_domain.ErrInvalidID
	}
	return nil
}

// BulkPurchaseParams es el body de la compra masiva: la misma cuenta y producto, y cuántos bonos comprar.
type BulkPurchaseParams struct {
	PurchaseParams
	Quantity int `json:"quantity"`
}

// Validation exige lo mismo que la compra individual y una cantidad entre 1 y maxQuantity.
func (p BulkPurchaseParams) Validation(maxQuantity int) error {
	if err := p.PurchaseParams.Validation(); err != nil {
		return err
	}
	if p.Quantity < 1 || p.Quantity > maxQuantity {
		return ErrInvalidQuantity
	}
	return nil
}

// PurchaseResult es la respuesta de la compra: el bono creado y el saldo que le queda a la cuenta.
type PurchaseResult struct {
	BondBoundID      string  `json:"bondBoundId"`
	Key              string  `json:"key"`
	AccountAvailable float64 `json:"accountAvailable"`
}

// BulkPurchaseItem es un bono de la compra masiva con la referencia de su movimiento de saldo.
type BulkPurchaseItem struct {
	ID        string `json:"_id"`
	Key       string `json:"key"`
	Reference string `json:"reference"`
}

// BulkPurchaseResult es la respuesta de la compra masiva: los bonos creados, el total cobrado y el saldo
// que le queda a la cuenta.
type BulkPurchaseResult struct {
	Items            []BulkPurchaseItem `json:"items"`
	Total            float64            `json:"total"`
	AccountAvailable float64            `json:"accountAvailable"`
}

// BondBoundChart son las guías de la gráfica de un bono comprado.
type BondBoundChart struct {
	Guides []shared_domain.ChartGuide `json:"guides"`
}
