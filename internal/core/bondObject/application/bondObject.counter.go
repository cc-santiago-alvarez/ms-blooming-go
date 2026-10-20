package bondObject_application

import (
	"context"

	bondType_ports "github.com/futurapps/ms-blooming-go/internal/core/bondType/ports"
)

var _ bondType_ports.BondObjectCounterPort = (*BondObjectApplication)(nil)

// CountActiveByType cuenta, por tipo, los productos no borrados y activos que userID puede ver, de
// cualquier agencia.
func (app *BondObjectApplication) CountActiveByType(ctx context.Context, bondTypeIDs []string, userID string) (map[string]int, error) {
	return app.Repository.CountActiveByType(ctx, bondTypeIDs, userID)
}

// ExistsNotRemovedByType indica si algún producto no borrado usa el tipo.
func (app *BondObjectApplication) ExistsNotRemovedByType(ctx context.Context, bondTypeID string) (bool, error) {
	return app.Repository.ExistsNotRemovedByType(ctx, bondTypeID)
}
