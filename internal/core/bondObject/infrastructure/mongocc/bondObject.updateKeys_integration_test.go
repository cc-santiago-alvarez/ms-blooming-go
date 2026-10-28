//go:build integration

package bondObject_mongocc_repository_test

import (
	"context"
	"testing"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// UpdateKeys con 3 ids, uno de ellos borrado, devuelve {matched: 2, modified: 2}; el borrado y el producto
// que no está en la lista conservan su key. CA-02.8
func TestUpdateKeys(t *testing.T) {
	ctx := context.Background()
	repo := setup(t)
	insert(t, objectCollection,
		objectDoc(meteor1, "Meteor", typeID, bson.A{agencyA}, nil),
		objectDoc(uuid1, "UUID", legacyTypeID, agencyA, nil),
		objectDoc(meteor2, "Borrado", typeID, bson.A{agencyA}, bson.M{"isRemove": true}),
		objectDoc(uuid2, "Fuera de la lista", typeID, bson.A{agencyA}, nil),
	)

	got, err := repo.UpdateKeys(ctx, []string{meteor1, uuid1, meteor2}, "BP24", nil)

	if err != nil {
		t.Fatalf("UpdateKeys() = %v", err)
	}
	if want := (bondObject_domain.UpdateKeysResult{Matched: 2, Modified: 2}); got != want {
		t.Fatalf("UpdateKeys() = %+v, se esperaba %+v", got, want)
	}
	wantKeys := map[string]string{meteor1: "BP24", uuid1: "BP24", meteor2: "BP12", uuid2: "BP12"}
	for id, key := range wantKeys {
		if got := rawDoc(t, objectCollection, id)["key"]; got != key {
			t.Errorf("key de %s = %v, se esperaba %q", id, got, key)
		}
	}
}
