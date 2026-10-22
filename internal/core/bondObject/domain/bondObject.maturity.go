package bondObject_domain

import (
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// maturityDateLayout es el formato legacy "D MMM YYYY", con el mes abreviado en inglés.
const maturityDateLayout = "2 Jan 2006"

// NewBondObjectForInvestor suma al producto la fecha en que maduraría si se compra en now: el inicio del
// día en la zona utcDiff más maturityTimeInDays días, y esa fecha en la misma zona con formato
// "D MMM YYYY".
func NewBondObjectForInvestor(o BondObject, now time.Time, utcDiff int) BondObjectForInvestor {
	maturity := shared_domain.StartOfDay(now, utcDiff).AddDate(0, 0, o.MaturityTimeInDays)
	return BondObjectForInvestor{
		BondObject:            o,
		MaturityDateEstimated: maturity,
		MaturityDateFormatted: maturity.In(time.FixedZone("", utcDiff*3600)).Format(maturityDateLayout),
	}
}
