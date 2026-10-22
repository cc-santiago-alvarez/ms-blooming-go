package bondObject_application

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// ListByType devuelve los productos activos del tipo que ve la sesión, del más reciente al más antiguo,
// con la fecha en que maduraría cada uno si se compra hoy. Sin sesión o sin agencia no hay productos. Un
// bondTypeId inválido es ErrInvalidID.
func (app *BondObjectApplication) ListByType(ctx context.Context, bondTypeID string) ([]bondObject_domain.BondObjectForInvestor, error) {
	if !shared_domain.IsValidID(bondTypeID) {
		return nil, shared_domain.ErrInvalidID
	}

	result := []bondObject_domain.BondObjectForInvestor{}
	f, ok := visibleFilter(ctx)
	if !ok {
		return result, nil
	}
	f.BondTypeID = bondTypeID
	f.OnlyActive = true

	objects, err := app.Repository.FindVisible(ctx, f)
	if err != nil {
		return nil, err
	}

	now, utcDiff := shared_domain.Now(), session_domain.UtcDiff(ctx)
	for _, o := range objects {
		result = append(result, bondObject_domain.NewBondObjectForInvestor(*o, now, utcDiff))
	}
	return result, nil
}

// visibleFilter es el filtro de los productos que ve la sesión: los de su agencia y, de los privados,
// solo donde está autorizada. Sin sesión o sin agencia devuelve false.
func visibleFilter(ctx context.Context) (bondObject_domain.VisibleFilter, bool) {
	session, ok := session_domain.GetUserSession(ctx)
	if !ok || session.AgencyID == "" {
		return bondObject_domain.VisibleFilter{}, false
	}
	return bondObject_domain.VisibleFilter{AgencyID: session.AgencyID, UserID: session.UserID}, true
}
