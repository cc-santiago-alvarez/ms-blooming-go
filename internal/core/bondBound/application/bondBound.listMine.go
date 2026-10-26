package bondBound_application

import (
	"context"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// ListMine devuelve la página p de los bonos no borrados del usuario de la sesión en su agencia, por
// createdAt descendente, con su producto y los derivados a hoy en la zona de la sesión. Un estado desconocido
// es ErrInvalidStatus.
func (app *BondBoundApplication) ListMine(ctx context.Context, f bondBound_domain.MineFilters, p shared_domain.Pagination) (shared_domain.Paginated[bondBound_ports.BondBoundMineItem], error) {
	f.Normalize()
	if err := f.Validation(); err != nil {
		return shared_domain.Paginated[bondBound_ports.BondBoundMineItem]{}, err
	}
	p.Normalize()
	session, err := sessionUser(ctx)
	if err != nil {
		return shared_domain.Paginated[bondBound_ports.BondBoundMineItem]{}, err
	}

	rows, err := app.Repository.ListMine(ctx, session.UserID, session.AgencyID, f.StatusIDs, p)
	if err != nil {
		return shared_domain.Paginated[bondBound_ports.BondBoundMineItem]{}, err
	}

	now, utcDiff := shared_domain.Now(), session_domain.UtcDiff(ctx)
	items := make([]bondBound_ports.BondBoundMineItem, len(rows.Items))
	for i, row := range rows.Items {
		items[i] = newMineItem(row, now, utcDiff)
	}
	return shared_domain.NewPaginated(items, rows.Total, p), nil
}

// newMineItem arma el bono del usuario con lo acumulado a now, si está creciendo o ya creció, y sus fechas
// formateadas, todo en la zona utcDiff.
func newMineItem(row bondBound_ports.BondBoundWithObject, now time.Time, utcDiff int) bondBound_ports.BondBoundMineItem {
	b := row.Bound
	daysPassed, accumulated := bondBound_domain.Accumulated(b, now, utcDiff)
	growing, grew := bondBound_domain.Progress(b, now, utcDiff)
	item := bondBound_ports.BondBoundMineItem{
		ID:                        b.ID,
		Key:                       b.Key,
		StatusID:                  b.StatusID,
		BondObjectID:              b.BondObjectID,
		Amount:                    b.Amount,
		InitialDate:               b.InitialDate,
		MaturityDate:              b.MaturityDate,
		MaturityRewardsPercentage: b.MaturityRewardsPercentage,
		MaturityTimeInDays:        b.MaturityTimeInDays,
		EarlyCashTerm:             b.EarlyCashTerm,
		BondObject:                row.BondObject,
		DaysPassed:                daysPassed,
		AcumulateAmount:           accumulated,
		Growing:                   growing,
		Grew:                      grew,
	}
	if !b.InitialDate.IsZero() {
		item.InitialDateFormatted = bondBound_domain.FormatShortDate(b.InitialDate, utcDiff)
	}
	if !b.MaturityDate.IsZero() {
		item.MaturityDateFormatted = bondBound_domain.FormatShortDate(b.MaturityDate, utcDiff)
	}
	return item
}
