package bondBound_domain_test

import (
	"testing"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// quotePurchase es la compra de los bonos cotizados: 13 de octubre a las 22:00 en UTC-5 y 14 de octubre a
// las 17:00 en UTC+14.
var quotePurchase = time.Date(2026, 10, 14, 3, 0, 0, 0, time.UTC)

// quoteAt es las 08:00 del día n desde la compra en la zona utcDiff.
func quoteAt(n, utcDiff int) time.Time {
	zone := time.FixedZone("", utcDiff*3600)
	y, m, d := quotePurchase.In(zone).Date()
	return time.Date(y, m, d+n, 8, 0, 0, 0, zone)
}

// quoteBound es un bono comprado en quotePurchase con los términos dados, que vence m días después.
func quoteBound(amount float64, m int, r float64, p int, pf float64, withdraw int) bondBound_domain.BondBound {
	return bondBound_domain.BondBound{
		Amount:                    amount,
		MaturityTimeInDays:        m,
		MaturityRewardsPercentage: r,
		EarlyCashTerm: shared_domain.EarlyCashTerm{
			InitialPenaltyPercentage:    100,
			InflectionPoint:             p,
			InflectionPenaltyPercentage: pf,
			InitialDayForWithdraw:       withdraw,
		},
		InitialDate:  quotePurchase,
		MaturityDate: quotePurchase.AddDate(0, 0, m),
	}
}

// QuoteSale cotiza igual que preSale del legado: la inversión antes de la inflexión, el interés penalizado
// desde la inflexión y el completo desde el vencimiento, con los días contados en la zona de la sesión y
// topados en el plazo. Las tasas con decimales no se truncan y en el vencimiento la ganancia es el interés
// completo. CA-05.1, CA-05.2, CA-05.3, CA-05.4
func TestQuoteSaleMatchesLegacy(t *testing.T) {
	// salida de preSale (bond.bound.js) evaluado con Node y moment con las mismas fechas, con parseInt
	// cambiado por Number y redondeada con toFixed(2)
	example := quoteBound(1000, 100, 10, 0, 50, 0)
	lateInflection := quoteBound(1000, 100, 10, 30, 50, 0)
	fractional := quoteBound(1000.5, 365, 7.5, 30, 62.5, 15)

	tests := []struct {
		name     string
		bound    bondBound_domain.BondBound
		utcDiff  int
		day      int
		profit   float64
		atMoment float64
		amount   float64
		days     int
	}{
		{name: "día 50", bound: example, utcDiff: -5, day: 50, profit: 25, atMoment: 1050, amount: 1025, days: 50},
		{name: "víspera del vencimiento", bound: example, utcDiff: -5, day: 99, profit: 49.5, atMoment: 1099, amount: 1049.5, days: 99},
		// el legado informa profit 50, penalizado
		{name: "vencimiento", bound: example, utcDiff: -5, day: 100, profit: 100, atMoment: 1100, amount: 1100, days: 100},
		// el legado informa profit 50, penalizado
		{name: "después del vencimiento", bound: example, utcDiff: -5, day: 150, profit: 100, atMoment: 1100, amount: 1100, days: 100},
		{name: "antes de la inflexión", bound: lateInflection, utcDiff: -5, day: 20, profit: 0, atMoment: 1000, amount: 1000, days: 20},
		{name: "día de la inflexión", bound: lateInflection, utcDiff: -5, day: 30, profit: 0, atMoment: 1000, amount: 1000, days: 30},
		{name: "después de la inflexión", bound: lateInflection, utcDiff: -5, day: 65, profit: 25, atMoment: 1050, amount: 1025, days: 65},
		// el legado informa profit 50, penalizado
		{name: "vencimiento con inflexión", bound: lateInflection, utcDiff: -5, day: 100, profit: 100, atMoment: 1100, amount: 1100, days: 100},
		{name: "tasa con decimales, día mínimo de retiro", bound: fractional, utcDiff: 14, day: 15, profit: 0, atMoment: 1000.5, amount: 1000.5, days: 15},
		{name: "tasa con decimales, día 100", bound: fractional, utcDiff: 14, day: 100, profit: 5.88, atMoment: 1016.18, amount: 1006.38, days: 100},
		{name: "tasa con decimales, víspera del vencimiento", bound: fractional, utcDiff: 14, day: 364, profit: 28.06, atMoment: 1075.31, amount: 1028.56, days: 364},
		// el legado informa profit 28.14, penalizado
		{name: "tasa con decimales, vencimiento", bound: fractional, utcDiff: 14, day: 365, profit: 75.04, atMoment: 1075.54, amount: 1075.54, days: 365},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bondBound_domain.QuoteSale(tt.bound, quoteAt(tt.day, tt.utcDiff), tt.utcDiff)
			if err != nil {
				t.Fatalf("error %v, se esperaba la cotización", err)
			}

			want := bondBound_domain.SaleProjection{
				Inversion:         tt.bound.Amount,
				Interest:          tt.bound.MaturityRewardsPercentage,
				PenaltyPercentage: tt.bound.EarlyCashTerm.InflectionPenaltyPercentage,
				Profit:            tt.profit,
				AtMoment:          tt.atMoment,
				Amount:            tt.amount,
				TotalDaysPassed:   tt.days,
			}
			if got != want {
				t.Errorf("cotización %+v, se esperaba %+v", got, want)
			}
		})
	}
}
