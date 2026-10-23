//go:build integration

package bondUserSetting_mongocc_repository_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/codecraftkit/nexus"
	bondUserSettings_handlers "github.com/futurapps/ms-blooming-go/cmd/api/handlers/bondUserSettings"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	middlewares_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/middlewares"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// testSecret es el API_SECRET del servidor de estos tests.
const testSecret = "s3cr3t-blooming-it"

// countingVerifier acepta cualquier token y cuenta las llamadas.
type countingVerifier struct {
	shared_ports.SessionVerifier

	calls atomic.Int32
}

func (v *countingVerifier) Verify(context.Context, string) (*session_domain.UserSession, error) {
	v.calls.Add(1)
	return &session_domain.UserSession{UserID: userID, AgencyID: agencyA}, nil
}

// newInternalAPI arma las rutas de bondUserSetting sobre los adapters reales, con la validación de
// x-secret y de sesión en el mismo orden que el composition root.
func newInternalAPI(verifier shared_ports.SessionVerifier) http.Handler {
	server := &nexus.Server{
		ServerName: "blooming-it",
		Secret:     testSecret,
		Settings:   &nexus.Settings{PathPrefix: "/api/v1"},
	}
	bondUserSettings_handlers.NewBondUserSettingsHandler(server, newApp())
	server.Use(middlewares_adapters.ValidateApiSecret)
	server.Use(middlewares_adapters.VerifySession(verifier))
	return server.ApplyMiddlewares(server.PrepareEndpoints())
}

// syncAccount manda el PUT interno de Meteor para userID con accountID y los headers dados.
func syncAccount(h http.Handler, userID, accountID string, headers map[string]string) *httptest.ResponseRecorder {
	body := bytes.NewBufferString(`{"selectedAccountId":"` + accountID + `"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/internal/bond-user-settings/"+userID+"/selected-account", body)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// assertUpdated verifica que la respuesta sea 200 {updated: want}.
func assertUpdated(t *testing.T, rec *httptest.ResponseRecorder, want bool) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, se esperaba 200 (body %s)", rec.Code, rec.Body)
	}
	var body map[string]bool
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("el body no es JSON: %v (body %s)", err, rec.Body)
	}
	if got, ok := body["updated"]; !ok || got != want || len(body) != 1 {
		t.Fatalf("body %s, se esperaba {\"updated\":%t}", rec.Body, want)
	}
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

// insertSyncFixtures guarda el setting de userID con la cuenta accountOld, sus dos cuentas y la de
// user2ID, que no tiene setting.
func insertSyncFixtures(t *testing.T) {
	t.Helper()
	insert(t, settingCollection, settingDoc(settingID, userID, agencyA, bson.M{"selectedAccountId": accountOld}))
	insert(t, accountCollection,
		accountDoc(accountOld, userID, agencyA, time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC), nil),
		accountDoc(accountNew, userID, agencyA, time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC), nil),
		accountDoc(accountOfUser2, user2ID, agencyA, time.Date(2025, 2, 5, 0, 0, 0, 0, time.UTC), nil),
	)
}

// El PUT interno con x-secret válido y sin x-token actualiza el setting del usuario y responde
// {updated: true}, sin validar sesión. Si el usuario no tiene setting responde {updated: false} y no crea
// nada. Sin x-secret responde 401 y con la cuenta de otro usuario 404 ERR_ACCOUNT_NOT_FOUND, y en los dos
// casos el setting no cambia. CA-09.10
func TestInternalSelectedAccount(t *testing.T) {
	secret := map[string]string{"x-secret": testSecret}

	t.Run("con setting", func(t *testing.T) {
		setup(t)
		insertSyncFixtures(t)
		verifier := &countingVerifier{}

		rec := syncAccount(newInternalAPI(verifier), userID, accountNew, secret)

		assertUpdated(t, rec, true)
		raw := rawDoc(t, settingCollection, settingID)
		if v, ok := raw.Lookup("selectedAccountId").StringValueOK(); !ok || v != accountNew {
			t.Errorf("selectedAccountId guardado como %q, se esperaba %q", v, accountNew)
		}
		if n := verifier.calls.Load(); n != 0 {
			t.Errorf("la sesión se validó %d veces en un endpoint interno, se esperaba 0", n)
		}
	})

	t.Run("sin setting", func(t *testing.T) {
		setup(t)
		insertSyncFixtures(t)

		rec := syncAccount(newInternalAPI(&countingVerifier{}), user2ID, accountOfUser2, secret)

		assertUpdated(t, rec, false)
		if n := countSettings(t, bson.M{"userId": user2ID}); n != 0 {
			t.Fatalf("hay %d settings de %s, se esperaba 0", n, user2ID)
		}
	})

	rejected := []struct {
		name      string
		headers   map[string]string
		accountID string
		status    int
		code      string
	}{
		{name: "sin x-secret", headers: map[string]string{}, accountID: accountNew,
			status: http.StatusUnauthorized, code: "UNAUTHORIZED_INVALID_SECRET"},
		{name: "cuenta de otro usuario", headers: secret, accountID: accountOfUser2,
			status: http.StatusNotFound, code: "ERR_ACCOUNT_NOT_FOUND"},
	}
	for _, tt := range rejected {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)
			insertSyncFixtures(t)
			before := rawDoc(t, settingCollection, settingID)

			rec := syncAccount(newInternalAPI(&countingVerifier{}), userID, tt.accountID, tt.headers)

			assertError(t, rec, tt.status, tt.code)
			if after := rawDoc(t, settingCollection, settingID); !bytes.Equal(before, after) {
				t.Fatalf("el setting cambió:\n antes   %s\n después %s", before, after)
			}
		})
	}
}
