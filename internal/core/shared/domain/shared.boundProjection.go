package shared_domain

import (
	"math"
	"time"
)

// todayBulletClass es la clase de amCharts del punto de hoy.
const todayBulletClass = "lastBullet"

// BoundProjection arma la evolución de un bono comprado día por día, desde su fecha inicial hasta el
// vencimiento inclusive, en la zona utcDiff. Hasta now lleva lo acumulado, con y sin penalidad; después,
// lo proyectado. El punto de now lleva los valores del momento y el último, la ganancia final.
func BoundProjection(terms ProjectionTerms, amount float64, initialDate, now time.Time, utcDiff int) []BoundChartPoint {
	m := terms.MaturityTimeInDays
	if m < 0 {
		return []BoundChartPoint{}
	}
	p := terms.EarlyCashTerm.InflectionPoint
	pf := terms.EarlyCashTerm.InflectionPenaltyPercentage

	// sin días después de la inflexión no hay interés que sumar
	var daily float64
	if p < m {
		daily = terms.MaturityRewardsPercentage / float64(m-p) / 100
	}

	start := StartOfDay(initialDate, utcDiff)
	elapsed := DaysBetween(initialDate, now, utcDiff)
	points := make([]BoundChartPoint, 0, m+1)
	current := 0.0
	for i := 0; i <= m; i++ {
		point := BoundChartPoint{Date: formatDate(start.Add(time.Duration(i)*day), utcDiff)}
		if i > p {
			// la conversión explícita evita que el compilador fusione la operación (FMA) y mantiene
			// el resultado de JavaScript
			current += float64(amount * daily)
		}
		valor := Round2(current + amount)

		// hasta la inflexión no hay interés, así que la ganancia es el monto
		if i <= p || i > elapsed {
			ganancia := valor
			point.Ganancia = &ganancia
		}
		if i <= elapsed {
			inversion, perdida := valor, Round2(math.Abs(float64(current*(pf/100))-current)+amount)
			point.Inversion, point.Perdida = &inversion, &perdida
			if i == elapsed {
				atMoment, perdidaAtMoment := valor, perdida
				point.AtMoment, point.PerdidaAtMoment = &atMoment, &perdidaAtMoment
				point.BulletClass = todayBulletClass
			}
		}
		if i == m {
			gananciaFinal := valor
			point.GananciaFinal = &gananciaFinal
		}
		points = append(points, point)
	}
	return points
}
