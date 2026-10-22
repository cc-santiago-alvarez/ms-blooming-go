package bondObject_application

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
)

// latestLimit es cuántos productos recientes se devuelven.
const latestLimit = 3

// Latest devuelve los últimos productos publicados que ve la sesión, del más reciente al más antiguo. Sin
// sesión o sin agencia no hay productos.
func (app *BondObjectApplication) Latest(ctx context.Context) ([]*bondObject_domain.BondObject, error) {
	f, ok := visibleFilter(ctx)
	if !ok {
		return []*bondObject_domain.BondObject{}, nil
	}
	f.Limit = latestLimit
	return app.Repository.FindVisible(ctx, f)
}
