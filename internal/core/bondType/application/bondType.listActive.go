package bondType_application

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
)

// ListActive devuelve los tipos activos de la agencia de la sesión, con la cantidad de bonos que el
// usuario puede ver, contados en una sola consulta. Sin sesión o sin agencia no hay tipos.
func (app *BondTypeApplication) ListActive(ctx context.Context) ([]bondType_domain.ActiveBondType, error) {
	result := []bondType_domain.ActiveBondType{}
	session, ok := session_domain.GetUserSession(ctx)
	if !ok || session.AgencyID == "" {
		return result, nil
	}

	types, err := app.Repository.Find(ctx, []string{session.AgencyID}, true)
	if err != nil {
		return nil, err
	}
	if len(types) == 0 {
		return result, nil
	}

	ids := make([]string, len(types))
	for i, t := range types {
		ids[i] = t.ID
	}
	counts, err := app.BondObjects.CountActiveByType(ctx, ids, session.UserID)
	if err != nil {
		return nil, err
	}

	for _, t := range types {
		result = append(result, bondType_domain.ActiveBondType{BondType: *t, NumOfBonds: counts[t.ID]})
	}
	return result, nil
}
