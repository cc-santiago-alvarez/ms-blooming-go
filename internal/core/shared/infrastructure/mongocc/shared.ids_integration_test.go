//go:build integration

package shared_mongocc_repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/codecraftkit/mongocc"
	shared_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/mongocc"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// idCollection es una colección legacy donde conviven _id Meteor y UUID.
const idCollection = "agency"

// idDoc tiene los campos que comparten las entidades: _id string, una referencia, isRemove y fechas.
type idDoc struct {
	ID        string    `bson:"_id"`
	Name      string    `bson:"name"`
	CountryID string    `bson:"countryId"`
	IsRemove  bool      `bson:"isRemove"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// idFixtures son una agencia Meteor que apunta a un país UUID y una UUID que apunta a uno Meteor.
var idFixtures = []idDoc{
	{ID: meteorAgencyID, Name: "Agencia legacy", CountryID: uuidCountryID},
	{ID: uuidAgencyID, Name: "Agencia nueva", CountryID: meteorCountryID},
}

var idCreatedAt = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

func insertIDFixtures(t *testing.T, db *mongocc.MongoQueries) {
	t.Helper()
	for _, f := range idFixtures {
		doc := bson.M{
			"_id":       f.ID,
			"name":      f.Name,
			"countryId": f.CountryID,
			"isRemove":  false,
			"createdAt": idCreatedAt,
			"updatedAt": idCreatedAt,
		}
		if _, err := db.InsertOne(context.Background(), idCollection, doc); err != nil {
			t.Fatalf("no se pudo guardar %s: %v", f.ID, err)
		}
	}
}

// findOneByID lee como los repositorios: por _id y sin borrados; nil, nil si no existe.
func findOneByID(ctx context.Context, db *mongocc.MongoQueries, id string) (*idDoc, error) {
	var doc idDoc
	if err := db.FindOne(ctx, idCollection, bson.M{"_id": id, "isRemove": false}, nil).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &doc, nil
}

// updateOne actualiza como los repositorios: $set de campos por _id, sin tocar el _id.
func updateOne(ctx context.Context, db *mongocc.MongoQueries, id, name string, at time.Time) (*mongo.UpdateResult, error) {
	update := bson.M{"$set": bson.M{"name": name, "updatedAt": at}}
	return db.UpdateOne(ctx, idCollection, bson.M{"_id": id, "isRemove": false}, update, nil)
}

// En una misma colección se leen y actualizan documentos con _id UUID y Meteor, y el _id legacy no
// cambia. CA-00.32
func TestIDFormatsInSameCollection(t *testing.T) {
	ctx := context.Background()

	t.Run("FindOneByID encuentra los dos formatos", func(t *testing.T) {
		db := setup(t)
		insertIDFixtures(t, db)

		for _, f := range idFixtures {
			doc, err := findOneByID(ctx, db, f.ID)
			if err != nil {
				t.Fatalf("findOneByID(%s) = %v", f.ID, err)
			}
			if doc == nil {
				t.Fatalf("findOneByID(%s) no encontró el documento", f.ID)
			}
			if doc.ID != f.ID || doc.CountryID != f.CountryID {
				t.Fatalf("findOneByID(%s) = {_id %s, countryId %s}, se esperaba {%s, %s}",
					f.ID, doc.ID, doc.CountryID, f.ID, f.CountryID)
			}
		}

		doc, err := findOneByID(ctx, db, missingID)
		if err != nil || doc != nil {
			t.Fatalf("findOneByID(inexistente) = %v, %v, se esperaba nil, nil", doc, err)
		}
	})

	tests := []struct {
		name    string
		id      string
		otherID string
	}{
		{name: "legacy Meteor", id: meteorAgencyID, otherID: uuidAgencyID},
		{name: "UUID", id: uuidAgencyID, otherID: meteorAgencyID},
	}
	for _, tt := range tests {
		t.Run("UpdateOne conserva el _id "+tt.name, func(t *testing.T) {
			db := setup(t)
			insertIDFixtures(t, db)
			otherBefore := rawDoc(t, db, idCollection, tt.otherID)
			runner := &shared_mongocc_repository.TransactionRunner{DB: db}
			updatedAt := idCreatedAt.Add(48 * time.Hour)

			var result *mongo.UpdateResult
			err := runner.RunInTransaction(ctx, func(txCtx context.Context) error {
				var err error
				result, err = updateOne(txCtx, db, tt.id, "Agencia renombrada", updatedAt)
				return err
			})

			if err != nil {
				t.Fatalf("RunInTransaction() = %v", err)
			}
			if result.MatchedCount != 1 || result.ModifiedCount != 1 {
				t.Fatalf("UpdateOne coincidió con %d y modificó %d, se esperaba 1 y 1",
					result.MatchedCount, result.ModifiedCount)
			}
			doc, err := findOneByID(ctx, db, tt.id)
			if err != nil || doc == nil {
				t.Fatalf("findOneByID(%s) tras actualizar = %v, %v", tt.id, doc, err)
			}
			if doc.ID != tt.id || doc.Name != "Agencia renombrada" || !doc.UpdatedAt.Equal(updatedAt) {
				t.Fatalf("documento %+v, se esperaba _id %s con el nombre y la fecha nuevos", doc, tt.id)
			}
			raw := rawDoc(t, db, idCollection, tt.id)
			assertFieldType(t, raw, "_id", bson.TypeString)
			if n := countDocs(t, db, idCollection); n != int64(len(idFixtures)) {
				t.Fatalf("%s tiene %d documentos, se esperaban %d", idCollection, n, len(idFixtures))
			}
			assertDocUnchanged(t, db, idCollection, tt.otherID, otherBefore)
		})
	}
}
