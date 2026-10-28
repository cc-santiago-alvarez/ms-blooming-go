package bondObject_application

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// List devuelve los productos no borrados del alcance de agencia de la sesión, por name. agencyIDs solo
// filtra si la agencia de la sesión es la principal; sin sesión no hay productos. Un agencyID inválido es
// ErrInvalidID.
func (app *BondObjectApplication) List(ctx context.Context, agencyIDs []string) ([]*bondObject_domain.BondObject, error) {
	ids, ok, err := agencyFilter(ctx, agencyIDs)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []*bondObject_domain.BondObject{}, nil
	}
	return app.Repository.Find(ctx, ids)
}

// agencyFilter devuelve las agencias por las que filtra un listado de administración: las pedidas, ya
// limpias, si la agencia de la sesión es la principal, y si no, solo la de la sesión. ok es false sin
// sesión, que no ve nada. Un id inválido es ErrInvalidID.
func agencyFilter(ctx context.Context, agencyIDs []string) (ids []string, ok bool, err error) {
	ids = shared_domain.AgencyIDs(agencyIDs).Normalize()
	for _, id := range ids {
		if !shared_domain.IsValidID(id) {
			return nil, false, shared_domain.ErrInvalidID
		}
	}

	scope := shared_application.AgencyScope(ctx)
	switch {
	case scope == nil:
		return ids, true, nil
	case *scope == "":
		return nil, false, nil
	}
	return []string{*scope}, true, nil
}
