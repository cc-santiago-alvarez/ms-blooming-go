package shared_domain

import (
	"math"
	"strconv"
	"time"
)

// Umbrales de tiempo relativo de moment.
const (
	fewSecondsThreshold = 44
	minutesThreshold    = 45
	hoursThreshold      = 22
	daysThreshold       = 26
	monthsThreshold     = 11
)

// FromNow describe en inglés cuánto pasó (o falta) de now a t, igual que fromNow() de moment: "a few
// seconds ago", "2 days ago", "in a month". Los meses se cuentan por calendario en UTC.
func FromNow(t, now time.Time) string {
	// milisegundos en UTC, como un Date de JavaScript
	t = time.UnixMilli(t.UnixMilli()).UTC()
	now = time.UnixMilli(now.UnixMilli()).UTC()

	future := t.After(now)
	var months int
	var ms int64
	if future {
		months, ms = calendarDiff(now, t)
	} else {
		months, ms = calendarDiff(t, now)
	}

	text := relativeTime(months, float64(ms))
	if future {
		return "in " + text
	}
	return text + " ago"
}

// relativeTime elige el texto de una duración de months meses de calendario más ms milisegundos.
func relativeTime(months int, ms float64) string {
	// la conversión explícita evita que el compilador fusione la operación (FMA) y mantiene el
	// resultado de JavaScript
	days := jsRound(float64(months) * 146097 / 4800)
	seconds := jsRound(float64(days*86400) + ms/1000)
	minutes := jsRound(float64(days*1440) + ms/6e4)
	hours := jsRound(float64(days*24) + ms/36e5)
	wholeDays := jsRound(days + ms/864e5)
	asMonths := float64(months) + ms/864e5*4800/146097
	roundMonths := jsRound(asMonths)
	years := jsRound(asMonths / 12)

	switch {
	case seconds <= fewSecondsThreshold:
		return "a few seconds"
	case minutes <= 1:
		return "a minute"
	case minutes < minutesThreshold:
		return count(minutes, "minutes")
	case hours <= 1:
		return "an hour"
	case hours < hoursThreshold:
		return count(hours, "hours")
	case wholeDays <= 1:
		return "a day"
	case wholeDays < daysThreshold:
		return count(wholeDays, "days")
	case roundMonths <= 1:
		return "a month"
	case roundMonths < monthsThreshold:
		return count(roundMonths, "months")
	case years <= 1:
		return "a year"
	}
	return count(years, "years")
}

// calendarDiff devuelve los meses de calendario completos de base a other (base <= other) y los
// milisegundos que sobran.
func calendarDiff(base, other time.Time) (months int, ms int64) {
	months = (other.Year()-base.Year())*12 + int(other.Month()) - int(base.Month())
	if addMonths(base, months).After(other) {
		months--
	}
	return months, other.Sub(addMonths(base, months)).Milliseconds()
}

// addMonths suma n meses como moment: si el día no existe en el mes destino, queda en el último.
func addMonths(t time.Time, n int) time.Time {
	first := time.Date(t.Year(), t.Month()+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
	lastDay := first.AddDate(0, 1, -1).Day()
	return time.Date(first.Year(), first.Month(), min(t.Day(), lastDay), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
}

// jsRound redondea como Math.round de JavaScript: los empates van hacia +∞.
func jsRound(x float64) float64 {
	f := math.Floor(x)
	if x-f >= 0.5 {
		return f + 1
	}
	return f
}

// count escribe "N unidad".
func count(n float64, unit string) string {
	return strconv.FormatFloat(n, 'f', 0, 64) + " " + unit
}
