package bondObject_application

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// GetByIDIncludingRemoved devuelve el producto aunque esté borrado, o ErrBondObjectNotFound. Un id
// inválido es ErrInvalidID.
func (app *BondObjectApplication) GetByIDIncludingRemoved(ctx context.Context, id string) (*bondObject_domain.BondObject, error) {
	if !shared_domain.IsValidID(id) {
		return nil, shared_domain.ErrInvalidID
	}

	o, err := app.Repository.FindOneByIDIncludingRemoved(ctx, id)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, bondObject_domain.ErrBondObjectNotFound
	}
	return o, nil
}
