package bondType_application

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
)

// Create normaliza y valida p, guarda el tipo nuevo y lo devuelve.
func (app *BondTypeApplication) Create(ctx context.Context, p bondType_domain.CreateParams) (*bondType_domain.BondType, error) {
	p.Normalize()
	if err := p.Validation(); err != nil {
		return nil, err
	}

	t := bondType_domain.NewBondType(p)
	if err := app.Repository.InsertOne(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}
