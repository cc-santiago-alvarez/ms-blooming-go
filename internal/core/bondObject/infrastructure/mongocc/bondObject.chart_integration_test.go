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
// ErrBondObjectNotFound; el autorizado y un admin de otra agencia sí la reciben. CA-03.7
func TestChartVisibility(t *testing.T) {
	setup(t)
	insert(t, objectCollection, objectDoc(meteor4, "Privado", typeID, bson.A{agencyA},
		bson.M{"isPrivateBond": true, "authorizedUserIds": bson.A{user2ID}}))
	admin := session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID:        "adminAAAAAAAAAAAA",
		AgencyID:      agencyB,
		AdminBlooming: true,
	})

	tests := []struct {
		name    string
		ctx     context.Context
		visible bool
	}{
		{name: "inversionista sin autorizar", ctx: investorSession(userID, agencyA)},
		{name: "inversionista autorizado", ctx: investorSession(user2ID, agencyA), visible: true},
		{name: "admin de otra agencia", ctx: admin, visible: true},
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
