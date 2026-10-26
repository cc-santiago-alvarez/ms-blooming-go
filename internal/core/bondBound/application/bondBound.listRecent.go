package bondBound_application

import (
	"context"

	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// recentLimit es cuántos bonos devuelve ListRecent.
const recentLimit = 5

// ListRecent devuelve los cinco bonos iniciados o madurados más recientes del usuario de la sesión, de
// cualquier agencia, con su producto y los derivados a hoy en la zona de la sesión.
func (app *BondBoundApplication) ListRecent(ctx context.Context) ([]bondBound_ports.BondBoundMineItem, error) {
	session, err := sessionUser(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := app.Repository.ListRecent(ctx, session.UserID, recentLimit)
	if err != nil {
		return nil, err
	}

	now, utcDiff := shared_domain.Now(), session_domain.UtcDiff(ctx)
	items := make([]bondBound_ports.BondBoundMineItem, len(rows))
	for i, row := range rows {
		items[i] = newMineItem(row, now, utcDiff)
	}
	return items, nil
}
