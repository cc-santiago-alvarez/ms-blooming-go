package bondObject_application

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// ChartByID devuelve las guías del producto a partir de hoy y, con simulation, su simulación diaria. Si
// la sesión no lo ve, es ErrBondObjectNotFound. Un id inválido es ErrInvalidID.
func (app *BondObjectApplication) ChartByID(ctx context.Context, id string, simulation bool) (bondObject_domain.BondObjectChart, error) {
	o, err := app.findVisibleByID(ctx, id)
	if err != nil {
		return bondObject_domain.BondObjectChart{}, err
	}
	return buildChart(ctx, o.ProjectionTerms(), o.Amount, simulation), nil
}

// findVisibleByID devuelve el producto si la sesión lo ve: un admin, cualquiera no borrado; el resto,
// solo uno activo de su agencia y, si es privado, donde está autorizado. Si no, es ErrBondObjectNotFound.
// Un id inválido es ErrInvalidID.
func (app *BondObjectApplication) findVisibleByID(ctx context.Context, id string) (*bondObject_domain.BondObject, error) {
	if session, ok := session_domain.GetUserSession(ctx); ok && session.AdminBlooming {
		return app.findNotRemoved(ctx, id)
	}
	if !shared_domain.IsValidID(id) {
		return nil, shared_domain.ErrInvalidID
	}

	f, ok := visibleFilter(ctx)
	if !ok {
		return nil, bondObject_domain.ErrBondObjectNotFound
	}
	f.IDs = []string{id}
	f.OnlyActive = true
	f.Limit = 1

	objects, err := app.Repository.FindVisible(ctx, f)
	if err != nil {
		return nil, err
	}
	if len(objects) == 0 {
		return nil, bondObject_domain.ErrBondObjectNotFound
	}
	return objects[0], nil
}
