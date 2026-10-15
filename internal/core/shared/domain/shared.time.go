package shared_domain

import "time"

const day = 24 * time.Hour

// Now devuelve la hora actual en UTC.
func Now() time.Time {
	return time.Now().UTC()
}

// StartOfDay devuelve el inicio del día de t en la zona utcDiff (horas), expresado en UTC.
func StartOfDay(t time.Time, utcDiff int) time.Time {
	local := t.In(time.FixedZone("", utcDiff*3600))
	y, m, d := local.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, local.Location()).UTC()
}

// DaysBetween devuelve los días calendario de a hasta b en la zona utcDiff; es negativo si b es
// anterior a a.
func DaysBetween(a, b time.Time, utcDiff int) int {
	return int(StartOfDay(b, utcDiff).Sub(StartOfDay(a, utcDiff)) / day)
}
