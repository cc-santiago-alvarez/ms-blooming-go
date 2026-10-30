package bondBound_application_test

import (
	"errors"
	"testing"
	"time"

	bondBound_application "github.com/futurapps/ms-blooming-go/internal/core/bondBound/application"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// purchase es una compra de ownerID: el body, y la cuenta y el producto guardados (nil si no existe).
type purchase struct {
	params  bondBound_domain.PurchaseParams
	account *shared_domain.BalanceAccount
	object  *bondObject_domain.BondObject
}

// validPurchase es una compra que pasa todas las validaciones.
func validPurchase() *purchase {
	return &purchase{
		params:  bondBound_domain.PurchaseParams{AccountID: accountID, BondObjectID: objectID},
		account: ownAccount(),
		object:  purchasableObject(),
	}
}

// run guarda la cuenta y el producto de la compra y compra en la sesión de ownerID.
func (p *purchase) run() (*fakes, bondBound_domain.PurchaseResult, error) {
	f := purchaseFakes(p.account, p.object)
	result, err := f.app().Purchase(withSession(ownerID, -5), p.params)
	return f, result, err
}

// purchaseStep es una validación de la compra: cómo hacerla fallar y su error. body indica que falla el
// body, antes de leer la cuenta.
type purchaseStep struct {
	name string
	fail func(p *purchase)
	err  error
	body bool
}

// purchaseSteps son las validaciones de la compra en el orden en que se hacen.
func purchaseSteps() []purchaseStep {
	tomorrow := time.Now().Add(24 * time.Hour)
	yesterday := time.Now().Add(-24 * time.Hour)

	return []purchaseStep{
		{name: "sin accountId", fail: func(p *purchase) { p.params.AccountID = "" }, err: bondBound_domain.ErrAccountIDRequired, body: true},
		{name: "sin bondObjectId", fail: func(p *purchase) { p.params.BondObjectID = "" }, err: bondBound_domain.ErrBondObjectIDRequired, body: true},
		{name: "bondObjectId inválido", fail: func(p *purchase) { p.params.BondObjectID = "abc" }, err: shared_domain.ErrInvalidID, body: true},
		{name: "cuenta de otro usuario de la agencia", fail: func(p *purchase) { p.account.UserID = otherUserID }, err: shared_domain.ErrAccountNotFound},
		{name: "cuenta de otra agencia", fail: func(p *purchase) { p.account.AgencyID = agencyB }, err: shared_domain.ErrAccountNotFound},
		{name: "producto inexistente", fail: func(p *purchase) { p.object = nil }, err: bondBound_domain.ErrBondObjectNotFound},
		{name: "privado sin autorizar", fail: func(p *purchase) {
			p.object.IsPrivateBond = true
			p.object.AuthorizedUserIDs = []string{otherUserID}
		}, err: bondBound_domain.ErrBondPrivateNotAuthorized},
		{name: "available 0", fail: func(p *purchase) { p.account.Available = 0 }, err: bondBound_domain.ErrAccountWithoutFunds},
		{name: "cuenta con statusId 2", fail: func(p *purchase) { p.account.StatusID = "2" }, err: bondBound_domain.ErrAccountNotActive},
		{name: "producto inactivo", fail: func(p *purchase) { p.object.Active = false }, err: bondBound_domain.ErrBondInactive},
		{name: "available 500 con amount 1000", fail: func(p *purchase) { p.account.Available = 500 }, err: bondBound_domain.ErrInsufficientAmount},
		{name: "saleInitDate mañana", fail: func(p *purchase) { p.object.SaleInitDate = &tomorrow }, err: bondBound_domain.ErrBondNotAvailableByDate},
		{name: "saleEndDate ayer", fail: func(p *purchase) { p.object.SaleEndDate = &yesterday }, err: bondBound_domain.ErrBondSaleEnded},
	}
}

// Cada validación de la compra, sola, responde su error sin escribir nada; las del body no leen la
// cuenta. CA-04.2, CA-04.3, CA-04.4
func TestPurchaseValidations(t *testing.T) {
	for _, step := range purchaseSteps() {
		t.Run(step.name, func(t *testing.T) {
			p := validPurchase()
			step.fail(p)

			f, _, err := p.run()

			if !errors.Is(err, step.err) {
				t.Fatalf("Purchase = %v, se esperaba %v", err, step.err)
			}
			assertNothingWritten(t, f)
			if step.body && len(f.balance.reads) != 0 {
				t.Errorf("GetAccount se llamó con %+v con un body inválido", f.balance.reads)
			}
		})
	}
}

// Si fallan a la vez una validación y todas las que le siguen, la compra responde el error de la primera.
// CA-04.2
func TestPurchaseValidationOrder(t *testing.T) {
	steps := purchaseSteps()
	for i, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			p := validPurchase()
			// del último al primero, para que available 0 quede sobre available 500
			for j := len(steps) - 1; j >= i; j-- {
				steps[j].fail(p)
			}

			f, _, err := p.run()

			if !errors.Is(err, step.err) {
				t.Fatalf("Purchase = %v, se esperaba %v", err, step.err)
			}
			assertNothingWritten(t, f)
		})
	}
}

// Un producto público y activo de otra agencia se compra, y uno privado también si el usuario está
// autorizado: el bono es del usuario y la agencia de la sesión, y se cobra su monto. CA-04.4
func TestPurchaseAllowed(t *testing.T) {
	tests := []struct {
		name   string
		modify func(o *bondObject_domain.BondObject)
	}{
		{name: "público de otra agencia"},
		{name: "privado con el usuario autorizado", modify: func(o *bondObject_domain.BondObject) {
			o.IsPrivateBond = true
			o.AuthorizedUserIDs = []string{otherUserID, ownerID}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validPurchase()
			if tt.modify != nil {
				tt.modify(p.object)
			}

			f, result, err := p.run()

			if err != nil {
				t.Fatalf("Purchase: %v", err)
			}
			if len(f.repo.inserted) != 1 {
				t.Fatalf("se insertaron %d bonos, se esperaba 1", len(f.repo.inserted))
			}
			b := f.repo.inserted[0]
			if b.UserID != ownerID || b.AgencyID != agencyA || b.BondObjectID != objectID || b.Key != "BP12-1" {
				t.Errorf("bono de %q en %q del producto %q con clave %q, se esperaba de %q en %q del producto %q con clave \"BP12-1\"",
					b.UserID, b.AgencyID, b.BondObjectID, b.Key, ownerID, agencyA, objectID)
			}
			want := shared_domain.WithdrawParams{
				AccountID:     accountID,
				UserID:        ownerID,
				AgencyID:      agencyA,
				DescriptionID: shared_domain.BalanceDescriptionBondMD,
				Amount:        1000,
				Comment:       `Compra de Bono "BP12-1" (` + b.ID + `)`,
			}
			if len(f.balance.withdrawals) != 1 || f.balance.withdrawals[0] != want {
				t.Errorf("retiros %+v, se esperaba uno %+v", f.balance.withdrawals, want)
			}
			if result.BondBoundID != b.ID || result.Key != "BP12-1" || result.AccountAvailable != 4000 {
				t.Errorf("Purchase devolvió %+v, se esperaba el bono %s con clave BP12-1 y 4000 disponibles", result, b.ID)
			}
		})
	}
}

// Una compra individual o masiva con la cuenta de otro usuario de la misma agencia es ErrAccountNotFound:
// la cuenta se busca con el usuario de la sesión y su saldo no cambia. CA-04.3
func TestPurchaseForeignAccount(t *testing.T) {
	ctx := withSession(ownerID, -5)
	params := bondBound_domain.PurchaseParams{AccountID: accountID, BondObjectID: objectID}

	tests := []struct {
		name     string
		purchase func(f *fakes) error
	}{
		{name: "compra individual", purchase: func(f *fakes) error {
			_, err := f.app().Purchase(ctx, params)
			return err
		}},
		{name: "compra masiva", purchase: func(f *fakes) error {
			_, err := f.app().BulkPurchase(ctx, bondBound_domain.BulkPurchaseParams{PurchaseParams: params, Quantity: 2})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := ownAccount()
			account.UserID = otherUserID
			f := purchaseFakes(account, purchasableObject())

			err := tt.purchase(f)

			if !errors.Is(err, shared_domain.ErrAccountNotFound) {
				t.Fatalf("la compra = %v, se esperaba ErrAccountNotFound", err)
			}
			want := []accountRead{{accountID: accountID, userID: ownerID}}
			if len(f.balance.reads) != 1 || f.balance.reads[0] != want[0] {
				t.Errorf("GetAccount se llamó con %+v, se esperaba %+v", f.balance.reads, want)
			}
			if available := f.balance.accounts[accountID].Available; available != 5000 {
				t.Errorf("la cuenta ajena quedó con %v disponibles, se esperaba 5000", available)
			}
			assertNothingWritten(t, f)
		})
	}
}

// La compra masiva exige una cantidad entre 1 y el tope antes de leer la cuenta, y valida el saldo contra
// el total. CA-04.6
func TestBulkPurchaseValidations(t *testing.T) {
	tests := []struct {
		name      string
		quantity  int
		available float64
		err       error
	}{
		{name: "quantity 0", quantity: 0, available: 5000, err: bondBound_domain.ErrInvalidQuantity},
		{name: "quantity negativa", quantity: -1, available: 5000, err: bondBound_domain.ErrInvalidQuantity},
		{name: "quantity sobre el tope", quantity: maxQuantity + 1, available: 10000, err: bondBound_domain.ErrInvalidQuantity},
		{name: "available 2500 con 3 de 1000", quantity: 3, available: 2500, err: bondBound_domain.ErrInsufficientAmount},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := ownAccount()
			account.Available = tt.available
			f := purchaseFakes(account, purchasableObject())

			_, err := f.app().BulkPurchase(withSession(ownerID, -5), bondBound_domain.BulkPurchaseParams{
				PurchaseParams: bondBound_domain.PurchaseParams{AccountID: accountID, BondObjectID: objectID},
				Quantity:       tt.quantity,
			})

			if !errors.Is(err, tt.err) {
				t.Fatalf("BulkPurchase = %v, se esperaba %v", err, tt.err)
			}
			assertNothingWritten(t, f)
			if errors.Is(tt.err, bondBound_domain.ErrInvalidQuantity) && len(f.balance.reads) != 0 {
				t.Errorf("GetAccount se llamó con %+v con una cantidad inválida", f.balance.reads)
			}
		})
	}
}

// Una compra masiva del tope, con el saldo justo, crea todos los bonos con un cobro cada uno. CA-04.6
func TestBulkPurchaseMaxQuantity(t *testing.T) {
	f := purchaseFakes(ownAccount(), purchasableObject())

	result, err := f.app().BulkPurchase(withSession(ownerID, -5), bondBound_domain.BulkPurchaseParams{
		PurchaseParams: bondBound_domain.PurchaseParams{AccountID: accountID, BondObjectID: objectID},
		Quantity:       maxQuantity,
	})

	if err != nil {
		t.Fatalf("BulkPurchase: %v", err)
	}
	if len(f.repo.inserted) != maxQuantity || len(f.balance.withdrawals) != maxQuantity || len(result.Items) != maxQuantity {
		t.Fatalf("%d bonos, %d retiros y %d items, se esperaban %d de cada uno",
			len(f.repo.inserted), len(f.balance.withdrawals), len(result.Items), maxQuantity)
	}
	if result.Total != 5000 || result.AccountAvailable != 0 {
		t.Errorf("total %v y disponible %v, se esperaban 5000 y 0", result.Total, result.AccountAvailable)
	}
}

// Con la compra apagada, la individual y la masiva responden ErrTradingDisabled antes de validar el body o
// leer la cuenta, y no escriben nada ni avanzan el contador; encendida, la misma compra se hace. CA-04.15
func TestPurchaseTradingDisabled(t *testing.T) {
	ctx := withSession(ownerID, -5)
	valid := bondBound_domain.PurchaseParams{AccountID: accountID, BondObjectID: objectID}
	tests := []struct {
		name string
		buy  func(app *bondBound_application.BondBoundApplication) error
	}{
		{name: "compra", buy: func(app *bondBound_application.BondBoundApplication) error {
			_, err := app.Purchase(ctx, valid)
			return err
		}},
		{name: "compra sin accountId", buy: func(app *bondBound_application.BondBoundApplication) error {
			_, err := app.Purchase(ctx, bondBound_domain.PurchaseParams{BondObjectID: objectID})
			return err
		}},
		{name: "compra masiva", buy: func(app *bondBound_application.BondBoundApplication) error {
			_, err := app.BulkPurchase(ctx, bondBound_domain.BulkPurchaseParams{PurchaseParams: valid, Quantity: 3})
			return err
		}},
		{name: "compra masiva con quantity 0", buy: func(app *bondBound_application.BondBoundApplication) error {
			_, err := app.BulkPurchase(ctx, bondBound_domain.BulkPurchaseParams{PurchaseParams: valid})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := purchaseFakes(ownAccount(), purchasableObject())
			app := f.app()
			app.TradingEnabled = false

			err := tt.buy(app)

			if !errors.Is(err, bondBound_domain.ErrTradingDisabled) {
				t.Fatalf("la compra = %v, se esperaba ErrTradingDisabled", err)
			}
			assertNothingWritten(t, f)
			if len(f.balance.reads) != 0 {
				t.Errorf("GetAccount se llamó con %+v con la compra apagada", f.balance.reads)
			}
		})
	}

	t.Run("encendida", func(t *testing.T) {
		f := purchaseFakes(ownAccount(), purchasableObject())

		if _, err := f.app().Purchase(ctx, valid); err != nil {
			t.Fatalf("Purchase: %v", err)
		}
		if len(f.repo.inserted) != 1 || len(f.keys.reserved) != 1 {
			t.Errorf("%d bonos y reservas %v, se esperaba un bono y una reserva", len(f.repo.inserted), f.keys.reserved)
		}
	})
}
