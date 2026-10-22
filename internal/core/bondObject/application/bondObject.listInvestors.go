package bondObject_application

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// ListInvestors devuelve los usuarios con bonos vigentes del producto en la agencia de la sesión, con
// cuántos tiene cada uno y su perfil público. Un id inválido es ErrInvalidID.
func (app *BondObjectApplication) ListInvestors(ctx context.Context, id string) ([]bondObject_domain.BondObjectInvestor, error) {
	if !shared_domain.IsValidID(id) {
		return nil, shared_domain.ErrInvalidID
	}
	return app.BondBoundStats.InvestorsByObject(ctx, id, shared_application.AgencyScope(ctx))
}
