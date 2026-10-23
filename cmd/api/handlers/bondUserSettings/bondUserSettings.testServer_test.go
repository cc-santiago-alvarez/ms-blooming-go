package bondUserSettings_handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/codecraftkit/nexus"
	bondUserSettings_handlers "github.com/futurapps/ms-blooming-go/cmd/api/handlers/bondUserSettings"
	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

const (
	agencyA     = "agencyAAAAAAAAAAA"
	userID      = "user1AAAAAAAAAAAA"
	user2ID     = "user2AAAAAAAAAAAA"
	settingID   = "6a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d"
	accountID   = "account1AAAAAAAAA"
	bondBoundID = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"

	settingsPath = "/api/v1/bond-user-settings"
	settingPath  = settingsPath + "/" + settingID
	minePath     = settingsPath + "/me"
)

var (
	admin    = &session_domain.UserSession{UserID: "adminAAAAAAAAAAAA", AgencyID: agencyA, AdminBlooming: true}
	investor = &session_domain.UserSession{UserID: userID, AgencyID: agencyA}
)

// fakeApp responde lo configurado y registra con qué la llamó el handler.
type fakeApp struct {
	bondUserSetting_ports.BondUserSettingApplicationPorts

	setting *bondUserSetting_domain.BondUserSetting
	detail  *bondUserSetting_domain.BondUserSettingDetail
	page    shared_domain.Paginated[*bondUserSetting_domain.BondUserSetting]
	err     error

	calls      []string
	id         string
	pagination shared_domain.Pagination
	account    bondUserSetting_domain.SetSelectedAccountParams
	bondBound  bondUserSetting_domain.SetSelectedBondBoundParams
	language   bondUserSetting_domain.SetLanguageParams
}

func (a *fakeApp) GetMine(_ context.Context) (*bondUserSetting_domain.BondUserSettingDetail, error) {
	a.calls = append(a.calls, "GetMine")
	return a.detail, a.err
}

func (a *fakeApp) SetSelectedAccount(_ context.Context, p bondUserSetting_domain.SetSelectedAccountParams) (*bondUserSetting_domain.BondUserSetting, error) {
	a.calls = append(a.calls, "SetSelectedAccount")
	a.account = p
	return a.setting, a.err
}

func (a *fakeApp) SetSelectedBondBound(_ context.Context, p bondUserSetting_domain.SetSelectedBondBoundParams) (*bondUserSetting_domain.BondUserSetting, error) {
	a.calls = append(a.calls, "SetSelectedBondBound")
	a.bondBound = p
	return a.setting, a.err
}

func (a *fakeApp) SetLanguage(_ context.Context, p bondUserSetting_domain.SetLanguageParams) (*bondUserSetting_domain.BondUserSetting, error) {
	a.calls = append(a.calls, "SetLanguage")
	a.language = p
	return a.setting, a.err
}

func (a *fakeApp) List(_ context.Context, p shared_domain.Pagination) (shared_domain.Paginated[*bondUserSetting_domain.BondUserSetting], error) {
	a.calls = append(a.calls, "List")
	a.pagination = p
	return a.page, a.err
}

func (a *fakeApp) GetByID(_ context.Context, id string) (*bondUserSetting_domain.BondUserSetting, error) {
	a.calls = append(a.calls, "GetByID")
	a.id = id
	return a.setting, a.err
}

func (a *fakeApp) Remove(_ context.Context, id string) error {
	a.calls = append(a.calls, "Remove")
	a.id = id
	return a.err
}

// newTestAPI registra las rutas de /bond-user-settings sobre app y atiende cada request con la sesión dada.
func newTestAPI(app *fakeApp, session *session_domain.UserSession) http.Handler {
	server := &nexus.Server{
		ServerName: "blooming-test",
		Settings:   &nexus.Settings{PathPrefix: "/api/v1"},
	}
	bondUserSettings_handlers.NewBondUserSettingsHandler(server, app)
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

// sample es el setting de userID en la agencia A, con la cuenta accountID y sin bono, creado el 10 de enero.
func sample() *bondUserSetting_domain.BondUserSetting {
	createdAt := time.Date(2026, 1, 10, 15, 0, 0, 0, time.UTC)
	account := accountID
	return &bondUserSetting_domain.BondUserSetting{
		ID:                settingID,
		UserID:            userID,
		AgencyID:          agencyA,
		Language:          bondUserSetting_domain.DefaultLanguage,
		SelectedAccountID: &account,
		CreatedAt:         createdAt,
		UpdatedAt:         createdAt,
	}
}
