package bondObject_application

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// GetByID devuelve el producto no borrado del alcance de agencia de la sesión con los perfiles públicos de
// sus usuarios autorizados, leídos en una sola consulta, o ErrBondObjectNotFound. Un id inválido es
// ErrInvalidID.
func (app *BondObjectApplication) GetByID(ctx context.Context, id string) (*bondObject_domain.BondObjectDetail, error) {
	o, err := app.findInScope(ctx, id)
	if err != nil {
		return nil, err
	}

	profiles, err := app.Profiles.GetPublic(ctx, o.AuthorizedUserIDs...)
	if err != nil {
		return nil, err
	}
	return &bondObject_domain.BondObjectDetail{BondObject: *o, AuthorizedUser: profiles}, nil
}

// findInScope devuelve el producto no borrado del alcance de agencia de la sesión, o ErrBondObjectNotFound.
// Un id inválido es ErrInvalidID.
func (app *BondObjectApplication) findInScope(ctx context.Context, id string) (*bondObject_domain.BondObject, error) {
	if !shared_domain.IsValidID(id) {
		return nil, shared_domain.ErrInvalidID
	}

	o, err := app.Repository.FindOneByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if o == nil || !shared_application.InAgencyScope(ctx, o.AgencyID...) {
		return nil, bondObject_domain.ErrBondObjectNotFound
	}
	return o, nil
}
