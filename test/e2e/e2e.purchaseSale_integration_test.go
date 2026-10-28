//go:build integration

package e2e_test

import (
	"net/http"
	"testing"
)

// Un inversionista elige su cuenta destino, compra un bono y lo vende a mitad de plazo: la compra descuenta
// el precio, la venta acredita lo que cotizó pre-sale, cada paso avisa por ms-notification y un bono vendido
// no se vende otra vez. RF-04.1, RF-05.3
func TestPurchaseThenSale(t *testing.T) {
	setup(t)
	seed(t)
	selectAccount(t)

	bought := purchase(t)
	if bought.Key != productKey+"-1" || bought.AccountAvailable != initialAvailable-price {
		t.Fatalf("compra = %+v, se esperaba la clave %s-1 y %v disponible", bought, productKey, initialAvailable-price)
	}
	assertAvailable(t, initialAvailable-price)
	assertBoundStatus(t, bought.BondBoundID, "1")
	notifications.wait(t, notifyPurchased, notifyWithdrawUser, notifyWithdrawAdmin)

	// a los 50 de 100 días el interés de 1000 al 10 % es 50, y con la penalidad del 50 % se acreditan 25
	backdate(t, bought.BondBoundID, 50)
	var quote struct {
		Amount float64 `json:"amount"`
		Profit float64 `json:"profit"`
	}
	call(t, http.MethodGet, "/bond-bounds/"+bought.BondBoundID+"/pre-sale", investorToken, nil).expect(t, http.StatusOK, &quote)
	if quote.Amount != 1025 || quote.Profit != 25 {
		t.Fatalf("pre-sale = %+v, se esperaba amount 1025 y profit 25", quote)
	}

	sold := sell(t, bought.BondBoundID)
	want := saleResult{
		BondBoundID: bought.BondBoundID,
		StatusID:    "5",
		Amount:      quote.Amount,
		Profit:      quote.Profit,
		AccountID:   accountID,
		Available:   initialAvailable - price + quote.Amount,
	}
	if sold != want {
		t.Fatalf("venta = %+v, se esperaba %+v", sold, want)
	}
	assertAvailable(t, want.Available)
	assertBoundStatus(t, bought.BondBoundID, "5")
	notifications.wait(t, notifySold)

	call(t, http.MethodPost, "/bond-bounds/"+bought.BondBoundID+"/sale", investorToken, nil).
		expectError(t, http.StatusConflict, "ERR_BOND_ALREADY_SOLD")
	assertAvailable(t, want.Available)
}
