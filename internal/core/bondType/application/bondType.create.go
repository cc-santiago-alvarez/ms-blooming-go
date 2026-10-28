package bondType_application

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Create normaliza y valida p, guarda el tipo nuevo y lo devuelve. Fuera de la agencia principal, p solo
// puede tener la agencia de la sesión; si no, es ErrForbidden.
func (app *BondTypeApplication) Create(ctx context.Context, p bondType_domain.CreateParams) (*bondType_domain.BondType, error) {
	p.Normalize()
	if err := p.Validation(); err != nil {
		return nil, err
	}
	if !shared_application.OnlyInAgencyScope(ctx, p.AgencyID...) {
		return nil, shared_domain.ErrForbidden
	}

	t := bondType_domain.NewBondType(p)
	if err := app.Repository.InsertOne(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}
