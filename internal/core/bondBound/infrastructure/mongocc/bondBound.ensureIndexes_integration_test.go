//go:build integration

package bondBound_mongocc_repository_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// keySeqIndex son las claves del índice único parcial del número de clave por tipo.
const keySeqIndex = "bondTypeId:1,keySeq:1"

// boundIndexes son las claves de los índices que tiene que dejar EnsureIndexes, además del de _id.
var boundIndexes = []string{
	"_id:1",
	"userId:1,isRemove:1,statusId:1,createdAt:-1",
	"agencyId:1,isRemove:1,statusId:1,createdAt:-1",
	"bondObjectId:1,isRemove:1,statusId:1",
	"statusId:1,maturityDate:1",
	keySeqIndex,
}

// indexSpec es lo que se revisa de un índice de la colección.
type indexSpec struct {
	Key     bson.D   `bson:"key"`
	Unique  bool     `bson:"unique"`
	Partial bson.Raw `bson:"partialFilterExpression"`
}

// listIndexes devuelve los índices de bondBound por sus claves, como "campo:orden,...".
func listIndexes(t *testing.T) map[string]indexSpec {
	t.Helper()
	ctx := context.Background()
	cursor, err := testDB.GetCollection(boundCollection).Indexes().List(ctx)
	if err != nil {
		t.Fatalf("no se pudieron listar los índices: %v", err)
	}
	var specs []indexSpec
	if err := cursor.All(ctx, &specs); err != nil {
		t.Fatalf("no se pudieron decodificar los índices: %v", err)
	}
	out := make(map[string]indexSpec, len(specs))
	for _, s := range specs {
		keys := make([]string, len(s.Key))
		for i, e := range s.Key {
			keys[i] = fmt.Sprintf("%s:%v", e.Key, e.Value)
		}
		out[strings.Join(keys, ",")] = s
	}
	return out
}

// assertBoundIndexes verifica que la colección tenga exactamente los índices de boundIndexes, con el de
// keySeq único y parcial sobre los bonos que tienen keySeq.
func assertBoundIndexes(t *testing.T) {
	t.Helper()
	got := listIndexes(t)
	if len(got) != len(boundIndexes) {
		t.Fatalf("hay %d índices, se esperaban %d: %v", len(got), len(boundIndexes), got)
	}
	for _, key := range boundIndexes {
		spec, ok := got[key]
		if !ok {
			t.Fatalf("falta el índice %s; hay %v", key, got)
		}
		if spec.Unique != (key == keySeqIndex) {
			t.Errorf("el índice %s tiene unique %v", key, spec.Unique)
		}
	}
	partial := got[keySeqIndex].Partial
	if exists, ok := partial.Lookup("keySeq", "$exists").BooleanOK(); !ok || !exists {
		t.Errorf("el índice %s tiene el filtro parcial %s, se esperaba {keySeq: {$exists: true}}", keySeqIndex, partial)
	}
}

// EnsureIndexes crea los índices de bondBound, con el de keySeq único y parcial, y se puede llamar de nuevo
// sin error, también si ya existe un índice igual con el nombre por defecto. CA-04.13
func TestEnsureIndexes(t *testing.T) {
	ctx := context.Background()
	indexes := testDB.GetCollection(boundCollection).Indexes()

	t.Run("colección sin índices", func(t *testing.T) {
		repo := setup(t)
		if err := indexes.DropAll(ctx); err != nil {
			t.Fatalf("no se pudieron borrar los índices: %v", err)
		}

		if err := repo.EnsureIndexes(ctx); err != nil {
			t.Fatalf("EnsureIndexes: %v", err)
		}

		assertBoundIndexes(t)
	})

	t.Run("segunda llamada", func(t *testing.T) {
		repo := setup(t)
		if err := repo.EnsureIndexes(ctx); err != nil {
			t.Fatalf("primer EnsureIndexes: %v", err)
		}

		if err := repo.EnsureIndexes(ctx); err != nil {
			t.Fatalf("segundo EnsureIndexes: %v", err)
		}

		assertBoundIndexes(t)
	})

	t.Run("con un índice igual ya creado", func(t *testing.T) {
		repo := setup(t)
		if err := indexes.DropAll(ctx); err != nil {
			t.Fatalf("no se pudieron borrar los índices: %v", err)
		}
		existing := mongo.IndexModel{Keys: bson.D{{Key: "statusId", Value: 1}, {Key: "maturityDate", Value: 1}}}
		if _, err := indexes.CreateOne(ctx, existing); err != nil {
			t.Fatalf("no se pudo crear el índice previo: %v", err)
		}

		if err := repo.EnsureIndexes(ctx); err != nil {
			t.Fatalf("EnsureIndexes: %v", err)
		}

		assertBoundIndexes(t)
	})
}

// El índice de keySeq no deja repetir un número de clave dentro de un tipo, pero sí en otro tipo, y no
// aplica a los bonos legacy sin keySeq, aunque compartan la clave. CA-04.13
func TestKeySeqUnique(t *testing.T) {
	ctx := context.Background()
	insertBound := func(doc bson.M) error {
		_, err := testDB.InsertOne(ctx, boundCollection, doc)
		return err
	}

	tests := []struct {
		name      string
		first     bson.M
		second    bson.M
		duplicate bool
	}{
		{
			name:      "mismo tipo y keySeq",
			first:     boundDoc(boundID, typeID, "BP12-8", bson.M{"keySeq": 8}),
			second:    boundDoc(legacyBoundID, typeID, "BP12-8", bson.M{"keySeq": 8}),
			duplicate: true,
		},
		{
			name:   "mismo keySeq en otro tipo",
			first:  boundDoc(boundID, typeID, "BP12-8", bson.M{"keySeq": 8}),
			second: boundDoc(legacyBoundID, legacyTypeID, "BP24-8", bson.M{"keySeq": 8}),
		},
		{
			name:   "legacy sin keySeq con la misma clave",
			first:  boundDoc(legacyBoundID, typeID, "BP12-3", nil),
			second: boundDoc(legacyBound2ID, typeID, "BP12-3", nil),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setup(t)
			if err := repo.EnsureIndexes(ctx); err != nil {
				t.Fatalf("EnsureIndexes: %v", err)
			}
			if err := insertBound(tt.first); err != nil {
				t.Fatalf("no se pudo guardar el primer bono: %v", err)
			}

			err := insertBound(tt.second)

			if tt.duplicate {
				if !mongo.IsDuplicateKeyError(err) {
					t.Fatalf("el segundo bono se guardó con %v, se esperaba un error de clave duplicada", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("no se pudo guardar el segundo bono: %v", err)
			}
		})
	}
}
