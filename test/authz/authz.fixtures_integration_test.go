//go:build integration

package authz_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// collections se crean antes de levantar el servicio para que ninguna transacción tenga que crearlas, y se
// vacían antes de sembrar.
var collections = []string{
	"bondType", "bondObject", "bondBound", "bondKeyCounter", "bondHistory", "bondUserSetting", "bondJobLock",
	"balanceAccount", "balanceDetail", "balanceHistory", "balanceOperation",
	"notice", "activity", "profile", "agency", "country",
}

// IDs de los fixtures: Meteor (17 alfanuméricos) y UUID v4, con referencias cruzadas mixtas. Todo lo que se
// consulta es de la agencia A.
const (
	agencyA    = "tDr6NwSsDe4Lnccp8"
	agencyB    = "0b6f8c2e-3d4a-4e5b-9c6d-7e8f9a0b1c2d"
	agencyMain = "Mq4Hs8Kd2Lp6Wn3Rb"
	countryID  = "Zp5Rk1Mf7Gc3Qs8Ju"

	userID      = "Hn4Wq8Ls2Dx6Vb9Ty"
	ownerID     = "6f1e2d3c-4b5a-4c9d-8e7f-0a1b2c3d4e5f"
	adminAID    = "8d7c6b5a-4f3e-4d2c-a1b0-9f8e7d6c5b4a"
	adminBID    = "Ad8Bn2Cv6Xz4Lk9Jq"
	adminMainID = "2a3b4c5d-6e7f-4a8b-9c0d-1e2f3a4b5c6d"

	accountID       = "9d8c7b6a-5f4e-4d3c-8b2a-1f0e9d8c7b6a"
	typeID          = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"
	emptyTypeID     = "kQ3xT9mPz2LwA7bNc"
	objectID        = "Bd9Xc4Vn7Mz2Lk5Jh"
	boundID         = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"
	settingID       = "Rv6Tb1Yn5Um9Io3Pa"
	objectHistoryID = "5c4b3a29-1807-4f6e-9d5c-4b3a29180716"
	boundHistoryID  = "Wm3Ex7Rt1Yu5Io9Pq"
)

var fixtureIDs = []string{
	agencyA, agencyB, agencyMain, countryID, userID, ownerID, adminAID, adminBID, adminMainID,
	accountID, typeID, emptyTypeID, objectID, boundID, settingID, objectHistoryID, boundHistoryID,
}

// fixtureAt es la fecha de alta del catálogo, la cuenta y los perfiles.
var fixtureAt = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

// utcDiff es la diferencia horaria de las sesiones.
const utcDiff = -5

// role es quién llama a la ruta.
type role string

const (
	// roleUser es un inversionista de A que no es dueño de nada.
	roleUser role = "usuario"
	// roleOwner es un inversionista de A, dueño del bono, la cuenta y el setting.
	roleOwner role = "dueño"
	// roleAdminA es un admin de A, la agencia de los recursos, que no es la principal.
	roleAdminA role = "admin de A"
	// roleAdminB es un admin de B, otra agencia que no es la principal.
	roleAdminB role = "admin de otra agencia"
	// roleAdminMain es un admin de la agencia principal, que ve todas.
	roleAdminMain role = "admin de la principal"
)

// roles son las columnas de la matriz, en orden.
var roles = []role{roleUser, roleOwner, roleAdminA, roleAdminB, roleAdminMain}

// tokens son los x-token de cada rol.
var tokens = map[role]string{
	roleUser:      "tok-authz-user",
	roleOwner:     "tok-authz-owner",
	roleAdminA:    "tok-authz-admin-a",
	roleAdminB:    "tok-authz-admin-b",
	roleAdminMain: "tok-authz-admin-main",
}

// sessions son las sesiones de ms-oauth por token.
var sessions = map[string]map[string]any{
	tokens[roleUser]:      session(userID, roleUser, agencyA, false, false),
	tokens[roleOwner]:     session(ownerID, roleOwner, agencyA, false, false),
	tokens[roleAdminA]:    session(adminAID, roleAdminA, agencyA, false, true),
	tokens[roleAdminB]:    session(adminBID, roleAdminB, agencyB, false, true),
	tokens[roleAdminMain]: session(adminMainID, roleAdminMain, agencyMain, true, true),
}

// session arma la sesión de ms-oauth de un usuario de agencyID; isMain indica si es la agencia principal.
func session(id string, r role, agencyID string, isMain, admin bool) map[string]any {
	return map[string]any{
		"userId":        id,
		"token":         tokens[r],
		"agencyId":      agencyID,
		"agency":        map[string]any{"_id": agencyID, "name": "Agencia " + agencyID, "isMain": isMain},
		"countryCode":   "CO",
		"first_name":    string(r),
		"last_name":     "Prueba",
		"email":         id + "@futurapps.test",
		"adminBlooming": admin,
		"utcDiff":       utcDiff,
	}
}

// earlyCashTerm son los términos del producto y del bono: se puede retirar desde el primer día, sin punto
// de inflexión y con el 50 % de penalidad.
var earlyCashTerm = bson.M{
	"initialPenaltyPercentage":    100.0,
	"inflectionPoint":             0,
	"inflectionPenaltyPercentage": 50.0,
	"blockWithdrawals":            false,
	"initialDayForWithdraw":       0,
}

// dirty indica que el último request pudo escribir y hay que volver a sembrar.
var dirty = true

// prepare deja la base con los fixtures antes de un request: siempre antes de uno que escribe, y antes de
// uno que lee solo si el anterior pudo escribir. Sin Mongo, salta el test.
func prepare(t *testing.T, method string) {
	t.Helper()
	if skipReason != "" {
		t.Skip("sin Mongo para integración: " + skipReason)
	}
	if dirty || method != http.MethodGet {
		reset(t)
	}
	dirty = method != http.MethodGet
}

// reset vacía las colecciones, los índices quedan, y vuelve a sembrar los fixtures.
func reset(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	for _, name := range collections {
		if _, err := testDB.GetCollection(name).DeleteMany(ctx, bson.M{}); err != nil {
			t.Fatalf("no se pudo vaciar %s: %v", name, err)
		}
	}
	seed(t)
}

// seed guarda las tres agencias y el país, los perfiles, el catálogo de A (un tipo con un producto público y
// otro sin productos), la cuenta y el setting del dueño, su bono comprado hace 50 días, que hoy se puede
// vender, y un registro de historial del producto y otro del bono.
func seed(t *testing.T) {
	t.Helper()
	insert(t, "agency",
		bson.M{"_id": agencyA, "name": "Agencia A", "countryCode": "CO", "main": false, "isRemove": false},
		bson.M{"_id": agencyB, "name": "Agencia B", "countryCode": "CO", "main": false, "isRemove": false},
		bson.M{"_id": agencyMain, "name": "Agencia principal", "countryCode": "CO", "main": true, "isRemove": false},
	)
	insert(t, "country", bson.M{"_id": countryID, "name_es": "Colombia", "code": "CO", "symbolCurrency": "Pesos $", "isRemove": false})
	for _, p := range []struct{ id, agencyID string }{
		{userID, agencyA}, {ownerID, agencyA}, {adminAID, agencyA}, {adminBID, agencyB}, {adminMainID, agencyMain},
	} {
		insert(t, "profile", bson.M{
			"_id": p.id, "id": "CC-" + p.id[:6], "first_name": "Usuario", "last_name": p.id[:6],
			"email": p.id + "@futurapps.test", "agencyId": p.agencyID,
			"createdAt": fixtureAt, "updatedAt": fixtureAt, "isRemove": false,
		})
	}

	for _, id := range []string{typeID, emptyTypeID} {
		insert(t, "bondType", bson.M{
			"_id": id, "name": "Renta fija " + id[:4], "description": "Bonos de renta fija", "agencyId": bson.A{agencyA},
			"active": true, "createdAt": fixtureAt, "updatedAt": fixtureAt, "isRemove": false,
		})
	}
	insert(t, "bondObject", bson.M{
		"_id": objectID, "name": "Bono Plus 12M", "description": "Renta fija 100 días", "agencyId": bson.A{agencyA},
		"key": "BP12", "bondTypeId": typeID, "amount": 1000.0, "isPrivateBond": false, "authorizedUserIds": bson.A{},
		"maturityTimeInDays": 100, "maturityRewardsPercentage": 10.0, "active": true, "available": true,
		"saleInitDate": nil, "saleEndDate": nil, "earlyCashTerm": earlyCashTerm,
		"createdAt": fixtureAt, "updatedAt": fixtureAt, "isRemove": false,
	})

	insert(t, "balanceAccount", bson.M{
		"_id": accountID, "userId": ownerID, "agencyId": agencyA, "number": "4000123412341234",
		"statusId": "1", "typeId": "1", "available": 5000.0, "blocked": 0.0, "deferred": 0.0, "average": 0.0,
		"global": 5000.0, "initialAmount": 0.0, "initialMonthAmount": 0.0, "openedBy": ownerID,
		"createdAt": fixtureAt, "updatedAt": fixtureAt, "isRemove": false,
	})
	insert(t, "bondUserSetting", bson.M{
		"_id": settingID, "userId": ownerID, "agencyId": agencyA, "language": "es",
		"selectedAccountId": accountID, "selectedBondBoundId": nil,
		"createdAt": fixtureAt, "updatedAt": fixtureAt, "isRemove": false,
	})

	initial := time.Now().UTC().AddDate(0, 0, -50)
	insert(t, "bondBound", bson.M{
		"_id": boundID, "bondObjectId": objectID, "bondTypeId": typeID, "key": "BP12-1", "keySeq": 1,
		"token": "Ab3dE6", "userId": ownerID, "agencyId": agencyA, "statusId": "1", "amount": 1000.0,
		"maturityTimeInDays": 100, "maturityRewardsPercentage": 10.0, "earlyCashTerm": earlyCashTerm,
		"initialDate": initial, "maturityDate": initial.AddDate(0, 0, 100),
		"createdAt": initial, "updatedAt": initial, "isRemove": false,
	})
	// el contador del tipo va por el bono ya comprado, como lo deja la siembra de contadores
	insert(t, "bondKeyCounter", bson.M{"_id": typeID, "seq": 1})

	insert(t, "bondHistory",
		bson.M{
			"_id": objectHistoryID, "relatedId": objectID, "userId": adminAID, "typeId": string(bondHistory_domain.HistoryBond),
			"agencyId": agencyA, "text": shared_domain.TextBondAdded, "params": bson.A{},
			"createdAt": fixtureAt, "updatedAt": fixtureAt, "isRemove": false,
		},
		bson.M{
			"_id": boundHistoryID, "relatedId": boundID, "userId": ownerID, "typeId": string(bondHistory_domain.HistoryBondBound),
			"agencyId": agencyA, "text": shared_domain.TextBondBoundPurchased, "params": bson.A{},
			"createdAt": initial, "updatedAt": initial, "isRemove": false,
		},
	)
}

// insert guarda los documentos crudos en collection.
func insert(t *testing.T, collection string, docs ...bson.M) {
	t.Helper()
	for _, doc := range docs {
		if _, err := testDB.InsertOne(context.Background(), collection, doc); err != nil {
			t.Fatalf("no se pudo guardar %v en %s: %v", doc["_id"], collection, err)
		}
	}
}

// apiClient es el cliente de los tests; el job de maduración puede tardar más que un request común.
var apiClient = &http.Client{Timeout: 30 * time.Second}

// response es la respuesta de la API.
type response struct {
	status int
	body   []byte
}

// call manda un request a apiURL+path; con secret manda x-secret, y con token, x-token. body, si no es nil,
// va como JSON.
func call(t *testing.T, method, path string, secret bool, token string, body any) response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("no se pudo serializar el body de %s %s: %v", method, path, err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, apiURL+path, reader)
	if err != nil {
		t.Fatalf("no se pudo armar %s %s: %v", method, path, err)
	}
	if secret {
		req.Header.Set("x-secret", apiSecret)
	}
	if token != "" {
		req.Header.Set("x-token", token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := apiClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("no se pudo leer la respuesta de %s %s: %v", method, path, err)
	}
	return response{status: resp.StatusCode, body: data}
}
