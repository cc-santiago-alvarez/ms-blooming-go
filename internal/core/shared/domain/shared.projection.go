package shared_domain

import (
	"math"
	"time"
)

// ProjectionTerms son los términos de un bono que usan la simulación y las guías.
type ProjectionTerms struct {
	MaturityTimeInDays        int
	MaturityRewardsPercentage float64
	EarlyCashTerm             EarlyCashTerm
}

// SimulateProjection simula el bono día por día, desde hoy hasta el vencimiento inclusive, en la zona
// utcDiff. Ganancia y Perdida aparecen desde el punto de inflexión. Los términos llegan validados.
func SimulateProjection(terms ProjectionTerms, amount float64, today time.Time, utcDiff int) []ChartPoint {
	m := terms.MaturityTimeInDays
	if m < 0 {
		return []ChartPoint{}
	}
	r := terms.MaturityRewardsPercentage
	p := terms.EarlyCashTerm.InflectionPoint
	pf := terms.EarlyCashTerm.InflectionPenaltyPercentage

	var daily float64
	if p == 0 {
		pf = terms.EarlyCashTerm.InitialPenaltyPercentage
		daily = r / float64(m) / 100
	} else {
		daily = r / float64(m-p) / 100
	}

	start := StartOfDay(today, utcDiff)
	inversion := Round2(amount)
	points := make([]ChartPoint, 0, m+1)
	current := 0.0
	for i := 0; i <= m; i++ {
		point := ChartPoint{
			Date:      formatDate(start.Add(time.Duration(i)*day), utcDiff),
			Inversion: inversion,
		}
		switch {
		case i == p:
			ganancia, perdida := inversion, inversion
			point.Ganancia, point.Perdida = &ganancia, &perdida
		case i > p:
			// la conversión explícita evita que el compilador fusione la operación (FMA) y mantiene
			// el resultado de JavaScript
			current += float64(amount * daily)
			penalty := float64(current*(pf/100)) - current
			ganancia, perdida := Round2(current+amount), Round2(math.Abs(penalty)+amount)
			point.Ganancia, point.Perdida = &ganancia, &perdida
		}
		points = append(points, point)
	}
	return points
}
