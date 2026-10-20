package bondTypes_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// GET /bond-types pasa a la aplicación cada agencyId del query y responde los tipos; un agencyId
// inválido es 400 ERR_INVALID_ID. CA-01.1
func TestListAgencyFilter(t *testing.T) {
	t.Run("filtro por agencias", func(t *testing.T) {
		app := &fakeApp{types: []*bondType_domain.BondType{sample()}}

		rec := serve(newTestAPI(app, admin), http.MethodGet, typesPath+"?agencyId="+agencyA+"&agencyId="+agencyB, "")

		var body []map[string]any
		decodeJSON(t, rec, http.StatusOK, &body)
		if !slices.Equal(app.agencyIDs, []string{agencyA, agencyB}) {
			t.Fatalf("la aplicación recibió %v, se esperaba [%s %s]", app.agencyIDs, agencyA, agencyB)
		}
		if len(body) != 1 || body[0]["_id"] != typeID {
			t.Fatalf("body %s, se esperaba el tipo %s", rec.Body, typeID)
		}
	})

	t.Run("agencyId inválido", func(t *testing.T) {
		app := &fakeApp{err: shared_domain.ErrInvalidID}

		rec := serve(newTestAPI(app, admin), http.MethodGet, typesPath+"?agencyId=abc", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_ID")
		if !slices.Equal(app.agencyIDs, []string{"abc"}) {
			t.Fatalf("la aplicación recibió %v, se esperaba [abc]", app.agencyIDs)
		}
	})
}
