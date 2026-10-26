package bondBound_application_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Cuentas seleccionadas de otros usuarios.
const (
	adminAccountID = "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d"
	otherAccountID = "2b3c4d5e-6f7a-4b8c-9d0e-1f2a3b4c5d6e"
)

// withSelectedAccount agrega a f una cuenta activa de userID con 5000 disponibles, seleccionada en su setting.
func withSelectedAccount(f *fakes, userID, id string) {
	account := ownAccount()
	account.ID, account.UserID = id, userID
	f.balance.accounts[id] = account
	f.userSettings.settings[userID] = &bondUserSetting_domain.BondUserSetting{UserID: userID, AgencyID: agencyA, SelectedAccountID: &id}
}

// saleFakes arma las dependencias de la venta: el bono, la cuenta de ownerID seleccionada en su setting y el
// producto del bono.
func saleFakes(b *bondBound_domain.BondBound) *fakes {
	f := newFakes(b)
	withSelectedAccount(f, ownerID, accountID)
	object := purchasableObject()
	f.objects.objects[object.ID] = object
	return f
}

// assertNotSold verifica que la venta no escribió nada: el bono guardado sigue en el estado de want y todas
// las cuentas conservan sus 5000 disponibles.
func assertNotSold(t *testing.T, f *fakes, want *bondBound_domain.BondBound) {
	t.Helper()
	assertNothingWritten(t, f)
	if b := f.repo.bounds[want.ID]; b.StatusID != want.StatusID || (b.SoldDate == nil) != (want.SoldDate == nil) ||
		(b.TookDate == nil) != (want.TookDate == nil) {
		t.Errorf("el bono quedó en %q con soldDate %v y tookDate %v, se esperaba %q con soldDate %v y tookDate %v",
			b.StatusID, b.SoldDate, b.TookDate, want.StatusID, want.SoldDate, want.TookDate)
	}
	for id, a := range f.balance.accounts {
		if a.Available != 5000 {
			t.Errorf("la cuenta %s quedó con %v disponibles, se esperaba 5000", id, a.Available)
		}
	}
}

// notSellable es un bono que no se puede vender: cómo dejarlo así y su error.
type notSellable struct {
	name   string
	modify func(b *bondBound_domain.BondBound)
	err    error
}

// notSellableCases son los bonos que no se cotizan ni se venden, por su estado o por sus fechas.
func notSellableCases() []notSellable {
	at := time.Now()
	status := func(s bondBound_domain.BondBoundStatus) func(b *bondBound_domain.BondBound) {
		return func(b *bondBound_domain.BondBound) { b.StatusID = s }
	}

	return []notSellable{
		{name: "bloqueado", modify: status(bondBound_domain.StatusBlocked), err: bondBound_domain.ErrBondStatusNotSellable},
		{name: "desligado", modify: status(bondBound_domain.StatusUnbound), err: bondBound_domain.ErrBondStatusNotSellable},
		{name: "estado desconocido", modify: status("9"), err: bondBound_domain.ErrBondStatusNotSellable},
		{name: "vendido", modify: status(bondBound_domain.StatusSold), err: bondBound_domain.ErrBondAlreadySold},
		{name: "canjeado", modify: status(bondBound_domain.StatusTook), err: bondBound_domain.ErrBondAlreadyTook},
		{name: "iniciado con soldDate", modify: func(b *bondBound_domain.BondBound) { b.SoldDate = &at }, err: bondBound_domain.ErrBondAlreadySold},
		{name: "madurado con tookDate", modify: func(b *bondBound_domain.BondBound) {
			b.StatusID = bondBound_domain.StatusFinalized
			b.TookDate = &at
		}, err: bondBound_domain.ErrBondAlreadyTook},
	}
}

// Un bono bloqueado, desligado o con un estado desconocido es ErrBondStatusNotSellable, y uno vendido o
// canjeado, por estado o por fecha, ErrBondAlreadySold o ErrBondAlreadyTook. En ninguno se escribe nada ni se
// lee la cuenta destino. CA-05.9
func TestSaleNotSellable(t *testing.T) {
	for _, tt := range notSellableCases() {
		t.Run(tt.name, func(t *testing.T) {
			b := boundOnDay(50)
			tt.modify(b)
			f := saleFakes(b)

			_, err := f.app().Sale(withSession(ownerID, -5), boundID)

			if !errors.Is(err, tt.err) {
				t.Fatalf("Sale = %v, se esperaba %v", err, tt.err)
			}
			assertNotSold(t, f, b)
			if len(f.userSettings.reads) != 0 {
				t.Errorf("se leyó el setting de %v", f.userSettings.reads)
			}
		})
	}
}

// Solo el dueño vende su bono: cambia el estado, acredita la cotización en su cuenta seleccionada y registra
// el historial. Otro usuario, o un admin, recibe ErrForbidden y no se acredita nada en ninguna cuenta, aunque
// tengan una seleccionada. CA-05.10
func TestSaleOwnerOnly(t *testing.T) {
	t.Run("dueño", func(t *testing.T) {
		b := boundOnDay(50)
		f := saleFakes(b)
		quote, err := bondBound_domain.QuoteSale(*b, time.Now(), -5)
		if err != nil {
			t.Fatalf("QuoteSale: %v", err)
		}

		result, err := f.app().Sale(withSession(ownerID, -5), boundID)

		if err != nil {
			t.Fatalf("Sale: %v", err)
		}
		if len(f.repo.changes) != 1 {
			t.Fatalf("cambios de estado %+v, se esperaba uno", f.repo.changes)
		}
		if c := f.repo.changes[0]; c.id != boundID || c.change.From != bondBound_domain.StatusInitiated ||
			c.change.To != bondBound_domain.StatusSold || c.change.SoldDate == nil || c.change.TookDate != nil {
			t.Errorf("cambio de estado %+v, se esperaba %s de '1' a '5' con soldDate", c, boundID)
		}
		wantDeposit := shared_domain.DepositParams{
			AccountID:     accountID,
			UserID:        ownerID,
			AgencyID:      agencyA,
			DescriptionID: shared_domain.BalanceDescriptionBondSaleMC,
			Amount:        quote.Amount,
			Comment:       `Bono "BP12-1" vendido.`,
		}
		if len(f.balance.deposits) != 1 || f.balance.deposits[0] != wantDeposit {
			t.Errorf("depósitos %+v, se esperaba uno %+v", f.balance.deposits, wantDeposit)
		}
		if len(f.history.added) != 1 || f.history.added[0].RelatedID != boundID ||
			!slices.Equal(f.history.added[0].Params, []string{"1", "5"}) {
			t.Errorf("historial %+v, se esperaba el cambio de '1' a '5' de %s", f.history.added, boundID)
		}
		want := bondBound_domain.SaleResult{
			BondBoundID: boundID,
			StatusID:    bondBound_domain.StatusSold,
			Amount:      quote.Amount,
			Profit:      quote.Profit,
			AccountID:   accountID,
			Available:   shared_domain.Round2(5000 + quote.Amount),
		}
		if result != want {
			t.Errorf("Sale devolvió %+v, se esperaba %+v", result, want)
		}
	})

	owner := withSession(ownerID, -5)
	rejected := []struct {
		name string
		ctx  context.Context
		id   string
		err  error
	}{
		{name: "otro usuario", ctx: withSession(otherUserID, -5), id: boundID, err: shared_domain.ErrForbidden},
		{name: "admin que no es dueño", ctx: withAdmin(), id: boundID, err: shared_domain.ErrForbidden},
		{name: "inexistente", ctx: owner, id: missingID, err: bondBound_domain.ErrBondBoundNotFound},
		{name: "id inválido", ctx: owner, id: "abc", err: shared_domain.ErrInvalidID},
	}
	for _, tt := range rejected {
		t.Run(tt.name, func(t *testing.T) {
			b := boundOnDay(50)
			f := saleFakes(b)
			withSelectedAccount(f, adminID, adminAccountID)
			withSelectedAccount(f, otherUserID, otherAccountID)

			_, err := f.app().Sale(tt.ctx, tt.id)

			if !errors.Is(err, tt.err) {
				t.Fatalf("Sale = %v, se esperaba %v", err, tt.err)
			}
			assertNotSold(t, f, b)
			if len(f.userSettings.reads) != 0 || len(f.balance.reads) != 0 {
				t.Errorf("se leyó la cuenta destino: settings de %v y cuentas %+v", f.userSettings.reads, f.balance.reads)
			}
			if tt.id == "abc" && len(f.repo.reads) != 0 {
				t.Errorf("FindOneByID se llamó con %v con un id inválido", f.repo.reads)
			}
		})
	}
}

// Antes del día mínimo de retiro la venta es ErrInvalidWithdrawDate: el bono sigue iniciado, la cuenta conserva
// su saldo y la cuenta destino ni se lee. CA-05.11
func TestSaleBeforeWithdrawDate(t *testing.T) {
	b := boundOnDay(10)
	b.EarlyCashTerm.InitialDayForWithdraw = 30
	f := saleFakes(b)

	_, err := f.app().Sale(withSession(ownerID, -5), boundID)

	if !errors.Is(err, bondBound_domain.ErrInvalidWithdrawDate) {
		t.Fatalf("Sale = %v, se esperaba ErrInvalidWithdrawDate", err)
	}
	assertNotSold(t, f, b)
	if len(f.userSettings.reads) != 0 {
		t.Errorf("se leyó el setting de %v", f.userSettings.reads)
	}
}

// Sin setting, o con selectedAccountId nulo o vacío, la venta es ErrUserSettingAccountRequired; una cuenta
// seleccionada que no existe o que no es del usuario, ErrAccountNotFound. En ninguno se escribe nada.
// CA-05.12
func TestSaleDestinationAccount(t *testing.T) {
	empty := ""

	tests := []struct {
		name        string
		modify      func(f *fakes)
		err         error
		accountRead bool
	}{
		{name: "sin setting", modify: func(f *fakes) { delete(f.userSettings.settings, ownerID) },
			err: bondBound_domain.ErrUserSettingAccountRequired},
		{name: "selectedAccountId nulo", modify: func(f *fakes) { f.userSettings.settings[ownerID].SelectedAccountID = nil },
			err: bondBound_domain.ErrUserSettingAccountRequired},
		{name: "selectedAccountId vacío", modify: func(f *fakes) { f.userSettings.settings[ownerID].SelectedAccountID = &empty },
			err: bondBound_domain.ErrUserSettingAccountRequired},
		{name: "cuenta inexistente", modify: func(f *fakes) { delete(f.balance.accounts, accountID) },
			err: shared_domain.ErrAccountNotFound, accountRead: true},
		{name: "cuenta de otro usuario", modify: func(f *fakes) { f.balance.accounts[accountID].UserID = otherUserID },
			err: shared_domain.ErrAccountNotFound, accountRead: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := boundOnDay(50)
			f := saleFakes(b)
			tt.modify(f)

			_, err := f.app().Sale(withSession(ownerID, -5), boundID)

			if !errors.Is(err, tt.err) {
				t.Fatalf("Sale = %v, se esperaba %v", err, tt.err)
			}
			assertNotSold(t, f, b)
			if !slices.Equal(f.userSettings.reads, []string{ownerID}) {
				t.Errorf("se leyó el setting de %v, se esperaba el de %s", f.userSettings.reads, ownerID)
			}
			var wantReads []accountRead
			if tt.accountRead {
				wantReads = []accountRead{{accountID: accountID, userID: ownerID}}
			}
			if !slices.Equal(f.balance.reads, wantReads) {
				t.Errorf("GetAccount se llamó con %+v, se esperaba %+v", f.balance.reads, wantReads)
			}
		})
	}
}
