package bondBounds_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// POST /bond-bounds/bulk-purchase pasa el body, con quantity, a la aplicación y responde 201 con los bonos,
// el total y el saldo que le queda a la cuenta. CA-04.6
func TestBulkPurchase(t *testing.T) {
	app := &fakeApp{bulkResult: bondBound_domain.BulkPurchaseResult{
		Items: []bondBound_domain.BulkPurchaseItem{
			{ID: boundID, Key: "BP12-59", Reference: "REF001"},
			{ID: legacyBoundID, Key: "BP12-60", Reference: "REF002"},
			{ID: "1b2c3d4e-5f6a-4b7c-9d8e-0f1a2b3c4d5e", Key: "BP12-61", Reference: "REF003"},
		},
		Total:            3000,
		AccountAvailable: 1000,
	}}

	rec := serve(newTestAPI(app, investor), http.MethodPost, bulkPurchasePath, bulkBody)

	var body bondBound_domain.BulkPurchaseResult
	decodeJSON(t, rec, http.StatusCreated, &body)
	if len(body.Items) != 3 || body.Items[0].Reference != "REF001" || body.Total != 3000 || body.AccountAvailable != 1000 {
		t.Errorf("body %s, se esperaba el resultado de la aplicación", rec.Body)
	}
	p := app.bulk
	if !slices.Equal(app.calls, []string{"BulkPurchase"}) || p.AccountID != accountID || p.BondObjectID != objectID || p.Quantity != 3 {
		t.Errorf("la aplicación recibió %v con %+v, se esperaba BulkPurchase con el body", app.calls, p)
	}
}

// Una cantidad fuera del rango responde 400 ERR_INVALID_QUANTITY, el saldo insuficiente para el total 422
// ERR_INSUFFICIENT_AMOUNT y la cuenta ajena 404 ERR_ACCOUNT_NOT_FOUND. CA-04.3, CA-04.6
func TestBulkPurchaseRejected(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{err: bondBound_domain.ErrInvalidQuantity, status: http.StatusBadRequest},
		{err: bondBound_domain.ErrInsufficientAmount, status: http.StatusUnprocessableEntity},
		{err: shared_domain.ErrAccountNotFound, status: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			app := &fakeApp{err: tt.err}

			rec := serve(newTestAPI(app, investor), http.MethodPost, bulkPurchasePath, bulkBody)

			assertError(t, rec, tt.status, tt.err.Error())
		})
	}
}

// Una quantity que no es un entero responde 400 INVALID_REQUEST_BODY sin llegar a la aplicación. CA-04.6
func TestBulkPurchaseInvalidQuantityType(t *testing.T) {
	app := &fakeApp{}
	body := `{"accountId":"` + accountID + `","bondObjectId":"` + objectID + `","quantity":"3"}`

	rec := serve(newTestAPI(app, investor), http.MethodPost, bulkPurchasePath, body)

	assertError(t, rec, http.StatusBadRequest, "INVALID_REQUEST_BODY")
	if len(app.calls) != 0 {
		t.Fatalf("la aplicación recibió %v con quantity no numérica", app.calls)
	}
}
