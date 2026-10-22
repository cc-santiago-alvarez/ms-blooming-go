//go:build integration

package bondObject_mongocc_repository_test

import (
	"context"
	"maps"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Para un usuario de la agencia A y el tipo T1 con tres productos activos (uno privado sin el usuario
// autorizado), uno inactivo y uno borrado, CountActiveByType cuenta 2 en T1; un privado donde el usuario
// está autorizado sí cuenta, y un tipo sin productos no aparece. CA-01.3
func TestCountActiveByType(t *testing.T) {
	ctx := context.Background()
	repo := setup(t)
	insert(t, objectCollection,
		objectDoc(meteor1, "Público", typeID, bson.A{agencyA}, nil),
		objectDoc(uuid1, "Público legacy", typeID, agencyA, nil),
		objectDoc(meteor2, "Privado de otro", typeID, bson.A{agencyA},
			bson.M{"isPrivateBond": true, "authorizedUserIds": bson.A{user2ID}}),
		objectDoc(uuid2, "Inactivo", typeID, bson.A{agencyA}, bson.M{"active": false}),
		objectDoc(meteor3, "Borrado", typeID, bson.A{agencyA}, bson.M{"isRemove": true}),
		objectDoc(uuid3, "Privado propio", legacyTypeID, bson.A{agencyA},
			bson.M{"isPrivateBond": true, "authorizedUserIds": bson.A{userID}}),
	)

	got, err := repo.CountActiveByType(ctx, []string{typeID, legacyTypeID, otherTypeID}, userID)

	if err != nil {
		t.Fatalf("CountActiveByType() = %v", err)
	}
	want := map[string]int{typeID: 2, legacyTypeID: 1}
	if !maps.Equal(got, want) {
		t.Fatalf("CountActiveByType() = %v, se esperaba %v", got, want)
	}
}
