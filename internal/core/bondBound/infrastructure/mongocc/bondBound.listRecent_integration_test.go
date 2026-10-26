//go:build integration

package bondBound_mongocc_repository_test

import (
	"context"
	"slices"
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Los recientes son los limit bonos iniciados o madurados más nuevos del usuario, de cualquier agencia, con
// el nombre y los términos de retiro de su producto, también los de un producto legacy guardados como
// string; el bono sin producto sale sin bondObject. No salen los vendidos, los borrados ni los de otro
// usuario, y limit 0 no limita. CA-06.9
func TestListRecent(t *testing.T) {
	repo := setup(t)
	insert(t, objectCollection,
		objectDoc(objectID, "Bono Plus 12M", "BP12", typeID, nil),
		objectDoc(legacyObjectID, "Bono Clásico 6M", "BC06", typeID, bson.M{"earlyCashTerm": bson.M{
			"initialPenaltyPercentage":    "100",
			"inflectionPoint":             "45",
			"inflectionPenaltyPercentage": "50",
			"blockWithdrawals":            false,
			"initialDayForWithdraw":       "15",
		}}))
	var eligible []bson.M
	for i := range 7 {
		extra := bson.M{"statusId": "1"}
		if i%2 == 1 {
			extra["statusId"], extra["bondObjectId"] = "2", legacyObjectID
		}
		eligible = append(eligible, boundAt(nextBoundID(), i, extra))
	}
	inB := boundAt(nextBoundID(), 12, bson.M{"agencyId": agencyB})
	noObject := boundAt(nextBoundID(), 13, bson.M{"statusId": "2", "bondObjectId": missingID})
	insert(t, boundCollection, eligible...)
	insert(t, boundCollection,
		boundAt(nextBoundID(), 10, bson.M{"statusId": "5"}),
		boundAt(nextBoundID(), 11, bson.M{"isRemove": true}),
		inB, noObject,
		boundAt(nextBoundID(), 14, bson.M{"userId": user2ID}))

	tests := []struct {
		name  string
		limit int
		want  []string
	}{
		{name: "límite 5", limit: 5, want: ids(noObject, inB, eligible[6], eligible[5], eligible[4])},
		{name: "sin límite", limit: 0, want: ids(noObject, inB, eligible[6], eligible[5], eligible[4], eligible[3], eligible[2], eligible[1], eligible[0])},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := repo.ListRecent(context.Background(), userID, tt.limit)
			if err != nil {
				t.Fatalf("ListRecent: %v", err)
			}

			if got := withObjectIDs(rows); !slices.Equal(got, tt.want) {
				t.Fatalf("bonos %v, se esperaban %v", got, tt.want)
			}
			if rows[0].BondObject != nil {
				t.Errorf("el bono sin producto trae bondObject %+v", rows[0].BondObject)
			}
			plus := shared_domain.EarlyCashTerm{InitialPenaltyPercentage: 100, InflectionPoint: 90, InflectionPenaltyPercentage: 75, InitialDayForWithdraw: 30}
			classic := shared_domain.EarlyCashTerm{InitialPenaltyPercentage: 100, InflectionPoint: 45, InflectionPenaltyPercentage: 50, InitialDayForWithdraw: 15}
			for _, row := range rows[1:] {
				wantName, wantTerm := "Bono Plus 12M", plus
				if row.Bound.BondObjectID == legacyObjectID {
					wantName, wantTerm = "Bono Clásico 6M", classic
				}
				if row.BondObject == nil || row.BondObject.Name != wantName || row.BondObject.EarlyCashTerm == nil || *row.BondObject.EarlyCashTerm != wantTerm {
					t.Errorf("bono %s con producto %+v, se esperaba %q con %+v", row.Bound.ID, row.BondObject, wantName, wantTerm)
				}
			}
		})
	}
}
