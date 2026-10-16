package handlers_shared_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
)

const (
	uuidID   = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"
	meteorID = "tDr6NwSsDe4Lnccp8"
)

// fakeRepo registra los ids con los que el handler llega al repositorio.
type fakeRepo struct {
	ids []string
}

func (f *fakeRepo) FindOneByID(_ context.Context, id string) {
	f.ids = append(f.ids, id)
}

// newTestAPI arma un nexus.Server con un endpoint por id y otro con un parámetro de otro nombre.
func newTestAPI(repo *fakeRepo) http.Handler {
	byParam := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			id, ok := handlers_shared.PathID(w, r, name)
			if !ok {
				return
			}
			repo.FindOneByID(r.Context(), id)
			_ = nexus.ResponseWithJSON(w, http.StatusOK, map[string]string{"id": id})
		}
	}

	server := &nexus.Server{
		ServerName: "blooming-test",
		Settings:   &nexus.Settings{PathPrefix: "/api/v1"},
	}
	server.Group("/bond-types", []nexus.Endpoint{
		{Path: "GET /{id}", HandlerFunc: byParam("id")},
	})
	server.Group("/users", []nexus.Endpoint{
		{Path: "GET /{userId}/bond-bounds", HandlerFunc: byParam("userId")},
	})
	return server.PrepareEndpoints()
}

func get(h http.Handler, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

// Un id fuera de formato responde 400 ERR_INVALID_ID y el handler no llega al repositorio. CA-00.31
func TestPathIDRejectsInvalidID(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{name: "texto corto", target: "/api/v1/bond-types/abc"},
		{name: "ObjectId", target: "/api/v1/bond-types/64b7f0c2a1e4d3b2c1a0f9e8"},
		{name: "17 caracteres con guion", target: "/api/v1/bond-types/tDr6NwSsDe4Lncc-8"},
		{name: "UUID v1", target: "/api/v1/bond-types/6ba7b810-9dad-11d1-80b4-00c04fd430c8"},
		{name: "Meteor con espacio codificado", target: "/api/v1/bond-types/" + meteorID + "%20"},
		{name: "parámetro userId", target: "/api/v1/users/abc/bond-bounds"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}

			rec := get(newTestAPI(repo), tt.target)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status %d, se esperaba %d (body %s)", rec.Code, http.StatusBadRequest, rec.Body)
			}
			var body nexus.ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("el body no es un error de nexus: %v (body %s)", err, rec.Body)
			}
			if body.Message != "ERR_INVALID_ID" {
				t.Fatalf("código %q, se esperaba ERR_INVALID_ID", body.Message)
			}
			if len(repo.ids) != 0 {
				t.Fatalf("el repositorio recibió %d llamadas, se esperaba 0: %v", len(repo.ids), repo.ids)
			}
		})
	}
}

// Un id UUID v4 o Meteor llega al repositorio tal cual y una sola vez. CA-00.31
func TestPathIDAcceptsBothFormats(t *testing.T) {
	tests := []struct {
		name   string
		target string
		id     string
	}{
		{name: "UUID v4", target: "/api/v1/bond-types/" + uuidID, id: uuidID},
		{name: "Meteor", target: "/api/v1/bond-types/" + meteorID, id: meteorID},
		{name: "Meteor en userId", target: "/api/v1/users/" + meteorID + "/bond-bounds", id: meteorID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}

			rec := get(newTestAPI(repo), tt.target)

			if rec.Code != http.StatusOK {
				t.Fatalf("status %d, se esperaba %d (body %s)", rec.Code, http.StatusOK, rec.Body)
			}
			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("el body no es JSON: %v (body %s)", err, rec.Body)
			}
			if body["id"] != tt.id {
				t.Fatalf("id %q en la respuesta, se esperaba %q", body["id"], tt.id)
			}
			if !slices.Equal(repo.ids, []string{tt.id}) {
				t.Fatalf("el repositorio recibió %v, se esperaba [%s]", repo.ids, tt.id)
			}
		})
	}
}
