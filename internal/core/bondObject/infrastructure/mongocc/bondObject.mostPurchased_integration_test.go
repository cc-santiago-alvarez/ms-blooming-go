//go:build integration

package bondObject_mongocc_repository_test

import (
	"slices"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Con 4 productos de la agencia con 5, 3, 8 y 1 compras, MostPurchased devuelve los de 8, 5 y 3 en ese
// orden, sin contar bonos borrados ni de otra agencia; si el de 8 es privado y el usuario no está
// autorizado, no aparece ni se reemplaza. CA-02.12
func TestMostPurchased(t *testing.T) {
	five, three, eight, one := meteor1, uuid1, meteor2, uuid2

	tests := []struct {
		name           string
		eightIsPrivate bool
		want           []string
	}{
		{name: "los tres más comprados", want: []string{eight, five, three}},
		{name: "privado sin autorizar", eightIsPrivate: true, want: []string{five, three}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)
			var eightExtra bson.M
			if tt.eightIsPrivate {
				eightExtra = bson.M{"isPrivateBond": true, "authorizedUserIds": bson.A{user2ID}}
			}
			insert(t, objectCollection,
				objectDoc(five, "Cinco", typeID, bson.A{agencyA}, nil),
				objectDoc(three, "Tres", typeID, agencyA, nil),
				objectDoc(eight, "Ocho", legacyTypeID, bson.A{agencyA}, eightExtra),
				objectDoc(one, "Uno", typeID, bson.A{agencyA, agencyB}, nil),
			)
			insert(t, boundCollection, bounds(5, five, userID, agencyA, nil)...)
			insert(t, boundCollection, bounds(3, three, user2ID, agencyA, nil)...)
			insert(t, boundCollection, bounds(8, eight, user2ID, agencyA, nil)...)
			insert(t, boundCollection, bounds(1, one, userID, agencyA, nil)...)
			// si se contaran, uno pasaría al primer lugar
			insert(t, boundCollection, bounds(10, one, userID, agencyA, bson.M{"isRemove": true})...)
			insert(t, boundCollection, bounds(10, one, user2ID, agencyB, nil)...)

			got, err := newApp().MostPurchased(investorSession(userID, agencyA))

			if err != nil {
				t.Fatalf("MostPurchased() = %v", err)
			}
			if !slices.Equal(ids(got), tt.want) {
				t.Fatalf("MostPurchased() = %v, se esperaba %v", ids(got), tt.want)
			}
		})
	}
}
