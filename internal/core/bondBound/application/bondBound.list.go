package bondBound_application

import (
	"context"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// List devuelve la página p de los bonos no borrados del alcance de la sesión que pasan f, por createdAt
// descendente, con su dueño, su producto, su tipo y los días y el monto acumulados a hoy; las fechas usan la
// zona de la sesión. Un filtro inválido es el error de su Validation.
func (app *BondBoundApplication) List(ctx context.Context, f bondBound_domain.ListFilters, p shared_domain.Pagination) (shared_domain.Paginated[bondBound_ports.BondBoundAdminItem], error) {
	f.Normalize()
	if err := f.Validation(); err != nil {
		return shared_domain.Paginated[bondBound_ports.BondBoundAdminItem]{}, err
	}
	p.Normalize()

	utcDiff := session_domain.UtcDiff(ctx)
	rows, err := app.Repository.ListAdmin(ctx, shared_application.AgencyScope(ctx), f, p, utcDiff)
	if err != nil {
		return shared_domain.Paginated[bondBound_ports.BondBoundAdminItem]{}, err
	}

	now := shared_domain.Now()
	items := make([]bondBound_ports.BondBoundAdminItem, len(rows.Items))
	for i, row := range rows.Items {
		items[i] = newAdminItem(row, now, utcDiff)
	}
	return shared_domain.NewPaginated(items, rows.Total, p), nil
}

// newAdminItem arma el ítem del listado con el nombre del estado y lo acumulado a now en la zona utcDiff.
func newAdminItem(row bondBound_ports.BondBoundAdminRow, now time.Time, utcDiff int) bondBound_ports.BondBoundAdminItem {
	b := row.Bound
	daysPassed, accumulated := bondBound_domain.Accumulated(b, now, utcDiff)
	return bondBound_ports.BondBoundAdminItem{
		ID:                        b.ID,
		Key:                       b.Key,
		Status:                    b.StatusID.Ref(),
		Amount:                    b.Amount,
		MaturityTimeInDays:        b.MaturityTimeInDays,
		InitialDate:               b.InitialDate,
		MaturityRewardsPercentage: b.MaturityRewardsPercentage,
		Profile:                   row.Profile,
		BondType:                  row.BondType,
		BondObject:                row.BondObject,
		CreatedAt:                 b.CreatedAt,
		Month:                     row.Month,
		Year:                      row.Year,
		DaysPassed:                daysPassed,
		AcumulateAmount:           accumulated,
	}
}
