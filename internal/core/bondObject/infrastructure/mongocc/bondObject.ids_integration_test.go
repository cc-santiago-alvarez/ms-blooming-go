//go:build integration

package bondObject_mongocc_repository_test

import (
	"context"
	"slices"
	"testing"
	"time"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Un producto legacy con _id Meteor y bondTypeId Meteor se lee por id con sus valores legacy decodificados,
// y el listado de su tipo lo devuelve junto al producto nuevo con _id UUID. CA-02.15
func TestLegacyMeteorIDs(t *testing.T) {
	ctx := context.Background()
	repo := setup(t)
	insert(t, objectCollection, objectDoc(meteor1, "Legacy", legacyTypeID, agencyA, bson.M{
		"amount":                    "1500.5",
		"maturityTimeInDays":        "180",
		"maturityRewardsPercentage": "12.5",
		"authorizedUserIds":         nil,
		"saleInitDate":              "2024-06-01T00:00:00.000Z",
		"earlyCashTerm": bson.M{
			"initialPenaltyPercentage":    "100",
			"inflectionPoint":             "90",
			"inflectionPenaltyPercentage": "75",
			"blockWithdrawals":            false,
			"initialDayForWithdraw":       "30",
		},
	}))

	created := bondObject_domain.NewBondObject(bondObject_domain.CreateParams{
		Name:               "Nuevo",
		AgencyID:           shared_domain.AgencyIDs{agencyA},
		Key:                "BP12",
		BondTypeID:         legacyTypeID,
		Amount:             1000,
		AuthorizedUserIDs:  []string{},
		MaturityTimeInDays: 365,
		Active:             true,
		EarlyCashTerm:      shared_domain.EarlyCashTerm{InitialPenaltyPercentage: 100, InflectionPenaltyPercentage: 100},
	})
	if err := repo.InsertOne(ctx, created); err != nil {
		t.Fatalf("InsertOne() = %v", err)
	}
	if shared_domain.IDFormat(created.ID) != shared_domain.IDKindUUID {
		t.Fatalf("_id del producto nuevo = %q, se esperaba un UUID v4", created.ID)
	}

	t.Run("se lee por id", func(t *testing.T) {
		got, err := repo.FindOneByID(ctx, meteor1)
		if err != nil || got == nil {
			t.Fatalf("FindOneByID(%s) = %v, %v", meteor1, got, err)
		}
		saleInit := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		wantTerm := shared_domain.EarlyCashTerm{
			InitialPenaltyPercentage:    100,
			InflectionPoint:             90,
			InflectionPenaltyPercentage: 75,
			InitialDayForWithdraw:       30,
		}
		switch {
		case got.ID != meteor1 || got.BondTypeID != legacyTypeID:
			t.Errorf("_id %q y bondTypeId %q, se esperaban %q y %q", got.ID, got.BondTypeID, meteor1, legacyTypeID)
		case !slices.Equal(got.AgencyID, []string{agencyA}):
			t.Errorf("agencyId = %#v, se esperaba [%s]", got.AgencyID, agencyA)
		case got.Amount != 1500.5 || got.MaturityTimeInDays != 180 || got.MaturityRewardsPercentage != 12.5:
			t.Errorf("amount %v, maturityTimeInDays %d y maturityRewardsPercentage %v; se esperaban 1500.5, 180 y 12.5",
				got.Amount, got.MaturityTimeInDays, got.MaturityRewardsPercentage)
		case got.EarlyCashTerm != wantTerm:
			t.Errorf("earlyCashTerm = %+v, se esperaba %+v", got.EarlyCashTerm, wantTerm)
		case got.SaleInitDate == nil || !got.SaleInitDate.Equal(saleInit) || got.SaleEndDate != nil:
			t.Errorf("saleInitDate %v y saleEndDate %v, se esperaban %v y nil", got.SaleInitDate, got.SaleEndDate, saleInit)
		case got.AuthorizedUserIDs == nil || len(got.AuthorizedUserIDs) != 0:
			t.Errorf("authorizedUserIds = %#v, se esperaba []", got.AuthorizedUserIDs)
		}

		fresh, err := repo.FindOneByID(ctx, created.ID)
		if err != nil || fresh == nil || fresh.BondTypeID != legacyTypeID {
			t.Fatalf("FindOneByID(%s) = %v, %v; se esperaba el producto nuevo", created.ID, fresh, err)
		}
	})

	t.Run("el listado del tipo lo incluye con el nuevo", func(t *testing.T) {
		got, err := newApp().ListByType(investorSession(userID, agencyA), legacyTypeID)
		if err != nil {
			t.Fatalf("ListByType(%s) = %v", legacyTypeID, err)
		}
		gotIDs := make([]string, len(got))
		for i, o := range got {
			gotIDs[i] = o.ID
		}
		// el nuevo se creó hoy y el legacy en 2024
		if want := []string{created.ID, meteor1}; !slices.Equal(gotIDs, want) {
			t.Fatalf("ListByType(%s) = %v, se esperaba %v", legacyTypeID, gotIDs, want)
		}
	})
}
