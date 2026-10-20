//go:build integration

package bondType_mongocc_repository_test

import (
	"context"
	"slices"
	"testing"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Un tipo legacy con _id Meteor se lee, se edita y se borra igual que uno nuevo y conserva su _id; el
// tipo creado después tiene _id UUID. CA-01.9
func TestLegacyMeteorID(t *testing.T) {
	ctx := context.Background()
	repo := setup(t)
	insertTypes(t,
		legacyType(legacyTypeID, "Legacy", bson.A{agencyA}, bson.M{"label": "campo legacy"}),
		legacyType(t1ID, "Nuevo", bson.A{agencyB}, nil),
	)
	otherBefore := rawDoc(t, t1ID)

	bt, err := repo.FindOneByID(ctx, legacyTypeID)
	if err != nil || bt == nil {
		t.Fatalf("FindOneByID(%s) = %v, %v", legacyTypeID, bt, err)
	}

	bt.Apply(bondType_domain.UpdateParams{Name: "Legacy renombrado", Description: "editado", Active: false, AgencyID: []string{agencyA, agencyB}})
	if ok, err := repo.UpdateOne(ctx, bt); err != nil || !ok {
		t.Fatalf("UpdateOne(%s) = %v, %v", legacyTypeID, ok, err)
	}
	stored := rawDoc(t, legacyTypeID)
	if stored["_id"] != legacyTypeID || stored["name"] != "Legacy renombrado" || stored["active"] != false {
		t.Fatalf("documento editado = %v", stored)
	}
	if createdAt, _ := stored["createdAt"].(bson.DateTime); !createdAt.Time().Equal(fixtureAt) {
		t.Fatalf("createdAt = %v, se esperaba %v", stored["createdAt"], fixtureAt)
	}
	if updatedAt, _ := stored["updatedAt"].(bson.DateTime); !updatedAt.Time().After(fixtureAt) {
		t.Fatalf("updatedAt = %v no avanzó", stored["updatedAt"])
	}
	if stored["label"] != "campo legacy" {
		t.Fatalf("se perdió un campo legacy: %v", stored)
	}
	if other := rawDoc(t, t1ID); other["name"] != otherBefore["name"] || other["updatedAt"] != otherBefore["updatedAt"] {
		t.Fatalf("se modificó otro tipo: %v", other)
	}

	created := bondType_domain.NewBondType(bondType_domain.CreateParams{Name: "Creado", AgencyID: []string{agencyA}})
	if err := repo.InsertOne(ctx, created); err != nil {
		t.Fatalf("InsertOne() = %v", err)
	}
	if shared_domain.IDFormat(created.ID) != shared_domain.IDKindUUID {
		t.Fatalf("_id del tipo nuevo = %q, se esperaba un UUID v4", created.ID)
	}
	many, err := repo.FindManyByIDs(ctx, []string{legacyTypeID, created.ID, missingID})
	if err != nil {
		t.Fatalf("FindManyByIDs() = %v", err)
	}
	got := ids(many)
	slices.Sort(got)
	want := []string{created.ID, legacyTypeID}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("FindManyByIDs() = %v, se esperaba %v", got, want)
	}

	if err := repo.SoftDelete(ctx, legacyTypeID); err != nil {
		t.Fatalf("SoftDelete(%s) = %v", legacyTypeID, err)
	}
	if removed := rawDoc(t, legacyTypeID); removed["_id"] != legacyTypeID || removed["isRemove"] != true {
		t.Fatalf("documento borrado = %v", removed)
	}
	if bt, err := repo.FindOneByID(ctx, legacyTypeID); bt != nil || err != nil {
		t.Fatalf("FindOneByID(borrado) = %v, %v, se esperaba nil, nil", bt, err)
	}
	bt.Apply(bondType_domain.UpdateParams{Name: "Otra vez", AgencyID: []string{agencyA}})
	if ok, err := repo.UpdateOne(ctx, bt); err != nil || ok {
		t.Fatalf("UpdateOne(borrado) = %v, %v, se esperaba false", ok, err)
	}
	if after := rawDoc(t, legacyTypeID); after["name"] != "Legacy renombrado" {
		t.Fatalf("UpdateOne modificó un tipo borrado: %v", after)
	}
}
