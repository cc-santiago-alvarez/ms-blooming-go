package bondBounds_handlers_test

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// POST /bond-bounds/{id}/sale, sin body, pasa el id del path a la aplicación y responde 200 con lo acreditado
// y el saldo de la cuenta destino; un id inválido responde 400 ERR_INVALID_ID sin llegar a la aplicación.
// CA-05.10
func TestSale(t *testing.T) {
	t.Run("dueño", func(t *testing.T) {
		app := &fakeApp{saleResult: bondBound_domain.SaleResult{
			BondBoundID: boundID, StatusID: bondBound_domain.StatusSold, Amount: 1025, Profit: 25, AccountID: accountID, Available: 6025,
		}}

		rec := serve(newTestAPI(app, investor), http.MethodPost, salePath, "")

		var body map[string]any
		decodeJSON(t, rec, http.StatusOK, &body)
		want := map[string]any{
			"bondBoundId": boundID, "statusId": "5", "amount": 1025.0, "profit": 25.0, "accountId": accountID, "available": 6025.0,
		}
		if !maps.Equal(body, want) {
			t.Errorf("body %s, se esperaba %v", rec.Body, want)
		}
		if !slices.Equal(app.calls, []string{"Sale"}) || app.id != boundID {
			t.Errorf("la aplicación recibió %v con el id %q, se esperaba Sale con %q", app.calls, app.id, boundID)
		}
	})

	t.Run("id inválido", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, investor), http.MethodPost, boundsPath+"/abc/sale", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_ID")
		if len(app.calls) != 0 {
			t.Fatalf("la aplicación recibió %v con un id inválido", app.calls)
		}
	})
}

// Los errores de la venta responden su status y su código: el bono ajeno 403, el que no se puede vender por
// su estado 409, sin cuenta seleccionada 422 y la cuenta seleccionada inexistente o ajena 404. CA-05.9,
// CA-05.10, CA-05.12
func TestSaleRejected(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{err: shared_domain.ErrForbidden, status: http.StatusForbidden},
		{err: bondBound_domain.ErrBondBoundNotFound, status: http.StatusNotFound},
		{err: bondBound_domain.ErrBondStatusNotSellable, status: http.StatusConflict},
		{err: bondBound_domain.ErrBondAlreadySold, status: http.StatusConflict},
		{err: bondBound_domain.ErrBondAlreadyTook, status: http.StatusConflict},
		{err: bondBound_domain.ErrUserSettingAccountRequired, status: http.StatusUnprocessableEntity},
		{err: shared_domain.ErrAccountNotFound, status: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			app := &fakeApp{err: tt.err}

			rec := serve(newTestAPI(app, investor), http.MethodPost, salePath, "")

			assertError(t, rec, tt.status, tt.err.Error())
		})
	}
}

// Antes del día mínimo de retiro la venta responde 422 ERR_INVALID_WITHDRAW_DATE con la fecha de hoy en
// DD/MM/YYYY, en la zona de la sesión, en errors.detail. CA-05.11
func TestSaleBeforeWithdrawDate(t *testing.T) {
	for _, utcDiff := range []int{-5, 14} {
		t.Run(fmt.Sprintf("UTC%+d", utcDiff), func(t *testing.T) {
			app := &fakeApp{err: bondBound_domain.ErrInvalidWithdrawDate}

			rec := serve(newTestAPI(app, investorIn(utcDiff)), http.MethodPost, salePath, "")

			assertWithdrawDateDetail(t, rec, utcDiff)
		})
	}
}
