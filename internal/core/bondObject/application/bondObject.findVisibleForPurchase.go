package bondObject_application

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// FindVisibleForPurchase devuelve el producto no borrado, de cualquier agencia, o nil, nil si no existe.
// No valida privados: lo hace quien compra. Un id inválido es ErrInvalidID.
func (app *BondObjectApplication) FindVisibleForPurchase(ctx context.Context, id string) (*bondObject_domain.BondObject, error) {
	if !shared_domain.IsValidID(id) {
		return nil, shared_domain.ErrInvalidID
	}
	return app.Repository.FindOneByID(ctx, id)
}
