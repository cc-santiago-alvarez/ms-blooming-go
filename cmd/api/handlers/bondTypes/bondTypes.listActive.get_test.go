package bondTypes_handlers_test

import (
	"net/http"
	"testing"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
)

// GET /bond-types/active responde cada tipo con sus campos y numOfBonds en el mismo objeto. CA-01.3
func TestListActiveNumOfBonds(t *testing.T) {
	app := &fakeApp{active: []bondType_domain.ActiveBondType{{BondType: *sample(), NumOfBonds: 2}}}

	rec := serve(newTestAPI(app, investor), http.MethodGet, typesPath+"/active", "")

	var body []map[string]any
	decodeJSON(t, rec, http.StatusOK, &body)
	if len(body) != 1 {
		t.Fatalf("body %s, se esperaba un tipo", rec.Body)
	}
	want := map[string]any{"_id": typeID, "name": "Bono Plus", "active": true, "isRemove": false, "numOfBonds": 2.0}
	for k, w := range want {
		if body[0][k] != w {
			t.Errorf("%s = %v, se esperaba %v", k, body[0][k], w)
		}
	}
	if agencies, ok := body[0]["agencyId"].([]any); !ok || len(agencies) != 1 || agencies[0] != agencyA {
		t.Errorf("agencyId = %v, se esperaba [%s]", body[0]["agencyId"], agencyA)
	}
}
