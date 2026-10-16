package session_domain_test

import (
	"encoding/json"
	"reflect"
	"testing"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
)

// oauthSessionJSON tiene la forma de la sesión que responde ms-oauth en GET /api/v1/session.
const oauthSessionJSON = `{
	"_id": "user-1",
	"userId": "user-1",
	"token": "tok-1",
	"extraData": "tok-1",
	"agency": {"_id": "agency-1", "name": "Agencia Uno", "isMain": true},
	"agencyId": "agency-1",
	"email": "ana@futurapps.com",
	"name": "Ana",
	"lastName": "Gómez",
	"id": 42,
	"countryCode": "MX",
	"first_name": "Ana",
	"last_name": "Gómez",
	"status": true,
	"rolesIds": ["r1"],
	"adminBalance": true,
	"adminBlooming": true,
	"symbolCurrency": "$",
	"access": [{"key": "blooming"}],
	"accessUser": {"adminAccess": ["blooming"], "isAdmin": {"blooming": true}, "keys": ["public"]},
	"userLocation": {"location": {"_id": "loc-1"}},
	"updatedAt": "2026-10-16T12:00:00.000Z"
}`

// La sesión de ms-oauth se decodifica con los campos esperados e ignora los demás. CA-00.5
func TestUserSessionDecodesOAuthPayload(t *testing.T) {
	var got session_domain.UserSession
	if err := json.Unmarshal([]byte(oauthSessionJSON), &got); err != nil {
		t.Fatalf("json.Unmarshal() = %v", err)
	}

	want := session_domain.UserSession{
		UserID:         "user-1",
		Token:          "tok-1",
		AgencyID:       "agency-1",
		Agency:         session_domain.SessionAgency{ID: "agency-1", Name: "Agencia Uno", IsMain: true},
		CountryCode:    "MX",
		FirstName:      "Ana",
		LastName:       "Gómez",
		Email:          "ana@futurapps.com",
		AdminBlooming:  true,
		SymbolCurrency: "$",
		AccessUser: session_domain.AccessUser{
			AdminAccess: []string{"blooming"},
			IsAdmin:     map[string]bool{"blooming": true},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sesión decodificada:\n got  %+v\n want %+v", got, want)
	}
}
