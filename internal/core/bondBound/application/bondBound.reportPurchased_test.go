package bondBound_application_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// purchasedRow es un bono iniciado del reporte de comprados, con su dueño, su producto y su tipo.
func purchasedRow() bondBound_ports.PurchasedReportRow {
	return bondBound_ports.PurchasedReportRow{
		Key:                       "BP12-1",
		Status:                    "Iniciado",
		Amount:                    1000,
		MaturityTimeInDays:        365,
		InitialDate:               boundCreatedAt,
		MaturityDate:              boundCreatedAt.AddDate(0, 0, 365),
		MaturityRewardsPercentage: 15,
		Profile:                   &bondBound_ports.ProfileName{FirstName: "Juan", LastName: "Pérez"},
		BondType:                  bondBound_ports.BondTypeRef{Name: "Plus"},
		BondObject:                bondBound_ports.BondObjectRef{Name: "Bono Plus 12M"},
		CreatedAt:                 boundCreatedAt,
	}
}

// El reporte consulta la agencia de la sesión, aunque sea la principal, con los filtros recortados, el estado
// iniciado si no viene y todos con '0', en la zona de la sesión. Envía a purchasedBonds las filas, con su
// estado por nombre, y la moneda de la agencia, y devuelve el archivo de ms-exporter. CA-07.1
func TestReportPurchasedQueryAndExport(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		filters bondBound_domain.ReportFilters
		want    bondBound_domain.ReportFilters
	}{
		{name: "sin statusId", ctx: withAdmin(), want: bondBound_domain.ReportFilters{StatusID: bondBound_domain.StatusInitiated}},
		{name: "statusId 0", ctx: withAdmin(), filters: bondBound_domain.ReportFilters{StatusID: "0"}, want: bondBound_domain.ReportFilters{StatusID: bondBound_domain.ReportAllStatuses}},
		{
			name: "filtros con espacios",
			ctx:  withAdmin(),
			filters: bondBound_domain.ReportFilters{
				StatusID: " 2 ",
				UserID:   " " + ownerID + " ",
				Search:   " plus ",
				Date:     bondBound_domain.DateFilter{Kind: " aMonth ", Month: 10, Year: 2026},
			},
			want: bondBound_domain.ReportFilters{
				StatusID: bondBound_domain.StatusFinalized,
				UserID:   ownerID,
				Search:   "plus",
				Date:     bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterAMonth, Month: 10, Year: 2026},
			},
		},
		{name: "agencia principal", ctx: withMainAdmin(), want: bondBound_domain.ReportFilters{StatusID: bondBound_domain.StatusInitiated}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()
			rows := []bondBound_ports.PurchasedReportRow{purchasedRow()}
			f.repo.purchasedRows = rows

			file, err := f.app().ReportPurchased(tt.ctx, tt.filters)
			if err != nil {
				t.Fatalf("ReportPurchased: %v", err)
			}

			want := []repoQuery{{method: "ReportPurchasedRows", agencyID: agency(agencyA), reportFilters: tt.want, utcDiff: -5}}
			if !reflect.DeepEqual(f.repo.queries, want) {
				t.Errorf("consultas %+v, se esperaba %+v", f.repo.queries, want)
			}
			assertExported(t, f, "purchasedBonds", rows)
			if !slices.Equal(file.Content, f.exporter.file) {
				t.Errorf("contenido %v, se esperaba el de ms-exporter %v", file.Content, f.exporter.file)
			}
		})
	}
}

// El nombre es reporte-purchased-bonds_ con la hora actual en la zona de la sesión y la extensión .xlsx.
// CA-07.1
func TestReportPurchasedFileName(t *testing.T) {
	for _, utcDiff := range []int{14, -12} {
		t.Run(fmt.Sprintf("UTC%+d", utcDiff), func(t *testing.T) {
			f := newFakes()

			before := time.Now()
			file, err := f.app().ReportPurchased(withSession(adminID, utcDiff), bondBound_domain.ReportFilters{})
			after := time.Now()

			if err != nil {
				t.Fatalf("ReportPurchased: %v", err)
			}
			assertReportName(t, file.Name, "reporte-purchased-bonds_", utcDiff, before, after)
		})
	}
}

// Un statusId fuera del enum y distinto de '0', un userId fuera de formato, una búsqueda de más de 100
// caracteres o un filtro de fecha inválido son el error de la validación, y no consultan ni generan el archivo.
// CA-07.1
func TestReportPurchasedInvalidFilters(t *testing.T) {
	tests := []struct {
		name    string
		filters bondBound_domain.ReportFilters
		wantErr error
	}{
		{name: "estado desconocido", filters: bondBound_domain.ReportFilters{StatusID: "9"}, wantErr: bondBound_domain.ErrInvalidStatus},
		{name: "userId inválido", filters: bondBound_domain.ReportFilters{UserID: "abc"}, wantErr: shared_domain.ErrInvalidID},
		{name: "búsqueda de 101 caracteres", filters: bondBound_domain.ReportFilters{Search: strings.Repeat("a", 101)}, wantErr: bondBound_domain.ErrInvalidSearch},
		{name: "tipo de filtro desconocido", filters: bondBound_domain.ReportFilters{Date: bondBound_domain.DateFilter{Kind: "aWeek"}}, wantErr: bondBound_domain.ErrInvalidDateFilter},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()

			file, err := f.app().ReportPurchased(withAdmin(), tt.filters)

			if !errors.Is(err, tt.wantErr) || file.Name != "" || file.Content != nil {
				t.Fatalf("ReportPurchased = %+v, %v, se esperaba %v", file, err, tt.wantErr)
			}
			assertNoQueries(t, f)
			assertNotExported(t, f)
		})
	}
}

// Si ms-exporter falla, el reporte devuelve ErrExportFailed y ningún archivo. CA-07.4
func TestReportPurchasedExportFailed(t *testing.T) {
	f := newFakes()
	f.exporter.err = shared_domain.ErrExportFailed

	file, err := f.app().ReportPurchased(withAdmin(), bondBound_domain.ReportFilters{})

	if !errors.Is(err, shared_domain.ErrExportFailed) || file.Name != "" || file.Content != nil {
		t.Fatalf("ReportPurchased = %+v, %v, se esperaba ErrExportFailed", file, err)
	}
	if len(f.exporter.calls) != 1 {
		t.Errorf("ms-exporter recibió %+v, se esperaba una llamada", f.exporter.calls)
	}
}

// Un error del repositorio o al leer la moneda se devuelve sin cambiarlo y no llega a ms-exporter; sin sesión
// es un error y no consulta. CA-07.1
func TestReportPurchasedDependencyErrors(t *testing.T) {
	t.Run("repositorio", func(t *testing.T) {
		f := newFakes()
		f.repo.err = errors.New("mongo caído")

		_, err := f.app().ReportPurchased(withAdmin(), bondBound_domain.ReportFilters{})

		if !errors.Is(err, f.repo.err) {
			t.Fatalf("ReportPurchased = %v, se esperaba el error del repositorio", err)
		}
		assertNotExported(t, f)
	})

	t.Run("moneda", func(t *testing.T) {
		f := newFakes()
		f.agencies.err = errors.New("mongo caído")

		_, err := f.app().ReportPurchased(withAdmin(), bondBound_domain.ReportFilters{})

		if !errors.Is(err, f.agencies.err) {
			t.Fatalf("ReportPurchased = %v, se esperaba el error de la moneda", err)
		}
		if len(f.exporter.calls) != 0 {
			t.Errorf("ms-exporter recibió %+v, se esperaba ninguna llamada", f.exporter.calls)
		}
	})

	t.Run("sin sesión", func(t *testing.T) {
		f := newFakes()

		_, err := f.app().ReportPurchased(context.Background(), bondBound_domain.ReportFilters{})

		if err == nil {
			t.Fatal("ReportPurchased sin sesión no devolvió error")
		}
		assertNoQueries(t, f)
		assertNotExported(t, f)
	})
}
