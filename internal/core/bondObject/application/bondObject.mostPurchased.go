package bondObject_application

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
)

// mostPurchasedLimit es cuántos productos más comprados se buscan.
const mostPurchasedLimit = 3

// MostPurchased devuelve los productos con más bonos en la agencia de la sesión, de más a menos compras.
// Un producto que la sesión no ve, como un privado donde no está autorizada, no aparece ni se reemplaza.
// Sin sesión o sin agencia no hay productos.
func (app *BondObjectApplication) MostPurchased(ctx context.Context) ([]*bondObject_domain.BondObject, error) {
	result := []*bondObject_domain.BondObject{}
	f, ok := visibleFilter(ctx)
	if !ok {
		return result, nil
	}

	ids, err := app.BondBoundStats.TopPurchasedObjectIDs(ctx, f.AgencyID, mostPurchasedLimit)
	if err != nil {
		return nil, err
	}
	// sin ids, FindVisible no filtraría por _id
	if len(ids) == 0 {
		return result, nil
	}
	f.IDs = ids

	objects, err := app.Repository.FindVisible(ctx, f)
	if err != nil {
		return nil, err
	}

	byID := make(map[string]*bondObject_domain.BondObject, len(objects))
	for _, o := range objects {
		byID[o.ID] = o
	}
	for _, id := range ids {
		if o, ok := byID[id]; ok {
			result = append(result, o)
		}
	}
	return result, nil
}
