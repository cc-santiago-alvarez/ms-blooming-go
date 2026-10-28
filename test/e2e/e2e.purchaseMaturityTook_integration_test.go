//go:build integration

package e2e_test

import (
	"net/http"
	"testing"
)

// maturityResult es la respuesta del job de maduración.
type maturityResult struct {
	Matured int `json:"matured"`
	Users   int `json:"users"`
	Failed  int `json:"failed"`
}

// runMaturity corre el job de maduración como ms-cronjob-go: solo con x-secret.
func runMaturity(t *testing.T) maturityResult {
	t.Helper()
	var result maturityResult
	call(t, http.MethodPost, "/internal/jobs/bond-maturity/run", "", nil).expect(t, http.StatusOK, &result)
	return result
}

// Un inversionista compra dos bonos y uno vence: el job de maduración madura solo el vencido y lo avisa, el
// inversionista lo canjea por el precio más el rendimiento completo y una segunda corrida no madura nada.
// RF-04.1, RF-10.3, RF-05.3
func TestPurchaseMaturityThenTook(t *testing.T) {
	setup(t)
	seed(t)
	selectAccount(t)

	due := purchase(t)
	pending := purchase(t)
	if pending.AccountAvailable != initialAvailable-2*price {
		t.Fatalf("disponible tras dos compras = %v, se esperaba %v", pending.AccountAvailable, initialAvailable-2*price)
	}
	// due venció ayer; pending vence dentro de 100 días
	backdate(t, due.BondBoundID, maturityDays+1)

	if got, want := runMaturity(t), (maturityResult{Matured: 1, Users: 1}); got != want {
		t.Fatalf("maduración = %+v, se esperaba %+v", got, want)
	}
	assertBoundStatus(t, due.BondBoundID, "2")
	assertBoundStatus(t, pending.BondBoundID, "1")
	notifications.wait(t, notifyMatured)

	// el canje paga el rendimiento completo: 1000 más el 10 %
	took := sell(t, due.BondBoundID)
	want := saleResult{
		BondBoundID: due.BondBoundID,
		StatusID:    "6",
		Amount:      price + price*rewards/100,
		Profit:      price * rewards / 100,
		AccountID:   accountID,
		Available:   initialAvailable - 2*price + price + price*rewards/100,
	}
	if took != want {
		t.Fatalf("canje = %+v, se esperaba %+v", took, want)
	}
	assertAvailable(t, want.Available)
	assertBoundStatus(t, due.BondBoundID, "6")
	notifications.wait(t, notifyTook)

	if got := runMaturity(t); got != (maturityResult{}) {
		t.Fatalf("segunda maduración = %+v, se esperaba ninguna", got)
	}
	assertBoundStatus(t, pending.BondBoundID, "1")
}
