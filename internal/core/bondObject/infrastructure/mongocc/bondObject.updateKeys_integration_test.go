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

// Con la agencia A, UpdateKeys solo cambia los productos que son únicamente de A, con agencyId arreglo o
// string legacy: {matched: 2, modified: 2}, y el compartido y el de B conservan su key. Sin agencia cambia
// los cuatro. CA-02.20
func TestUpdateKeysAgencyScope(t *testing.T) {
	agency := agencyA
	tests := []struct {
		name     string
		agencyID *string
		want     bondObject_domain.UpdateKeysResult
		wantKeys map[string]string
	}{
		{
			name:     "agencia A",
			agencyID: &agency,
			want:     bondObject_domain.UpdateKeysResult{Matched: 2, Modified: 2},
			wantKeys: map[string]string{meteor1: "BP24", uuid1: "BP24", meteor2: "BP12", uuid2: "BP12"},
		},
		{
			name:     "sin agencia",
			want:     bondObject_domain.UpdateKeysResult{Matched: 4, Modified: 4},
			wantKeys: map[string]string{meteor1: "BP24", uuid1: "BP24", meteor2: "BP24", uuid2: "BP24"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setup(t)
			insert(t, objectCollection,
				objectDoc(meteor1, "Solo de A", typeID, bson.A{agencyA}, nil),
				objectDoc(uuid1, "Solo de A legacy", typeID, agencyA, nil),
				objectDoc(meteor2, "Compartido", typeID, bson.A{agencyA, agencyB}, nil),
				objectDoc(uuid2, "Solo de B", typeID, bson.A{agencyB}, nil),
			)

			got, err := repo.UpdateKeys(context.Background(), []string{meteor1, uuid1, meteor2, uuid2}, "BP24", tt.agencyID)

			if err != nil {
				t.Fatalf("UpdateKeys() = %v", err)
			}
			if got != tt.want {
				t.Fatalf("UpdateKeys() = %+v, se esperaba %+v", got, tt.want)
			}
			for id, key := range tt.wantKeys {
				if got := rawDoc(t, objectCollection, id)["key"]; got != key {
					t.Errorf("key de %s = %v, se esperaba %q", id, got, key)
				}
			}
		})
	}
}
