package bondType_application

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// List devuelve los tipos no borrados por name; con agencyIDs, solo los que tienen alguna de esas
// agencias. Un agencyID inválido es ErrInvalidID.
func (app *BondTypeApplication) List(ctx context.Context, agencyIDs []string) ([]*bondType_domain.BondType, error) {
	ids := shared_domain.AgencyIDs(agencyIDs).Normalize()
	for _, id := range ids {
		if !shared_domain.IsValidID(id) {
			return nil, shared_domain.ErrInvalidID
		}
	}
	return app.Repository.Find(ctx, ids, false)
}
