package bondType_application

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
)

// Remove borra el tipo de forma lógica si ningún producto no borrado lo usa. Si no existe devuelve
// ErrBondTypeNotFound, y si tiene productos, ErrBondTypeHasBondObjects.
func (app *BondTypeApplication) Remove(ctx context.Context, id string) error {
	t, err := app.GetByID(ctx, id)
	if err != nil {
		return err
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
