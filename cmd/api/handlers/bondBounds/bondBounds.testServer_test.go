package bondBounds_handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/codecraftkit/nexus"
	bondBounds_handlers "github.com/futurapps/ms-blooming-go/cmd/api/handlers/bondBounds"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

const (
	agencyA   = "agencyAAAAAAAAAAA"
	userID    = "user1AAAAAAAAAAAA"
	boundID   = "5a6b7c8d-9e0f-4a1b-8c2d-3e4f5a6b7c8d"
	accountID = "9d8c7b6a-5f4e-4d3c-8b2a-1f0e9d8c7b6a"
	objectID  = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"

	// legacyBoundID es un _id Meteor de 17 caracteres.
	legacyBoundID = "Kp3Wq8Zr5Tn2Yx7Lm"

	boundsPath       = "/api/v1/bond-bounds"
	boundPath        = boundsPath + "/" + boundID
	purchasePath     = boundsPath + "/purchase"
	bulkPurchasePath = boundsPath + "/bulk-purchase"
	preSalePath      = boundPath + "/pre-sale"
	salePath         = boundPath + "/sale"

	// purchaseBody es la compra de un bono del producto con la cuenta.
	purchaseBody = `{"accountId":"` + accountID + `","bondObjectId":"` + objectID + `"}`
	// bulkBody es la compra de 3 bonos del producto con la cuenta.
	bulkBody = `{"accountId":"` + accountID + `","bondObjectId":"` + objectID + `","quantity":3}`
)

var investor = &session_domain.UserSession{UserID: userID, AgencyID: agencyA}

// investorIn es la sesión del inversionista en la zona utcDiff.
func investorIn(utcDiff int) *session_domain.UserSession {
	return &session_domain.UserSession{
		UserID:   userID,
		AgencyID: agencyA,
		UtcDiff:  session_domain.OptionalUtcDiff{Hours: utcDiff, Valid: true},
	}
}

// fakeApp responde lo configurado y registra con qué la llamó el handler.
type fakeApp struct {
	bondBound_ports.BondBoundApplicationPorts

	chart          bondBound_domain.BondBoundChart
	purchaseResult bondBound_domain.PurchaseResult
	bulkResult     bondBound_domain.BulkPurchaseResult
	projection     bondBound_domain.SaleProjection
	saleResult     bondBound_domain.SaleResult
	history        []bondHistory_domain.HistoryEntry
	err            error

	calls    []string
	id       string
	purchase bondBound_domain.PurchaseParams
	bulk     bondBound_domain.BulkPurchaseParams
}

func (a *fakeApp) Purchase(_ context.Context, p bondBound_domain.PurchaseParams) (bondBound_domain.PurchaseResult, error) {
	a.calls = append(a.calls, "Purchase")
	a.purchase = p
	return a.purchaseResult, a.err
}

func (a *fakeApp) BulkPurchase(_ context.Context, p bondBound_domain.BulkPurchaseParams) (bondBound_domain.BulkPurchaseResult, error) {
	a.calls = append(a.calls, "BulkPurchase")
	a.bulk = p
	return a.bulkResult, a.err
}

func (a *fakeApp) ChartGuides(_ context.Context, id string) (bondBound_domain.BondBoundChart, error) {
	a.calls = append(a.calls, "ChartGuides")
	a.id = id
	return a.chart, a.err
}

func (a *fakeApp) PreSale(_ context.Context, id string) (bondBound_domain.SaleProjection, error) {
	a.calls = append(a.calls, "PreSale")
	a.id = id
	return a.projection, a.err
}

func (a *fakeApp) Sale(_ context.Context, id string) (bondBound_domain.SaleResult, error) {
	a.calls = append(a.calls, "Sale")
	a.id = id
	return a.saleResult, a.err
}

func (a *fakeApp) HistoryOf(_ context.Context, id string) ([]bondHistory_domain.HistoryEntry, error) {
	a.calls = append(a.calls, "HistoryOf")
	a.id = id
	return a.history, a.err
}

// newTestAPI registra las rutas de /bond-bounds sobre app y atiende cada request con la sesión dada.
func newTestAPI(app *fakeApp, session *session_domain.UserSession) http.Handler {
	server := &nexus.Server{
		ServerName: "blooming-test",
		Settings:   &nexus.Settings{PathPrefix: "/api/v1"},
	}
	bondBounds_handlers.NewBondBoundsHandler(server, app)
	mux := server.PrepareEndpoints()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if session != nil {
			r = r.WithContext(session_domain.SetUserSession(r.Context(), session))
		}
		mux.ServeHTTP(w, r)
	})
}

// serve manda un request al handler y devuelve la respuesta. Un body vacío es un request sin body.
func serve(h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	if body != "" {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// assertError verifica el status y el código de error de una respuesta de nexus.
func assertError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status %d, se esperaba %d (body %s)", rec.Code, status, rec.Body)
	}
	var body nexus.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("el body no es un error de nexus: %v (body %s)", err, rec.Body)
	}
	if body.Message != code {
		t.Fatalf("código %q, se esperaba %q", body.Message, code)
	}
}

// assertWithdrawDateDetail verifica un 422 ERR_INVALID_WITHDRAW_DATE con el mensaje de la fecha de hoy, en la
// zona utcDiff, en errors.detail.
func assertWithdrawDateDetail(t *testing.T, rec *httptest.ResponseRecorder, utcDiff int) {
	t.Helper()
	assertError(t, rec, http.StatusUnprocessableEntity, "ERR_INVALID_WITHDRAW_DATE")
	var body nexus.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("el body no es un error de nexus: %v (body %s)", err, rec.Body)
	}
	today := time.Now().In(time.FixedZone("", utcDiff*3600)).Format("02/01/2006")
	if want := "El bono no puede ser vendido en la fecha actual (" + today + ")"; body.Errors["detail"] != want {
		t.Errorf("errors.detail %q, se esperaba %q", body.Errors["detail"], want)
	}
}

// decodeJSON verifica el status y decodifica el body en v.
func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, status int, v any) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status %d, se esperaba %d (body %s)", rec.Code, status, rec.Body)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("el body no es JSON: %v (body %s)", err, rec.Body)
	}
}

// sampleChart son las guías de un bono comprado: el punto de crecimiento y la línea de retiro, sin toDate
// ni fillColor.
func sampleChart() bondBound_domain.BondBoundChart {
	thickness := 1.5
	return bondBound_domain.BondBoundChart{Guides: []shared_domain.ChartGuide{
		{
			Date:          "2026-01-31",
			LineColor:     "#CC0000",
			FillColor:     "#CC0000",
			LabelRotation: 90,
			LineThickness: &thickness,
			Label:         "Punto de Crecimiento",
		},
		{
			Date:          "2026-01-11",
			LineColor:     "#4ec74c",
			LabelRotation: 90,
			LineThickness: &thickness,
			Label:         "Zona de Retiro",
		},
	}}
}
