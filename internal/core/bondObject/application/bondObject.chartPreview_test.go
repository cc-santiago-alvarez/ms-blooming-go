package bondObject_application_test

import (
	"errors"
	"testing"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// previewParams es una vista previa válida de 365 días con inflexión en el día 90 y retiro desde el 30.
func previewParams(simulation bool) bondObject_domain.ChartPreviewParams {
	return bondObject_domain.ChartPreviewParams{
		Amount:                    1000,
		MaturityTimeInDays:        365,
		MaturityRewardsPercentage: 15,
		EarlyCashTerm: shared_domain.EarlyCashTerm{
			InitialPenaltyPercentage:    100,
			InflectionPoint:             90,
			InflectionPenaltyPercentage: 75,
			InitialDayForWithdraw:       30,
		},
		Simulation: simulation,
	}
}

// Los términos fuera de las reglas del alta se rechazan con su error; un inflectionPoint igual o mayor que
// maturityTimeInDays es ErrBondObjectInvalidInflectionPoint. CA-03.6
func TestChartPreviewRejectsInvalidTerms(t *testing.T) {
	tests := []struct {
		name   string
		modify func(p *bondObject_domain.ChartPreviewParams)
		want   error
	}{
		{name: "inflectionPoint igual al plazo", modify: func(p *bondObject_domain.ChartPreviewParams) {
			p.EarlyCashTerm.InflectionPoint = p.MaturityTimeInDays
		}, want: bondObject_domain.ErrBondObjectInvalidInflectionPoint},
		{name: "inflectionPoint mayor que el plazo", modify: func(p *bondObject_domain.ChartPreviewParams) {
			p.EarlyCashTerm.InflectionPoint = p.MaturityTimeInDays + 1
		}, want: bondObject_domain.ErrBondObjectInvalidInflectionPoint},
		{name: "amount en 0", modify: func(p *bondObject_domain.ChartPreviewParams) {
			p.Amount = 0
		}, want: bondObject_domain.ErrBondObjectInvalidAmount},
		{name: "plazo en 0", modify: func(p *bondObject_domain.ChartPreviewParams) {
			p.MaturityTimeInDays = 0
		}, want: bondObject_domain.ErrBondObjectInvalidMaturity},
		{name: "día de retiro después del plazo", modify: func(p *bondObject_domain.ChartPreviewParams) {
			p.EarlyCashTerm.InitialDayForWithdraw = p.MaturityTimeInDays + 1
		}, want: bondObject_domain.ErrBondObjectInvalidWithdrawDay},
		{name: "penalidad mayor que 100", modify: func(p *bondObject_domain.ChartPreviewParams) {
			p.EarlyCashTerm.InflectionPenaltyPercentage = 101
		}, want: bondObject_domain.ErrBondObjectInvalidPercentage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := previewParams(true)
			tt.modify(&p)

			chart, err := newFakes().app().ChartPreview(withAdmin(agencyA), p)

			if !errors.Is(err, tt.want) {
				t.Fatalf("ChartPreview = %v, se esperaba %v", err, tt.want)
			}
			if chart.Guides != nil || chart.Data != nil {
				t.Errorf("ChartPreview devolvió una gráfica con error: %+v", chart)
			}
		})
	}
}

// Sin simulation la vista previa solo trae las guías; con simulation trae además un punto por día, del
// día 0 al vencimiento. CA-03.6
func TestChartPreviewSimulation(t *testing.T) {
	app := newFakes().app()

	t.Run("sin simulation", func(t *testing.T) {
		chart, err := app.ChartPreview(withAdmin(agencyA), previewParams(false))

		if err != nil {
			t.Fatalf("ChartPreview: %v", err)
		}
		if chart.Data != nil {
			t.Errorf("Data tiene %d puntos, se esperaba nil", len(chart.Data))
		}
		if len(chart.Guides) != 2 {
			t.Errorf("Guides tiene %d guías, se esperaban la de crecimiento y la de retiro", len(chart.Guides))
		}
	})

	t.Run("con simulation", func(t *testing.T) {
		chart, err := app.ChartPreview(withAdmin(agencyA), previewParams(true))

		if err != nil {
			t.Fatalf("ChartPreview: %v", err)
		}
		if len(chart.Data) != 366 || len(chart.Guides) != 2 {
			t.Errorf("la gráfica tiene %d puntos y %d guías, se esperaban 366 y 2", len(chart.Data), len(chart.Guides))
		}
	})
}

// La vista previa simula con amount redondeado a 2 decimales y con las penalidades en 0 llevadas a 100,
// como el alta. CA-03.10
func TestChartPreviewDefaults(t *testing.T) {
	p := bondObject_domain.ChartPreviewParams{
		Amount:                    1000.004,
		MaturityTimeInDays:        10,
		MaturityRewardsPercentage: 100,
		Simulation:                true,
	}

	chart, err := newFakes().app().ChartPreview(withAdmin(agencyA), p)

	if err != nil {
		t.Fatalf("ChartPreview: %v", err)
	}
	if len(chart.Data) != 11 {
		t.Fatalf("Data tiene %d puntos, se esperaban 11", len(chart.Data))
	}
	last := chart.Data[10]
	if last.Ganancia == nil || last.Perdida == nil {
		t.Fatalf("el día 10 no tiene ganancia o pérdida: %+v", last)
	}
	if last.Inversion != 1000 || *last.Ganancia != 2000 || *last.Perdida != 1000 {
		t.Errorf("día 10 = inversion %v, ganancia %v, perdida %v; se esperaban 1000, 2000 y 1000",
			last.Inversion, *last.Ganancia, *last.Perdida)
	}
}
