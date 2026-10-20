//go:build integration

package bondType_mongocc_repository_test

import (
	"context"
	"slices"
	"testing"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Con agencyId=A, Find devuelve solo los tipos no borrados de A, con _id UUID y Meteor. CA-01.1
func TestFindByAgency(t *testing.T) {
	ctx := context.Background()
	repo := setup(t)
	insertTypes(t,
		legacyType(t1ID, "Bono Plus", bson.A{agencyA}, nil),
		legacyType(t2ID, "Ahorro", bson.A{agencyB}, nil),
		legacyType(t3ID, "Borrado", bson.A{agencyA}, bson.M{"isRemove": true}),
	)
	// la consulta corre con el índice ya creado, y crearlo dos veces no falla
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
		{name: "solo los de A", agencyIDs: []string{agencyA}, want: []string{t1ID}},
		{name: "cualquiera de A o B, por name", agencyIDs: []string{agencyA, agencyB}, want: []string{t2ID, t1ID}},
		{name: "sin filtro de agencia", agencyIDs: nil, want: []string{t2ID, t1ID}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Find(ctx, tt.agencyIDs, false)
			if err != nil {
				t.Fatalf("Find(%v) = %v", tt.agencyIDs, err)
			}
			if !slices.Equal(ids(got), tt.want) {
				t.Fatalf("Find(%v) = %v, se esperaba %v", tt.agencyIDs, ids(got), tt.want)
			}
		})
	}
}

// Un tipo legacy con agencyId string se lee como arreglo, aparece al filtrar por su agencia y al
// editarlo queda guardado como arreglo. CA-01.8
func TestLegacyAgencyIDString(t *testing.T) {
	ctx := context.Background()

	t.Run("Find lo incluye con agencyId como arreglo", func(t *testing.T) {
		repo := setup(t)
		insertTypes(t,
			legacyType(legacyTypeID, "Legacy", agencyA, nil),
			legacyType(inactiveID, "Inactivo", bson.A{agencyA}, bson.M{"active": false}),
		)

		all, err := repo.Find(ctx, []string{agencyA}, false)
		if err != nil {
			t.Fatalf("Find(A) = %v", err)
		}
		if want := []string{inactiveID, legacyTypeID}; !slices.Equal(ids(all), want) {
			t.Fatalf("Find(A) = %v, se esperaba %v", ids(all), want)
		}
		if got := all[1].AgencyID; !slices.Equal(got, []string{agencyA}) {
			t.Fatalf("agencyId = %#v, se esperaba [%s]", got, agencyA)
		}

		active, err := repo.Find(ctx, []string{agencyA}, true)
		if err != nil {
			t.Fatalf("Find(A, activos) = %v", err)
		}
		if want := []string{legacyTypeID}; !slices.Equal(ids(active), want) {
			t.Fatalf("Find(A, activos) = %v, se esperaba %v", ids(active), want)
		}
	})

	t.Run("FindOneByID lo devuelve como arreglo", func(t *testing.T) {
		repo := setup(t)
		insertTypes(t, legacyType(legacyTypeID, "Legacy", agencyA, nil))

		got, err := repo.FindOneByID(ctx, legacyTypeID)
		if err != nil || got == nil {
			t.Fatalf("FindOneByID() = %v, %v", got, err)
		}
		if !slices.Equal(got.AgencyID, []string{agencyA}) {
			t.Fatalf("agencyId = %#v, se esperaba [%s]", got.AgencyID, agencyA)
		}
	})

	t.Run("UpdateOne lo guarda como arreglo", func(t *testing.T) {
		repo := setup(t)
		insertTypes(t, legacyType(legacyTypeID, "Legacy", agencyA, nil))

		bt, err := repo.FindOneByID(ctx, legacyTypeID)
		if err != nil || bt == nil {
			t.Fatalf("FindOneByID() = %v, %v", bt, err)
		}
		bt.Apply(bondType_domain.UpdateParams{Name: "Legacy", Active: true, AgencyID: bt.AgencyID})
		if ok, err := repo.UpdateOne(ctx, bt); err != nil || !ok {
			t.Fatalf("UpdateOne() = %v, %v", ok, err)
		}

		stored, isArray := rawDoc(t, legacyTypeID)["agencyId"].(bson.A)
		if !isArray || len(stored) != 1 || stored[0] != agencyA {
			t.Fatalf("agencyId guardado = %#v, se esperaba [%s]", rawDoc(t, legacyTypeID)["agencyId"], agencyA)
		}
	})
}
