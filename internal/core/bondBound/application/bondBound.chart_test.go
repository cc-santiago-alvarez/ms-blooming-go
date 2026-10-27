package bondBound_application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// La gráfica trae las mismas guías que ChartGuides y la evolución de M+1 días desde la fecha inicial, con
// las fechas en la zona de la sesión. CA-03.16
func TestChartGuidesAndEvolution(t *testing.T) {
	tests := []struct {
		name    string
		utcDiff int
		first   string
		last    string
	}{
		{name: "UTC-5", utcDiff: -5, first: "2026-01-01", last: "2027-01-01"},
		{name: "UTC+14, ya es 2 de enero", utcDiff: 14, first: "2026-01-02", last: "2027-01-02"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := withSession(ownerID, tt.utcDiff)
			app := newFakes(storedBound(boundID)).app()

			chart, err := app.Chart(ctx, boundID)
			if err != nil {
				t.Fatalf("Chart: %v", err)
			}
			guides, err := app.ChartGuides(ctx, boundID)
			if err != nil {
				t.Fatalf("ChartGuides: %v", err)
			}

			if !reflect.DeepEqual(chart.Guides, guides.Guides) {
				t.Errorf("Chart devolvió las guías %+v, ChartGuides devuelve %+v", chart.Guides, guides.Guides)
			}
			if len(chart.Data) != 366 {
				t.Fatalf("la evolución tiene %d puntos, se esperaban 366", len(chart.Data))
			}
			if first, last := chart.Data[0].Date, chart.Data[365].Date; first != tt.first || last != tt.last {
				t.Errorf("la evolución va del %s al %s, se esperaba del %s al %s", first, last, tt.first, tt.last)
			}
		})
	}
}

// La evolución llega hasta hoy y parte de la fecha inicial del bono, mientras que las guías parten de su
// createdAt. CA-03.16
func TestChartEvolutionBase(t *testing.T) {
	t.Run("punto de hoy", func(t *testing.T) {
		chart, err := newFakes(boundOnDay(40)).app().Chart(withSession(ownerID, -5), boundID)
		if err != nil {
			t.Fatalf("Chart: %v", err)
		}

		var today []int
		for i, p := range chart.Data {
			if p.BulletClass != "" {
				today = append(today, i)
			}
		}
		if len(today) != 1 || today[0] != 40 {
			t.Fatalf("puntos de hoy en los días %v, se esperaba solo el día 40", today)
		}
		if p := chart.Data[40]; p.BulletClass != "lastBullet" || p.AtMoment == nil || p.PerdidaAtMoment == nil {
			t.Errorf("el día 40 es %+v, se esperaban los valores del momento y lastBullet", p)
		}
	})

	t.Run("fecha inicial distinta de createdAt", func(t *testing.T) {
		b := storedBound(boundID)
		b.InitialDate = boundCreatedAt.AddDate(0, 0, 2)

		chart, err := newFakes(b).app().Chart(withSession(ownerID, -5), boundID)
		if err != nil {
			t.Fatalf("Chart: %v", err)
		}

		if len(chart.Data) == 0 || chart.Data[0].Date != "2026-01-03" {
			t.Errorf("la evolución empieza en %+v, se esperaba el 2026-01-03", chart.Data[:min(1, len(chart.Data))])
		}
		if len(chart.Guides) == 0 || chart.Guides[0].Date != "2026-01-31" {
			t.Errorf("las guías son %+v, se esperaba el punto de crecimiento el 2026-01-31", chart.Guides)
		}
	})
}

// El dueño y un admin reciben la gráfica; otro usuario, o una llamada sin sesión, es ErrForbidden, y un
// bono inexistente o borrado es ErrBondBoundNotFound aunque la sesión sea admin. CA-03.16
func TestChartAccess(t *testing.T) {
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

			chart, err := newFakes(stored...).app().Chart(tt.ctx, tt.id)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Chart = %v, se esperaba %v", err, tt.wantErr)
				}
				if chart.Guides != nil || chart.Data != nil {
					t.Errorf("Chart devolvió %d guías y %d puntos con error", len(chart.Guides), len(chart.Data))
				}
				return
			}
			if err != nil {
				t.Fatalf("Chart: %v", err)
			}
			if len(chart.Guides) != 2 || len(chart.Data) != 366 {
				t.Errorf("Chart devolvió %d guías y %d puntos, se esperaban 2 y 366", len(chart.Guides), len(chart.Data))
			}
		})
	}
}

// Un bono con _id legacy de 17 caracteres da la misma gráfica que uno con UUID y los mismos datos. CA-03.16
func TestChartLegacyIDMatchesNew(t *testing.T) {
	app := newFakes(storedBound(boundID), storedBound(legacyBoundID)).app()
	ctx := withSession(ownerID, -5)

	current, err := app.Chart(ctx, boundID)
	if err != nil {
		t.Fatalf("Chart con UUID: %v", err)
	}
	legacy, err := app.Chart(ctx, legacyBoundID)
	if err != nil {
		t.Fatalf("Chart con id legacy: %v", err)
	}

	if !reflect.DeepEqual(legacy, current) {
		t.Errorf("la gráfica del id legacy difiere de la del UUID:\n%+v\n%+v", legacy, current)
	}
}

// Un id fuera de formato es ErrInvalidID y no llega al repositorio, ni siquiera para un admin. CA-03.16
func TestChartInvalidID(t *testing.T) {
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

			_, err := f.app().Chart(s.ctx, "abc")

			if !errors.Is(err, shared_domain.ErrInvalidID) {
				t.Fatalf("Chart = %v, se esperaba ErrInvalidID", err)
			}
			if len(f.repo.reads) != 0 {
				t.Errorf("FindOneByID se llamó con %v, se esperaba ninguna lectura", f.repo.reads)
			}
		})
	}
}

// Un error del repositorio se devuelve sin cambiarlo. CA-03.16
func TestChartRepositoryError(t *testing.T) {
	f := newFakes(storedBound(boundID))
	f.repo.err = errors.New("mongo caído")

	chart, err := f.app().Chart(withSession(ownerID, -5), boundID)

	if !errors.Is(err, f.repo.err) {
		t.Fatalf("Chart = %v, se esperaba el error del repositorio", err)
	}
	if chart.Guides != nil || chart.Data != nil {
		t.Errorf("Chart devolvió %d guías y %d puntos con error", len(chart.Guides), len(chart.Data))
	}
}
