//go:build integration

package bondObject_mongocc_repository_test

import (
	"context"
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Con un bono comprado del producto X por 1000, editar X a 2000 cambia solo el producto: el bondBound
// sigue con amount 1000 y su updatedAt, y no se le manda ningún update. CA-02.7
func TestUpdateOneKeepsBondBounds(t *testing.T) {
	ctx := context.Background()
	repo := setup(t)
	boundID := nextBoundID()
	insert(t, objectCollection, objectDoc(meteor1, "Bono X", typeID, bson.A{agencyA}, nil))
	insert(t, boundCollection, boundDoc(boundID, meteor1, userID, agencyA, "1", nil))
	before := rawDoc(t, boundCollection, boundID)

	o, err := repo.FindOneByID(ctx, meteor1)
	if err != nil || o == nil {
		t.Fatalf("FindOneByID(%s) = %v, %v", meteor1, o, err)
	}
	o.Amount = 2000
	o.UpdatedAt = shared_domain.Now()
	commands.reset()

	if ok, err := repo.UpdateOne(ctx, o); err != nil || !ok {
		t.Fatalf("UpdateOne(%s) = %v, %v", meteor1, ok, err)
	}

	if amount := rawDoc(t, objectCollection, meteor1)["amount"]; amount != 2000.0 {
		t.Fatalf("amount del producto = %v, se esperaba 2000", amount)
	}
	after := rawDoc(t, boundCollection, boundID)
	if after["amount"] != 1000.0 || after["amount"] != before["amount"] || after["updatedAt"] != before["updatedAt"] {
		t.Fatalf("el bono cambió: amount %v y updatedAt %v, antes %v y %v",
			after["amount"], after["updatedAt"], before["amount"], before["updatedAt"])
	}
	for _, name := range []string{"update", "findAndModify"} {
		if n := len(commands.sent(name, boundCollection)); n != 0 {
			t.Fatalf("se mandaron %d %s sobre %s al editar el producto", n, name, boundCollection)
		}
	}
}
