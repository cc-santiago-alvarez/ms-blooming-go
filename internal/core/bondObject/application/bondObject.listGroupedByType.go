package bondObject_application

import (
	"context"

	bondObject_ports "github.com/futurapps/ms-blooming-go/internal/core/bondObject/ports"
)

// ListGroupedByType agrupa por tipo, en una sola consulta, los productos no borrados de la agencia de la
// sesión. agencyIDs solo filtra si la agencia de la sesión es la principal; sin sesión no hay grupos. Un
// agencyID inválido es ErrInvalidID.
func (app *BondObjectApplication) ListGroupedByType(ctx context.Context, agencyIDs []string) ([]bondObject_ports.BondObjectGroup, error) {
	ids, ok, err := agencyFilter(ctx, agencyIDs)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []bondObject_ports.BondObjectGroup{}, nil
	}
	return app.Repository.FindGroupedByType(ctx, ids)
}
