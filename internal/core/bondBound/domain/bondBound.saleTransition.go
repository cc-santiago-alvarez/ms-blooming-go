package bondBound_domain

import (
	"fmt"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// SaleTransition es la salida de un bono: el cambio de estado y el depósito que lo acompaña.
type SaleTransition struct {
	From          BondBoundStatus
	To            BondBoundStatus
	DescriptionID string // descriptionId del depósito
	Comment       string // comment del depósito
}

// SaleTransitionFor devuelve la venta de un bono iniciado o el canje de uno madurado. Un bono vendido (por
// soldDate o por estado) es ErrBondAlreadySold, uno canjeado, ErrBondAlreadyTook, y cualquier otro estado,
// ErrBondStatusNotSellable.
func SaleTransitionFor(bound BondBound) (SaleTransition, error) {
	// las fechas mandan sobre el statusId
	switch {
	case bound.SoldDate != nil:
		return SaleTransition{}, ErrBondAlreadySold
	case bound.TookDate != nil:
		return SaleTransition{}, ErrBondAlreadyTook
	}

	switch bound.StatusID {
	case StatusInitiated:
		return SaleTransition{
			From:          StatusInitiated,
			To:            StatusSold,
			DescriptionID: shared_domain.BalanceDescriptionBondSaleMC,
			Comment:       fmt.Sprintf("Bono \"%s\" vendido.", bound.Key),
		}, nil
	case StatusFinalized:
		return SaleTransition{
			From:          StatusFinalized,
			To:            StatusTook,
			DescriptionID: shared_domain.BalanceDescriptionBondMC,
			Comment:       fmt.Sprintf("Bono \"%s\" canjeado.", bound.Key),
		}, nil
	case StatusSold:
		return SaleTransition{}, ErrBondAlreadySold
	case StatusTook:
		return SaleTransition{}, ErrBondAlreadyTook
	}
	return SaleTransition{}, ErrBondStatusNotSellable
}

// Dates devuelve la fecha que escribe la transición, at, en soldDate o en tookDate; la otra queda en nil.
func (t SaleTransition) Dates(at time.Time) (soldDate, tookDate *time.Time) {
	if t.To == StatusTook {
		return nil, &at
	}
	return &at, nil
}
