//go:build integration

package shared_mongocc_repository_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const depositComment = "Venta del bono AB-3"

// depositAccount es una cuenta activa con 700 disponibles.
var depositAccount = accountState{statusID: "1", available: 700, global: 1000, deferred: 200}

// depositParams arma el depósito de una venta de bonos por amount.
func depositParams(a accountFixture, amount float64) shared_domain.DepositParams {
	return shared_domain.DepositParams{
		AccountID:     a.id,
		UserID:        a.userID,
		AgencyID:      a.agencyID,
		DescriptionID: shared_domain.BalanceDescriptionBondSaleMC,
		Amount:        amount,
		Comment:       depositComment,
	}
}

// Un depósito suma a available y global y registra el movimiento con effectiveDate null; el estado
// de la cuenta no se revisa. CA-00.18
func TestDeposit(t *testing.T) {
	ctx := context.Background()

	for _, a := range accountFixtures {
		t.Run(a.name, func(t *testing.T) {
			db := setup(t)
			insertAccount(t, db, a, depositAccount)

			mov, err := newBalanceRepo(db).Deposit(ctx, depositParams(a, 500))

			if err != nil {
				t.Fatalf("Deposit() = %v", err)
			}

			account := readAccount(t, db, a.id)
			if account.Available != 1200 || account.Global != 1500 || account.Deferred != 200 {
				t.Fatalf("cuenta available %v, global %v, deferred %v; se esperaba 1200, 1500, 200",
					account.Available, account.Global, account.Deferred)
			}

			details := findAll[shared_domain.BalanceDetail](t, db, "balanceDetail")
			if len(details) != 1 {
				t.Fatalf("hay %d balanceDetail, se esperaba 1", len(details))
			}
			d := details[0]
			assertUUIDv4(t, d.ID)
			assertFieldType(t, rawDoc(t, db, "balanceDetail", d.ID), "effectiveDate", bson.TypeNull)
			if !referencePattern.MatchString(d.Reference) {
				t.Fatalf("reference %q no tiene 16 alfanuméricos", d.Reference)
			}
			wantDetail := shared_domain.BalanceDetail{
				ID:               d.ID,
				BalanceAccountID: a.id,
				OperationTypeID:  "2",
				Reference:        d.Reference,
				DescriptionID:    "8",
				Amount:           500,
				Balance:          1200,
				AgencyID:         a.agencyID,
				UserID:           a.userID,
				Comment:          depositComment,
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
			if h.TypeID != "2" || h.Text != "BALANCE_ACCOUNT_DEPOSIT" || !reflect.DeepEqual(h.Params, []string{d.ID}) ||
				h.BalanceAccountID != a.id || h.UserID != a.userID || h.AgencyID != a.agencyID {
				t.Fatalf("balanceHistory: %+v", h)
			}

			operations := findAll[shared_domain.BalanceOperation](t, db, "balanceOperation")
			if len(operations) != 1 {
				t.Fatalf("hay %d balanceOperation, se esperaba 1", len(operations))
			}
			o := operations[0]
			if o.TypeID != "1" || o.CategoryID != "2" || !reflect.DeepEqual(o.Details, []string{a.id, d.ID}) ||
				o.Concept != depositComment || o.UserID != a.userID || o.AgencyID != a.agencyID {
				t.Fatalf("balanceOperation: %+v", o)
			}

			wantMov := &shared_domain.BalanceMovement{
				DetailID:  d.ID,
				Reference: d.Reference,
				Amount:    500,
				Balance:   1200,
				Available: 1200,
			}
			if !reflect.DeepEqual(mov, wantMov) {
				t.Fatalf("Deposit() = %+v, se esperaba %+v", mov, wantMov)
			}
		})

		t.Run(a.name+"/cuenta suspendida", func(t *testing.T) {
			db := setup(t)
			suspended := depositAccount
			suspended.statusID = "2"
			insertAccount(t, db, a, suspended)

			if _, err := newBalanceRepo(db).Deposit(ctx, depositParams(a, 500)); err != nil {
				t.Fatalf("Deposit() = %v, el estado de la cuenta no se revisa", err)
			}
			if got := readAccount(t, db, a.id).Available; got != 1200 {
				t.Fatalf("available %v, se esperaba 1200", got)
			}
		})
	}
}

// Un depósito con monto inválido o sobre una cuenta ajena no escribe nada. CA-00.33
func TestDepositRejected(t *testing.T) {
	ctx := context.Background()
	removed := depositAccount
	removed.isRemove = true

	tests := []struct {
		name    string
		state   accountState
		amount  float64
		modify  func(p *shared_domain.DepositParams)
		wantErr error
	}{
		{name: "monto cero", state: depositAccount, amount: 0, wantErr: shared_domain.ErrAccountDepositAmountInvalid},
		{name: "monto negativo", state: depositAccount, amount: -10, wantErr: shared_domain.ErrAccountDepositAmountInvalid},
		{name: "monto que redondea a cero", state: depositAccount, amount: 0.004, wantErr: shared_domain.ErrAccountDepositAmountInvalid},
		{name: "NaN", state: depositAccount, amount: math.NaN(), wantErr: shared_domain.ErrAccountDepositAmountInvalid},
		{name: "infinito", state: depositAccount, amount: math.Inf(1), wantErr: shared_domain.ErrAccountDepositAmountInvalid},
		{name: "cuenta de otro usuario", state: depositAccount, amount: 500,
			modify: func(p *shared_domain.DepositParams) { p.UserID = otherUserID }, wantErr: shared_domain.ErrAccountNotFound},
		{name: "cuenta borrada", state: removed, amount: 500, wantErr: shared_domain.ErrAccountNotFound},
		{name: "cuenta inexistente", state: depositAccount, amount: 500,
			modify: func(p *shared_domain.DepositParams) { p.AccountID = missingID }, wantErr: shared_domain.ErrAccountNotFound},
	}
	for _, a := range accountFixtures {
		for _, tt := range tests {
			t.Run(a.name+"/"+tt.name, func(t *testing.T) {
				db := setup(t)
				insertAccount(t, db, a, tt.state)
				before := rawDoc(t, db, "balanceAccount", a.id)
				p := depositParams(a, tt.amount)
				if tt.modify != nil {
					tt.modify(&p)
				}

				mov, err := newBalanceRepo(db).Deposit(ctx, p)

				if !errors.Is(err, tt.wantErr) || mov != nil {
					t.Fatalf("Deposit() = (%+v, %v), se esperaba (nil, %v)", mov, err, tt.wantErr)
				}
				assertDocUnchanged(t, db, "balanceAccount", a.id, before)
				assertNoMovements(t, db)
			})
		}
	}
}
