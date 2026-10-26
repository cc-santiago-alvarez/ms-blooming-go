package bondBound_application_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// summaryRow es el total comprado por ownerID, con su perfil.
func summaryRow() bondBound_ports.SummaryReportRow {
	return bondBound_ports.SummaryReportRow{
		UserID:  ownerID,
		Total:   1500,
		Profile: &bondBound_ports.ProfileName{FirstName: "Juan", LastName: "Pérez"},
	}
}

// Sin parámetros, el resumen consulta los bonos iniciados de la agencia de la sesión, aunque sea la principal,
// y con '0' todos los estados. Envía a summaryPurchasedBonds las filas {_id, total, profile} y la moneda de la
// agencia, y devuelve el archivo de ms-exporter. CA-07.3
func TestReportSummaryQueryAndExport(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		filters bondBound_domain.ReportFilters
		want    bondBound_domain.ReportFilters
	}{
		{name: "sin parámetros", ctx: withAdmin(), want: bondBound_domain.ReportFilters{StatusID: bondBound_domain.StatusInitiated}},
		{name: "statusId 0", ctx: withAdmin(), filters: bondBound_domain.ReportFilters{StatusID: " 0 "}, want: bondBound_domain.ReportFilters{StatusID: bondBound_domain.ReportAllStatuses}},
		{name: "agencia principal", ctx: withMainAdmin(), want: bondBound_domain.ReportFilters{StatusID: bondBound_domain.StatusInitiated}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()
			rows := []bondBound_ports.SummaryReportRow{summaryRow()}
			f.repo.summaryRows = rows

			file, err := f.app().ReportSummary(tt.ctx, tt.filters)
			if err != nil {
				t.Fatalf("ReportSummary: %v", err)
			}

			want := []repoQuery{{method: "ReportSummaryRows", agencyID: agency(agencyA), reportFilters: tt.want}}
			if !reflect.DeepEqual(f.repo.queries, want) {
				t.Errorf("consultas %+v, se esperaba %+v", f.repo.queries, want)
			}
			assertExported(t, f, "summaryPurchasedBonds", rows)
			if !slices.Equal(file.Content, f.exporter.file) {
				t.Errorf("contenido %v, se esperaba el de ms-exporter %v", file.Content, f.exporter.file)
			}
		})
	}
}

// El nombre es reporte-summary-purchased-bonds_ con la hora actual en la zona de la sesión y la extensión
// .xlsx. CA-07.1
func TestReportSummaryFileName(t *testing.T) {
	f := newFakes()

	before := time.Now()
	file, err := f.app().ReportSummary(withSession(adminID, 14), bondBound_domain.ReportFilters{})
	after := time.Now()

	if err != nil {
		t.Fatalf("ReportSummary: %v", err)
	}
	assertReportName(t, file.Name, "reporte-summary-purchased-bonds_", 14, before, after)
}

// Un statusId fuera del enum y distinto de '0' es ErrInvalidStatus y no consulta ni genera el archivo. CA-07.3
func TestReportSummaryInvalidStatus(t *testing.T) {
	f := newFakes()

	file, err := f.app().ReportSummary(withAdmin(), bondBound_domain.ReportFilters{StatusID: "9"})

	if !errors.Is(err, bondBound_domain.ErrInvalidStatus) || file.Content != nil {
		t.Fatalf("ReportSummary = %+v, %v, se esperaba ErrInvalidStatus", file, err)
	}
	assertNoQueries(t, f)
	assertNotExported(t, f)
}

// Si ms-exporter falla, el resumen devuelve ErrExportFailed y ningún archivo. CA-07.4
func TestReportSummaryExportFailed(t *testing.T) {
	f := newFakes()
	f.exporter.err = shared_domain.ErrExportFailed

	file, err := f.app().ReportSummary(withAdmin(), bondBound_domain.ReportFilters{})

	if !errors.Is(err, shared_domain.ErrExportFailed) || file.Name != "" || file.Content != nil {
		t.Fatalf("ReportSummary = %+v, %v, se esperaba ErrExportFailed", file, err)
	}
}
