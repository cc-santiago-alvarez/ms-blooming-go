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

// GET /bond-bounds/{id}/pre-sale pasa el id del path a la aplicación y responde 200 con la cotización; un id
// inválido responde 400 ERR_INVALID_ID sin llegar a la aplicación. CA-05.6
func TestPreSale(t *testing.T) {
	t.Run("dueño o admin", func(t *testing.T) {
		app := &fakeApp{projection: bondBound_domain.SaleProjection{
			Inversion: 1000, Interest: 10, PenaltyPercentage: 50, Profit: 25, AtMoment: 1050, Amount: 1025, TotalDaysPassed: 50,
		}}

		rec := serve(newTestAPI(app, investor), http.MethodGet, preSalePath, "")

		var body map[string]any
		decodeJSON(t, rec, http.StatusOK, &body)
		want := map[string]any{
			"inversion": 1000.0, "interest": 10.0, "penaltyPercentage": 50.0, "profit": 25.0,
			"atMoment": 1050.0, "amount": 1025.0, "totalDaysPassed": 50.0,
		}
		if !maps.Equal(body, want) {
			t.Errorf("body %s, se esperaba %v", rec.Body, want)
		}
		if !slices.Equal(app.calls, []string{"PreSale"}) || app.id != boundID {
			t.Errorf("la aplicación recibió %v con el id %q, se esperaba PreSale con %q", app.calls, app.id, boundID)
		}
	})

	t.Run("id inválido", func(t *testing.T) {
		app := &fakeApp{}

		rec := serve(newTestAPI(app, investor), http.MethodGet, boundsPath+"/abc/pre-sale", "")

		assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_ID")
		if len(app.calls) != 0 {
			t.Fatalf("la aplicación recibió %v con un id inválido", app.calls)
		}
	})
}

// Los errores de la cotización responden su status y su código: el bono ajeno 403, el inexistente 404 y el
// que no se puede vender por su estado 409. CA-05.6, CA-05.9
func TestPreSaleRejected(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{err: shared_domain.ErrForbidden, status: http.StatusForbidden},
		{err: bondBound_domain.ErrBondBoundNotFound, status: http.StatusNotFound},
		{err: bondBound_domain.ErrBondStatusNotSellable, status: http.StatusConflict},
		{err: bondBound_domain.ErrBondAlreadySold, status: http.StatusConflict},
		{err: bondBound_domain.ErrBondAlreadyTook, status: http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			app := &fakeApp{err: tt.err}

			rec := serve(newTestAPI(app, investor), http.MethodGet, preSalePath, "")

			assertError(t, rec, tt.status, tt.err.Error())
		})
	}
}

// Antes del día mínimo de retiro la cotización responde 422 ERR_INVALID_WITHDRAW_DATE con la fecha de hoy en
// DD/MM/YYYY, en la zona de la sesión, en errors.detail. CA-05.5
func TestPreSaleBeforeWithdrawDate(t *testing.T) {
	for _, utcDiff := range []int{-5, 14} {
		t.Run(fmt.Sprintf("UTC%+d", utcDiff), func(t *testing.T) {
			app := &fakeApp{err: bondBound_domain.ErrInvalidWithdrawDate}

			rec := serve(newTestAPI(app, investorIn(utcDiff)), http.MethodGet, preSalePath, "")

			assertWithdrawDateDetail(t, rec, utcDiff)
		})
	}
}
