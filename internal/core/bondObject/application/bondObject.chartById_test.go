package bondObject_application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// El inversionista solo ve la gráfica de un producto activo de su agencia y, si es privado, donde está
// autorizado; el admin ve la de cualquier producto no borrado de su alcance de agencia, y el de la agencia
// principal, la de cualquiera. Lo que no ve es ErrBondObjectNotFound. CA-03.7, CA-03.17
func TestChartByIDVisibility(t *testing.T) {
	private := func(authorized ...string) func(o *bondObject_domain.BondObject) {
		return func(o *bondObject_domain.BondObject) {
			o.IsPrivateBond = true
			o.AuthorizedUserIDs = authorized
		}
	}
	inactive := func(o *bondObject_domain.BondObject) { o.Active = false }
	removed := func(o *bondObject_domain.BondObject) { o.IsRemove = true }
	investor := withSession(userID, agencyA)

	tests := []struct {
		name    string
		ctx     context.Context
		modify  func(o *bondObject_domain.BondObject)
		id      string
		visible bool
	}{
		{name: "público de su agencia", ctx: investor, id: objectID, visible: true},
		{name: "privado sin autorizar", ctx: investor, modify: private(user2ID), id: objectID},
		{name: "privado autorizado", ctx: investor, modify: private(userID), id: objectID, visible: true},
		{name: "privado sin autorizar, admin de su agencia", ctx: withAdmin(agencyA, false), modify: private(user2ID), id: objectID, visible: true},
		{name: "privado sin autorizar, admin de la principal", ctx: withAdmin(agencyMain, true), modify: private(user2ID), id: objectID, visible: true},
		{name: "público, admin de otra agencia", ctx: withAdmin(agencyB, false), id: objectID},
		{name: "de otra agencia", ctx: withSession(userID, agencyB), id: objectID},
		{name: "inactivo", ctx: investor, modify: inactive, id: objectID},
		{name: "inactivo, admin", ctx: withAdmin(agencyA, false), modify: inactive, id: objectID, visible: true},
		{name: "borrado", ctx: investor, modify: removed, id: objectID},
		{name: "borrado, admin", ctx: withAdmin(agencyA, false), modify: removed, id: objectID},
		{name: "inexistente", ctx: investor, id: missingID},
		{name: "inexistente, admin", ctx: withAdmin(agencyA, false), id: missingID},
		{name: "sin sesión", ctx: context.Background(), id: objectID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stored := storedObject(objectID)
			if tt.modify != nil {
				tt.modify(stored)
			}

			chart, err := newFakes(stored).app().ChartByID(tt.ctx, tt.id, true)

			if !tt.visible {
				if !errors.Is(err, bondObject_domain.ErrBondObjectNotFound) {
					t.Fatalf("ChartByID = %v, se esperaba ErrBondObjectNotFound", err)
				}
				if chart.Guides != nil || chart.Data != nil {
					t.Errorf("ChartByID devolvió una gráfica con error: %+v", chart)
				}
				return
			}
			if err != nil {
				t.Fatalf("ChartByID: %v", err)
			}
			if len(chart.Data) != stored.MaturityTimeInDays+1 || len(chart.Guides) == 0 {
				t.Errorf("la gráfica tiene %d puntos y %d guías, se esperaban %d puntos y guías",
					len(chart.Data), len(chart.Guides), stored.MaturityTimeInDays+1)
			}
		})
	}
}

// El inversionista busca el producto con el filtro del catálogo, acotado al id, a los activos y a uno
// solo; el admin de la agencia principal lo lee por id sin ese filtro. CA-03.7
func TestChartByIDUsesCatalogFilter(t *testing.T) {
	t.Run("inversionista", func(t *testing.T) {
		f := newFakes(storedObject(objectID))

		if _, err := f.app().ChartByID(withSession(userID, agencyA), objectID, false); err != nil {
			t.Fatalf("ChartByID: %v", err)
		}

		want := []bondObject_domain.VisibleFilter{{
			AgencyID:   agencyA,
			UserID:     userID,
			IDs:        []string{objectID},
			OnlyActive: true,
			Limit:      1,
		}}
		if !reflect.DeepEqual(f.repo.visible, want) {
			t.Errorf("FindVisible recibió %+v, se esperaba %+v", f.repo.visible, want)
		}
		if len(f.repo.reads) != 0 {
			t.Errorf("FindOneByID se llamó con %v, se esperaba ninguna lectura por id", f.repo.reads)
		}
	})

	t.Run("admin", func(t *testing.T) {
		f := newFakes(storedObject(objectID))

		if _, err := f.app().ChartByID(withAdmin(agencyMain, true), objectID, false); err != nil {
			t.Fatalf("ChartByID: %v", err)
		}

		if len(f.repo.reads) != 1 || f.repo.reads[0] != objectID {
			t.Errorf("FindOneByID se llamó con %v, se esperaba [%s]", f.repo.reads, objectID)
		}
		if len(f.repo.visible) != 0 {
			t.Errorf("FindVisible recibió %+v, se esperaba ninguna consulta", f.repo.visible)
		}
	})
}

// Un producto legacy, con _id de 17 caracteres, da la misma gráfica que uno con _id UUID y los mismos
// términos; un id fuera de los dos formatos es ErrInvalidID y no llega al repositorio. CA-03.9
func TestChartByIDLegacyID(t *testing.T) {
	withTerms := func(id string) *bondObject_domain.BondObject {
		o := storedObject(id)
		o.EarlyCashTerm = shared_domain.EarlyCashTerm{
			InitialPenaltyPercentage:    100,
			InflectionPoint:             90,
			InflectionPenaltyPercentage: 75,
			InitialDayForWithdraw:       30,
		}
		return o
	}

	t.Run("misma gráfica", func(t *testing.T) {
		app := newFakes(withTerms(legacyObjectID), withTerms(objectID)).app()
		ctx := withSession(userID, agencyA)

		legacy, err := app.ChartByID(ctx, legacyObjectID, true)
		if err != nil {
			t.Fatalf("ChartByID(%s): %v", legacyObjectID, err)
		}
		fresh, err := app.ChartByID(ctx, objectID, true)
		if err != nil {
			t.Fatalf("ChartByID(%s): %v", objectID, err)
		}

		if len(legacy.Data) != 366 || len(legacy.Guides) != 2 {
			t.Fatalf("la gráfica legacy tiene %d puntos y %d guías, se esperaban 366 y 2", len(legacy.Data), len(legacy.Guides))
		}
		if !reflect.DeepEqual(undated(legacy), undated(fresh)) {
			t.Errorf("la gráfica legacy difiere de la nueva:\nlegacy %+v\nnueva  %+v", undated(legacy), undated(fresh))
		}
	})

	sessions := []struct {
		name string
		ctx  context.Context
	}{
		{name: "inversionista", ctx: withSession(userID, agencyA)},
		{name: "admin", ctx: withAdmin(agencyA, false)},
	}
	for _, s := range sessions {
		t.Run("id fuera de formato, "+s.name, func(t *testing.T) {
			f := newFakes(withTerms(objectID))

			_, err := f.app().ChartByID(s.ctx, "abc", true)

			if !errors.Is(err, shared_domain.ErrInvalidID) {
				t.Fatalf("ChartByID = %v, se esperaba ErrInvalidID", err)
			}
			if len(f.repo.reads) != 0 || len(f.repo.visible) != 0 {
				t.Errorf("el repositorio recibió %v por id y %+v con filtro, se esperaba ninguna consulta",
					f.repo.reads, f.repo.visible)
			}
		})
	}
}

// undated devuelve una copia de la gráfica sin fechas, que dependen del día en que corre el test.
func undated(c bondObject_domain.BondObjectChart) bondObject_domain.BondObjectChart {
	out := bondObject_domain.BondObjectChart{
		Guides: make([]shared_domain.ChartGuide, len(c.Guides)),
		Data:   make([]shared_domain.ChartPoint, len(c.Data)),
	}
	for i, g := range c.Guides {
		g.Date, g.ToDate = "", ""
		out.Guides[i] = g
	}
	for i, p := range c.Data {
		p.Date = ""
		out.Data[i] = p
	}
	return out
}
