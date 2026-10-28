//go:build integration

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// collections se crean antes de levantar el servicio para que ninguna transacción tenga que crearlas, y se
// vacían antes de cada test.
var collections = []string{
	"bondType", "bondObject", "bondBound", "bondKeyCounter", "bondHistory", "bondUserSetting", "bondJobLock",
	"balanceAccount", "balanceDetail", "balanceHistory", "balanceOperation",
	"notice", "activity", "profile", "agency", "country",
}

// IDs de los fixtures: Meteor (17 alfanuméricos) y UUID v4, con referencias cruzadas mixtas.
const (
	agencyID   = "tDr6NwSsDe4Lnccp8"
	investorID = "Hn4Wq8Ls2Dx6Vb9Ty"
	adminID    = "8d7c6b5a-4f3e-4d2c-a1b0-9f8e7d6c5b4a"
	accountID  = "9d8c7b6a-5f4e-4d3c-8b2a-1f0e9d8c7b6a"
	typeID     = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"
	objectID   = "Bd9Xc4Vn7Mz2Lk5Jh"
	countryID  = "Zp5Rk1Mf7Gc3Qs8Ju"
)

var fixtureIDs = []string{agencyID, investorID, adminID, accountID, typeID, objectID, countryID}

// fixtureAt es la fecha de alta del catálogo, la cuenta y el perfil.
var fixtureAt = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

// Tokens de las sesiones que responde ms-oauth.
const (
	investorToken = "tok-e2e-investor"
	adminToken    = "tok-e2e-admin"
)

// utcDiff es la diferencia horaria de las sesiones.
const utcDiff = -5

// sessions son las sesiones de ms-oauth por token: un inversionista y un admin de la misma agencia.
var sessions = map[string]map[string]any{
	investorToken: session(investorID, investorToken, "Ana", "Gómez", false),
	adminToken:    session(adminID, adminToken, "Luis", "Pérez", true),
}

// session arma la sesión de ms-oauth de un usuario de la agencia de los fixtures.
func session(userID, token, firstName, lastName string, admin bool) map[string]any {
	return map[string]any{
		"userId":        userID,
		"token":         token,
		"agencyId":      agencyID,
		"agency":        map[string]any{"_id": agencyID, "name": agencyName, "isMain": false},
		"countryCode":   "CO",
		"first_name":    firstName,
		"last_name":     lastName,
		"email":         firstName + "@futurapps.test",
		"adminBlooming": admin,
		"utcDiff":       utcDiff,
	}
}

// Catálogo, cuenta y agencia de los fixtures.
const (
	agencyName  = "Futurapps Colombia"
	currency    = "Pesos $"
	typeName    = "Renta fija"
	productName = "Bono Plus 12M"
	productKey  = "BP12"
	// price es el precio del producto.
	price = 1000.0
	// maturityDays y rewards son el plazo y el rendimiento del producto.
	maturityDays = 100
	rewards      = 10.0
	// initialAvailable es el saldo de la cuenta antes de comprar.
	initialAvailable = 5000.0
)

// Avisos de ms-notification, por path.
const (
	notifyPurchased     = "/notify/user-ids/notify-blooming-bondPurchased"
	notifyBulkPurchased = "/notify/user-ids/notify-blooming-bondBulkPurchased"
	notifyWithdrawUser  = "/notify/user-ids/notify-blooming-withdrawUser"
	notifyWithdrawAdmin = "/notify/to-raw-emails/notify-blooming-withdrawAdmin"
	notifySold          = "/notify/user-ids/notify-blooming-bondSold"
	notifyTook          = "/notify/user-ids/notify-blooming-bondTook"
	notifyMatured       = "/notify/internal/user-ids/notify-blooming-bondMatured"
)

// setup vacía las colecciones, los índices quedan, y los registros de los servicios falsos. Sin Mongo,
// salta el test.
func setup(t *testing.T) {
	t.Helper()
	if skipReason != "" {
		t.Skip("sin Mongo para integración: " + skipReason)
	}
	ctx := context.Background()
	for _, name := range collections {
		if _, err := testDB.GetCollection(name).DeleteMany(ctx, bson.M{}); err != nil {
			t.Fatalf("no se pudo vaciar %s: %v", name, err)
		}
	}
	notifications.reset()
	exporter.reset()
}

// seed guarda el tipo y el producto, la cuenta y el perfil del inversionista, y la agencia con su país. El
// producto es público, sin ventana de venta y se puede retirar desde el primer día, sin punto de inflexión
// y con el 50 % de penalidad.
func seed(t *testing.T) {
	t.Helper()
	insert(t, "bondType", bson.M{
		"_id":         typeID,
		"name":        typeName,
		"description": "Bonos de renta fija",
		"agencyId":    bson.A{agencyID},
		"active":      true,
		"createdAt":   fixtureAt,
		"updatedAt":   fixtureAt,
		"isRemove":    false,
	})
	insert(t, "bondObject", bson.M{
		"_id":                       objectID,
		"name":                      productName,
		"description":               "Renta fija 100 días",
		"agencyId":                  bson.A{agencyID},
		"key":                       productKey,
		"bondTypeId":                typeID,
		"amount":                    price,
		"isPrivateBond":             false,
		"authorizedUserIds":         bson.A{},
		"maturityTimeInDays":        maturityDays,
		"maturityRewardsPercentage": rewards,
		"active":                    true,
		"available":                 true,
		"saleInitDate":              nil,
		"saleEndDate":               nil,
		"earlyCashTerm": bson.M{
			"initialPenaltyPercentage":    100.0,
			"inflectionPoint":             0,
			"inflectionPenaltyPercentage": 50.0,
			"blockWithdrawals":            false,
			"initialDayForWithdraw":       0,
		},
		"createdAt": fixtureAt,
		"updatedAt": fixtureAt,
		"isRemove":  false,
	})
	insert(t, "balanceAccount", bson.M{
		"_id":                accountID,
		"userId":             investorID,
		"agencyId":           agencyID,
		"number":             "4000123412341234",
		"statusId":           "1",
		"typeId":             "1",
		"available":          initialAvailable,
		"blocked":            0.0,
		"deferred":           0.0,
		"average":            0.0,
		"global":             initialAvailable,
		"initialAmount":      0.0,
		"initialMonthAmount": 0.0,
		"openedBy":           investorID,
		"createdAt":          fixtureAt,
		"updatedAt":          fixtureAt,
		"isRemove":           false,
	})
	insert(t, "profile", bson.M{
		"_id":        investorID,
		"id":         "CC-1020304050",
		"first_name": "Ana",
		"last_name":  "Gómez",
		"email":      "ana@futurapps.test",
		"agencyId":   agencyID,
		"createdAt":  fixtureAt,
		"updatedAt":  fixtureAt,
		"isRemove":   false,
	})
	insert(t, "agency", bson.M{"_id": agencyID, "name": agencyName, "countryCode": "CO", "isRemove": false})
	insert(t, "country", bson.M{"_id": countryID, "name_es": "Colombia", "code": "CO", "symbolCurrency": currency, "isRemove": false})
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

// rawDoc lee el documento tal como quedó en Mongo.
func rawDoc(t *testing.T, collection, id string) bson.M {
	t.Helper()
	var doc bson.M
	if err := testDB.FindOne(context.Background(), collection, bson.M{"_id": id}, nil).Decode(&doc); err != nil {
		t.Fatalf("no se pudo leer %s de %s: %v", id, collection, err)
	}
	return doc
}

// available devuelve el saldo disponible guardado en la cuenta del inversionista.
func available(t *testing.T) float64 {
	t.Helper()
	var account struct {
		Available float64 `bson:"available"`
	}
	if err := testDB.FindOne(context.Background(), "balanceAccount", bson.M{"_id": accountID}, nil).Decode(&account); err != nil {
		t.Fatalf("no se pudo leer la cuenta %s: %v", accountID, err)
	}
	return account.Available
}

// backdate corre la compra del bono days días al pasado; el servicio usa su propio reloj.
func backdate(t *testing.T, boundID string, days int) {
	t.Helper()
	doc := rawDoc(t, "bondBound", boundID)
	set := bson.M{}
	for _, field := range []string{"initialDate", "maturityDate", "createdAt"} {
		date, ok := doc[field].(bson.DateTime)
		if !ok {
			t.Fatalf("%s del bono %s = %v, se esperaba una fecha", field, boundID, doc[field])
		}
		set[field] = date.Time().UTC().AddDate(0, 0, -days)
	}
	if _, err := testDB.GetCollection("bondBound").UpdateOne(context.Background(), bson.M{"_id": boundID}, bson.M{"$set": set}); err != nil {
		t.Fatalf("no se pudo correr las fechas del bono %s: %v", boundID, err)
	}
}

// dateRange es el filtro de fechas de ayer a mañana en la zona de las sesiones: incluye lo comprado
// durante el test aunque cruce la medianoche.
func dateRange() string {
	today := time.Now().In(time.FixedZone("", utcDiff*3600))
	return "filter=betweenDates&startDate=" + today.AddDate(0, 0, -1).Format(time.DateOnly) +
		"&endDate=" + today.AddDate(0, 0, 1).Format(time.DateOnly)
}

// apiClient es el cliente de los tests; el job de maduración puede tardar más que un request común.
var apiClient = &http.Client{Timeout: 30 * time.Second}

// response es la respuesta de la API a request, "MÉTODO /path".
type response struct {
	request string
	status  int
	header  http.Header
	body    []byte
}

// call manda un request a /api/v1 con x-secret y, si hay token, x-token; body, si no es nil, va como JSON.
func call(t *testing.T, method, path, token string, body any) response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("no se pudo serializar el body de %s %s: %v", method, path, err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, apiURL+"/api/v1"+path, reader)
	if err != nil {
		t.Fatalf("no se pudo armar %s %s: %v", method, path, err)
	}
	req.Header.Set("x-secret", apiSecret)
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
	return response{request: method + " " + path, status: resp.StatusCode, header: resp.Header, body: data}
}

// expect exige status y decodifica el body JSON en out, si no es nil.
func (r response) expect(t *testing.T, status int, out any) {
	t.Helper()
	if r.status != status {
		t.Fatalf("%s = %d, se esperaba %d; body: %s", r.request, r.status, status, r.body)
	}
	if out != nil {
		if err := json.Unmarshal(r.body, out); err != nil {
			t.Fatalf("%s: body %s inválido: %v", r.request, r.body, err)
		}
	}
}

// expectError exige status y el código de error en message.
func (r response) expectError(t *testing.T, status int, code string) {
	t.Helper()
	var body struct {
		Message string `json:"message"`
	}
	r.expect(t, status, &body)
	if body.Message != code {
		t.Fatalf("%s respondió %q, se esperaba %q", r.request, body.Message, code)
	}
}

// purchaseResult es la respuesta de la compra.
type purchaseResult struct {
	BondBoundID      string  `json:"bondBoundId"`
	Key              string  `json:"key"`
	AccountAvailable float64 `json:"accountAvailable"`
}

// saleResult es la respuesta de la venta o el canje.
type saleResult struct {
	BondBoundID string  `json:"bondBoundId"`
	StatusID    string  `json:"statusId"`
	Amount      float64 `json:"amount"`
	Profit      float64 `json:"profit"`
	AccountID   string  `json:"accountId"`
	Available   float64 `json:"available"`
}

// selectAccount elige la cuenta del inversionista como destino de sus ventas.
func selectAccount(t *testing.T) {
	t.Helper()
	call(t, http.MethodPatch, "/bond-user-settings/me/selected-account", investorToken,
		map[string]string{"selectedAccountId": accountID}).expect(t, http.StatusOK, nil)
}

// purchase compra el producto con la cuenta del inversionista.
func purchase(t *testing.T) purchaseResult {
	t.Helper()
	var bought purchaseResult
	call(t, http.MethodPost, "/bond-bounds/purchase", investorToken,
		map[string]string{"accountId": accountID, "bondObjectId": objectID}).expect(t, http.StatusCreated, &bought)
	return bought
}

// sell vende o canjea el bono como el inversionista.
func sell(t *testing.T, boundID string) saleResult {
	t.Helper()
	var sold saleResult
	call(t, http.MethodPost, "/bond-bounds/"+boundID+"/sale", investorToken, nil).expect(t, http.StatusOK, &sold)
	return sold
}

// assertBoundStatus verifica el estado del bono que ve su dueño.
func assertBoundStatus(t *testing.T, boundID, want string) {
	t.Helper()
	var bound struct {
		StatusID string `json:"statusId"`
	}
	call(t, http.MethodGet, "/bond-bounds/"+boundID, investorToken, nil).expect(t, http.StatusOK, &bound)
	if bound.StatusID != want {
		t.Fatalf("statusId del bono %s = %q, se esperaba %q", boundID, bound.StatusID, want)
	}
}

// assertAvailable verifica el saldo guardado en la cuenta del inversionista.
func assertAvailable(t *testing.T, want float64) {
	t.Helper()
	if got := available(t); got != want {
		t.Fatalf("saldo guardado = %v, se esperaba %v", got, want)
	}
}
