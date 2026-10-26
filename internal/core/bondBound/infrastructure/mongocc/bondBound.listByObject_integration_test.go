//go:build integration

package bondBound_mongocc_repository_test

import (
	"context"
	"slices"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Los compradores son los bonos no borrados del producto por createdAt descendente, de cualquier dueño; los
// de otra agencia solo salen sin alcance de agencia, y los de otro producto nunca. CA-06.11
func TestListByObject(t *testing.T) {
	repo := setup(t)
	first := boundAt(nextBoundID(), 0, nil)
	second := boundAt(nextBoundID(), 1, bson.M{"userId": user2ID})
	third := boundAt(nextBoundID(), 2, nil)
	inB := boundAt(nextBoundID(), 3, bson.M{"agencyId": agencyB})
	otherObject := boundAt(nextBoundID(), 5, bson.M{"bondObjectId": legacyObjectID})
	insert(t, boundCollection, first, second, third, inB, otherObject,
		boundAt(nextBoundID(), 4, bson.M{"isRemove": true}))

	tests := []struct {
		name     string
		objectID string
		scope    *string
		want     []string
	}{
		{name: "alcance de la agencia A", objectID: objectID, scope: agencyScope(agencyA), want: ids(third, second, first)},
		{name: "sin alcance", objectID: objectID, want: ids(inB, third, second, first)},
		{name: "producto legacy", objectID: legacyObjectID, scope: agencyScope(agencyA), want: ids(otherObject)},
		{name: "producto sin compradores", objectID: missingID, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bounds, err := repo.ListByObject(context.Background(), tt.objectID, tt.scope)
			if err != nil {
				t.Fatalf("ListByObject: %v", err)
			}

			if got := boundIDs(bounds); !slices.Equal(got, tt.want) {
				t.Errorf("bonos %v, se esperaban %v", got, tt.want)
			}
		})
	}
}
