package bondType_application

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
)

// GetManyByIDs devuelve los tipos no borrados que existan entre ids; los demás no aparecen.
func (app *BondTypeApplication) GetManyByIDs(ctx context.Context, ids []string) ([]*bondType_domain.BondType, error) {
	return app.Repository.FindManyByIDs(ctx, ids)
}
