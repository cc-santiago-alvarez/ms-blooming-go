//go:build integration

package bondObject_mongocc_repository_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Un producto legacy, con _id Meteor y los numéricos guardados como string, da la misma gráfica que uno con
// _id UUID y los mismos términos numéricos. CA-03.9
func TestChartLegacyDocument(t *testing.T) {
	setup(t)
	insert(t, objectCollection,
		objectDoc(meteor1, "Legacy", legacyTypeID, agencyA, bson.M{
			"amount":                    "1500.5",
			"maturityTimeInDays":        "180",
			"maturityRewardsPercentage": "12.5",
			"earlyCashTerm": bson.M{
				"initialPenaltyPercentage":    "100",
				"inflectionPoint":             "90",
				"inflectionPenaltyPercentage": "75",
				"blockWithdrawals":            false,
				"initialDayForWithdraw":       "30",
			},
		}),
		objectDoc(uuid1, "Nuevo", typeID, bson.A{agencyA}, bson.M{
			"amount":                    1500.5,
			"maturityTimeInDays":        180,
			"maturityRewardsPercentage": 12.5,
			"earlyCashTerm": bson.M{
				"initialPenaltyPercentage":    100.0,
				"inflectionPoint":             90,
				"inflectionPenaltyPercentage": 75.0,
				"blockWithdrawals":            false,
				"initialDayForWithdraw":       30,
			},
		}),
	)
	app, ctx := newApp(), investorSession(userID, agencyA)

	legacy, err := app.ChartByID(ctx, meteor1, true)
	if err != nil {
		t.Fatalf("ChartByID(%s) = %v", meteor1, err)
	}
	fresh, err := app.ChartByID(ctx, uuid1, true)
	if err != nil {
		t.Fatalf("ChartByID(%s) = %v", uuid1, err)
	}

	if len(legacy.Data) != 181 || len(legacy.Guides) != 2 {
		t.Fatalf("la gráfica legacy tiene %d puntos y %d guías, se esperaban 181 y 2", len(legacy.Data), len(legacy.Guides))
	}
	if !reflect.DeepEqual(undated(legacy), undated(fresh)) {
		t.Errorf("la gráfica legacy difiere de la nueva:\nlegacy %+v\nnueva  %+v", undated(legacy), undated(fresh))
	}
}

// La gráfica de un privado no se entrega a un inversionista que no está autorizado, que recibe
// ErrBondObjectNotFound; el autorizado, un admin de la agencia del producto y uno de la principal sí la
// reciben. CA-03.7
func TestChartVisibility(t *testing.T) {
	setup(t)
	insert(t, objectCollection, objectDoc(meteor4, "Privado", typeID, bson.A{agencyA},
		bson.M{"isPrivateBond": true, "authorizedUserIds": bson.A{user2ID}}))

	tests := []struct {
		name    string
		ctx     context.Context
		visible bool
	}{
		{name: "inversionista sin autorizar", ctx: investorSession(userID, agencyA)},
		{name: "inversionista autorizado", ctx: investorSession(user2ID, agencyA), visible: true},
		{name: "admin de su agencia", ctx: adminSession(agencyA, false), visible: true},
		{name: "admin de la principal", ctx: adminSession(agencyB, true), visible: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chart, err := newApp().ChartByID(tt.ctx, meteor4, true)

			if !tt.visible {
				if !errors.Is(err, bondObject_domain.ErrBondObjectNotFound) {
					t.Fatalf("ChartByID = %v, se esperaba ErrBondObjectNotFound", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ChartByID = %v", err)
			}
			if len(chart.Data) != 366 {
				t.Errorf("la gráfica tiene %d puntos, se esperaban 366", len(chart.Data))
			}
		})
	}
}

// Un admin de una agencia no principal no recibe la gráfica de un producto activo y público de otra
// agencia, que es ErrBondObjectNotFound; uno de la agencia principal recibe la de cualquiera, aunque esté
// inactivo o sea privado. CA-03.17
func TestChartAgencyScope(t *testing.T) {
	setup(t)
	insert(t, objectCollection,
		objectDoc(meteor1, "Público de A", typeID, bson.A{agencyA}, nil),
		objectDoc(meteor4, "Inactivo y privado de A", typeID, bson.A{agencyA},
			bson.M{"active": false, "isPrivateBond": true, "authorizedUserIds": bson.A{user2ID}}),
	)

	t.Run("admin de otra agencia", func(t *testing.T) {
		_, err := newApp().ChartByID(adminSession(agencyB, false), meteor1, true)

		if !errors.Is(err, bondObject_domain.ErrBondObjectNotFound) {
			t.Fatalf("ChartByID = %v, se esperaba ErrBondObjectNotFound", err)
		}
	})

	t.Run("admin de la principal", func(t *testing.T) {
		chart, err := newApp().ChartByID(adminSession(agencyB, true), meteor4, true)

		if err != nil {
			t.Fatalf("ChartByID = %v", err)
		}
		if len(chart.Data) != 366 {
			t.Errorf("la gráfica tiene %d puntos, se esperaban 366", len(chart.Data))
		}
	})
}

// adminSession es un context con la sesión de un admin de Blooming en agencyID; isMain indica si es la
// agencia principal.
func adminSession(agencyID string, isMain bool) context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID:        "adminAAAAAAAAAAAA",
		AgencyID:      agencyID,
		Agency:        session_domain.SessionAgency{ID: agencyID, IsMain: isMain},
		AdminBlooming: true,
	})
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
