package bondBound_application

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// PreSale cotiza la venta anticipada o el canje del bono a hoy, en la zona de la sesión. Si no existe o está
// fuera del alcance de agencia de la sesión es ErrBondBoundNotFound; si no es del usuario ni es admin,
// ErrForbidden; si su estado no se puede vender, el error de SaleTransitionFor, y si la cotización falla, el
// de QuoteSale. Un id inválido es ErrInvalidID.
func (app *BondBoundApplication) PreSale(ctx context.Context, id string) (bondBound_domain.SaleProjection, error) {
	b, err := app.findOwnedOrAdmin(ctx, id)
	if err != nil {
		return bondBound_domain.SaleProjection{}, err
	}
	if _, err := bondBound_domain.SaleTransitionFor(*b); err != nil {
		return bondBound_domain.SaleProjection{}, err
	}
	return bondBound_domain.QuoteSale(*b, shared_domain.Now(), session_domain.UtcDiff(ctx))
}
