package shared_domain

// ChartPoint es un día de la simulación de un bono. Ganancia y Perdida solo aparecen desde el punto de
// inflexión.
type ChartPoint struct {
	Date      string   `json:"date"`
	Inversion float64  `json:"inversion"`
	Ganancia  *float64 `json:"ganancia,omitempty"`
	Perdida   *float64 `json:"perdida,omitempty"`
}

// ChartGuide es una guía de amCharts: una línea en una fecha o una zona entre Date y ToDate.
type ChartGuide struct {
	Date          string   `json:"date"`
	ToDate        string   `json:"toDate,omitempty"`
	LineColor     string   `json:"lineColor"`
	LineAlpha     float64  `json:"lineAlpha"`
	FillAlpha     float64  `json:"fillAlpha"`
	FillColor     string   `json:"fillColor,omitempty"`
	DashLength    int      `json:"dashLength"`
	Inside        bool     `json:"inside"`
	LabelRotation int      `json:"labelRotation"`
	LineThickness *float64 `json:"lineThickness,omitempty"`
	Position      string   `json:"position,omitempty"`
	Label         string   `json:"label"`
}
