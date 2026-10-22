package bondObject_application

import (
	"context"

	bondObject_ports "github.com/futurapps/ms-blooming-go/internal/core/bondObject/ports"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
)

// ListGroupedByType agrupa por tipo, en una sola consulta, los productos no borrados de la agencia de la
// sesión. agencyIDs solo filtra si la agencia de la sesión es la principal; sin sesión no hay grupos. Un
// agencyID inválido es ErrInvalidID.
func (app *BondObjectApplication) ListGroupedByType(ctx context.Context, agencyIDs []string) ([]bondObject_ports.BondObjectGroup, error) {
	ids, err := agencyFilter(agencyIDs)
	if err != nil {
		return nil, err
	}

	if scope := shared_application.AgencyScope(ctx); scope != nil {
		if *scope == "" {
			return []bondObject_ports.BondObjectGroup{}, nil
		}
		ids = []string{*scope}
	}
	return app.Repository.FindGroupedByType(ctx, ids)
}
