//go:build load

package load_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"slices"
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
	// otherTypeID es el segundo tipo de la siembra del listado.
	otherTypeID = "Tq7Wm2Xc9Vb4Nz6Lp"
)

var fixtureIDs = []string{agencyID, investorID, adminID, accountID, typeID, objectID, countryID, otherTypeID}

// fixtureAt es la fecha de alta del catálogo, la cuenta y los perfiles.
var fixtureAt = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

// Tokens de las sesiones que responde ms-oauth.
const (
	investorToken = "tok-load-investor"
	adminToken    = "tok-load-admin"
)

// utcDiff es la diferencia horaria de las sesiones.
const utcDiff = -5

// sessions son las sesiones de ms-oauth por token: un inversionista y un admin de la misma agencia, que no
// es la principal.
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

// Catálogo y agencia de los fixtures.
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
)

// earlyCashTerm son las condiciones de retiro de los productos y de sus bonos: se puede retirar desde el
// primer día, sin punto de inflexión y con el 50 % de penalidad.
var earlyCashTerm = bson.M{
	"initialPenaltyPercentage":    100.0,
	"inflectionPoint":             0,
	"inflectionPenaltyPercentage": 50.0,
	"blockWithdrawals":            false,
	"initialDayForWithdraw":       0,
}

// setup vacía las colecciones; los índices quedan. Sin Mongo, salta el test.
func setup(t *testing.T) {
	t.Helper()
	if skipReason != "" {
		t.Skip("sin Mongo para la prueba de carga: " + skipReason)
	}
	ctx := context.Background()
	for _, name := range collections {
		if _, err := testDB.GetCollection(name).DeleteMany(ctx, bson.M{}); err != nil {
			t.Fatalf("no se pudo vaciar %s: %v", name, err)
		}
	}
}

// seedCatalog guarda el tipo y el producto, público y sin ventana de venta, el perfil del inversionista y la
// agencia con su país.
func seedCatalog(t *testing.T) {
	t.Helper()
	insert(t, "bondType", typeDoc(typeID, typeName))
	insert(t, "bondObject", objectDoc(objectID, typeID, productName, productKey))
	insert(t, "profile", profileDoc(investorID, "CC-1020304050", "Ana", "Gómez", "ana@futurapps.test"))
	insert(t, "agency", bson.M{"_id": agencyID, "name": agencyName, "countryCode": "CO", "isRemove": false})
	insert(t, "country", bson.M{"_id": countryID, "name_es": "Colombia", "code": "CO", "symbolCurrency": currency, "isRemove": false})
}

// typeDoc es un tipo activo de la agencia.
func typeDoc(id, name string) bson.M {
	return bson.M{
		"_id":         id,
		"name":        name,
		"description": "Bonos de " + name,
		"agencyId":    bson.A{agencyID},
		"active":      true,
		"createdAt":   fixtureAt,
		"updatedAt":   fixtureAt,
		"isRemove":    false,
	}
}

// objectDoc es un producto público y activo de la agencia, al precio de los fixtures.
func objectDoc(id, typeID, name, key string) bson.M {
	return bson.M{
		"_id":                       id,
		"name":                      name,
		"description":               "Renta fija 100 días",
		"agencyId":                  bson.A{agencyID},
		"key":                       key,
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
		"earlyCashTerm":             earlyCashTerm,
		"createdAt":                 fixtureAt,
		"updatedAt":                 fixtureAt,
		"isRemove":                  false,
	}
}

// profileDoc es el perfil de un usuario de la agencia.
func profileDoc(id, document, firstName, lastName, email string) bson.M {
	return bson.M{
		"_id":        id,
		"id":         document,
		"first_name": firstName,
		"last_name":  lastName,
		"email":      email,
		"agencyId":   agencyID,
		"createdAt":  fixtureAt,
		"updatedAt":  fixtureAt,
		"isRemove":   false,
	}
}

// insertAccount guarda la cuenta activa del inversionista con el saldo disponible.
func insertAccount(t *testing.T, available float64) {
	t.Helper()
	insert(t, "balanceAccount", bson.M{
		"_id":                accountID,
		"userId":             investorID,
		"agencyId":           agencyID,
		"number":             "4000123412341234",
		"statusId":           "1",
		"typeId":             "1",
		"available":          available,
		"blocked":            0.0,
		"deferred":           0.0,
		"average":            0.0,
		"global":             available,
		"initialAmount":      0.0,
		"initialMonthAmount": 0.0,
		"openedBy":           investorID,
		"createdAt":          fixtureAt,
		"updatedAt":          fixtureAt,
		"isRemove":           false,
	})
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

// count devuelve la cantidad de documentos de collection que pasan filter.
func count(t *testing.T, collection string, filter bson.M) int64 {
	t.Helper()
	n, err := testDB.GetCollection(collection).CountDocuments(context.Background(), filter)
	if err != nil {
		t.Fatalf("no se pudo contar %s: %v", collection, err)
	}
	return n
}

// apiClient es el cliente de los tests; mantiene abiertas las conexiones de todos los clientes concurrentes.
var apiClient = &http.Client{
	Timeout:   30 * time.Second,
	Transport: &http.Transport{MaxIdleConns: 64, MaxIdleConnsPerHost: 64},
}

// response es la respuesta de la API a request, "MÉTODO /path", y lo que tardó.
type response struct {
	request string
	status  int
	body    []byte
	took    time.Duration
}

// send manda un request a /api/v1 con x-secret y, si hay token, x-token; body, si no es nil, va como JSON.
// No usa t, así que se puede llamar desde cualquier goroutine.
func send(method, path, token string, body any) (response, error) {
	res := response{request: method + " " + path}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return res, fmt.Errorf("no se pudo serializar el body de %s: %w", res.request, err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, apiURL+"/api/v1"+path, reader)
	if err != nil {
		return res, fmt.Errorf("no se pudo armar %s: %w", res.request, err)
	}
	req.Header.Set("x-secret", apiSecret)
	if token != "" {
		req.Header.Set("x-token", token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	resp, err := apiClient.Do(req)
	if err != nil {
		return res, fmt.Errorf("%s: %w", res.request, err)
	}
	defer resp.Body.Close()
	res.body, err = io.ReadAll(resp.Body)
	res.took = time.Since(start)
	if err != nil {
		return res, fmt.Errorf("no se pudo leer la respuesta de %s: %w", res.request, err)
	}
	res.status = resp.StatusCode
	return res, nil
}

// decode exige status y decodifica el body JSON en out, si no es nil.
func (r response) decode(status int, out any) error {
	if r.status != status {
		return fmt.Errorf("%s = %d, se esperaba %d; body: %s", r.request, r.status, status, r.body)
	}
	if out != nil {
		if err := json.Unmarshal(r.body, out); err != nil {
			return fmt.Errorf("%s: body %s inválido: %w", r.request, r.body, err)
		}
	}
	return nil
}

// message devuelve el código de error de la respuesta.
func (r response) message() string {
	var body struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(r.body, &body)
	return body.Message
}

// percentile devuelve el percentil p (0 a 100) de durations por rango más cercano.
func percentile(durations []time.Duration, p float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	sorted := slices.Sorted(slices.Values(durations))
	rank := int(math.Ceil(float64(len(sorted))*p/100)) - 1
	return sorted[min(max(rank, 0), len(sorted)-1)]
}

// logLatencies registra la mediana, el p95 y el máximo de durations.
func logLatencies(t *testing.T, name string, durations []time.Duration) {
	t.Helper()
	if len(durations) == 0 {
		return
	}
	t.Logf("%s: %d requests, p50 %v, p95 %v, máx %v", name, len(durations),
		percentile(durations, 50).Round(time.Millisecond),
		percentile(durations, 95).Round(time.Millisecond),
		slices.Max(durations).Round(time.Millisecond))
}
