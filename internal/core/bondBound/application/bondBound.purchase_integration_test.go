//go:build integration

package bondBound_application_test

import (
	"errors"
	"net/http"
	"slices"
	"sync"
	"testing"

	bondBound_application "github.com/futurapps/ms-blooming-go/internal/core/bondBound/application"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// purchaseParams es la compra de un bono del producto con la cuenta.
func purchaseParams(accountID, objectID string) bondBound_domain.PurchaseParams {
	return bondBound_domain.PurchaseParams{AccountID: accountID, BondObjectID: objectID}
}

// Una compra con una cuenta propia de 5000 y un producto de 1000 deja la cuenta en 4000, un bono iniciado del
// usuario con los términos del producto, un retiro de 1000 por compra de bonos y el historial del bono y del
// producto. CA-04.1
func TestPurchaseCommits(t *testing.T) {
	setupDB(t)
	insertAccount(t, accountID, ownerID, agencyA, 5000, false)
	insertObject(t, objectID, typeID)

	result, err := integrationApp(realBalance(), realKeyCounter()).Purchase(withSession(ownerID, -5), purchaseParams(accountID, objectID))

	if err != nil {
		t.Fatalf("Purchase: %v", err)
	}
	if result.AccountAvailable != 4000 || availableOf(t, accountID) != 4000 {
		t.Errorf("accountAvailable %v y la cuenta quedó en %v, se esperaba 4000", result.AccountAvailable, availableOf(t, accountID))
	}

	bounds := findAll[bondBound_domain.BondBound](t, boundCollection)
	if len(bounds) != 1 {
		t.Fatalf("hay %d bonos, se esperaba 1", len(bounds))
	}
	b := bounds[0]
	if b.ID != result.BondBoundID || b.Key != result.Key || b.Key != "BP12-1" {
		t.Errorf("bono %s con clave %q y respuesta %+v, se esperaba la clave BP12-1 en los dos", b.ID, b.Key, result)
	}
	if b.StatusID != bondBound_domain.StatusInitiated || b.UserID != ownerID || b.AgencyID != agencyA {
		t.Errorf("statusId %q, userId %q y agencyId %q, se esperaban iniciado, %q y %q", b.StatusID, b.UserID, b.AgencyID, ownerID, agencyA)
	}
	wantTerms := shared_domain.EarlyCashTerm{
		InitialPenaltyPercentage:    100,
		InflectionPoint:             90,
		InflectionPenaltyPercentage: 75,
		InitialDayForWithdraw:       30,
	}
	if b.BondObjectID != objectID || b.BondTypeID != typeID || b.Amount != 1000 || b.MaturityTimeInDays != 365 ||
		b.MaturityRewardsPercentage != 15 || b.EarlyCashTerm != wantTerms {
		t.Errorf("bono %+v, se esperaban los términos del producto", b)
	}

	details := findAll[shared_domain.BalanceDetail](t, detailCollection)
	wantComment := `Compra de Bono "BP12-1" (` + b.ID + `)`
	if len(details) != 1 || details[0].Amount != -1000 || details[0].DescriptionID != shared_domain.BalanceDescriptionBondMD ||
		details[0].BalanceAccountID != accountID || details[0].Comment != wantComment {
		t.Errorf("balanceDetail %+v, se esperaba uno de -1000 con descriptionId 7 y comment %q", details, wantComment)
	}

	histories := findAll[bondHistory_domain.BondHistory](t, historyCollection)
	if len(histories) != 2 {
		t.Fatalf("hay %d bondHistory, se esperaban 2", len(histories))
	}
	for _, h := range histories {
		switch h.TypeID {
		case bondHistory_domain.HistoryBondBound:
			if h.RelatedID != b.ID || h.Text != shared_domain.TextBondBoundSold {
				t.Errorf("historial del bono %+v, se esperaba BOND_BOUND_SOLD de %s", h, b.ID)
			}
		case bondHistory_domain.HistoryBond:
			if h.RelatedID != objectID || h.Text != shared_domain.TextBondSold || !slices.Equal(h.Params, []string{b.ID}) {
				t.Errorf("historial del producto %+v, se esperaba BOND_SOLD de %s con params [%s]", h, objectID, b.ID)
			}
		default:
			t.Errorf("historial con typeId %q inesperado: %+v", h.TypeID, h)
		}
	}
}

// Una compra masiva de 3 con el contador del tipo en 58 crea BP12-59, BP12-60 y BP12-61 con un retiro de
// 1000 cada uno, y la respuesta trae la referencia de cada retiro. CA-04.5
func TestBulkPurchaseKeys(t *testing.T) {
	setupDB(t)
	insertAccount(t, accountID, ownerID, agencyA, 5000, false)
	insertObject(t, objectID, typeID)
	seedCounter(t, typeID, 58)

	result, err := integrationApp(realBalance(), realKeyCounter()).BulkPurchase(withSession(ownerID, -5),
		bondBound_domain.BulkPurchaseParams{PurchaseParams: purchaseParams(accountID, objectID), Quantity: 3})

	if err != nil {
		t.Fatalf("BulkPurchase: %v", err)
	}
	wantKeys := []string{"BP12-59", "BP12-60", "BP12-61"}
	keys := make([]string, len(result.Items))
	references := make([]string, len(result.Items))
	for i, item := range result.Items {
		keys[i], references[i] = item.Key, item.Reference
	}
	if !slices.Equal(keys, wantKeys) {
		t.Errorf("items con claves %v, se esperaban %v", keys, wantKeys)
	}

	bounds := findAll[bondBound_domain.BondBound](t, boundCollection)
	seqs := make([]int, 0, len(bounds))
	for _, b := range bounds {
		if b.KeySeq != nil {
			seqs = append(seqs, *b.KeySeq)
		}
	}
	slices.Sort(seqs)
	if !slices.Equal(seqs, []int{59, 60, 61}) {
		t.Errorf("bonos con keySeq %v, se esperaban 59, 60 y 61", seqs)
	}

	details := findAll[shared_domain.BalanceDetail](t, detailCollection)
	detailRefs := make([]string, 0, len(details))
	for _, d := range details {
		if d.Amount != -1000 {
			t.Errorf("balanceDetail de %v, se esperaba -1000", d.Amount)
		}
		detailRefs = append(detailRefs, d.Reference)
	}
	slices.Sort(detailRefs)
	slices.Sort(references)
	if len(details) != 3 || !slices.Equal(references, detailRefs) {
		t.Errorf("referencias de los items %v y de los balanceDetail %v, se esperaban las mismas 3", references, detailRefs)
	}

	if result.Total != 3000 || result.AccountAvailable != 2000 || availableOf(t, accountID) != 2000 {
		t.Errorf("total %v, accountAvailable %v y cuenta en %v, se esperaban 3000, 2000 y 2000",
			result.Total, result.AccountAvailable, availableOf(t, accountID))
	}
	if seq, _ := counterSeq(t, typeID); seq != 61 {
		t.Errorf("el contador quedó en %d, se esperaba 61", seq)
	}
}

// Con el contador sembrado en 7 la siguiente compra es BP12-8, y dos compras concurrentes del mismo tipo
// obtienen claves distintas. CA-04.7
func TestPurchaseAfterSeed(t *testing.T) {
	t.Run("una compra", func(t *testing.T) {
		setupDB(t)
		insertAccount(t, accountID, ownerID, agencyA, 5000, false)
		insertObject(t, objectID, typeID)
		seedCounter(t, typeID, 7)

		result, err := integrationApp(realBalance(), realKeyCounter()).Purchase(withSession(ownerID, -5), purchaseParams(accountID, objectID))

		if err != nil {
			t.Fatalf("Purchase: %v", err)
		}
		if seq, _ := counterSeq(t, typeID); result.Key != "BP12-8" || seq != 8 {
			t.Errorf("clave %q y contador en %d, se esperaban BP12-8 y 8", result.Key, seq)
		}
	})

	t.Run("dos compras concurrentes", func(t *testing.T) {
		setupDB(t)
		insertAccount(t, accountID, ownerID, agencyA, 5000, false)
		insertObject(t, objectID, typeID)
		seedCounter(t, typeID, 7)
		app := integrationApp(realBalance(), newGatedKeyCounter(2))

		results, errs := concurrentPurchases(app, 2)

		for i, err := range errs {
			if err != nil {
				t.Fatalf("la compra %d falló: %v", i, err)
			}
		}
		keys := []string{results[0].Key, results[1].Key}
		slices.Sort(keys)
		if !slices.Equal(keys, []string{"BP12-8", "BP12-9"}) {
			t.Errorf("claves %v, se esperaban BP12-8 y BP12-9", keys)
		}
		if seq, _ := counterSeq(t, typeID); seq != 9 || countDocs(t, boundCollection) != 2 {
			t.Errorf("contador en %d y %d bonos, se esperaban 9 y 2", seq, countDocs(t, boundCollection))
		}
	})
}

// Si en una compra masiva de 3 falla el tercer retiro, no queda ningún bono, movimiento de saldo, historial
// ni aviso, el contador no avanza y el saldo queda igual. CA-04.9
func TestBulkPurchaseRollback(t *testing.T) {
	setupDB(t)
	insertAccount(t, accountID, ownerID, agencyA, 5000, false)
	insertObject(t, objectID, typeID)
	seedCounter(t, typeID, 58)
	balance := &failingBalance{BalancePort: realBalance(), failOn: 3}

	_, err := integrationApp(balance, realKeyCounter()).BulkPurchase(withSession(ownerID, -5),
		bondBound_domain.BulkPurchaseParams{PurchaseParams: purchaseParams(accountID, objectID), Quantity: 3})

	if !errors.Is(err, errWithdraw) {
		t.Fatalf("BulkPurchase = %v, se esperaba el error del tercer retiro", err)
	}
	for _, c := range append([]string{boundCollection, historyCollection, noticeCollection}, movementCollections...) {
		if n := countDocs(t, c); n != 0 {
			t.Errorf("%s tiene %d documentos, se esperaba ninguno", c, n)
		}
	}
	if seq, _ := counterSeq(t, typeID); seq != 58 {
		t.Errorf("el contador quedó en %d, se esperaba 58", seq)
	}
	if available := availableOf(t, accountID); available != 5000 {
		t.Errorf("la cuenta quedó en %v, se esperaba 5000", available)
	}
}

// Con 1500 disponibles, de dos compras concurrentes de 1000 exactamente una se confirma; la otra responde
// 422 y el saldo final es 500. CA-04.10
func TestPurchaseConcurrentBalance(t *testing.T) {
	setupDB(t)
	insertAccount(t, accountID, ownerID, agencyA, 1500, false)
	insertObject(t, objectID, typeID)
	app := integrationApp(realBalance(), newGatedKeyCounter(2))

	_, errs := concurrentPurchases(app, 2)

	var failed []error
	for _, err := range errs {
		if err != nil {
			failed = append(failed, err)
		}
	}
	if len(failed) != 1 {
		t.Fatalf("fallaron %d compras (%v), se esperaba exactamente una", len(failed), errs)
	}
	// las dos ya validaron el saldo: a la segunda la frena el retiro dentro de su transacción
	if !errors.Is(failed[0], shared_domain.ErrAccountWithdrawAmountMoreThanAvailable) {
		t.Errorf("la compra rechazada falló con %v, se esperaba ErrAccountWithdrawAmountMoreThanAvailable", failed[0])
	}
	if status := bondBound_domain.MapToHTTPStatus(failed[0]); status != http.StatusUnprocessableEntity {
		t.Errorf("la compra rechazada respondería %d, se esperaba 422", status)
	}
	if available := availableOf(t, accountID); available != 500 {
		t.Errorf("la cuenta quedó en %v, se esperaba 500", available)
	}
	if bounds, details := countDocs(t, boundCollection), countDocs(t, detailCollection); bounds != 1 || details != 1 {
		t.Errorf("%d bonos y %d balanceDetail, se esperaba 1 de cada uno", bounds, details)
	}
}

// La compra de un producto legacy, con _id y bondTypeId de 17 caracteres, con una cuenta Meteor de montos
// int32, crea un bono con _id UUID que guarda los ids legacy tal cual, y el contador usa el bondTypeId
// legacy como _id. CA-04.14
func TestPurchaseLegacyIDs(t *testing.T) {
	setupDB(t)
	insertAccount(t, legacyAccountID, ownerID, agencyA, 5000, true)
	insertObject(t, legacyObjectID, legacyTypeID)

	result, err := integrationApp(realBalance(), realKeyCounter()).Purchase(withSession(ownerID, -5), purchaseParams(legacyAccountID, legacyObjectID))

	if err != nil {
		t.Fatalf("Purchase: %v", err)
	}
	if shared_domain.IDFormat(result.BondBoundID) != shared_domain.IDKindUUID {
		t.Errorf("bondBoundId %q, se esperaba un UUID v4", result.BondBoundID)
	}
	raw := rawDoc(t, boundCollection, result.BondBoundID)
	objectRef, _ := raw.Lookup("bondObjectId").StringValueOK()
	typeRef, _ := raw.Lookup("bondTypeId").StringValueOK()
	if objectRef != legacyObjectID || typeRef != legacyTypeID {
		t.Errorf("bondObjectId %q y bondTypeId %q, se esperaban los legacy %q y %q", objectRef, typeRef, legacyObjectID, legacyTypeID)
	}
	if seq, ok := counterSeq(t, legacyTypeID); !ok || seq != 1 || result.Key != "BP12-1" {
		t.Errorf("contador de %s en %d (existe %v) y clave %q, se esperaban 1 y BP12-1", legacyTypeID, seq, ok, result.Key)
	}
	if available := availableOf(t, legacyAccountID); available != 4000 || result.AccountAvailable != 4000 {
		t.Errorf("la cuenta quedó en %v y la respuesta dice %v, se esperaba 4000", available, result.AccountAvailable)
	}
}

// concurrentPurchases lanza n compras a la vez del producto con la cuenta de ownerID y espera a todas.
func concurrentPurchases(app *bondBound_application.BondBoundApplication, n int) ([]bondBound_domain.PurchaseResult, []error) {
	results := make([]bondBound_domain.PurchaseResult, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			results[i], errs[i] = app.Purchase(withSession(ownerID, -5), purchaseParams(accountID, objectID))
		})
	}
	wg.Wait()
	return results, errs
}
