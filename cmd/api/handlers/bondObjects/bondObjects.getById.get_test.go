package bondObjects_handlers_test

import (
	"net/http"
	"testing"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// GET /bond-objects/{id} de un producto con un autorizado responde authorizedUser[0] con first_name y
// email, y sin password. CA-02.3
func TestGetByID(t *testing.T) {
	object := sample()
	object.IsPrivateBond = true
	object.AuthorizedUserIDs = []string{userID}
	app := &fakeApp{detail: &bondObject_domain.BondObjectDetail{
		BondObject:     *object,
		AuthorizedUser: []shared_domain.PublicProfile{{ID: userID, FirstName: "Ana", Email: "ana@futurapps.com"}},
	}}

	rec := serve(newTestAPI(app, admin), http.MethodGet, objectPath, "")

	var body struct {
		ID             string           `json:"_id"`
		AuthorizedUser []map[string]any `json:"authorizedUser"`
	}
	decodeJSON(t, rec, http.StatusOK, &body)
	if app.id != objectID {
		t.Fatalf("la aplicación recibió el id %q, se esperaba %q", app.id, objectID)
	}
	if body.ID != objectID || len(body.AuthorizedUser) != 1 {
		t.Fatalf("body %s, se esperaba el producto %s con un autorizado", rec.Body, objectID)
	}
	user := body.AuthorizedUser[0]
	if user["first_name"] != "Ana" || user["email"] != "ana@futurapps.com" {
		t.Errorf("authorizedUser[0] = %v, se esperaban first_name y email del perfil", user)
	}
	if _, ok := user["password"]; ok {
		t.Errorf("authorizedUser[0] trae password: %v", user)
	}
}
