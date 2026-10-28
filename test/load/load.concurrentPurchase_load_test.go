//go:build load

package load_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	// purchases es la cantidad de compras que llegan a la vez.
	purchases = 50
	// affordable es la cantidad de compras que alcanza a pagar la cuenta.
	affordable = 30
)

// insufficientFunds son los códigos de una compra sin saldo: la validación previa o el retiro dentro de la
// transacción, según cuándo vio el saldo.
var insufficientFunds = []string{
	"ERR_INSUFFICIENT_AMOUNT", "ERR_ACCOUNT_WITHOUT_FUNDS", "ERR_ACCOUNT_WITHDRAW_AMOUNT_MORE_THAN_AVAILABLE",
}

// De 50 compras concurrentes de 1000 sobre una cuenta con 30 000, exactamente 30 se confirman, con las
// claves BP12-1 a BP12-30 sin huecos ni repetidas, y cada una ve el saldo que dejó la anterior; las otras 20
// responden 422 por falta de saldo y ninguna responde 5xx. La cuenta queda en 0, el contador en 30 y cada
// bono con su movimiento y sus dos historiales. RF-04.5
func TestConcurrentPurchasesSameAccount(t *testing.T) {
	setup(t)
	seedCatalog(t)
	insertAccount(t, affordable*price)

	results := make([]response, purchases)
	errs := make([]error, purchases)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range purchases {
		wg.Go(func() {
			<-start
			results[i], errs[i] = send(http.MethodPost, "/bond-bounds/purchase", investorToken,
				map[string]string{"accountId": accountID, "bondObjectId": objectID})
		})
	}
	close(start)
	wg.Wait()

	var (
		ids, keys, rejected []string
		balances            []float64
		took                []time.Duration
	)
	for i, res := range results {
		if errs[i] != nil {
			t.Fatalf("compra %d: %v", i, errs[i])
		}
		took = append(took, res.took)
		switch res.status {
		case http.StatusCreated:
			var bought struct {
				BondBoundID      string  `json:"bondBoundId"`
				Key              string  `json:"key"`
				AccountAvailable float64 `json:"accountAvailable"`
			}
			if err := res.decode(http.StatusCreated, &bought); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, bought.BondBoundID)
			keys = append(keys, bought.Key)
			balances = append(balances, bought.AccountAvailable)
		case http.StatusUnprocessableEntity:
			rejected = append(rejected, res.message())
		default:
			t.Errorf("compra %d respondió %d: %s", i, res.status, res.body)
		}
	}
	logLatencies(t, "compras concurrentes", took)
	if t.Failed() {
		t.FailNow()
	}

	if len(keys) != affordable || len(rejected) != purchases-affordable {
		t.Fatalf("%d compras confirmadas y %d rechazadas, se esperaban %d y %d",
			len(keys), len(rejected), affordable, purchases-affordable)
	}
	for _, code := range rejected {
		if !slices.Contains(insufficientFunds, code) {
			t.Errorf("una compra rechazada respondió %q, se esperaba uno de %v", code, insufficientFunds)
		}
	}

	wantKeys := make([]string, affordable)
	wantBalances := make([]float64, affordable)
	for i := range affordable {
		wantKeys[i] = fmt.Sprintf("%s-%d", productKey, i+1)
		wantBalances[i] = float64(i) * price
	}
	slices.Sort(keys)
	slices.Sort(wantKeys)
	if !slices.Equal(keys, wantKeys) {
		t.Errorf("claves = %v, se esperaban %v", keys, wantKeys)
	}
	slices.Sort(balances)
	if !slices.Equal(balances, wantBalances) {
		t.Errorf("saldos tras cada compra = %v, se esperaban %v", balances, wantBalances)
	}
	if distinct := slices.Compact(slices.Sorted(slices.Values(ids))); len(distinct) != affordable {
		t.Errorf("%d bondBoundId distintos, se esperaban %d", len(distinct), affordable)
	}

	ctx := context.Background()
	var account struct {
		Available float64 `bson:"available"`
	}
	if err := testDB.FindOne(ctx, "balanceAccount", bson.M{"_id": accountID}, nil).Decode(&account); err != nil {
		t.Fatalf("no se pudo leer la cuenta %s: %v", accountID, err)
	}
	if account.Available != 0 {
		t.Errorf("saldo guardado = %v, se esperaba 0", account.Available)
	}
	var counter struct {
		Seq int64 `bson:"seq"`
	}
	if err := testDB.FindOne(ctx, "bondKeyCounter", bson.M{"_id": typeID}, nil).Decode(&counter); err != nil {
		t.Fatalf("no se pudo leer el contador del tipo %s: %v", typeID, err)
	}
	if counter.Seq != affordable {
		t.Errorf("contador del tipo = %d, se esperaba %d", counter.Seq, affordable)
	}
	for collection, want := range map[string]int64{"bondBound": affordable, "balanceDetail": affordable, "bondHistory": 2 * affordable} {
		if n := count(t, collection, bson.M{}); n != want {
			t.Errorf("%s tiene %d documentos, se esperaban %d", collection, n, want)
		}
	}
}
