package bondObjects_handlers_test

import (
	"net/http"
	"testing"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
)

// Un POST sin isPrivateBond ni inflectionPenaltyPercentage llega a la aplicación sin defaults, que los
// aplica el caso de uso, y responde 201 con el documento creado. CA-02.4
func TestCreate(t *testing.T) {
	app := &fakeApp{object: sample()}

	rec := serve(newTestAPI(app, admin), http.MethodPost, objectsPath, validBody)

	var body struct {
		ID            string `json:"_id"`
		IsPrivateBond bool   `json:"isPrivateBond"`
		EarlyCashTerm struct {
			InflectionPenaltyPercentage float64 `json:"inflectionPenaltyPercentage"`
		} `json:"earlyCashTerm"`
	}
	decodeJSON(t, rec, http.StatusCreated, &body)
	if body.ID != objectID || body.IsPrivateBond || body.EarlyCashTerm.InflectionPenaltyPercentage != 100 {
		t.Errorf("body %s, se esperaba el producto %s que devolvió la aplicación", rec.Body, objectID)
	}
	p := app.params
	if p.Name != "Bono Plus 12M" || p.BondTypeID != typeID || p.Amount != 1000 || p.EarlyCashTerm.InflectionPoint != 90 {
		t.Errorf("la aplicación recibió %+v, se esperaba el body del request", p)
	}
	if p.IsPrivateBond || p.EarlyCashTerm.InflectionPenaltyPercentage != 0 {
		t.Errorf("la aplicación recibió isPrivateBond %v e inflectionPenaltyPercentage %v, se esperaban sin defaults",
			p.IsPrivateBond, p.EarlyCashTerm.InflectionPenaltyPercentage)
	}
}

// Un bondTypeId inexistente responde 422 ERR_BOND_OBJECT_INVALID_BOND_TYPE, e inflectionPoint igual a
// maturityTimeInDays responde 400 ERR_BOND_OBJECT_INVALID_INFLECTION_POINT. CA-02.5
func TestCreateRejected(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "bondTypeId inexistente", err: bondObject_domain.ErrBondObjectInvalidBondType, status: http.StatusUnprocessableEntity, code: "ERR_BOND_OBJECT_INVALID_BOND_TYPE"},
		{name: "inflectionPoint igual al plazo", err: bondObject_domain.ErrBondObjectInvalidInflectionPoint, status: http.StatusBadRequest, code: "ERR_BOND_OBJECT_INVALID_INFLECTION_POINT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &fakeApp{err: tt.err}

			rec := serve(newTestAPI(app, admin), http.MethodPost, objectsPath, validBody)

			assertError(t, rec, tt.status, tt.code)
		})
	}
}
