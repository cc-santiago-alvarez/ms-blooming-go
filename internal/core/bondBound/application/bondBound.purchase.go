package bondBound_application

import (
	"context"
	"fmt"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// tokenLength es la cantidad de caracteres del token de un bono.
const tokenLength = 6

// Purchase compra un bono del producto con una cuenta del usuario de la sesión: valida sin escribir nada y,
// en una transacción, crea el bono, cobra y registra el historial. Devuelve el bono creado y el saldo que le
// queda a la cuenta.
func (app *BondBoundApplication) Purchase(ctx context.Context, p bondBound_domain.PurchaseParams) (bondBound_domain.PurchaseResult, error) {
	if !app.TradingEnabled {
		return bondBound_domain.PurchaseResult{}, bondBound_domain.ErrTradingDisabled
	}
	p.Normalize()
	if err := p.Validation(); err != nil {
		return bondBound_domain.PurchaseResult{}, err
	}
	session, err := sessionUser(ctx)
	if err != nil {
		return bondBound_domain.PurchaseResult{}, err
	}

	now := shared_domain.Now()
	order, err := app.validatePurchase(ctx, session, p.AccountID, p.BondObjectID, 1, now)
	if err != nil {
		return bondBound_domain.PurchaseResult{}, err
	}
	bounds, movements, err := app.createBounds(ctx, order, now)
	if err != nil {
		return bondBound_domain.PurchaseResult{}, err
	}
	app.notifyPurchase(ctx, session, order, bounds, movements, now)

	return bondBound_domain.PurchaseResult{
		BondBoundID:      bounds[0].ID,
		Key:              bounds[0].Key,
		AccountAvailable: shared_domain.Round2(movements[len(movements)-1].Available),
	}, nil
}

// createBounds crea en una transacción los bonos de la orden: reserva sus claves, los guarda y, por cada
// uno, cobra su monto de la cuenta y registra el historial del bono y del producto. Si algo falla no queda
// nada escrito. Devuelve los bonos y sus movimientos de saldo, en el mismo orden.
func (app *BondBoundApplication) createBounds(ctx context.Context, order *purchaseOrder, now time.Time) ([]*bondBound_domain.BondBound, []*shared_domain.BalanceMovement, error) {
	snapshot := snapshotOf(order.object)
	var (
		bounds    []*bondBound_domain.BondBound
		movements []*shared_domain.BalanceMovement
	)

	err := app.Tx.RunInTransaction(ctx, func(txCtx context.Context) error {
		// el callback puede reintentarse: todo se arma de cero en cada intento
		start, err := app.KeyCounter.Reserve(txCtx, order.object.BondTypeID, order.quantity)
		if err != nil {
			return err
		}

		bounds = make([]*bondBound_domain.BondBound, order.quantity)
		for i := range bounds {
			bounds[i] = bondBound_domain.NewBoundFromObject(snapshot, order.buyer, start+i, app.Utils.RandomAlnum(tokenLength), now)
		}
		if err := app.Repository.InsertMany(txCtx, bounds); err != nil {
			return err
		}

		movements = make([]*shared_domain.BalanceMovement, 0, len(bounds))
		for _, b := range bounds {
			mov, err := app.Balance.Withdraw(txCtx, shared_domain.WithdrawParams{
				AccountID:     order.account.ID,
				UserID:        order.buyer.UserID,
				AgencyID:      order.buyer.AgencyID,
				DescriptionID: shared_domain.BalanceDescriptionBondMD,
				Amount:        b.Amount,
				Comment:       fmt.Sprintf("Compra de Bono \"%s\" (%s)", b.Key, b.ID),
			})
			if err != nil {
				return err
			}
			movements = append(movements, mov)

			if err := app.History.Add(txCtx, bondHistory_domain.AddParams{
				RelatedID: b.ID,
				TypeID:    bondHistory_domain.HistoryBondBound,
				Text:      shared_domain.TextBondBoundSold,
			}); err != nil {
				return err
			}
			if err := app.History.Add(txCtx, bondHistory_domain.AddParams{
				RelatedID: order.object.ID,
				TypeID:    bondHistory_domain.HistoryBond,
				Text:      shared_domain.TextBondSold,
				Params:    []string{b.ID},
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return bounds, movements, nil
}
