package bondBound_application

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// BulkPurchase compra quantity bonos del producto con una cuenta del usuario de la sesión: valida como la
// compra individual, con el saldo contra el total, y crea todos los bonos, sus cobros y su historial en una
// sola transacción. Devuelve cada bono con la referencia de su cobro, el total y el saldo que le queda a la
// cuenta.
func (app *BondBoundApplication) BulkPurchase(ctx context.Context, p bondBound_domain.BulkPurchaseParams) (bondBound_domain.BulkPurchaseResult, error) {
	p.Normalize()
	if err := p.Validation(app.BulkPurchaseMaxQuantity); err != nil {
		return bondBound_domain.BulkPurchaseResult{}, err
	}
	session, err := sessionUser(ctx)
	if err != nil {
		return bondBound_domain.BulkPurchaseResult{}, err
	}

	now := shared_domain.Now()
	order, err := app.validatePurchase(ctx, session, p.AccountID, p.BondObjectID, p.Quantity, now)
	if err != nil {
		return bondBound_domain.BulkPurchaseResult{}, err
	}
	bounds, movements, err := app.createBounds(ctx, order, now)
	if err != nil {
		return bondBound_domain.BulkPurchaseResult{}, err
	}

	items := make([]bondBound_domain.BulkPurchaseItem, len(bounds))
	var total float64
	for i, b := range bounds {
		items[i] = bondBound_domain.BulkPurchaseItem{ID: b.ID, Key: b.Key, Reference: movements[i].Reference}
		total += b.Amount
	}
	return bondBound_domain.BulkPurchaseResult{
		Items:            items,
		Total:            shared_domain.Round2(total),
		AccountAvailable: shared_domain.Round2(movements[len(movements)-1].Available),
	}, nil
}
