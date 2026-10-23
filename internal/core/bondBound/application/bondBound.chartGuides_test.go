package bondBound_application_test

import (
	"context"
	"errors"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Las guías parten del createdAt del bono en la zona de la sesión: el punto de crecimiento cae P días
// después y el retiro initialDayForWithdraw días después, como una línea sin toDate ni fillColor. CA-03.8
func TestChartGuidesDates(t *testing.T) {
	tests := []struct {
		name       string
		utcDiff    int
		inflection string
		withdraw   string
	}{
		{name: "UTC-5", utcDiff: -5, inflection: "2026-01-31", withdraw: "2026-01-11"},
		{name: "UTC+14, ya es 2 de enero", utcDiff: 14, inflection: "2026-02-01", withdraw: "2026-01-12"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chart, err := newFakes(storedBound(boundID)).app().ChartGuides(withSession(ownerID, tt.utcDiff), boundID)
			if err != nil {
				t.Fatalf("ChartGuides: %v", err)
			}

			if len(chart.Guides) != 2 {
				t.Fatalf("ChartGuides devolvió %d guías, se esperaban 2: %+v", len(chart.Guides), chart.Guides)
			}
			g1, g2 := chart.Guides[0], chart.Guides[1]
			if g1.Date != tt.inflection || g2.Date != tt.withdraw {
				t.Errorf("G1.date %q y G2.date %q, se esperaban %q y %q", g1.Date, g2.Date, tt.inflection, tt.withdraw)
			}
			if g2.ToDate != "" || g2.FillColor != "" {
				t.Errorf("G2 tiene toDate %q y fillColor %q, se esperaba una línea sin zona", g2.ToDate, g2.FillColor)
			}
		})
	}
}

// El dueño y un admin reciben las guías; otro usuario, o una llamada sin sesión, es ErrForbidden, y un
// bono inexistente o borrado es ErrBondBoundNotFound aunque la sesión sea admin. CA-03.8
func TestChartGuidesAccess(t *testing.T) {
	removed := func(b *bondBound_domain.BondBound) { b.IsRemove = true }
	owner := withSession(ownerID, -5)

	tests := []struct {
		name    string
		ctx     context.Context
		modify  func(b *bondBound_domain.BondBound)
		id      string
		wantErr error
	}{
		{name: "dueño", ctx: owner, id: boundID},
		{name: "admin que no es dueño", ctx: withAdmin(), id: boundID},
		{name: "dueño con id legacy", ctx: owner, id: legacyBoundID},
		{name: "otro usuario", ctx: withSession(otherUserID, -5), id: boundID, wantErr: shared_domain.ErrForbidden},
		{name: "sin sesión", ctx: context.Background(), id: boundID, wantErr: shared_domain.ErrForbidden},
		{name: "inexistente", ctx: owner, id: missingID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "inexistente, admin", ctx: withAdmin(), id: missingID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "borrado", ctx: owner, modify: removed, id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "borrado, admin", ctx: withAdmin(), modify: removed, id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stored := []*bondBound_domain.BondBound{storedBound(boundID), storedBound(legacyBoundID)}
			for _, b := range stored {
				if tt.modify != nil {
					tt.modify(b)
				}
			}

			chart, err := newFakes(stored...).app().ChartGuides(tt.ctx, tt.id)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ChartGuides = %v, se esperaba %v", err, tt.wantErr)
				}
				if chart.Guides != nil {
					t.Errorf("ChartGuides devolvió guías con error: %+v", chart.Guides)
				}
				return
			}
			if err != nil {
				t.Fatalf("ChartGuides: %v", err)
			}
			if len(chart.Guides) != 2 || chart.Guides[0].Date != "2026-01-31" {
				t.Errorf("ChartGuides devolvió %+v, se esperaban las 2 guías del bono", chart.Guides)
			}
		})
	}
}

// Un id fuera de formato es ErrInvalidID y no llega al repositorio, ni siquiera para un admin. CA-03.8
func TestChartGuidesInvalidID(t *testing.T) {
	sessions := []struct {
		name string
		ctx  context.Context
	}{
		{name: "dueño", ctx: withSession(ownerID, -5)},
		{name: "admin", ctx: withAdmin()},
	}
	for _, s := range sessions {
		t.Run(s.name, func(t *testing.T) {
			f := newFakes(storedBound(boundID))

			_, err := f.app().ChartGuides(s.ctx, "abc")

			if !errors.Is(err, shared_domain.ErrInvalidID) {
				t.Fatalf("ChartGuides = %v, se esperaba ErrInvalidID", err)
			}
			if len(f.repo.reads) != 0 {
				t.Errorf("FindOneByID se llamó con %v, se esperaba ninguna lectura", f.repo.reads)
			}
		})
	}
}

// Un error del repositorio se devuelve sin cambiarlo. CA-03.8
func TestChartGuidesRepositoryError(t *testing.T) {
	f := newFakes(storedBound(boundID))
	f.repo.err = errors.New("mongo caído")

	chart, err := f.app().ChartGuides(withSession(ownerID, -5), boundID)

	if !errors.Is(err, f.repo.err) {
		t.Fatalf("ChartGuides = %v, se esperaba el error del repositorio", err)
	}
	if chart.Guides != nil {
		t.Errorf("ChartGuides devolvió guías con error: %+v", chart.Guides)
	}
}
