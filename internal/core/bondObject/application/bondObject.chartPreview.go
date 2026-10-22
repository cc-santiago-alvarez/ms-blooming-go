package bondObject_application

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// ChartPreview normaliza y valida los términos de un producto sin guardar, con las reglas del alta, y
// devuelve sus guías y, con simulation, su simulación diaria.
func (app *BondObjectApplication) ChartPreview(ctx context.Context, p bondObject_domain.ChartPreviewParams) (bondObject_domain.BondObjectChart, error) {
	p.Normalize()
	if err := p.Validation(); err != nil {
		return bondObject_domain.BondObjectChart{}, err
	}
	return buildChart(ctx, p.ProjectionTerms(), p.Amount, p.Simulation), nil
}

// buildChart arma las guías a partir de hoy en la zona de la sesión y, con simulation, la simulación
// diaria hasta el vencimiento; sin simulation, Data queda en nil.
func buildChart(ctx context.Context, terms shared_domain.ProjectionTerms, amount float64, simulation bool) bondObject_domain.BondObjectChart {
	now, utcDiff := shared_domain.Now(), session_domain.UtcDiff(ctx)
	chart := bondObject_domain.BondObjectChart{Guides: shared_domain.BuildObjectGuides(terms, now, utcDiff)}
	if simulation {
		chart.Data = shared_domain.SimulateProjection(terms, amount, now, utcDiff)
	}
	return chart
}
