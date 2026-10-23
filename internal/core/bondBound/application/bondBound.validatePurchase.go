package bondBound_application

import (
	"context"
	"slices"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// purchaseOrder es una compra ya validada: quién compra, con qué cuenta, qué producto y cuántos bonos.
type purchaseOrder struct {
	buyer    bondBound_domain.Buyer
	account  *shared_domain.BalanceAccount
	object   *bondObject_domain.BondObject
	quantity int
}

// validatePurchase valida que el usuario de la sesión puede comprar quantity bonos del producto con la
// cuenta, sin escribir nada. Los ids y la cantidad ya vienen validados. El primer error que encuentra sigue
// este orden: cuenta propia, producto, privado autorizado, cuenta con saldo, cuenta activa, producto activo,
// saldo suficiente, venta iniciada y venta no terminada.
func (app *BondBoundApplication) validatePurchase(ctx context.Context, session *session_domain.UserSession, accountID, bondObjectID string, quantity int, now time.Time) (*purchaseOrder, error) {
	account, err := app.Balance.GetAccount(ctx, accountID, session.UserID)
	if err != nil {
		return nil, err
	}
	if account == nil || account.AgencyID != session.AgencyID {
		return nil, shared_domain.ErrAccountNotFound
	}

	object, err := app.BondObjects.FindVisibleForPurchase(ctx, bondObjectID)
	if err != nil {
		return nil, err
	}
	if object == nil {
		return nil, bondBound_domain.ErrBondObjectNotFound
	}

	total := shared_domain.Round2(shared_domain.Round2(object.Amount) * float64(quantity))
	switch {
	case object.IsPrivateBond && !slices.Contains(object.AuthorizedUserIDs, session.UserID):
		return nil, bondBound_domain.ErrBondPrivateNotAuthorized
	case account.Available <= 0:
		return nil, bondBound_domain.ErrAccountWithoutFunds
	case account.StatusID != shared_domain.BalanceAccountStatusActive:
		return nil, bondBound_domain.ErrAccountNotActive
	case !object.Active:
		return nil, bondBound_domain.ErrBondInactive
	case account.Available < total:
		return nil, bondBound_domain.ErrInsufficientAmount
	case object.SaleInitDate != nil && object.SaleInitDate.After(now):
		return nil, bondBound_domain.ErrBondNotAvailableByDate
	case object.SaleEndDate != nil && object.SaleEndDate.Before(now):
		return nil, bondBound_domain.ErrBondSaleEnded
	}

	return &purchaseOrder{
		buyer:    bondBound_domain.Buyer{UserID: session.UserID, AgencyID: session.AgencyID},
		account:  account,
		object:   object,
		quantity: quantity,
	}, nil
}

// snapshotOf devuelve los datos del producto que copia cada bono.
func snapshotOf(o *bondObject_domain.BondObject) bondBound_domain.ObjectSnapshot {
	return bondBound_domain.ObjectSnapshot{
		BondObjectID:              o.ID,
		BondTypeID:                o.BondTypeID,
		Key:                       o.Key,
		Amount:                    o.Amount,
		MaturityTimeInDays:        o.MaturityTimeInDays,
		MaturityRewardsPercentage: o.MaturityRewardsPercentage,
		EarlyCashTerm:             o.EarlyCashTerm,
	}
}
