package bondBound_domain

import (
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// SaleProjection es la cotización de la venta anticipada o el canje de un bono.
type SaleProjection struct {
	Inversion         float64 `json:"inversion"`
	Interest          float64 `json:"interest"`
	PenaltyPercentage float64 `json:"penaltyPercentage"`
	Profit            float64 `json:"profit"`
	AtMoment          float64 `json:"atMoment"`
	Amount            float64 `json:"amount"`
	TotalDaysPassed   int     `json:"totalDaysPassed"`
}

// QuoteSale cotiza el bono a la fecha now en la zona utcDiff, con los términos de su snapshot. Antes del
// punto de inflexión devuelve la inversión; desde la inflexión suma el interés penalizado, y desde el
// vencimiento, el completo. Si el plazo no supera al punto de inflexión es ErrBondInvalidTerms, y antes del
// día mínimo de retiro, ErrInvalidWithdrawDate. Amount, AtMoment y Profit salen redondeados.
func QuoteSale(bound BondBound, now time.Time, utcDiff int) (SaleProjection, error) {
	m := bound.MaturityTimeInDays
	p := bound.EarlyCashTerm.InflectionPoint
	r := bound.MaturityRewardsPercentage
	pf := bound.EarlyCashTerm.InflectionPenaltyPercentage
	if m-p <= 0 {
		return SaleProjection{}, ErrBondInvalidTerms
	}

	today := shared_domain.StartOfDay(now, utcDiff)
	start := shared_domain.StartOfDay(bound.InitialDate, utcDiff)
	if today.Before(start.AddDate(0, 0, bound.EarlyCashTerm.InitialDayForWithdraw)) {
		return SaleProjection{}, ErrInvalidWithdrawDate
	}

	amount := bound.Amount
	daysPassed := min(shared_domain.DaysBetween(start, today, utcDiff), m)
	// la conversión explícita evita que el compilador fusione la operación (FMA) y mantiene el resultado
	// de JavaScript
	interest := float64(amount / float64(m-p) * float64(daysPassed-p) * (r / 100))
	penalized := float64(interest * (1 - pf/100))

	atMoment, credited, profit := amount, amount, max(interest, 0)
	switch {
	case !today.Before(shared_domain.StartOfDay(bound.MaturityDate, utcDiff)):
		atMoment, credited = interest+amount, interest+amount
	case !today.Before(start.AddDate(0, 0, p)):
		atMoment, credited, profit = interest+amount, amount+penalized, max(penalized, 0)
	}

	return SaleProjection{
		Inversion:         amount,
		Interest:          r,
		PenaltyPercentage: pf,
		Profit:            shared_domain.Round2(profit),
		AtMoment:          shared_domain.Round2(atMoment),
		Amount:            shared_domain.Round2(credited),
		TotalDaysPassed:   daysPassed,
	}, nil
}
