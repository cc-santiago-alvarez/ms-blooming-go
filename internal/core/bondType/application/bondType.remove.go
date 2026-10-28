package bondType_application

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Remove borra de forma lógica el tipo del alcance de agencia de la sesión si ningún producto no borrado
// lo usa. Si no existe devuelve ErrBondTypeNotFound; si fuera de la agencia principal también es de otras
// agencias, ErrForbidden; y si tiene productos, ErrBondTypeHasBondObjects.
func (app *BondTypeApplication) Remove(ctx context.Context, id string) error {
	t, err := app.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !shared_application.OnlyInAgencyScope(ctx, t.AgencyID...) {
		return shared_domain.ErrForbidden
	}

	inUse, err := app.BondObjects.ExistsNotRemovedByType(ctx, t.ID)
	if err != nil {
		return err
	}
	if inUse {
		return bondType_domain.ErrBondTypeHasBondObjects
	}
	return app.Repository.SoftDelete(ctx, t.ID)
}
