package shared_domain

import (
	"fmt"
	"time"
)

// EffectiveDate devuelve la fecha efectiva de un retiro según el día de now en la zona utcDiff
// (horas): del 1 al 10, el 15 del mes; del 11 al 25, el último día del mes; desde el 26, el 15 del
// mes siguiente (diciembre pasa a enero del año siguiente). Formato legacy "YYYY-M-D".
func EffectiveDate(now time.Time, utcDiff int) string {
	// fecha local: inicio del día en la zona, desplazado para leerlo en UTC
	local := StartOfDay(now, utcDiff).Add(time.Duration(utcDiff) * time.Hour)
	y, m, d := local.Date()
	switch {
	case d <= 10:
		return formatEffectiveDate(y, m, 15)
	case d <= 25:
		// día 0 del mes siguiente = último día de este mes
		return formatEffectiveDate(y, m, time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day())
	default:
		// time.Date normaliza el mes 13 a enero del año siguiente
		next := time.Date(y, m+1, 15, 0, 0, 0, 0, time.UTC)
		return formatEffectiveDate(next.Year(), next.Month(), 15)
	}
}

func formatEffectiveDate(y int, m time.Month, d int) string {
	return fmt.Sprintf("%d-%d-%d", y, int(m), d)
}
