package shared_domain

import "time"

const (
	inflectionColor = "#CC0000"
	withdrawColor   = "#4ec74c"
	inflectionLabel = "Punto de Crecimiento"
	withdrawLabel   = "Zona de Retiro"
	guideThickness  = 1.5
)

// BuildObjectGuides arma las guías de un bondObject a partir de hoy: el punto de crecimiento y, si hay
// día inicial de retiro, la zona de retiro hasta el día anterior al vencimiento.
func BuildObjectGuides(terms ProjectionTerms, today time.Time, utcDiff int) []ChartGuide {
	start := StartOfDay(today, utcDiff)
	guides := []ChartGuide{inflectionGuide(terms.EarlyCashTerm.InflectionPoint, start, utcDiff)}

	if w := terms.EarlyCashTerm.InitialDayForWithdraw; w > 0 {
		guides = append(guides, ChartGuide{
			Date:          guideDate(start, w, utcDiff),
			ToDate:        guideDate(start, terms.MaturityTimeInDays-1, utcDiff),
			LineColor:     withdrawColor,
			LineAlpha:     1,
			FillAlpha:     0.2,
			FillColor:     withdrawColor,
			DashLength:    10,
			Inside:        true,
			LabelRotation: 0,
			Position:      "top",
			Label:         withdrawLabel,
		})
	}
	return guides
}

// BuildBoundGuides arma las guías de un bono comprado a partir de su createdAt; el retiro se marca con
// una línea en el día inicial, sin zona.
func BuildBoundGuides(terms ProjectionTerms, createdAt time.Time, utcDiff int) []ChartGuide {
	start := StartOfDay(createdAt, utcDiff)
	guides := []ChartGuide{inflectionGuide(terms.EarlyCashTerm.InflectionPoint, start, utcDiff)}

	if w := terms.EarlyCashTerm.InitialDayForWithdraw; w > 0 {
		thickness := guideThickness
		guides = append(guides, ChartGuide{
			Date:          guideDate(start, w, utcDiff),
			LineColor:     withdrawColor,
			LineAlpha:     1,
			FillAlpha:     0.2,
			DashLength:    10,
			Inside:        true,
			LabelRotation: 90,
			LineThickness: &thickness,
			Label:         withdrawLabel,
		})
	}
	return guides
}

// inflectionGuide es la línea del punto de crecimiento; sin punto de inflexión queda sin grosor ni texto.
func inflectionGuide(p int, start time.Time, utcDiff int) ChartGuide {
	thickness, label := 0.0, ""
	if p > 0 {
		thickness, label = guideThickness, inflectionLabel
	}
	return ChartGuide{
		Date:          guideDate(start, p, utcDiff),
		LineColor:     inflectionColor,
		LineAlpha:     1,
		FillAlpha:     0.2,
		FillColor:     inflectionColor,
		DashLength:    10,
		Inside:        true,
		LabelRotation: 90,
		LineThickness: &thickness,
		Label:         label,
	}
}

// guideDate devuelve la fecha YYYY-MM-DD que cae days días después de start en la zona utcDiff.
func guideDate(start time.Time, days, utcDiff int) string {
	return formatDate(start.Add(time.Duration(days)*day), utcDiff)
}
