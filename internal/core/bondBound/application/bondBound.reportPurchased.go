package bondBound_application

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
)

const (
	// purchasedBondsReport es el reporte de bonos comprados en ms-exporter.
	purchasedBondsReport = "purchasedBonds"
	// purchasedBondsFile es el nombre de su archivo, antes de la hora.
	purchasedBondsFile = "purchased-bonds"
)

// ReportPurchased genera el Excel de los bonos no borrados de la agencia de la sesión que pasan f, por
// createdAt descendente, con la moneda de la agencia; el filtro de fecha y la hora del nombre usan la zona de
// la sesión. Un filtro inválido es el error de su Validation, y si ms-exporter falla, ErrExportFailed.
func (app *BondBoundApplication) ReportPurchased(ctx context.Context, f bondBound_domain.ReportFilters) (bondBound_ports.ReportFile, error) {
	f.Normalize()
	if err := f.Validation(); err != nil {
		return bondBound_ports.ReportFile{}, err
	}
	session, err := sessionUser(ctx)
	if err != nil {
		return bondBound_ports.ReportFile{}, err
	}

	rows, err := app.Repository.ReportPurchasedRows(ctx, session.AgencyID, f, session_domain.UtcDiff(ctx))
	if err != nil {
		return bondBound_ports.ReportFile{}, err
	}
	return app.exportReport(ctx, session.AgencyID, purchasedBondsReport, purchasedBondsFile, rows)
}
