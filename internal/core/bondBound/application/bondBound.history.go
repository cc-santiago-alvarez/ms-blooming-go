package bondBound_application

import (
	"context"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
)

// HistoryOf devuelve el historial del bono en la agencia de la sesión, del más reciente al más antiguo y listo
// para mostrar. Si no existe es ErrBondBoundNotFound, y si no es del usuario ni es admin, ErrForbidden. Un id
// inválido es ErrInvalidID.
func (app *BondBoundApplication) HistoryOf(ctx context.Context, id string) ([]bondHistory_domain.HistoryEntry, error) {
	if _, err := app.findOwnedOrAdmin(ctx, id); err != nil {
		return nil, err
	}
	return app.History.ListByRelated(ctx, bondHistory_domain.HistoryBondBound, id)
}
