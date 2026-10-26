package bondBounds_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// GET /bond-bounds/me/recent responde 200 con los bonos recientes, cada uno con el nombre y los términos de
// retiro de su producto. CA-06.9
func TestListRecent(t *testing.T) {
	term := shared_domain.EarlyCashTerm{InitialPenaltyPercentage: 100, InflectionPoint: 90, InflectionPenaltyPercentage: 75, InitialDayForWithdraw: 30}
	app := &fakeApp{recent: []bondBound_ports.BondBoundMineItem{{
		ID:           boundID,
		Key:          "BP12-1",
		StatusID:     bondBound_domain.StatusInitiated,
		BondObjectID: objectID,
		BondObject:   &bondBound_ports.BondObjectRef{Name: "Bono Plus 12M", EarlyCashTerm: &term},
	}}}

	rec := serve(newTestAPI(app, investor), http.MethodGet, boundsPath+"/me/recent", "")

	var body []map[string]any
	decodeJSON(t, rec, http.StatusOK, &body)
	if len(body) != 1 || body[0]["_id"] != boundID || body[0]["bondObjectId"] != objectID {
		t.Fatalf("body %s, se esperaban los bonos que devolvió la aplicación", rec.Body)
	}
	bondObject, ok := body[0]["bondObject"].(map[string]any)
	if !ok || bondObject["name"] != "Bono Plus 12M" {
		t.Fatalf("bondObject %v, se esperaba el producto del bono", body[0]["bondObject"])
	}
	if ect, ok := bondObject["earlyCashTerm"].(map[string]any); !ok || ect["inflectionPoint"] != 90.0 {
		t.Errorf("bondObject.earlyCashTerm %v, se esperaban los términos de retiro del producto", bondObject["earlyCashTerm"])
	}
	if !slices.Equal(app.calls, []string{"ListRecent"}) {
		t.Errorf("la aplicación recibió %v, se esperaba ListRecent", app.calls)
	}
}
