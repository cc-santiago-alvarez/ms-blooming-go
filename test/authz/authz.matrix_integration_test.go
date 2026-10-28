//go:build integration

package authz_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
)

// prefix es el prefijo de las rutas de negocio.
const prefix = "/api/v1"

// sight indica si la celda revisa que el body contenga el recurso de A.
type sight int

const (
	// anySight no revisa el body.
	anySight sight = iota
	// shown exige que el body contenga el marcador de la fila.
	shown
	// hidden exige que no lo contenga.
	hidden
)

// cell es lo que responde una ruta a un rol.
type cell struct {
	status int
	sight  sight
}

// Celdas de la matriz.
var (
	ok        = cell{status: http.StatusOK}
	created   = cell{status: http.StatusCreated}
	forbidden = cell{status: http.StatusForbidden}
	notFound  = cell{status: http.StatusNotFound}
	// listed es un 200 cuyo body contiene el recurso de A, y unlisted, uno que no lo contiene.
	listed   = cell{status: http.StatusOK, sight: shown}
	unlisted = cell{status: http.StatusOK, sight: hidden}
)

// cells son las celdas de una fila, en el orden de roles: usuario, dueño, admin de A, admin de otra agencia
// y admin de la principal.
type cells [5]cell

// adminOnly es una ruta envuelta en RequireBloomingAdmin: el usuario y el dueño reciben 403.
func adminOnly(adminA, adminB, adminMain cell) cells {
	return cells{forbidden, forbidden, adminA, adminB, adminMain}
}

// allOK es una ruta que responde 200 a cualquier sesión.
var allOK = cells{ok, ok, ok, ok, ok}

// ownerOrAdmin es un bono de A por id: lo ven el dueño y los admins de su alcance; otro usuario recibe 403 y
// un admin de otra agencia, 404.
var ownerOrAdmin = cells{forbidden, ok, ok, notFound, ok}

// row es una fila de la matriz: una ruta registrada y un request concreto sobre los fixtures.
type row struct {
	// route es la ruta como se registra, "MÉTODO /patrón", sin el prefijo.
	route string
	// path es el path concreto del request, sin el prefijo y con su query.
	path string
	body any
	// marker es lo que contiene el body cuando el rol ve el recurso de A.
	marker string
	want   cells
}

// method devuelve el método de la ruta.
func (r row) method() string {
	method, _, _ := strings.Cut(r.route, " ")
	return method
}

// registered devuelve la ruta con el prefijo, como la lista /_routes.
func (r row) registered() string {
	method, pattern, _ := strings.Cut(r.route, " ")
	return method + " " + prefix + pattern
}

// name es el nombre del subtest: la ruta y, si tiene, la query del request.
func (r row) name() string {
	if _, query, ok := strings.Cut(r.path, "?"); ok {
		return r.route + "?" + query
	}
	return r.route
}

// bondTypeBody es un alta o una edición válida de un tipo de A.
func bondTypeBody() map[string]any {
	return map[string]any{"name": "Renta variable", "description": "Bonos de renta variable", "active": true, "agencyId": []string{agencyA}}
}

// bondObjectBody es un alta o una edición válida de un producto de A con el tipo de A.
func bondObjectBody() map[string]any {
	return map[string]any{
		"name": "Bono Max 6M", "description": "Renta fija 180 días", "agencyId": []string{agencyA}, "key": "BM06",
		"bondTypeId": typeID, "amount": 500, "isPrivateBond": false, "authorizedUserIds": []string{},
		"maturityTimeInDays": 180, "maturityRewardsPercentage": 8, "active": true, "available": true,
		"earlyCashTerm": map[string]any{
			"initialPenaltyPercentage": 100, "inflectionPoint": 30, "inflectionPenaltyPercentage": 50,
			"blockWithdrawals": false, "initialDayForWithdraw": 10,
		},
	}
}

// previewBody son los términos válidos de una vista previa sin simulación.
var previewBody = map[string]any{
	"amount": 1000, "maturityTimeInDays": 100, "maturityRewardsPercentage": 10, "simulation": false,
	"earlyCashTerm": map[string]any{
		"initialPenaltyPercentage": 100, "inflectionPoint": 0, "inflectionPenaltyPercentage": 50,
		"blockWithdrawals": false, "initialDayForWithdraw": 0,
	},
}

// purchaseBody es una compra del producto de A con la cuenta del dueño.
var purchaseBody = map[string]any{"accountId": accountID, "bondObjectId": objectID}

// matrix son las rutas de negocio. Todo lo que se consulta es de la agencia A: el tipo, el producto, el bono,
// la cuenta y el setting del dueño.
var matrix = []row{
	// tipos de bono
	{route: "GET /bond-types", path: "/bond-types", marker: typeID, want: adminOnly(listed, unlisted, listed)},
	{route: "GET /bond-types/active", path: "/bond-types/active", marker: typeID, want: cells{listed, listed, listed, unlisted, unlisted}},
	{route: "GET /bond-types/{id}", path: "/bond-types/" + typeID, want: adminOnly(ok, notFound, ok)},
	{route: "POST /bond-types", path: "/bond-types", body: bondTypeBody(), want: adminOnly(created, forbidden, created)},
	{route: "PUT /bond-types/{id}", path: "/bond-types/" + typeID, body: bondTypeBody(), want: adminOnly(ok, notFound, ok)},
	{route: "DELETE /bond-types/{id}", path: "/bond-types/" + emptyTypeID, want: adminOnly(ok, notFound, ok)},
	{route: "GET /bond-types/{bondTypeId}/bond-objects", path: "/bond-types/" + typeID + "/bond-objects", marker: objectID, want: cells{listed, listed, listed, unlisted, unlisted}},

	// productos
	{route: "GET /bond-objects", path: "/bond-objects", marker: objectID, want: adminOnly(listed, unlisted, listed)},
	{route: "GET /bond-objects/grouped-by-type", path: "/bond-objects/grouped-by-type", marker: objectID, want: adminOnly(listed, unlisted, listed)},
	{route: "GET /bond-objects/most-purchased", path: "/bond-objects/most-purchased", marker: objectID, want: cells{listed, listed, listed, unlisted, unlisted}},
	{route: "GET /bond-objects/latest", path: "/bond-objects/latest", marker: objectID, want: cells{listed, listed, listed, unlisted, unlisted}},
	{route: "GET /bond-objects/{id}", path: "/bond-objects/" + objectID, want: adminOnly(ok, notFound, ok)},
	{route: "GET /bond-objects/{id}/investors", path: "/bond-objects/" + objectID + "/investors", marker: ownerID, want: adminOnly(listed, unlisted, listed)},
	{route: "GET /bond-objects/{id}/chart", path: "/bond-objects/" + objectID + "/chart", want: cells{ok, ok, ok, notFound, ok}},
	{route: "GET /bond-objects/{id}/history", path: "/bond-objects/" + objectID + "/history", marker: objectHistoryID, want: adminOnly(listed, unlisted, listed)},
	{route: "GET /bond-objects/{id}/bond-bounds", path: "/bond-objects/" + objectID + "/bond-bounds", marker: boundID, want: adminOnly(listed, unlisted, listed)},
	{route: "POST /bond-objects", path: "/bond-objects", body: bondObjectBody(), want: adminOnly(created, forbidden, created)},
	{route: "POST /bond-objects/chart/preview", path: "/bond-objects/chart/preview", body: previewBody, want: adminOnly(ok, ok, ok)},
	{route: "PUT /bond-objects/{id}", path: "/bond-objects/" + objectID, body: bondObjectBody(), want: adminOnly(ok, notFound, ok)},
	{route: "PATCH /bond-objects/keys", path: "/bond-objects/keys", body: map[string]any{"bonds": []string{objectID}, "key": "BP99"}, marker: `"matched":1`, want: adminOnly(listed, unlisted, listed)},
	{route: "DELETE /bond-objects/{id}", path: "/bond-objects/" + objectID, want: adminOnly(ok, notFound, ok)},

	// preferencias
	{route: "GET /bond-user-settings/me", path: "/bond-user-settings/me", want: allOK},
	{route: "PATCH /bond-user-settings/me/selected-account", path: "/bond-user-settings/me/selected-account", body: map[string]any{"selectedAccountId": accountID}, want: cells{notFound, ok, notFound, notFound, notFound}},
	{route: "PATCH /bond-user-settings/me/selected-bond-bound", path: "/bond-user-settings/me/selected-bond-bound", body: map[string]any{"selectedBondBoundId": boundID}, want: allOK},
	{route: "PATCH /bond-user-settings/me/language", path: "/bond-user-settings/me/language", body: map[string]any{"language": "en"}, want: allOK},
	{route: "GET /bond-user-settings", path: "/bond-user-settings", marker: settingID, want: adminOnly(listed, unlisted, listed)},
	{route: "GET /bond-user-settings/{id}", path: "/bond-user-settings/" + settingID, want: adminOnly(ok, notFound, ok)},
	{route: "DELETE /bond-user-settings/{id}", path: "/bond-user-settings/" + settingID, want: adminOnly(ok, notFound, ok)},

	// bonos comprados
	{route: "GET /bond-bounds", path: "/bond-bounds", marker: boundID, want: adminOnly(listed, unlisted, listed)},
	{route: "GET /bond-bounds/statuses", path: "/bond-bounds/statuses", want: allOK},
	{route: "GET /bond-bounds/me", path: "/bond-bounds/me", marker: boundID, want: cells{unlisted, listed, unlisted, unlisted, unlisted}},
	{route: "GET /bond-bounds/me/recent", path: "/bond-bounds/me/recent", marker: boundID, want: cells{unlisted, listed, unlisted, unlisted, unlisted}},
	{route: "GET /bond-bounds/timeline", path: "/bond-bounds/timeline", marker: boundID, want: cells{unlisted, listed, unlisted, unlisted, unlisted}},
	{route: "GET /bond-bounds/timeline", path: "/bond-bounds/timeline?userId=" + ownerID, marker: boundID, want: cells{forbidden, listed, listed, unlisted, listed}},
	{route: "GET /bond-bounds/reports/purchased", path: "/bond-bounds/reports/purchased", want: adminOnly(ok, ok, ok)},
	{route: "GET /bond-bounds/reports/purchased-summary", path: "/bond-bounds/reports/purchased-summary", want: adminOnly(ok, ok, ok)},
	{route: "GET /bond-bounds/kpis/count-by-status", path: "/bond-bounds/kpis/count-by-status", want: allOK},
	{route: "GET /bond-bounds/{id}", path: "/bond-bounds/" + boundID, want: ownerOrAdmin},
	{route: "GET /bond-bounds/{id}/details", path: "/bond-bounds/" + boundID + "/details", want: ownerOrAdmin},
	{route: "GET /bond-bounds/{id}/chart-guides", path: "/bond-bounds/" + boundID + "/chart-guides", want: ownerOrAdmin},
	{route: "GET /bond-bounds/{id}/chart", path: "/bond-bounds/" + boundID + "/chart", want: ownerOrAdmin},
	{route: "GET /bond-bounds/{id}/pre-sale", path: "/bond-bounds/" + boundID + "/pre-sale", want: ownerOrAdmin},
	{route: "GET /bond-bounds/{id}/history", path: "/bond-bounds/" + boundID + "/history", want: ownerOrAdmin},
	{route: "DELETE /bond-bounds/{id}", path: "/bond-bounds/" + boundID, want: adminOnly(ok, notFound, ok)},
	{route: "POST /bond-bounds/{id}/sale", path: "/bond-bounds/" + boundID + "/sale", want: cells{forbidden, ok, forbidden, forbidden, forbidden}},
	{route: "POST /bond-bounds/purchase", path: "/bond-bounds/purchase", body: purchaseBody, want: cells{notFound, created, notFound, notFound, notFound}},
	{route: "POST /bond-bounds/bulk-purchase", path: "/bond-bounds/bulk-purchase", body: map[string]any{"accountId": accountID, "bondObjectId": objectID, "quantity": 2}, want: cells{notFound, created, notFound, notFound, notFound}},
	{route: "GET /users/{userId}/bond-bounds", path: "/users/" + ownerID + "/bond-bounds?bondObjectId=" + objectID, marker: boundID, want: adminOnly(listed, unlisted, listed)},
}

// internalRoute es una ruta interna: la llama otro servicio con x-secret y sin sesión.
type internalRoute struct {
	route string
	path  string
	body  any
}

// internalRoutes son las rutas internas.
var internalRoutes = []internalRoute{
	{route: "PUT /internal/bond-user-settings/{userId}/selected-account", path: "/internal/bond-user-settings/" + ownerID + "/selected-account", body: map[string]any{"selectedAccountId": accountID}},
	{route: "POST /internal/jobs/bond-maturity/run", path: "/internal/jobs/bond-maturity/run"},
}

// Cada ruta registrada en el servicio, salvo /_health y /_routes, tiene su fila en la matriz o en las rutas
// internas, y ninguna fila nombra una ruta que no esté registrada. Las rutas salen de /_routes, que con
// API_DEBUG lista las registradas sin pedir x-secret. CA-00.43
func TestMatrixCoversRegisteredRoutes(t *testing.T) {
	if skipReason != "" {
		t.Skip("sin Mongo para integración: " + skipReason)
	}
	res := call(t, http.MethodGet, "/_routes", false, "", nil)
	if res.status != http.StatusOK {
		t.Fatalf("GET /_routes = %d, se esperaba 200; body: %s", res.status, res.body)
	}
	var listed map[string]string
	if err := json.Unmarshal(res.body, &listed); err != nil {
		t.Fatalf("GET /_routes: body %s inválido: %v", res.body, err)
	}

	var registered []string
	for route := range listed {
		if route == "GET /_health" || route == "GET /_routes" || route == "GET /_routes/raw" {
			continue
		}
		registered = append(registered, route)
	}
	var covered []string
	for _, r := range matrix {
		covered = append(covered, r.registered())
	}
	for _, r := range internalRoutes {
		covered = append(covered, row{route: r.route}.registered())
	}

	var missing, unknown []string
	for _, route := range registered {
		if !slices.Contains(covered, route) {
			missing = append(missing, route)
		}
	}
	for _, route := range covered {
		if !slices.Contains(registered, route) {
			unknown = append(unknown, route)
		}
	}
	slices.Sort(missing)
	slices.Sort(unknown)
	if len(missing) > 0 {
		t.Errorf("rutas registradas sin fila en la matriz: %v", missing)
	}
	if len(unknown) > 0 {
		t.Errorf("filas de la matriz que no son rutas registradas: %v", slices.Compact(unknown))
	}
}

// Cada ruta de negocio responde a cada rol lo que piden las specs. Sin x-secret es 401
// UNAUTHORIZED_INVALID_SECRET y sin x-token, 401 TOKEN_IS_REQUIRED. Las rutas de admin responden 403 a quien
// no lo es. Lo que está fuera del alcance de agencia de la sesión no existe: un admin de otra agencia que no
// es la principal recibe 404 en el detalle, la edición y la baja, no ve el recurso en los listados y no puede
// crear para otra agencia, mientras que el admin de la principal ve todo. Los bonos por id son del dueño o de
// un admin de su alcance, la venta solo del dueño, y la compra y la cuenta destino solo con una cuenta propia.
// CA-00.2, CA-00.3, CA-00.7, CA-01.2, CA-01.10, CA-01.11, CA-02.18, CA-02.19, CA-02.20, CA-03.6, CA-03.8,
// CA-03.17, CA-03.18, CA-04.3, CA-05.6, CA-05.10, CA-05.18, CA-06.7, CA-06.10, CA-06.12, CA-06.13, CA-06.15,
// CA-06.16, CA-07.6, CA-07.7, CA-08.3, CA-08.4, CA-08.10, CA-09.5, CA-09.8, CA-09.9
func TestAuthorizationMatrix(t *testing.T) {
	for _, r := range matrix {
		t.Run(r.name(), func(t *testing.T) {
			t.Run("sin secret", func(t *testing.T) {
				prepare(t, http.MethodGet)
				res := call(t, r.method(), prefix+r.path, false, tokens[roleAdminMain], r.body)
				expectError(t, r, res, http.StatusUnauthorized, "UNAUTHORIZED_INVALID_SECRET")
			})
			t.Run("sin token", func(t *testing.T) {
				prepare(t, http.MethodGet)
				res := call(t, r.method(), prefix+r.path, true, "", r.body)
				expectError(t, r, res, http.StatusUnauthorized, "TOKEN_IS_REQUIRED")
			})
			for i, rl := range roles {
				want := r.want[i]
				t.Run(string(rl), func(t *testing.T) {
					prepare(t, r.method())

					res := call(t, r.method(), prefix+r.path, true, tokens[rl], r.body)

					if res.status != want.status {
						t.Fatalf("%s %s = %d, se esperaba %d; body: %s", r.method(), r.path, res.status, want.status, res.body)
					}
					switch seen := bytes.Contains(res.body, []byte(r.marker)); {
					case want.sight == shown && !seen:
						t.Errorf("%s %s no trae %s; body: %s", r.method(), r.path, r.marker, res.body)
					case want.sight == hidden && seen:
						t.Errorf("%s %s trae %s, que es de otra agencia o de otro usuario; body: %s", r.method(), r.path, r.marker, res.body)
					}
				})
			}
		})
	}
}

// Las rutas internas no piden sesión: con x-secret y sin x-token responden 200, y sin x-secret, 401
// UNAUTHORIZED_INVALID_SECRET. CA-00.2, CA-09.10, CA-10.8
func TestInternalRoutes(t *testing.T) {
	for _, r := range internalRoutes {
		rw := row{route: r.route, path: r.path, body: r.body}
		t.Run(r.route, func(t *testing.T) {
			t.Run("sin secret", func(t *testing.T) {
				prepare(t, http.MethodGet)
				res := call(t, rw.method(), prefix+r.path, false, "", r.body)
				expectError(t, rw, res, http.StatusUnauthorized, "UNAUTHORIZED_INVALID_SECRET")
			})
			t.Run("con secret y sin token", func(t *testing.T) {
				prepare(t, rw.method())
				res := call(t, rw.method(), prefix+r.path, true, "", r.body)
				if res.status != http.StatusOK {
					t.Fatalf("%s %s = %d, se esperaba 200; body: %s", rw.method(), r.path, res.status, res.body)
				}
			})
		})
	}
}

// expectError exige status y el código de error en message.
func expectError(t *testing.T, r row, res response, status int, code string) {
	t.Helper()
	var body struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(res.body, &body)
	if res.status != status || body.Message != code {
		t.Fatalf("%s %s = %d %q, se esperaba %d %q; body: %s", r.method(), r.path, res.status, body.Message, status, code, res.body)
	}
}
