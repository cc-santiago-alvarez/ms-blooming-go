package bondObject_application

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// List devuelve los productos no borrados por name; con agencyIDs, solo los que tienen alguna de esas
// agencias. Un agencyID inválido es ErrInvalidID.
func (app *BondObjectApplication) List(ctx context.Context, agencyIDs []string) ([]*bondObject_domain.BondObject, error) {
	ids, err := agencyFilter(agencyIDs)
	if err != nil {
		return nil, err
	}
	return app.Repository.Find(ctx, ids)
}

// agencyFilter limpia las agencias pedidas como filtro; un id inválido es ErrInvalidID.
func agencyFilter(agencyIDs []string) ([]string, error) {
	ids := shared_domain.AgencyIDs(agencyIDs).Normalize()
	for _, id := range ids {
		if !shared_domain.IsValidID(id) {
			return nil, shared_domain.ErrInvalidID
		}
	}
	return ids, nil
}
