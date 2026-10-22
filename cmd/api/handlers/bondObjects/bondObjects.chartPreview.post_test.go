package bondObjects_handlers_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
)

const previewPath = objectsPath + "/chart/preview"

// POST /bond-objects/chart/preview sin adminBlooming responde 403 ERR_FORBIDDEN y no llega a la
// aplicación. CA-03.6
func TestChartPreviewRequiresAdmin(t *testing.T) {
	app := &fakeApp{chart: sampleChart(true)}

	rec := serve(newTestAPI(app, investor), http.MethodPost, previewPath, previewBody)

	assertError(t, rec, http.StatusForbidden, "ERR_FORBIDDEN")
	if len(app.calls) != 0 {
		t.Fatalf("la aplicación recibió %v, se esperaba ninguna llamada", app.calls)
	}
}

// Con adminBlooming el body llega a la aplicación y la respuesta trae guides y data; un inflectionPoint
// fuera del plazo responde 400 ERR_BOND_OBJECT_INVALID_INFLECTION_POINT. CA-03.6
func TestChartPreview(t *testing.T) {
	t.Run("términos válidos", func(t *testing.T) {
		app := &fakeApp{chart: sampleChart(true)}

		rec := serve(newTestAPI(app, admin), http.MethodPost, previewPath, previewBody)

		var body bondObject_domain.BondObjectChart
		decodeJSON(t, rec, http.StatusOK, &body)
		if len(body.Guides) != 1 || len(body.Data) != 2 {
			t.Errorf("body %s, se esperaba la gráfica que devolvió la aplicación", rec.Body)
		}
		p, term := app.preview, app.preview.EarlyCashTerm
		if p.Amount != 1000.5 || p.MaturityTimeInDays != 365 || p.MaturityRewardsPercentage != 15 || !p.Simulation {
			t.Errorf("la aplicación recibió %+v, se esperaban los términos y simulation del body", p)
		}
		if term.InitialPenaltyPercentage != 100 || term.InflectionPoint != 90 ||
			term.InflectionPenaltyPercentage != 75 || term.InitialDayForWithdraw != 30 {
			t.Errorf("la aplicación recibió earlyCashTerm %+v, se esperaba el del body", term)
		}
	})

	t.Run("inflectionPoint fuera del plazo", func(t *testing.T) {
		app := &fakeApp{err: bondObject_domain.ErrBondObjectInvalidInflectionPoint}

		rec := serve(newTestAPI(app, admin), http.MethodPost, previewPath, previewBody)

		assertError(t, rec, http.StatusBadRequest, "ERR_BOND_OBJECT_INVALID_INFLECTION_POINT")
	})
}

// Con simulation false la respuesta trae guides y no tiene la clave data. CA-03.6
func TestChartPreviewWithoutSimulation(t *testing.T) {
	app := &fakeApp{chart: sampleChart(false)}
	body := strings.Replace(previewBody, `"simulation":true`, `"simulation":false`, 1)

	rec := serve(newTestAPI(app, admin), http.MethodPost, previewPath, body)

	var keys map[string]json.RawMessage
	decodeJSON(t, rec, http.StatusOK, &keys)
	if app.preview.Simulation {
		t.Errorf("la aplicación recibió simulation true, se esperaba false")
	}
	if _, ok := keys["guides"]; !ok {
		t.Errorf("body %s, se esperaba la clave guides", rec.Body)
	}
	if _, ok := keys["data"]; ok {
		t.Errorf("body %s, no se esperaba la clave data", rec.Body)
	}
}
