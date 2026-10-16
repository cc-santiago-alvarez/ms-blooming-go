//go:build integration

package shared_mongocc_repository_test

import (
	"context"
	"testing"

	"github.com/codecraftkit/mongocc"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/mongocc"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// insertDocs guarda los documentos en collection.
func insertDocs(t *testing.T, db *mongocc.MongoQueries, collection string, docs ...bson.M) {
	t.Helper()
	for _, doc := range docs {
		if _, err := db.InsertOne(context.Background(), collection, doc); err != nil {
			t.Fatalf("no se pudo guardar %s/%v: %v", collection, doc["_id"], err)
		}
	}
}

// SymbolCurrency devuelve el símbolo del país de la agencia, "$" si no hay uno válido, y el error
// de Mongo si la consulta falla. CA-00.36
func TestSymbolCurrency(t *testing.T) {
	db := setup(t)
	insertDocs(t, db, "country",
		bson.M{"_id": meteorCountryID, "name_es": "Colombia", "code": "CO", "dial_code": "+57", "symbolCurrency": "Pesos $", "isRemove": false},
		bson.M{"_id": uuidCountryID, "name_es": "Venezuela", "code": "VE", "dial_code": "+58", "isRemove": false},
		bson.M{"_id": shared_domain.NewID(), "name_es": "Argentina", "code": "AR", "symbolCurrency": "", "isRemove": false},
		bson.M{"_id": shared_domain.NewID(), "name_es": "Perú", "code": "PE", "symbolCurrency": "S/", "isRemove": true},
	)
	insertDocs(t, db, "agency",
		bson.M{"_id": meteorAgencyID, "name": "Futurapps Colombia", "countryCode": "CO", "isRemove": false},
		bson.M{"_id": uuidAgencyID, "name": "Futurapps Venezuela", "countryCode": "VE", "isRemove": false},
		bson.M{"_id": emptySymbolAgency, "name": "Futurapps Argentina", "countryCode": "AR", "isRemove": false},
		bson.M{"_id": removedCountryAgcy, "name": "Futurapps Perú", "countryCode": "PE", "isRemove": false},
		bson.M{"_id": removedAgencyID, "name": "Agencia cerrada", "countryCode": "CO", "isRemove": true},
		bson.M{"_id": noCountryAgencyID, "name": "Agencia sin país", "isRemove": false},
	)
	repo := &shared_mongocc_repository.AgencyMongoccRepository{DB: db}

	tests := []struct {
		name     string
		agencyID string
		want     string
	}{
		{name: "agencia Meteor de un país con símbolo", agencyID: meteorAgencyID, want: "Pesos $"},
		{name: "agencia UUID de un país sin símbolo", agencyID: uuidAgencyID, want: "$"},
		{name: "país con símbolo vacío", agencyID: emptySymbolAgency, want: "$"},
		{name: "país borrado", agencyID: removedCountryAgcy, want: "$"},
		{name: "agencia borrada", agencyID: removedAgencyID, want: "$"},
		{name: "agencia sin país", agencyID: noCountryAgencyID, want: "$"},
		{name: "agencia inexistente", agencyID: missingID, want: "$"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.SymbolCurrency(context.Background(), tt.agencyID)
			if err != nil || got != tt.want {
				t.Fatalf("SymbolCurrency(%s) = (%q, %v), se esperaba (%q, nil)", tt.agencyID, got, err, tt.want)
			}
		})
	}

	t.Run("error de Mongo", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		got, err := repo.SymbolCurrency(ctx, meteorAgencyID)

		if err == nil || got != "" {
			t.Fatalf("SymbolCurrency() con el ctx cancelado = (%q, %v), se esperaba un error", got, err)
		}
	})
}
