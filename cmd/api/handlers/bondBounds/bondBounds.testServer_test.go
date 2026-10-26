package bondBounds_handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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

	reportPurchasedPath = boundsPath + "/reports/purchased"
	reportSummaryPath   = boundsPath + "/reports/purchased-summary"
	countByStatusPath   = boundsPath + "/kpis/count-by-status"

	// xlsxContentType es el Content-Type de los reportes.
	xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

	// purchaseBody es la compra de un bono del producto con la cuenta.
	purchaseBody = `{"accountId":"` + accountID + `","bondObjectId":"` + objectID + `"}`
	// bulkBody es la compra de 3 bonos del producto con la cuenta.
	bulkBody = `{"accountId":"` + accountID + `","bondObjectId":"` + objectID + `","quantity":3}`
)

var (
	investor = &session_domain.UserSession{UserID: userID, AgencyID: agencyA}
	admin    = &session_domain.UserSession{UserID: "adminAAAAAAAAAAAA", AgencyID: agencyA, AdminBlooming: true}
)

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
	adminPage      shared_domain.Paginated[bondBound_ports.BondBoundAdminItem]
	minePage       shared_domain.Paginated[bondBound_ports.BondBoundMineItem]
	recent         []bondBound_ports.BondBoundMineItem
	bounds         []*bondBound_domain.BondBound
	buyers         []bondBound_ports.BondBoundWithProfile
	bound          *bondBound_domain.BondBound
	detail         *bondBound_ports.BondBoundDetail
	timeline       []bondBound_ports.TimelineItem
	reportFile     bondBound_ports.ReportFile
	statusCount    bondBound_ports.StatusCount
	err            error

	calls         []string
	id            string
	purchase      bondBound_domain.PurchaseParams
	bulk          bondBound_domain.BulkPurchaseParams
	userID        string
	bondObjectID  string
	listFilters   bondBound_domain.ListFilters
	mineFilters   bondBound_domain.MineFilters
	reportFilters bondBound_domain.ReportFilters
	kpiFilters    bondBound_domain.KpiFilters
	page          shared_domain.Pagination
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

func (a *fakeApp) List(_ context.Context, f bondBound_domain.ListFilters, p shared_domain.Pagination) (shared_domain.Paginated[bondBound_ports.BondBoundAdminItem], error) {
	a.calls = append(a.calls, "List")
	a.listFilters, a.page = f, p
	return a.adminPage, a.err
}

func (a *fakeApp) ListByUserAndObject(_ context.Context, userID, bondObjectID string) ([]*bondBound_domain.BondBound, error) {
	a.calls = append(a.calls, "ListByUserAndObject")
	a.userID, a.bondObjectID = userID, bondObjectID
	return a.bounds, a.err
}

func (a *fakeApp) ListByObject(_ context.Context, bondObjectID string) ([]bondBound_ports.BondBoundWithProfile, error) {
	a.calls = append(a.calls, "ListByObject")
	a.bondObjectID = bondObjectID
	return a.buyers, a.err
}

func (a *fakeApp) Remove(_ context.Context, id string) error {
	a.calls = append(a.calls, "Remove")
	a.id = id
	return a.err
}

// Statuses devuelve los estados del dominio.
func (a *fakeApp) Statuses() []bondBound_domain.StatusRef {
	a.calls = append(a.calls, "Statuses")
	return bondBound_domain.StatusRefs()
}

func (a *fakeApp) ListMine(_ context.Context, f bondBound_domain.MineFilters, p shared_domain.Pagination) (shared_domain.Paginated[bondBound_ports.BondBoundMineItem], error) {
	a.calls = append(a.calls, "ListMine")
	a.mineFilters, a.page = f, p
	return a.minePage, a.err
}

func (a *fakeApp) ListRecent(_ context.Context) ([]bondBound_ports.BondBoundMineItem, error) {
	a.calls = append(a.calls, "ListRecent")
	return a.recent, a.err
}

func (a *fakeApp) GetByID(_ context.Context, id string) (*bondBound_domain.BondBound, error) {
	a.calls = append(a.calls, "GetByID")
	a.id = id
	return a.bound, a.err
}

func (a *fakeApp) GetDetails(_ context.Context, id string) (*bondBound_ports.BondBoundDetail, error) {
	a.calls = append(a.calls, "GetDetails")
	a.id = id
	return a.detail, a.err
}

func (a *fakeApp) Timeline(_ context.Context, userID string) ([]bondBound_ports.TimelineItem, error) {
	a.calls = append(a.calls, "Timeline")
	a.userID = userID
	return a.timeline, a.err
}

func (a *fakeApp) ReportPurchased(_ context.Context, f bondBound_domain.ReportFilters) (bondBound_ports.ReportFile, error) {
	a.calls = append(a.calls, "ReportPurchased")
	a.reportFilters = f
	return a.reportFile, a.err
}

func (a *fakeApp) ReportSummary(_ context.Context, f bondBound_domain.ReportFilters) (bondBound_ports.ReportFile, error) {
	a.calls = append(a.calls, "ReportSummary")
	a.reportFilters = f
	return a.reportFile, a.err
}

func (a *fakeApp) CountByStatus(_ context.Context, f bondBound_domain.KpiFilters) (bondBound_ports.StatusCount, error) {
	a.calls = append(a.calls, "CountByStatus")
	a.kpiFilters = f
	return a.statusCount, a.err
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

// reportFile es un reporte de comprados como lo devuelve la aplicación.
func reportFile(name string) bondBound_ports.ReportFile {
	return bondBound_ports.ReportFile{Name: name, Content: []byte{80, 75, 3, 4}}
}

// assertReportResponse verifica un 200 con file como descarga de Excel: su tipo, su nombre, su largo y sus
// bytes.
func assertReportResponse(t *testing.T, rec *httptest.ResponseRecorder, file bondBound_ports.ReportFile) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, se esperaba 200 (body %s)", rec.Code, rec.Body)
	}
	header := rec.Header()
	if got := header.Get("Content-Type"); got != xlsxContentType {
		t.Errorf("Content-Type %q, se esperaba %q", got, xlsxContentType)
	}
	if got, want := header.Get("Content-Disposition"), `attachment; filename="`+file.Name+`"`; got != want {
		t.Errorf("Content-Disposition %q, se esperaba %q", got, want)
	}
	if got := header.Get("Content-Length"); got != strconv.Itoa(len(file.Content)) {
		t.Errorf("Content-Length %q, se esperaba %d", got, len(file.Content))
	}
	if !bytes.Equal(rec.Body.Bytes(), file.Content) {
		t.Errorf("body %v, se esperaba %v", rec.Body.Bytes(), file.Content)
	}
}

// assertExportFailed verifica un 502 ERR_EXPORT_FAILED en JSON, sin más detalle que el código y sin los
// headers de una descarga.
func assertExportFailed(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	assertError(t, rec, http.StatusBadGateway, "ERR_EXPORT_FAILED")
	var body nexus.ErrorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Errors) != 1 || body.Errors["error"] != "ERR_EXPORT_FAILED" {
		t.Errorf("errors %v, se esperaba solo {error: ERR_EXPORT_FAILED}", body.Errors)
	}
	if header := rec.Header(); header.Get("Content-Disposition") != "" || header.Get("Content-Type") == xlsxContentType {
		t.Errorf("headers %v, se esperaba una respuesta JSON sin descarga", header)
	}
}

// assertForbiddenAdmin verifica un 403 ERR_FORBIDDEN de una ruta de admin que no llegó a la aplicación.
func assertForbiddenAdmin(t *testing.T, rec *httptest.ResponseRecorder, app *fakeApp) {
	t.Helper()
	assertError(t, rec, http.StatusForbidden, "ERR_FORBIDDEN")
	if len(app.calls) != 0 {
		t.Fatalf("la aplicación recibió %v sin sesión admin", app.calls)
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
