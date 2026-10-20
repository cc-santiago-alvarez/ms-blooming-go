package bondTypes_handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/codecraftkit/nexus"
	bondTypes_handlers "github.com/futurapps/ms-blooming-go/cmd/api/handlers/bondTypes"
	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	bondType_ports "github.com/futurapps/ms-blooming-go/internal/core/bondType/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
)

const (
	agencyA = "agencyAAAAAAAAAAA"
	agencyB = "agencyBBBBBBBBBBB"
	typeID  = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"

	typesPath = "/api/v1/bond-types"
	typePath  = typesPath + "/" + typeID

	// validBody es un alta o edición válida.
	validBody = `{"name":"Bono Plus","description":"Renta fija 12 meses","active":true,"agencyId":["` + agencyA + `","` + agencyB + `"]}`
)

var (
	admin    = &session_domain.UserSession{UserID: "admin-1", AgencyID: agencyA, AdminBlooming: true}
	investor = &session_domain.UserSession{UserID: "user-1", AgencyID: agencyA}
)

// fakeApp responde lo configurado y registra con qué la llamó el handler.
type fakeApp struct {
	bondType_ports.BondTypeApplicationPorts

	types    []*bondType_domain.BondType
	active   []bondType_domain.ActiveBondType
	bondType *bondType_domain.BondType
	err      error

	calls     []string
	agencyIDs []string
	id        string
	params    bondType_domain.CreateParams
}

func (a *fakeApp) List(_ context.Context, agencyIDs []string) ([]*bondType_domain.BondType, error) {
	a.calls = append(a.calls, "List")
	a.agencyIDs = agencyIDs
	return a.types, a.err
}

func (a *fakeApp) ListActive(context.Context) ([]bondType_domain.ActiveBondType, error) {
	a.calls = append(a.calls, "ListActive")
	return a.active, a.err
}

func (a *fakeApp) GetByID(_ context.Context, id string) (*bondType_domain.BondType, error) {
	a.calls = append(a.calls, "GetByID")
	a.id = id
	return a.bondType, a.err
}

func (a *fakeApp) Create(_ context.Context, p bondType_domain.CreateParams) (*bondType_domain.BondType, error) {
	a.calls = append(a.calls, "Create")
	a.params = p
	return a.bondType, a.err
}

func (a *fakeApp) Update(_ context.Context, id string, p bondType_domain.UpdateParams) (*bondType_domain.BondType, error) {
	a.calls = append(a.calls, "Update")
	a.id = id
	a.params = p
	return a.bondType, a.err
}

func (a *fakeApp) Remove(_ context.Context, id string) error {
	a.calls = append(a.calls, "Remove")
	a.id = id
	return a.err
}

// newTestAPI registra las rutas de /bond-types sobre app y atiende cada request con la sesión dada.
func newTestAPI(app *fakeApp, session *session_domain.UserSession) http.Handler {
	server := &nexus.Server{
		ServerName: "blooming-test",
		Settings:   &nexus.Settings{PathPrefix: "/api/v1"},
	}
	bondTypes_handlers.NewBondTypesHandler(server, app)
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

// sample es un tipo de la agencia A creado el 10 de enero en UTC.
func sample() *bondType_domain.BondType {
	createdAt := time.Date(2026, 1, 10, 15, 0, 0, 0, time.UTC)
	return &bondType_domain.BondType{
		ID:          typeID,
		AgencyID:    []string{agencyA},
		Name:        "Bono Plus",
		Description: "Renta fija 12 meses",
		Active:      true,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
}
