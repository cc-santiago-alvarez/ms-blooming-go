package bondBounds_handlers_test

import (
	"net/http"
	"reflect"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// detailsPath es la ruta del detalle del bono.
const detailsPath = boundPath + "/details"

// sampleDetail es el detalle de un bono iniciado hace 50 días.
func sampleDetail() *bondBound_ports.BondBoundDetail {
	return &bondBound_ports.BondBoundDetail{
		ID:                 boundID,
		Key:                "BP12-1",
		AgencyID:           agencyA,
		Amount:             1000,
		MaturityTimeInDays: 365,
		BondObject:         bondBound_ports.BondObjectPublic{ID: objectID, Name: "Bono Plus 12M", AgencyID: shared_domain.AgencyIDs{agencyA}},
		BondType:           bondType_domain.BondType{Name: "Plus"},
		Status:             bondBound_domain.StatusInitiated.Ref(),
		DaysPassed:         50,
		AcumulateAmount:    157.53,
	}
}

// GET /bond-bounds/{id}/details responde 200 con el detalle, sin los ids de estado, dueño, producto y tipo
// ni los usuarios autorizados del producto; un bono ajeno responde 403 ERR_FORBIDDEN, uno inexistente o de
// otra agencia 404 ERR_BOND_BOUND_NOT_FOUND, y un id inválido 400 ERR_INVALID_ID sin llegar a la
// aplicación. CA-06.10
func TestGetDetails(t *testing.T) {
	t.Run("dueño", func(t *testing.T) {
		app := &fakeApp{detail: sampleDetail()}

		rec := serve(newTestAPI(app, investor), http.MethodGet, detailsPath, "")

		var body map[string]any
		decodeJSON(t, rec, http.StatusOK, &body)
		if !slices.Equal(app.calls, []string{"GetDetails"}) || app.id != boundID {
			t.Fatalf("la aplicación recibió %v con el id %q, se esperaba GetDetails con %q", app.calls, app.id, boundID)
		}
		for _, key := range []string{"statusId", "userId", "bondObjectId", "bondTypeId"} {
			if _, ok := body[key]; ok {
				t.Errorf("el detalle tiene %q: %s", key, rec.Body)
			}
		}
		if want := map[string]any{"_id": "1", "name": "Iniciado"}; !reflect.DeepEqual(body["status"], want) {
			t.Errorf("status %v, se esperaba %v", body["status"], want)
		}
		if body["daysPassed"] != 50.0 || body["acumulateAmount"] != 157.53 {
			t.Errorf("daysPassed %v y acumulateAmount %v, se esperaban 50 y 157.53", body["daysPassed"], body["acumulateAmount"])
		}
		bondObject, ok := body["bondObject"].(map[string]any)
		if !ok || bondObject["name"] != "Bono Plus 12M" {
			t.Fatalf("bondObject %v, se esperaba el producto del detalle", body["bondObject"])
		}
		if _, ok := bondObject["authorizedUserIds"]; ok {
			t.Errorf("el producto del detalle tiene authorizedUserIds: %s", rec.Body)
		}
	})

	rejected := []struct {
		name   string
		err    error
		status int
	}{
		{name: "bono ajeno", err: shared_domain.ErrForbidden, status: http.StatusForbidden},
		{name: "inexistente o de otra agencia", err: bondBound_domain.ErrBondBoundNotFound, status: http.StatusNotFound},
	}
	for _, tt := range rejected {
		t.Run(tt.name, func(t *testing.T) {
			app := &fakeApp{err: tt.err}

			rec := serve(newTestAPI(app, investor), http.MethodGet, detailsPath, "")

			assertError(t, rec, tt.status, tt.err.Error())
		})
	}

	t.Run("id inválido", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundsPath+"/abc/details", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_ID")
		if len(app.calls) != 0 {
			t.Fatalf("la aplicación recibió %v con un id inválido", app.calls)
		}
	})
}
