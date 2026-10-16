package shared_domain

import (
	"math"
	"math/big"
	"strconv"
)

var (
	hundred = big.NewFloat(100)
	half    = big.NewFloat(0.5)
)

// Round2 redondea a 2 decimales igual que toFixed(2) de JavaScript: sobre el valor binario exacto,
// con los empates lejos de cero. Es el único redondeo de montos y nunca devuelve -0.
func Round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return v
	}

	// |v|·100 es exacto con 128 bits: la mantisa de v ocupa 53 y 100 cabe en 7
	scaled := new(big.Float).SetPrec(128).SetFloat64(math.Abs(v))
	scaled.Mul(scaled, hundred)
	cents, _ := scaled.Int(nil)
	frac := new(big.Float).SetPrec(128).Sub(scaled, new(big.Float).SetInt(cents))
	if frac.Cmp(half) >= 0 {
		cents.Add(cents, big.NewInt(1))
	}
	if cents.Sign() == 0 {
		return 0
	}

	// ParseFloat da el float64 más cercano a cents/100, como Number() en JavaScript
	r, _ := strconv.ParseFloat(cents.String()+"e-2", 64)
	if v < 0 {
		return -r
	}
	return r
}
