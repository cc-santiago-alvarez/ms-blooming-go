//go:build integration

package bondBound_mongocc_repository_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Los bonos de un producto legacy se guardan con _id UUID y con bondObjectId y bondTypeId de 17 caracteres
// tal cual, con keySeq entero y fechas BSON, y se leen igual que se armaron. CA-04.14
func TestInsertManyLegacyProduct(t *testing.T) {
	repo := setup(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 13, 20, 30, 0, 0, time.UTC)
	snapshot := bondBound_domain.ObjectSnapshot{
		BondObjectID:              legacyObjectID,
		BondTypeID:                legacyTypeID,
		Key:                       "BP12",
		Amount:                    1000,
		MaturityTimeInDays:        365,
		MaturityRewardsPercentage: 15,
		EarlyCashTerm: shared_domain.EarlyCashTerm{
			InitialPenaltyPercentage:    100,
			InflectionPoint:             90,
			InflectionPenaltyPercentage: 75,
			InitialDayForWithdraw:       30,
		},
	}
	buyer := bondBound_domain.Buyer{UserID: user2ID, AgencyID: agencyA}
	bounds := []*bondBound_domain.BondBound{
		bondBound_domain.NewBoundFromObject(snapshot, buyer, 8, "Ab3dE6", now),
		bondBound_domain.NewBoundFromObject(snapshot, buyer, 9, "Fg7hJ2", now),
	}

	if err := repo.InsertMany(ctx, bounds); err != nil {
		t.Fatalf("InsertMany: %v", err)
	}

	if n := countDocs(t, boundCollection); n != 2 {
		t.Fatalf("hay %d bonos, se esperaban 2", n)
	}
	for _, want := range bounds {
		raw := rawDoc(t, boundCollection, want.ID)
		if id, _ := raw.Lookup("_id").StringValueOK(); shared_domain.IDFormat(id) != shared_domain.IDKindUUID {
			t.Errorf("_id %q, se esperaba un UUID v4", id)
		}
		objectRef, _ := raw.Lookup("bondObjectId").StringValueOK()
		typeRef, _ := raw.Lookup("bondTypeId").StringValueOK()
		if objectRef != legacyObjectID || typeRef != legacyTypeID {
			t.Errorf("bondObjectId %q y bondTypeId %q, se esperaban los legacy %q y %q", objectRef, typeRef, legacyObjectID, legacyTypeID)
		}
		if status, _ := raw.Lookup("statusId").StringValueOK(); status != "1" {
			t.Errorf("statusId %q, se esperaba \"1\"", status)
		}
		if seq, ok := raw.Lookup("keySeq").AsInt64OK(); !ok || seq != int64(*want.KeySeq) {
			t.Errorf("keySeq %v, se esperaba el entero %d", raw.Lookup("keySeq"), *want.KeySeq)
		}
		for _, key := range []string{"initialDate", "maturityDate", "createdAt", "updatedAt"} {
			if typ := raw.Lookup(key).Type; typ != bson.TypeDateTime {
				t.Errorf("%s es de tipo %s, se esperaba una fecha", key, typ)
			}
		}

		got, err := repo.FindOneByID(ctx, want.ID)
		if err != nil || got == nil {
			t.Fatalf("FindOneByID(%s) = %v, %v", want.ID, got, err)
		}
		if !sameBound(*got, *want) {
			t.Errorf("se leyó\n%+v\nse esperaba\n%+v", *got, *want)
		}
	}
}

// Una lista vacía no escribe nada. CA-04.14
func TestInsertManyEmpty(t *testing.T) {
	repo := setup(t)

	if err := repo.InsertMany(context.Background(), nil); err != nil {
		t.Fatalf("InsertMany(nil): %v", err)
	}
	if n := countDocs(t, boundCollection); n != 0 {
		t.Fatalf("hay %d bonos, se esperaba ninguno", n)
	}
}

// sameBound compara dos bonos con las fechas por instante y el resto de los campos por valor.
func sameBound(a, b bondBound_domain.BondBound) bool {
	times := func(x *bondBound_domain.BondBound) []*time.Time {
		return []*time.Time{&x.InitialDate, &x.MaturityDate, &x.CreatedAt, &x.UpdatedAt}
	}
	at, bt := times(&a), times(&b)
	for i := range at {
		if !at[i].Equal(*bt[i]) {
			return false
		}
		*at[i], *bt[i] = time.Time{}, time.Time{}
	}
	return reflect.DeepEqual(a, b)
}
