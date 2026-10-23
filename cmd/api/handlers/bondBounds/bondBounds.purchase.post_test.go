package bondBounds_handlers_test

import (
	"net/http"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// POST /bond-bounds/purchase pasa el body a la aplicación y responde 201 con el bono y el saldo que le
// queda a la cuenta. CA-04.4
func TestPurchase(t *testing.T) {
	app := &fakeApp{purchaseResult: bondBound_domain.PurchaseResult{BondBoundID: boundID, Key: "BP12-58", AccountAvailable: 4000}}

	rec := serve(newTestAPI(app, investor), http.MethodPost, purchasePath, purchaseBody)

	var body bondBound_domain.PurchaseResult
	decodeJSON(t, rec, http.StatusCreated, &body)
	if body.BondBoundID != boundID || body.Key != "BP12-58" || body.AccountAvailable != 4000 {
		t.Errorf("body %s, se esperaba el resultado de la aplicación", rec.Body)
	}
	if !slices.Equal(app.calls, []string{"Purchase"}) || app.purchase.AccountID != accountID || app.purchase.BondObjectID != objectID {
		t.Errorf("la aplicación recibió %v con %+v, se esperaba Purchase con el body", app.calls, app.purchase)
	}
}

// Los errores de la compra responden su status y su código: la cuenta ajena 404, el privado sin autorizar
// 403 y las validaciones de cuenta, producto, saldo y ventana de venta 422. CA-04.2, CA-04.3, CA-04.4
func TestPurchaseRejected(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{err: bondBound_domain.ErrAccountIDRequired, status: http.StatusBadRequest},
		{err: shared_domain.ErrAccountNotFound, status: http.StatusNotFound},
		{err: bondBound_domain.ErrBondObjectNotFound, status: http.StatusNotFound},
		{err: bondBound_domain.ErrBondPrivateNotAuthorized, status: http.StatusForbidden},
		{err: bondBound_domain.ErrAccountWithoutFunds, status: http.StatusUnprocessableEntity},
		{err: bondBound_domain.ErrAccountNotActive, status: http.StatusUnprocessableEntity},
		{err: bondBound_domain.ErrBondInactive, status: http.StatusUnprocessableEntity},
		{err: bondBound_domain.ErrInsufficientAmount, status: http.StatusUnprocessableEntity},
		{err: bondBound_domain.ErrBondNotAvailableByDate, status: http.StatusUnprocessableEntity},
		{err: bondBound_domain.ErrBondSaleEnded, status: http.StatusUnprocessableEntity},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			app := &fakeApp{err: tt.err}

			rec := serve(newTestAPI(app, investor), http.MethodPost, purchasePath, purchaseBody)

			assertError(t, rec, tt.status, tt.err.Error())
		})
	}
}

// Un request sin body responde 400 BODY_REQUIRED y uno con JSON inválido 400 INVALID_REQUEST_BODY, sin
// llegar a la aplicación. CA-04.2
func TestPurchaseInvalidBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		code string
	}{
		{name: "sin body", body: "", code: "BODY_REQUIRED"},
		{name: "JSON inválido", body: `{"accountId":`, code: "INVALID_REQUEST_BODY"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &fakeApp{}

			rec := serve(newTestAPI(app, investor), http.MethodPost, purchasePath, tt.body)

			assertError(t, rec, http.StatusBadRequest, tt.code)
			if len(app.calls) != 0 {
				t.Fatalf("la aplicación recibió %v con un body inválido", app.calls)
			}
		})
	}
}
