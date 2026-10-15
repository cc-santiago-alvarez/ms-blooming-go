package shared_domain

import "math"

// Round2 redondea a 2 decimales, con las mitades lejos de cero. Es el único redondeo de montos.
func Round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return v
	}
	r := math.Round(v*100) / 100
	if r == 0 {
		return 0 // evita -0
	}
	return r
}
