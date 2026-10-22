package bondObject_application

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// GetByID devuelve el producto no borrado con los perfiles públicos de sus usuarios autorizados, leídos en
// una sola consulta, o ErrBondObjectNotFound. Un id inválido es ErrInvalidID.
func (app *BondObjectApplication) GetByID(ctx context.Context, id string) (*bondObject_domain.BondObjectDetail, error) {
	if !shared_domain.IsValidID(id) {
		return nil, shared_domain.ErrInvalidID
	}

	o, err := app.Repository.FindOneByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, bondObject_domain.ErrBondObjectNotFound
	}

	profiles, err := app.Profiles.GetPublic(ctx, o.AuthorizedUserIDs...)
	if err != nil {
		return nil, err
	}
	return &bondObject_domain.BondObjectDetail{BondObject: *o, AuthorizedUser: profiles}, nil
}
