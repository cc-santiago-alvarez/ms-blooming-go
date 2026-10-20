package bondType_application

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// GetByID devuelve el tipo no borrado, o ErrBondTypeNotFound. Un id inválido es ErrInvalidID.
func (app *BondTypeApplication) GetByID(ctx context.Context, id string) (*bondType_domain.BondType, error) {
	if !shared_domain.IsValidID(id) {
		return nil, shared_domain.ErrInvalidID
	}

	t, err := app.Repository.FindOneByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, bondType_domain.ErrBondTypeNotFound
	}
	return t, nil
}
