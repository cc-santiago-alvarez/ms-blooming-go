//go:build integration

package bondBound_mongocc_repository_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// De los bonos del usuario en el producto solo salen los iniciados, madurados, bloqueados y canjeados, por
// createdAt descendente; no salen los borrados, los de otro usuario ni los de otro producto, y los de otra
// agencia solo sin alcance de agencia. CA-06.7
func TestListByUserAndObjectStatuses(t *testing.T) {
	repo := setup(t)
	byStatus := make([]bson.M, 7)
	for s := 1; s <= 6; s++ {
		byStatus[s] = boundAt(nextBoundID(), s, bson.M{"statusId": fmt.Sprint(s)})
		insert(t, boundCollection, byStatus[s])
	}
	otherObject := boundAt(nextBoundID(), 9, bson.M{"bondObjectId": legacyObjectID})
	inB := boundAt(nextBoundID(), 10, bson.M{"agencyId": agencyB})
	insert(t, boundCollection,
		boundAt(nextBoundID(), 7, bson.M{"isRemove": true}),
		boundAt(nextBoundID(), 8, bson.M{"userId": user2ID}),
		otherObject, inB)

	tests := []struct {
		name     string
		objectID string
		scope    *string
		want     []string
	}{
		{name: "alcance de la agencia A", objectID: objectID, scope: agencyScope(agencyA), want: ids(byStatus[6], byStatus[3], byStatus[2], byStatus[1])},
		{name: "sin alcance", objectID: objectID, want: ids(inB, byStatus[6], byStatus[3], byStatus[2], byStatus[1])},
		{name: "producto legacy", objectID: legacyObjectID, scope: agencyScope(agencyA), want: ids(otherObject)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bounds, err := repo.ListByUserAndObject(context.Background(), userID, tt.objectID, tt.scope)
			if err != nil {
				t.Fatalf("ListByUserAndObject: %v", err)
			}

			if got := boundIDs(bounds); !slices.Equal(got, tt.want) {
				t.Errorf("bonos %v, se esperaban %v", got, tt.want)
			}
		})
	}
}
