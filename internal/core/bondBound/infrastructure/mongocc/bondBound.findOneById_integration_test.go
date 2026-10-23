//go:build integration

package bondBound_mongocc_repository_test

import (
	"context"
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Un bono legacy, con _id Meteor, sin keySeq y con los numéricos como string o int32, se lee con los
// mismos valores que uno nuevo; uno con keySeq lo conserva. CA-04.13
func TestFindOneByIDLegacy(t *testing.T) {
	wantTerms := shared_domain.EarlyCashTerm{
		InitialPenaltyPercentage:    100,
		InflectionPoint:             90,
		InflectionPenaltyPercentage: 75,
		InitialDayForWithdraw:       30,
	}

	tests := []struct {
		name   string
		id     string
		extra  bson.M
		keySeq *int
	}{
		{name: "numéricos como string", id: legacyBoundID, extra: bson.M{
			"bondTypeId":                legacyTypeID,
			"amount":                    "1000",
			"maturityTimeInDays":        "365",
			"maturityRewardsPercentage": "15.5",
			"earlyCashTerm": bson.M{
				"initialPenaltyPercentage":    "100",
				"inflectionPoint":             "90",
				"inflectionPenaltyPercentage": "75",
				"blockWithdrawals":            false,
				"initialDayForWithdraw":       "30",
			},
		}},
		{name: "numéricos int32", id: legacyBound2ID, extra: bson.M{
			"amount":                    int32(1000),
			"maturityTimeInDays":        int32(365),
			"maturityRewardsPercentage": 15.5,
		}},
		{name: "bono nuevo con keySeq", id: boundID, keySeq: new(12), extra: bson.M{
			"maturityRewardsPercentage": 15.5,
			"keySeq":                    12,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setup(t)
			insert(t, boundCollection, boundDoc(tt.id, typeID, "BP12-7", tt.extra))

			b, err := repo.FindOneByID(context.Background(), tt.id)

			if err != nil {
				t.Fatalf("FindOneByID: %v", err)
			}
			if b == nil {
				t.Fatalf("FindOneByID(%s) = nil, se esperaba el bono", tt.id)
			}
			if b.ID != tt.id || b.Amount != 1000 || b.MaturityTimeInDays != 365 || b.MaturityRewardsPercentage != 15.5 {
				t.Errorf("_id %q, amount %v, maturityTimeInDays %d y maturityRewardsPercentage %v, se esperaban %q, 1000, 365 y 15.5",
					b.ID, b.Amount, b.MaturityTimeInDays, b.MaturityRewardsPercentage, tt.id)
			}
			if b.EarlyCashTerm != wantTerms {
				t.Errorf("earlyCashTerm %+v, se esperaba %+v", b.EarlyCashTerm, wantTerms)
			}
			switch {
			case tt.keySeq == nil && b.KeySeq != nil:
				t.Errorf("keySeq %d, se esperaba nil", *b.KeySeq)
			case tt.keySeq != nil && (b.KeySeq == nil || *b.KeySeq != *tt.keySeq):
				t.Errorf("keySeq %v, se esperaba %d", b.KeySeq, *tt.keySeq)
			}
			if !b.CreatedAt.Equal(fixtureAt) || !b.MaturityDate.Equal(fixtureAt.AddDate(0, 0, 365)) {
				t.Errorf("createdAt %v y maturityDate %v, se esperaban %v y un año después", b.CreatedAt, b.MaturityDate, fixtureAt)
			}
		})
	}
}

// Un bono borrado o inexistente se lee como nil, sin error. CA-04.13
func TestFindOneByIDNotFound(t *testing.T) {
	repo := setup(t)
	insert(t, boundCollection, boundDoc(legacyBoundID, typeID, "BP12-1", bson.M{"isRemove": true}))

	for _, id := range []string{legacyBoundID, missingID} {
		b, err := repo.FindOneByID(context.Background(), id)

		if err != nil || b != nil {
			t.Errorf("FindOneByID(%s) = %+v, %v; se esperaba nil, nil", id, b, err)
		}
	}
}
