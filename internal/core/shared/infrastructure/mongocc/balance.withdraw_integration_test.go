//go:build integration

package shared_mongocc_repository_test

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"slices"
	"sync"
	"testing"
	"time"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const withdrawComment = "Compra del bono AB-3"

var referencePattern = regexp.MustCompile(`^[A-Za-z0-9]{16}$`)

// withdrawParams arma el retiro de una compra de bonos por amount.
func withdrawParams(a accountFixture, amount float64) shared_domain.WithdrawParams {
	return shared_domain.WithdrawParams{
		AccountID:     a.id,
		UserID:        a.userID,
		AgencyID:      a.agencyID,
		DescriptionID: shared_domain.BalanceDescriptionBondMD,
		Amount:        amount,
		Comment:       withdrawComment,
	}
}

// mexicoSession devuelve un context con la sesión del dueño de la cuenta en México (utcDiff -7).
func mexicoSession(a accountFixture) context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID:      a.userID,
		AgencyID:    a.agencyID,
		CountryCode: "MX",
	})
}

// Un retiro descuenta available y global, no toca deferred y registra detail, history y operation.
// CA-00.14
func TestWithdraw(t *testing.T) {
	for _, a := range accountFixtures {
		t.Run(a.name, func(t *testing.T) {
			db := setup(t)
			insertAccount(t, db, a, activeAccount)

			effectiveBefore := shared_domain.EffectiveDate(shared_domain.Now(), -7)
			mov, err := newBalanceRepo(db).Withdraw(mexicoSession(a), withdrawParams(a, 300))
			effectiveAfter := shared_domain.EffectiveDate(shared_domain.Now(), -7)

			if err != nil {
				t.Fatalf("Withdraw() = %v", err)
			}

			account := readAccount(t, db, a.id)
			if account.Available != 700 || account.Global != 1200 || account.Deferred != 200 {
				t.Fatalf("cuenta available %v, global %v, deferred %v; se esperaba 700, 1200, 200",
					account.Available, account.Global, account.Deferred)
			}
			if !account.UpdatedAt.After(accountCreatedAt) {
				t.Fatalf("updatedAt de la cuenta no cambió: %v", account.UpdatedAt)
			}

			details := findAll[shared_domain.BalanceDetail](t, db, "balanceDetail")
			if len(details) != 1 {
				t.Fatalf("hay %d balanceDetail, se esperaba 1", len(details))
			}
			d := details[0]
			assertUUIDv4(t, d.ID)
			if !referencePattern.MatchString(d.Reference) {
				t.Fatalf("reference %q no tiene 16 alfanuméricos", d.Reference)
			}
			if d.EffectiveDate == nil || (*d.EffectiveDate != effectiveBefore && *d.EffectiveDate != effectiveAfter) {
				t.Fatalf("effectiveDate %v, se esperaba %q", d.EffectiveDate, effectiveBefore)
			}
			if time.Since(d.CreatedAt) > time.Minute {
				t.Fatalf("createdAt %v no es de ahora", d.CreatedAt)
			}
			wantDetail := shared_domain.BalanceDetail{
				ID:               d.ID,
				BalanceAccountID: a.id,
				OperationTypeID:  "1",
				Reference:        d.Reference,
				DescriptionID:    "7",
				Amount:           -300,
				Balance:          700,
				AgencyID:         a.agencyID,
				EffectiveDate:    d.EffectiveDate,
				UserID:           a.userID,
				Comment:          withdrawComment,
				CreatedAt:        d.CreatedAt,
				UpdatedAt:        d.CreatedAt,
			}
			if !reflect.DeepEqual(d, wantDetail) {
				t.Fatalf("balanceDetail:\n got  %+v\n want %+v", d, wantDetail)
			}

			histories := findAll[shared_domain.BalanceHistory](t, db, "balanceHistory")
			if len(histories) != 1 {
				t.Fatalf("hay %d balanceHistory, se esperaba 1", len(histories))
			}
			h := histories[0]
			assertUUIDv4(t, h.ID)
			wantHistory := shared_domain.BalanceHistory{
				ID:               h.ID,
				BalanceAccountID: a.id,
				UserID:           a.userID,
				TypeID:           "3",
				Text:             "BALANCE_ACCOUNT_WITHDRAW",
				Params:           []string{d.ID},
				AgencyID:         a.agencyID,
				CreatedAt:        h.CreatedAt,
				UpdatedAt:        h.CreatedAt,
			}
			if !reflect.DeepEqual(h, wantHistory) {
				t.Fatalf("balanceHistory:\n got  %+v\n want %+v", h, wantHistory)
			}

			operations := findAll[shared_domain.BalanceOperation](t, db, "balanceOperation")
			if len(operations) != 1 {
				t.Fatalf("hay %d balanceOperation, se esperaba 1", len(operations))
			}
			o := operations[0]
			assertUUIDv4(t, o.ID)
			assertFieldType(t, rawDoc(t, db, "balanceOperation", o.ID), "extraInfo", bson.TypeNull)
			wantOperation := shared_domain.BalanceOperation{
				ID:         o.ID,
				AgencyID:   a.agencyID,
				UserID:     a.userID,
				TypeID:     "1",
				CategoryID: "3",
				Details:    []string{a.id, d.ID},
				Concept:    withdrawComment,
				CreatedAt:  o.CreatedAt,
				UpdatedAt:  o.CreatedAt,
			}
			if !reflect.DeepEqual(o, wantOperation) {
				t.Fatalf("balanceOperation:\n got  %+v\n want %+v", o, wantOperation)
			}

			wantMov := &shared_domain.BalanceMovement{
				DetailID:  d.ID,
				Reference: d.Reference,
				Amount:    -300,
				Balance:   700,
				Available: 700,
			}
			if !reflect.DeepEqual(mov, wantMov) {
				t.Fatalf("Withdraw() = %+v, se esperaba %+v", mov, wantMov)
			}
		})
	}
}

// Un retiro inválido devuelve el error del legado, sin tocar la cuenta ni registrar movimientos.
// CA-00.15
func TestWithdrawRejected(t *testing.T) {
	withStatus := func(statusID string) accountState {
		s := activeAccount
		s.statusID = statusID
		return s
	}
	removed := activeAccount
	removed.isRemove = true

	tests := []struct {
		name    string
		state   accountState
		amount  float64
		modify  func(p *shared_domain.WithdrawParams)
		wantErr error
	}{
		{name: "cuenta suspendida", state: withStatus("2"), amount: 300, wantErr: shared_domain.ErrAccountSuspended},
		{name: "cuenta inactiva", state: withStatus("0"), amount: 300, wantErr: shared_domain.ErrAccountInactive},
		{name: "cuenta cancelada", state: withStatus("3"), amount: 300, wantErr: shared_domain.ErrAccountCancelled},
		{name: "estado desconocido", state: withStatus("7"), amount: 300, wantErr: shared_domain.ErrAccountStatus},
		{name: "monto mayor al disponible", state: activeAccount, amount: 1000.01, wantErr: shared_domain.ErrAccountWithdrawAmountMoreThanAvailable},
		{name: "monto cero", state: activeAccount, amount: 0, wantErr: shared_domain.ErrAccountWithdrawAmountInvalid},
		{name: "monto negativo", state: activeAccount, amount: -50, wantErr: shared_domain.ErrAccountWithdrawAmountInvalid},
		{name: "estado antes que monto", state: withStatus("2"), amount: 0, wantErr: shared_domain.ErrAccountSuspended},
		{name: "cuenta de otro usuario", state: activeAccount, amount: 300,
			modify: func(p *shared_domain.WithdrawParams) { p.UserID = otherUserID }, wantErr: shared_domain.ErrAccountNotFound},
		{name: "cuenta borrada", state: removed, amount: 300, wantErr: shared_domain.ErrAccountNotFound},
		{name: "cuenta inexistente", state: activeAccount, amount: 300,
			modify: func(p *shared_domain.WithdrawParams) { p.AccountID = missingID }, wantErr: shared_domain.ErrAccountNotFound},
	}
	for _, a := range accountFixtures {
		for _, tt := range tests {
			t.Run(a.name+"/"+tt.name, func(t *testing.T) {
				db := setup(t)
				insertAccount(t, db, a, tt.state)
				before := rawDoc(t, db, "balanceAccount", a.id)
				p := withdrawParams(a, tt.amount)
				if tt.modify != nil {
					tt.modify(&p)
				}

				mov, err := newBalanceRepo(db).Withdraw(mexicoSession(a), p)

				if !errors.Is(err, tt.wantErr) || mov != nil {
					t.Fatalf("Withdraw() = (%+v, %v), se esperaba (nil, %v)", mov, err, tt.wantErr)
				}
				assertDocUnchanged(t, db, "balanceAccount", a.id, before)
				assertNoMovements(t, db)
			})
		}
	}
}

// withdrawConcurrently lanza n retiros iguales a la vez y devuelve el error de cada uno.
func withdrawConcurrently(ctx context.Context, repo shared_ports.BalancePort, p shared_domain.WithdrawParams, n int) []error {
	start := make(chan struct{})
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			<-start
			_, errs[i] = repo.Withdraw(ctx, p)
		})
	}
	close(start)
	wg.Wait()
	return errs
}

// Retiros simultáneos nunca dejan el saldo negativo ni pierden actualizaciones. CA-00.16
func TestWithdrawConcurrent(t *testing.T) {
	for _, a := range accountFixtures {
		t.Run(a.name+"/dos retiros de 600 sobre 1000", func(t *testing.T) {
			for round := range 5 {
				db := setup(t)
				insertAccount(t, db, a, activeAccount)

				errs := withdrawConcurrently(mexicoSession(a), newBalanceRepo(db), withdrawParams(a, 600), 2)

				succeeded, rejected := 0, 0
				for _, err := range errs {
					switch {
					case err == nil:
						succeeded++
					case errors.Is(err, shared_domain.ErrAccountWithdrawAmountMoreThanAvailable):
						rejected++
					default:
						t.Fatalf("ronda %d: error inesperado: %v", round, err)
					}
				}
				if succeeded != 1 || rejected != 1 {
					t.Fatalf("ronda %d: %d retiros bien y %d rechazados, se esperaba 1 y 1", round, succeeded, rejected)
				}
				account := readAccount(t, db, a.id)
				if account.Available != 400 || account.Global != 900 {
					t.Fatalf("ronda %d: available %v y global %v, se esperaba 400 y 900", round, account.Available, account.Global)
				}
				for _, c := range movementCollections {
					if n := countDocs(t, db, c); n != 1 {
						t.Fatalf("ronda %d: %s tiene %d documentos, se esperaba 1", round, c, n)
					}
				}
			}
		})

		t.Run(a.name+"/diez retiros de 100 sobre 1000", func(t *testing.T) {
			db := setup(t)
			insertAccount(t, db, a, activeAccount)

			errs := withdrawConcurrently(mexicoSession(a), newBalanceRepo(db), withdrawParams(a, 100), 10)

			for i, err := range errs {
				if err != nil {
					t.Fatalf("retiro %d: %v", i, err)
				}
			}
			account := readAccount(t, db, a.id)
			if account.Available != 0 || account.Global != 500 {
				t.Fatalf("available %v y global %v, se esperaba 0 y 500", account.Available, account.Global)
			}
			var balances []float64
			for _, d := range findAll[shared_domain.BalanceDetail](t, db, "balanceDetail") {
				balances = append(balances, d.Balance)
			}
			slices.Sort(balances)
			want := []float64{0, 100, 200, 300, 400, 500, 600, 700, 800, 900}
			if !slices.Equal(balances, want) {
				t.Fatalf("balances de los detalles %v, se esperaba %v", balances, want)
			}
		})
	}
}
