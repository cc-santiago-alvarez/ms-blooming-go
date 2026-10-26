//go:build integration

package bondBound_mongocc_repository_test

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// El detalle de un bono legacy resuelve su producto y su tipo legacy, y el de un bono nuevo los suyos, con
// referencias cruzadas mixtas. El producto sale sin sus usuarios autorizados. CA-06.14
func TestFindDetailsLookups(t *testing.T) {
	tests := []struct {
		name       string
		bound      bson.M
		wantObject string
		wantName   string
		wantType   string
		wantTyName string
		wantAmount float64
	}{
		{
			name:       "bono legacy",
			bound:      boundDoc(legacyBoundID, legacyTypeID, "BC06-1", bson.M{"bondObjectId": legacyObjectID, "amount": "1000"}),
			wantObject: legacyObjectID, wantName: "Bono Clásico 6M",
			wantType: legacyTypeID, wantTyName: "Clásico",
			wantAmount: 1000,
		},
		{
			name:       "bono nuevo",
			bound:      boundDoc(boundID, typeID, "BP12-2", bson.M{"keySeq": 2}),
			wantObject: objectID, wantName: "Bono Plus 12M",
			wantType: typeID, wantTyName: "Plus",
			wantAmount: 1000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setup(t)
			seedCatalog(t)
			insert(t, boundCollection, tt.bound)
			id := tt.bound["_id"].(string)

			row, err := repo.FindDetails(context.Background(), id)
			if err != nil {
				t.Fatalf("FindDetails: %v", err)
			}
			if row == nil {
				t.Fatalf("FindDetails(%s) = nil, se esperaba el detalle", id)
			}

			if row.Bound.ID != id || row.Bound.Amount != tt.wantAmount || row.Bound.UserID != userID {
				t.Errorf("bono %q con amount %v y dueño %q, se esperaban %q, %v y %q",
					row.Bound.ID, row.Bound.Amount, row.Bound.UserID, id, tt.wantAmount, userID)
			}
			if row.BondObject.ID != tt.wantObject || row.BondObject.Name != tt.wantName || row.BondObject.EarlyCashTerm.InflectionPoint != 90 {
				t.Errorf("producto %q %q con inflexión %d, se esperaba %q %q con 90",
					row.BondObject.ID, row.BondObject.Name, row.BondObject.EarlyCashTerm.InflectionPoint, tt.wantObject, tt.wantName)
			}
			if row.BondType.ID != tt.wantType || row.BondType.Name != tt.wantTyName {
				t.Errorf("tipo %q %q, se esperaba %q %q", row.BondType.ID, row.BondType.Name, tt.wantType, tt.wantTyName)
			}
			// el producto guardado tiene dos autorizados; sin el campo, su lector lo deja vacío
			if len(row.BondObject.AuthorizedUserIDs) != 0 {
				t.Errorf("el producto trae authorizedUserIds %v", row.BondObject.AuthorizedUserIDs)
			}
		})
	}
}

// Un bono inexistente, borrado, sin producto o sin tipo no tiene detalle: nil, sin error. CA-06.10
func TestFindDetailsNotFound(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	removed := boundDoc(legacyBound2ID, typeID, "BP12-1", bson.M{"isRemove": true})
	noObject := boundDoc(nextBoundID(), typeID, "BP12-2", bson.M{"bondObjectId": missingID})
	noType := boundDoc(nextBoundID(), missingID, "BP12-3", nil)
	insert(t, boundCollection, removed, noObject, noType)

	for _, id := range append(ids(removed, noObject, noType), missingID) {
		row, err := repo.FindDetails(context.Background(), id)

		if err != nil || row != nil {
			t.Errorf("FindDetails(%s) = %+v, %v; se esperaba nil, nil", id, row, err)
		}
	}
}
