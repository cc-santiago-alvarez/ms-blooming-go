//go:build integration

package bondUserSetting_mongocc_repository_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Un usuario sin setting y con dos cuentas vivas en su agencia recibe, en GET /me, un setting nuevo con
// idioma "es", la cuenta más antigua de la agencia (ni la borrada ni la de otra agencia, aunque sean
// anteriores) y selectedBondBoundId null. Una segunda llamada devuelve el mismo _id. CA-09.1
func TestGetMineCreatesDefault(t *testing.T) {
	setup(t)
	insert(t, accountCollection,
		accountDoc(accountRemoved, userID, agencyA, time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), bson.M{"isRemove": true}),
		accountDoc(accountOtherAgency, userID, agencyB, time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC), nil),
		accountDoc(accountNew, userID, agencyA, time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC), nil),
		accountDoc(accountOld, userID, agencyA, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC), nil),
	)
	insert(t, profileCollection, profileDoc(userID, "Juan"))
	app := newApp()
	ctx := investorSession(userID, agencyA)

	got, err := app.GetMine(ctx)

	if err != nil {
		t.Fatalf("GetMine: %v", err)
	}
	if shared_domain.IDFormat(got.ID) != shared_domain.IDKindUUID {
		t.Errorf("_id %q no es un UUID v4", got.ID)
	}
	if got.UserID != userID || got.AgencyID != agencyA || got.Language != "es" ||
		val(got.SelectedAccountID) != accountOld || got.SelectedBondBoundID != nil {
		t.Fatalf("GetMine creó el setting de %s en %s, idioma %q, cuenta %s y bono %s; se esperaba el de %s en %s, \"es\", cuenta %s y bono <nil>",
			got.UserID, got.AgencyID, got.Language, val(got.SelectedAccountID), val(got.SelectedBondBoundID), userID, agencyA, accountOld)
	}
	if got.Account == nil || got.Account.ID != accountOld {
		t.Errorf("account %+v, se esperaba la cuenta %s", got.Account, accountOld)
	}

	raw := rawDoc(t, settingCollection, got.ID)
	if v, err := raw.LookupErr("selectedBondBoundId"); err != nil || v.Type != bson.TypeNull {
		t.Errorf("selectedBondBoundId guardado como %v (%v), se esperaba null", v, err)
	}
	if v, ok := raw.Lookup("selectedAccountId").StringValueOK(); !ok || v != accountOld {
		t.Errorf("selectedAccountId guardado como %q, se esperaba %q", v, accountOld)
	}

	again, err := app.GetMine(ctx)

	if err != nil {
		t.Fatalf("segunda llamada a GetMine: %v", err)
	}
	if again.ID != got.ID {
		t.Fatalf("la segunda llamada devolvió %s, se esperaba %s", again.ID, got.ID)
	}
	if n := countSettings(t, bson.M{"userId": userID}); n != 1 {
		t.Fatalf("hay %d settings de %s, se esperaba 1", n, userID)
	}
}

// Diez llamadas concurrentes a GET /me de un usuario nuevo crean exactamente un documento, y todas
// devuelven ese mismo setting. CA-09.3
func TestGetMineConcurrentCreatesOne(t *testing.T) {
	setup(t)
	insert(t, accountCollection, accountDoc(accountNew, newUserID, agencyA, time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC), nil))
	app := newApp()

	const calls = 10
	ids := make([]string, calls)
	errs := make([]error, calls)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range calls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			s, err := app.GetMine(investorSession(newUserID, agencyA))
			errs[i] = err
			if s != nil {
				ids[i] = s.ID
			}
		}()
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("la llamada %d falló: %v", i, err)
		}
	}
	for i, id := range ids {
		if id != ids[0] {
			t.Fatalf("la llamada %d devolvió %s y la 0 devolvió %s; se esperaba el mismo setting", i, id, ids[0])
		}
	}
	if n := countSettings(t, bson.M{"userId": newUserID}); n != 1 {
		t.Fatalf("hay %d settings de %s, se esperaba exactamente 1", n, newUserID)
	}
}

// Con bonos propios en estado '1' (1000 y 500), '2' (2000) y '5' (700), boundSummary es {investment 1500,
// matured 2000, profit 0}, calculado en una sola agregación; los bonos de otro usuario, de otra agencia o
// borrados no suman. account.number sale enmascarado y user no trae password. CA-09.4
func TestGetMineSummary(t *testing.T) {
	setup(t)
	insert(t, settingCollection, settingDoc(settingID, userID, agencyA, bson.M{"selectedAccountId": accountOld}))
	insert(t, accountCollection, accountDoc(accountOld, userID, agencyA, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC), nil))
	insert(t, profileCollection, profileDoc(userID, "Juan"))
	insert(t, boundCollection,
		boundDoc(userID, agencyA, "1", 1000.0, nil),
		boundDoc(userID, agencyA, "1", int32(500), nil),
		boundDoc(userID, agencyA, "2", 2000.0, nil),
		boundDoc(userID, agencyA, "5", 700.0, nil),
		boundDoc(user2ID, agencyA, "1", 9999.0, nil),
		boundDoc(userID, agencyB, "1", 9999.0, nil),
		boundDoc(userID, agencyA, "2", 9999.0, bson.M{"isRemove": true}),
	)
	commands.reset()

	got, err := newApp().GetMine(investorSession(userID, agencyA))

	if err != nil {
		t.Fatalf("GetMine: %v", err)
	}
	if want := (bondUserSetting_domain.BoundSummary{Investment: 1500, Matured: 2000, Profit: 0}); got.BoundSummary != want {
		t.Errorf("boundSummary %+v, se esperaba %+v", got.BoundSummary, want)
	}
	if n := commands.count("aggregate", boundCollection); n != 1 {
		t.Errorf("se mandaron %d agregaciones sobre %s, se esperaba 1", n, boundCollection)
	}
	if got.Account == nil || got.Account.ID != accountOld {
		t.Fatalf("account %+v, se esperaba la cuenta %s", got.Account, accountOld)
	}
	if got.Account.Number != shared_domain.MaskAccountNumber(accountNumber) || strings.Contains(got.Account.Number, accountNumber) {
		t.Errorf("account.number %q, se esperaba %q", got.Account.Number, shared_domain.MaskAccountNumber(accountNumber))
	}
	if got.User == nil || got.User.ID != userID {
		t.Fatalf("user %+v, se esperaba el perfil de %s", got.User, userID)
	}
	if got.UtcDiff != investorUtcDiff {
		t.Errorf("utcDiff %d, se esperaba %d", got.UtcDiff, investorUtcDiff)
	}

	body, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("no se pudo serializar el detalle: %v", err)
	}
	if bytes.Contains(body, []byte("password")) || bytes.Contains(body, []byte(passwordHash)) {
		t.Fatalf("la respuesta expone el password: %s", body)
	}
}
