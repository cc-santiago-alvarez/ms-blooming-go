//go:build integration

package bondObject_mongocc_repository_test

import (
	"context"
	"slices"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Con agencyId=A, Find devuelve por name los productos no borrados de A, con _id Meteor y UUID y
// agencyId legacy string leído como arreglo. CA-02.1
func TestFindByAgency(t *testing.T) {
	ctx := context.Background()
	repo := setup(t)
	alfa, zeta, beta, removed := uuid1, meteor1, uuid2, meteor2
	insert(t, objectCollection,
		objectDoc(zeta, "Zeta", typeID, agencyA, nil),
		objectDoc(alfa, "Alfa", legacyTypeID, bson.A{agencyA}, nil),
		objectDoc(beta, "Beta", typeID, bson.A{agencyB}, nil),
		objectDoc(removed, "Aaa borrado", typeID, bson.A{agencyA}, bson.M{"isRemove": true}),
	)
	// la consulta corre con los índices ya creados, y crearlos dos veces no falla
	for range 2 {
		if err := repo.EnsureIndexes(ctx); err != nil {
			t.Fatalf("EnsureIndexes() = %v", err)
		}
	}

	tests := []struct {
		name      string
		agencyIDs []string
		want      []string
	}{
		{name: "solo los de A", agencyIDs: []string{agencyA}, want: []string{alfa, zeta}},
		{name: "sin filtro de agencia", agencyIDs: nil, want: []string{alfa, beta, zeta}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Find(ctx, tt.agencyIDs)
			if err != nil {
				t.Fatalf("Find(%v) = %v", tt.agencyIDs, err)
			}
			if !slices.Equal(ids(got), tt.want) {
				t.Fatalf("Find(%v) = %v, se esperaba %v", tt.agencyIDs, ids(got), tt.want)
			}
			for _, o := range got {
				if o.ID == zeta && !slices.Equal(o.AgencyID, []string{agencyA}) {
					t.Fatalf("agencyId de Zeta = %#v, se esperaba [%s]", o.AgencyID, agencyA)
				}
			}
		})
	}
}
