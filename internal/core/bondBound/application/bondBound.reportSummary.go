package bondBound_application

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
)

const (
	// summaryPurchasedBondsReport es el reporte resumen por inversionista en ms-exporter.
	summaryPurchasedBondsReport = "summaryPurchasedBonds"
	// summaryPurchasedBondsFile es el nombre de su archivo, antes de la hora.
	summaryPurchasedBondsFile = "summary-purchased-bonds"
)

// ReportSummary genera el Excel del total comprado por cada inversionista en los bonos no borrados de la
// agencia de la sesión, del mayor al menor, con la moneda de la agencia; de f solo filtra el estado, y la hora
// del nombre usa la zona de la sesión. Un filtro inválido es el error de su Validation, y si ms-exporter falla,
// ErrExportFailed.
func (app *BondBoundApplication) ReportSummary(ctx context.Context, f bondBound_domain.ReportFilters) (bondBound_ports.ReportFile, error) {
	f.Normalize()
	if err := f.Validation(); err != nil {
		return bondBound_ports.ReportFile{}, err
	}
	session, err := sessionUser(ctx)
	if err != nil {
		return bondBound_ports.ReportFile{}, err
	}

	rows, err := app.Repository.ReportSummaryRows(ctx, session.AgencyID, f)
	if err != nil {
		return bondBound_ports.ReportFile{}, err
	}
	return app.exportReport(ctx, session.AgencyID, summaryPurchasedBondsReport, summaryPurchasedBondsFile, rows)
}
