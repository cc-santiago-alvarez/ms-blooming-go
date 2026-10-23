//go:build integration

package bondUserSetting_mongocc_repository_test

import (
	"testing"
	"time"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Un setting legacy, con _id Meteor de 17 caracteres, se devuelve en GET /me sin crear otro, y PATCH
// /me/selected-account lo actualiza conservando su _id y su createdAt. CA-09.12
func TestLegacySetting(t *testing.T) {
	setup(t)
	insert(t, settingCollection, settingDoc(legacySettingID, userID, agencyA, bson.M{"selectedAccountId": accountNew}))
	insert(t, accountCollection,
		accountDoc(accountOld, userID, agencyA, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC), nil),
		accountDoc(accountNew, userID, agencyA, time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC), nil),
	)
	insert(t, profileCollection, profileDoc(userID, "Juan"))
	app := newApp()
	ctx := investorSession(userID, agencyA)

	got, err := app.GetMine(ctx)

	if err != nil {
		t.Fatalf("GetMine: %v", err)
	}
	if got.ID != legacySettingID || val(got.SelectedAccountID) != accountNew {
		t.Fatalf("GetMine devolvió %s con la cuenta %s, se esperaba el setting legacy %s con %s",
			got.ID, val(got.SelectedAccountID), legacySettingID, accountNew)
	}
	if n := countSettings(t, bson.M{"userId": userID}); n != 1 {
		t.Fatalf("hay %d settings de %s después de GET /me, se esperaba 1", n, userID)
	}

	updated, err := app.SetSelectedAccount(ctx, bondUserSetting_domain.SetSelectedAccountParams{SelectedAccountID: accountOld})

	if err != nil {
		t.Fatalf("SetSelectedAccount: %v", err)
	}
	if updated.ID != legacySettingID || val(updated.SelectedAccountID) != accountOld {
		t.Fatalf("SetSelectedAccount devolvió %s con la cuenta %s, se esperaba %s con %s",
			updated.ID, val(updated.SelectedAccountID), legacySettingID, accountOld)
	}
	raw := rawDoc(t, settingCollection, legacySettingID)
	if v, ok := raw.Lookup("selectedAccountId").StringValueOK(); !ok || v != accountOld {
		t.Errorf("selectedAccountId guardado como %q, se esperaba %q", v, accountOld)
	}
	if v, ok := raw.Lookup("createdAt").TimeOK(); !ok || !v.Equal(fixtureAt) {
		t.Errorf("createdAt quedó en %v, se esperaba %v", v, fixtureAt)
	}
	if n := countSettings(t, bson.M{"userId": userID}); n != 1 {
		t.Fatalf("hay %d settings de %s después del cambio de cuenta, se esperaba 1", n, userID)
	}
}
