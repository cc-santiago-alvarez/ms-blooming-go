package bondBound_domain

import (
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// shortDateLayout es el formato legacy "D MMM YYYY", con el mes abreviado en inglés.
const shortDateLayout = "2 Jan 2006"

// Accumulated devuelve los días calendario desde la fecha inicial del bono hasta now, en la zona utcDiff, y
// el monto acumulado: el capital más el interés al vencimiento, prorrateado por día. Solo acumula un bono
// iniciado que no pasó su plazo; si no, o si no tiene plazo, el monto es 0. El monto sale redondeado.
func Accumulated(b BondBound, now time.Time, utcDiff int) (daysPassed int, amount float64) {
	daysPassed = shared_domain.DaysBetween(b.InitialDate, now, utcDiff)
	m := b.MaturityTimeInDays
	if b.StatusID != StatusInitiated || m <= 0 || daysPassed > m {
		return daysPassed, 0
	}
	// la conversión explícita evita que el compilador fusione la operación (FMA) y mantiene el resultado
	// de JavaScript
	total := float64(b.Amount*(b.MaturityRewardsPercentage/100)) + b.Amount
	return daysPassed, shared_domain.Round2(total / float64(m) * float64(daysPassed))
}

// Progress indica si el bono está creciendo, desde el punto de inflexión hasta el último día de su plazo, o
// si ya creció, desde el día siguiente. Cuenta los días calendario desde la fecha inicial hasta now, en la
// zona utcDiff.
func Progress(b BondBound, now time.Time, utcDiff int) (growing, grew bool) {
	days := shared_domain.DaysBetween(b.InitialDate, now, utcDiff)
	grew = days > b.MaturityTimeInDays
	growing = days >= b.EarlyCashTerm.InflectionPoint && !grew
	return growing, grew
}

// FormatShortDate devuelve la fecha de t en la zona utcDiff con el formato "D MMM YYYY" (por ejemplo
// "5 Mar 2027").
func FormatShortDate(t time.Time, utcDiff int) string {
	return t.In(time.FixedZone("", utcDiff*3600)).Format(shortDateLayout)
}
