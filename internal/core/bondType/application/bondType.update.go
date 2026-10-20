package bondType_application

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
)

// Update valida p, reemplaza los campos editables del tipo no borrado y lo devuelve actualizado, o
// ErrBondTypeNotFound.
func (app *BondTypeApplication) Update(ctx context.Context, id string, p bondType_domain.UpdateParams) (*bondType_domain.BondType, error) {
	p.Normalize()
	if err := p.Validation(); err != nil {
		return nil, err
	}

	t, err := app.GetByID(ctx, id)
	if err != nil {
		return nil, err
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
