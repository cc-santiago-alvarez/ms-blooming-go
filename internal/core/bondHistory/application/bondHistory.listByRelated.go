package bondHistory_application

import (
	"context"
	"fmt"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// ListByRelated devuelve el historial de relatedID en la agencia de la sesión, del más reciente al más
// antiguo y listo para mostrar. Lee los perfiles de todos los autores en una sola consulta.
func (app *BondHistoryApplication) ListByRelated(ctx context.Context, typeID bondHistory_domain.HistoryType, relatedID string) ([]bondHistory_domain.HistoryEntry, error) {
	if !typeID.IsValid() {
		return nil, fmt.Errorf("%w: typeId %q", bondHistory_domain.ErrBondHistoryInvalidParams, typeID)
	}

	histories, err := app.Repository.FindByRelated(ctx, typeID, relatedID, shared_application.AgencyScope(ctx))
	if err != nil {
		return nil, err
	}

	userIDs := make([]string, len(histories))
	for i, h := range histories {
		userIDs[i] = h.UserID
	}
	profiles, err := app.Profiles.GetPublic(ctx, userIDs...)
	if err != nil {
		return nil, err
	}

	return present(histories, profiles, shared_domain.Now()), nil
}
