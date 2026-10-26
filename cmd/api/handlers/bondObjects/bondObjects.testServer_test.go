package bondObjects_handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/codecraftkit/nexus"
	bondObjects_handlers "github.com/futurapps/ms-blooming-go/cmd/api/handlers/bondObjects"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondHistory_ports "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/ports"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	bondObject_ports "github.com/futurapps/ms-blooming-go/internal/core/bondObject/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

const (
	agencyA  = "agencyAAAAAAAAAAA"
	userID   = "user1AAAAAAAAAAAA"
	objectID = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"
	typeID   = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"

	objectsPath = "/api/v1/bond-objects"
	objectPath  = objectsPath + "/" + objectID

	// validBody es un alta válida sin isPrivateBond ni inflectionPenaltyPercentage.
	validBody = `{"name":"Bono Plus 12M","description":"Renta fija 12 meses","agencyId":["` + agencyA + `"],` +
		`"key":"BP12","bondTypeId":"` + typeID + `","amount":1000,"maturityTimeInDays":365,` +
		`"maturityRewardsPercentage":15,"active":true,"available":true,` +
		`"earlyCashTerm":{"initialPenaltyPercentage":100,"inflectionPoint":90,"initialDayForWithdraw":30}}`
)

const (
	// legacyObjectID es un _id Meteor de 17 caracteres.
	legacyObjectID = "Bd9Xc4Vn7Mz2Lk5Jh"

	// previewBody son los términos de un producto sin guardar, con simulación.
	previewBody = `{"amount":1000.5,"maturityTimeInDays":365,"maturityRewardsPercentage":15,` +
		`"earlyCashTerm":{"initialPenaltyPercentage":100,"inflectionPoint":90,"inflectionPenaltyPercentage":75,` +
		`"initialDayForWithdraw":30},"simulation":true}`
)

var (
	admin    = &session_domain.UserSession{UserID: "admin-1", AgencyID: agencyA, AdminBlooming: true}
	investor = &session_domain.UserSession{UserID: userID, AgencyID: agencyA}
)

// fakeApp responde lo configurado y registra con qué la llamó el handler.
type fakeApp struct {
	bondObject_ports.BondObjectApplicationPorts

	object *bondObject_domain.BondObject
	detail *bondObject_domain.BondObjectDetail
	chart  bondObject_domain.BondObjectChart
	err    error

	calls      []string
	id         string
	params     bondObject_domain.CreateParams
	preview    bondObject_domain.ChartPreviewParams
	simulation bool
}

func (a *fakeApp) ChartPreview(_ context.Context, p bondObject_domain.ChartPreviewParams) (bondObject_domain.BondObjectChart, error) {
	a.calls = append(a.calls, "ChartPreview")
	a.preview = p
	return a.chart, a.err
}

func (a *fakeApp) ChartByID(_ context.Context, id string, simulation bool) (bondObject_domain.BondObjectChart, error) {
	a.calls = append(a.calls, "ChartByID")
	a.id = id
	a.simulation = simulation
	return a.chart, a.err
}

func (a *fakeApp) GetByID(_ context.Context, id string) (*bondObject_domain.BondObjectDetail, error) {
	a.calls = append(a.calls, "GetByID")
	a.id = id
	return a.detail, a.err
}

func (a *fakeApp) Create(_ context.Context, p bondObject_domain.CreateParams) (*bondObject_domain.BondObject, error) {
	a.calls = append(a.calls, "Create")
	a.params = p
	return a.object, a.err
}

func (a *fakeApp) Update(_ context.Context, id string, p bondObject_domain.UpdateParams) (*bondObject_domain.BondObject, error) {
	a.calls = append(a.calls, "Update")
	a.id = id
	a.params = p
	return a.object, a.err
}

func (a *fakeApp) Remove(_ context.Context, id string) error {
	a.calls = append(a.calls, "Remove")
	a.id = id
	return a.err
}

// fakeHistory responde el historial configurado y registra con qué lo pidió el handler.
type fakeHistory struct {
	bondHistory_ports.BondHistoryApplicationPorts

	entries []bondHistory_domain.HistoryEntry
	err     error

	calls     []string
	typeID    bondHistory_domain.HistoryType
	relatedID string
}

func (h *fakeHistory) ListByRelated(_ context.Context, typeID bondHistory_domain.HistoryType, relatedID string) ([]bondHistory_domain.HistoryEntry, error) {
	h.calls = append(h.calls, "ListByRelated")
	h.typeID, h.relatedID = typeID, relatedID
	return h.entries, h.err
}

// newTestAPI registra las rutas de /bond-objects sobre app, sin historial, y atiende cada request con la
// sesión dada.
func newTestAPI(app *fakeApp, session *session_domain.UserSession) http.Handler {
	return newTestAPIWithHistory(app, nil, session)
}

// newTestAPIWithHistory registra las rutas de /bond-objects sobre app y el historial, y atiende cada request
// con la sesión dada. history nil deja la ruta del historial sin aplicación.
func newTestAPIWithHistory(app *fakeApp, history *fakeHistory, session *session_domain.UserSession) http.Handler {
	server := &nexus.Server{
		ServerName: "blooming-test",
		Settings:   &nexus.Settings{PathPrefix: "/api/v1"},
	}
	var historyApp bondHistory_ports.BondHistoryApplicationPorts
	if history != nil {
		historyApp = history
	}
	bondObjects_handlers.NewBondObjectsHandler(server, app, historyApp)
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

// sample es un producto de la agencia A creado el 10 de enero en UTC, con los defaults del legado.
func sample() *bondObject_domain.BondObject {
	createdAt := time.Date(2026, 1, 10, 15, 0, 0, 0, time.UTC)
	return &bondObject_domain.BondObject{
		ID:                        objectID,
		Name:                      "Bono Plus 12M",
		Description:               "Renta fija 12 meses",
		AgencyID:                  shared_domain.AgencyIDs{agencyA},
		Key:                       "BP12",
		BondTypeID:                typeID,
		Amount:                    1000,
		AuthorizedUserIDs:         []string{},
		MaturityTimeInDays:        365,
		MaturityRewardsPercentage: 15,
		Active:                    true,
		Available:                 true,
		EarlyCashTerm: shared_domain.EarlyCashTerm{
			InitialPenaltyPercentage:    100,
			InflectionPoint:             90,
			InflectionPenaltyPercentage: 100,
			InitialDayForWithdraw:       30,
		},
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

// sampleChart es una gráfica con la guía de crecimiento y, con data, dos días de simulación.
func sampleChart(data bool) bondObject_domain.BondObjectChart {
	thickness := 1.5
	chart := bondObject_domain.BondObjectChart{Guides: []shared_domain.ChartGuide{{
		Date:          "2027-01-11",
		LineColor:     "#CC0000",
		LineThickness: &thickness,
		Label:         "Punto de Crecimiento",
	}}}
	if data {
		chart.Data = []shared_domain.ChartPoint{
			{Date: "2026-10-13", Inversion: 1000},
			{Date: "2026-10-14", Inversion: 1000},
		}
	}
	return chart
}
