//go:build integration

package shared_mongocc_repository_test

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/codecraftkit/mongocc"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/mongocc"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const passwordHash = "$2b$10$Qm9ub3NBbHRvcy5IYXNoLmRlLnBydWViYQ"

// insertProfile guarda un perfil con la forma real de profile, password incluido.
func insertProfile(t *testing.T, db *mongocc.MongoQueries, id, firstName string, isRemove bool) {
	t.Helper()
	doc := bson.M{
		"_id":                         id,
		"id":                          "CC-" + firstName,
		"first_name":                  firstName,
		"last_name":                   "Gómez",
		"email":                       strings.ToLower(firstName) + "@futurapps.com",
		"avatar_url":                  "https://cdn.futurapps.com/avatars/" + id + ".png",
		"agencyId":                    meteorAgencyID,
		"password":                    passwordHash,
		"rolesIds":                    bson.A{"r1", "r2"},
		"phone":                       "3001234567",
		"status":                      true,
		"country_code":                "CO",
		"base_salary":                 1500000,
		"balanceDepositAuthorization": true,
		"createdAt":                   accountCreatedAt,
		"updatedAt":                   accountCreatedAt,
		"isRemove":                    isRemove,
	}
	if _, err := db.InsertOne(context.Background(), "profile", doc); err != nil {
		t.Fatalf("no se pudo guardar el perfil %s: %v", id, err)
	}
}

// publicProfile es lo que GetPublic debe devolver del perfil que guarda insertProfile.
func publicProfile(id, firstName string) shared_domain.PublicProfile {
	return shared_domain.PublicProfile{
		ID:        id,
		PublicID:  "CC-" + firstName,
		FirstName: firstName,
		LastName:  "Gómez",
		Email:     strings.ToLower(firstName) + "@futurapps.com",
		AvatarURL: "https://cdn.futurapps.com/avatars/" + id + ".png",
		AgencyID:  meteorAgencyID,
	}
}

// GetPublic devuelve solo los campos públicos de los perfiles existentes y nunca le pide password a
// Mongo. CA-00.24
func TestGetPublicProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("perfiles Meteor y UUID", func(t *testing.T) {
		db := setup(t)
		insertProfile(t, db, meteorUserID, "Ana", false)
		insertProfile(t, db, uuidUserID, "Luis", false)
		insertProfile(t, db, removedUserID, "Eva", true)
		commands.reset()
		repo := &shared_mongocc_repository.ProfileMongoccRepository{DB: db}

		got, err := repo.GetPublic(ctx, meteorUserID, uuidUserID, meteorUserID, "", removedUserID, missingID)

		if err != nil {
			t.Fatalf("GetPublic() = %v", err)
		}
		slices.SortFunc(got, func(a, b shared_domain.PublicProfile) int { return strings.Compare(a.ID, b.ID) })
		want := []shared_domain.PublicProfile{publicProfile(meteorUserID, "Ana"), publicProfile(uuidUserID, "Luis")}
		slices.SortFunc(want, func(a, b shared_domain.PublicProfile) int { return strings.Compare(a.ID, b.ID) })
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("GetPublic():\n got  %+v\n want %+v", got, want)
		}

		body, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("json.Marshal() = %v", err)
		}
		if strings.Contains(string(body), "password") || strings.Contains(string(body), passwordHash) {
			t.Fatalf("la respuesta expone el password: %s", body)
		}

		finds := commands.sent("find", "profile")
		if len(finds) != 1 {
			t.Fatalf("se mandaron %d find sobre profile, se esperaba 1", len(finds))
		}
		projection, ok := finds[0].Lookup("projection").DocumentOK()
		if !ok {
			t.Fatalf("el find sobre profile no tiene proyección: %s", finds[0])
		}
		elements, err := projection.Elements()
		if err != nil {
			t.Fatalf("proyección inválida: %v", err)
		}
		var fields []string
		for _, e := range elements {
			fields = append(fields, e.Key())
		}
		slices.Sort(fields)
		wantFields := []string{"_id", "agencyId", "avatar_url", "email", "first_name", "id", "last_name"}
		if !slices.Equal(fields, wantFields) {
			t.Fatalf("proyección %v, se esperaba %v", fields, wantFields)
		}

		ids, ok := finds[0].Lookup("filter", "_id", "$in").ArrayOK()
		if !ok {
			t.Fatalf("el filtro no tiene _id.$in: %s", finds[0])
		}
		if values, _ := ids.Values(); len(values) != 4 {
			t.Fatalf("el filtro busca %d ids, se esperaba 4 (sin repetidos ni vacíos)", len(values))
		}
	})

	t.Run("sin ids no consulta", func(t *testing.T) {
		db := setup(t)
		repo := &shared_mongocc_repository.ProfileMongoccRepository{DB: db}

		for _, ids := range [][]string{nil, {"", ""}} {
			got, err := repo.GetPublic(ctx, ids...)
			if err != nil || got == nil || len(got) != 0 {
				t.Fatalf("GetPublic(%q) = (%v, %v), se esperaba un slice vacío", ids, got, err)
			}
		}
		if finds := commands.sent("find", "profile"); len(finds) != 0 {
			t.Fatalf("se mandaron %d find sobre profile sin ids", len(finds))
		}
	})
}
