package bondType_application

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Update valida p, reemplaza los campos editables del tipo no borrado del alcance de agencia de la sesión
// y lo devuelve actualizado, o ErrBondTypeNotFound. Fuera de la agencia principal, el tipo y p solo pueden
// tener la agencia de la sesión; si no, es ErrForbidden.
func (app *BondTypeApplication) Update(ctx context.Context, id string, p bondType_domain.UpdateParams) (*bondType_domain.BondType, error) {
	p.Normalize()
	if err := p.Validation(); err != nil {
		return nil, err
	}

	t, err := app.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !shared_application.OnlyInAgencyScope(ctx, t.AgencyID...) || !shared_application.OnlyInAgencyScope(ctx, p.AgencyID...) {
		return nil, shared_domain.ErrForbidden
	}

	t.Apply(p)
	ok, err := app.Repository.UpdateOne(ctx, t)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, bondType_domain.ErrBondTypeNotFound
	}
	return t, nil
}
