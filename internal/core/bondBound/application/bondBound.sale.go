package bondBound_application

import (
	"context"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Sale vende el bono iniciado o canjea el madurado del usuario de la sesión: valida y cotiza sin escribir
// nada y, en una transacción, cambia su estado, acredita la cotización en la cuenta destino y registra el
// historial; después avisa al dueño. Devuelve lo acreditado y el saldo que queda en la cuenta.
func (app *BondBoundApplication) Sale(ctx context.Context, id string) (bondBound_domain.SaleResult, error) {
	if !shared_domain.IsValidID(id) {
		return bondBound_domain.SaleResult{}, shared_domain.ErrInvalidID
	}
	session, err := sessionUser(ctx)
	if err != nil {
		return bondBound_domain.SaleResult{}, err
	}

	b, err := app.Repository.FindOneByID(ctx, id)
	if err != nil {
		return bondBound_domain.SaleResult{}, err
	}
	if b == nil {
		return bondBound_domain.SaleResult{}, bondBound_domain.ErrBondBoundNotFound
	}
	// ni un admin vende bonos ajenos
	if b.UserID != session.UserID {
		return bondBound_domain.SaleResult{}, shared_domain.ErrForbidden
	}

	transition, err := bondBound_domain.SaleTransitionFor(*b)
	if err != nil {
		return bondBound_domain.SaleResult{}, err
	}
	now := shared_domain.Now()
	quote, err := bondBound_domain.QuoteSale(*b, now, session_domain.UtcDiff(ctx))
	if err != nil {
		return bondBound_domain.SaleResult{}, err
	}
	account, err := app.saleAccount(ctx, session.UserID)
	if err != nil {
		return bondBound_domain.SaleResult{}, err
	}

	mov, err := app.settleSale(ctx, session, b, transition, quote, account, now)
	if err != nil {
		return bondBound_domain.SaleResult{}, err
	}
	app.notifySale(ctx, session, b, transition, quote, account, mov)

	return bondBound_domain.SaleResult{
		BondBoundID: b.ID,
		StatusID:    transition.To,
		Amount:      quote.Amount,
		Profit:      quote.Profit,
		AccountID:   account.ID,
		Available:   shared_domain.Round2(mov.Available),
	}, nil
}

// saleAccount devuelve la cuenta destino del usuario: la seleccionada en su setting. Sin setting o sin
// cuenta seleccionada es ErrUserSettingAccountRequired, y si la cuenta no existe o no es suya,
// ErrAccountNotFound.
func (app *BondBoundApplication) saleAccount(ctx context.Context, userID string) (*shared_domain.BalanceAccount, error) {
	setting, err := app.UserSettings.GetByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if setting == nil || setting.SelectedAccountID == nil || *setting.SelectedAccountID == "" {
		return nil, bondBound_domain.ErrUserSettingAccountRequired
	}

	account, err := app.Balance.GetAccount(ctx, *setting.SelectedAccountID, userID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, shared_domain.ErrAccountNotFound
	}
	return account, nil
}

// settleSale aplica la venta en una transacción: cambia el estado del bono solo si nadie lo cambió antes,
// acredita la cotización en la cuenta y registra el cambio de estado en el historial. Si otra venta ganó la
// carrera es ErrBondAlreadySold; si algo falla no queda nada escrito. Devuelve el movimiento del depósito.
func (app *BondBoundApplication) settleSale(ctx context.Context, session *session_domain.UserSession, b *bondBound_domain.BondBound,
	t bondBound_domain.SaleTransition, quote bondBound_domain.SaleProjection, account *shared_domain.BalanceAccount, now time.Time,
) (*shared_domain.BalanceMovement, error) {
	var mov *shared_domain.BalanceMovement

	err := app.Tx.RunInTransaction(ctx, func(txCtx context.Context) error {
		soldDate, tookDate := t.Dates(now)
		changed, err := app.Repository.UpdateStatusIf(txCtx, b.ID, bondBound_ports.StatusChange{
			From:     t.From,
			To:       t.To,
			SoldDate: soldDate,
			TookDate: tookDate,
		})
		if err != nil {
			return err
		}
		if !changed {
			return bondBound_domain.ErrBondAlreadySold
		}

		mov, err = app.Balance.Deposit(txCtx, shared_domain.DepositParams{
			AccountID:     account.ID,
			UserID:        session.UserID,
			AgencyID:      session.AgencyID,
			DescriptionID: t.DescriptionID,
			Amount:        quote.Amount,
			Comment:       t.Comment,
		})
		if err != nil {
			return err
		}

		return app.History.Add(txCtx, bondHistory_domain.AddParams{
			RelatedID: b.ID,
			TypeID:    bondHistory_domain.HistoryBondBound,
			Text:      shared_domain.TextBondBoundStatusChanged,
			Params:    []string{string(t.From), string(t.To)},
		})
	})
	if err != nil {
		return nil, err
	}
	return mov, nil
}
