package shared_domain_test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// projectionToday es el 13 de octubre a las 10:00 en UTC-5.
var projectionToday = time.Date(2026, 10, 13, 15, 0, 0, 0, time.UTC)

// projectionTerms arma los términos de la simulación.
func projectionTerms(m int, r float64, p int, pi, pf float64) shared_domain.ProjectionTerms {
	return shared_domain.ProjectionTerms{
		MaturityTimeInDays:        m,
		MaturityRewardsPercentage: r,
		EarlyCashTerm: shared_domain.EarlyCashTerm{
			InitialPenaltyPercentage:    pi,
			InflectionPoint:             p,
			InflectionPenaltyPercentage: pf,
		},
	}
}

// projectionDay es la ganancia y la pérdida esperadas en un día de la simulación.
type projectionDay struct {
	day      int
	ganancia float64
	perdida  float64
}

// checkProjectionDays compara la ganancia y la pérdida de cada día esperado.
func checkProjectionDays(t *testing.T, points []shared_domain.ChartPoint, days []projectionDay) {
	t.Helper()
	for _, d := range days {
		p := points[d.day]
		if p.Ganancia == nil || p.Perdida == nil {
			t.Errorf("el día %d no tiene ganancia o pérdida", d.day)
			continue
		}
		if *p.Ganancia != d.ganancia || *p.Perdida != d.perdida {
			t.Errorf("día %d = %v/%v, se esperaba %v/%v", d.day, *p.Ganancia, *p.Perdida, d.ganancia, d.perdida)
		}
	}
}

// pointValue muestra un valor opcional del punto.
func pointValue(v *float64) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprint(*v)
}

// Sin punto de inflexión hay M+1 días consecutivos desde hoy en la zona utcDiff, con ganancia y
// pérdida desde el día 0; con penalidad inicial del 100 % la pérdida no crece. CA-03.1
func TestSimulateProjectionWithoutInflection(t *testing.T) {
	points := shared_domain.SimulateProjection(projectionTerms(10, 10, 0, 100, 100), 1000, projectionToday, -5)
	if len(points) != 11 {
		t.Fatalf("la simulación tiene %d puntos, se esperaban 11", len(points))
	}
	for i, p := range points {
		if want := fmt.Sprintf("2026-10-%d", 13+i); p.Date != want {
			t.Errorf("día %d con fecha %q, se esperaba %q", i, p.Date, want)
		}
		if p.Inversion != 1000 {
			t.Errorf("día %d con inversión %v, se esperaba 1000", i, p.Inversion)
		}
	}
	checkProjectionDays(t, points, []projectionDay{
		{day: 0, ganancia: 1000, perdida: 1000},
		{day: 1, ganancia: 1010, perdida: 1000},
		{day: 10, ganancia: 1100, perdida: 1000},
	})
}

// Con punto de inflexión, los días anteriores no tienen ganancia ni pérdida, el día de inflexión vale
// la inversión y desde ahí crecen con la tasa repartida en los días restantes. CA-03.2
func TestSimulateProjectionWithInflection(t *testing.T) {
	points := shared_domain.SimulateProjection(projectionTerms(10, 10, 5, 100, 50), 1000, projectionToday, -5)
	if len(points) != 11 {
		t.Fatalf("la simulación tiene %d puntos, se esperaban 11", len(points))
	}
	for i := 0; i < 5; i++ {
		if points[i].Ganancia != nil || points[i].Perdida != nil {
			t.Errorf("día %d = %s/%s, antes de la inflexión no hay valores", i, pointValue(points[i].Ganancia), pointValue(points[i].Perdida))
		}
	}
	checkProjectionDays(t, points, []projectionDay{
		{day: 5, ganancia: 1000, perdida: 1000},
		{day: 6, ganancia: 1020, perdida: 1010},
		{day: 10, ganancia: 1100, perdida: 1050},
	})
}

// Monto y tasa con decimales no se truncan: cada día coincide con la fórmula en float y ninguno con el
// cálculo sobre 1000 y 7 %. CA-03.3
func TestSimulateProjectionKeepsDecimals(t *testing.T) {
	amount, rate, pf := 1000.50, 7.5, 62.5
	m, p := 365, 30
	points := shared_domain.SimulateProjection(projectionTerms(m, rate, p, 100, pf), amount, projectionToday, -5)
	truncated := shared_domain.SimulateProjection(projectionTerms(m, 7, p, 100, pf), 1000, projectionToday, -5)
	if len(points) != m+1 || len(truncated) != m+1 {
		t.Fatalf("la simulación tiene %d y %d puntos, se esperaban %d", len(points), len(truncated), m+1)
	}

	if points[0].Inversion != 1000.5 {
		t.Errorf("inversión %v, se esperaba 1000.5", points[0].Inversion)
	}
	checkProjectionDays(t, points, []projectionDay{{day: p, ganancia: 1000.5, perdida: 1000.5}})

	daily := rate / float64(m-p) / 100
	current := 0.0
	for i := p + 1; i <= m; i++ {
		// la conversión explícita evita FMA, igual que la implementación
		current += float64(amount * daily)
		penalty := float64(current*(pf/100)) - current
		want := projectionDay{day: i, ganancia: shared_domain.Round2(current + amount), perdida: shared_domain.Round2(math.Abs(penalty) + amount)}
		checkProjectionDays(t, points, []projectionDay{want})

		if *points[i].Ganancia == *truncated[i].Ganancia || *points[i].Perdida == *truncated[i].Perdida {
			t.Errorf("día %d = %v/%v, igual al cálculo truncado", i, *points[i].Ganancia, *points[i].Perdida)
		}
	}
}

// La simulación da, día por día, los mismos valores y fechas que getDataChart del legado con monto y
// porcentajes sin truncar. CA-03.1, CA-03.2, CA-03.3
func TestSimulateProjectionMatchesLegacy(t *testing.T) {
	// salida de getDataChart (bond.object.js) evaluado con Node y moment, con parseInt cambiado por Number
	// en monto y porcentajes; ganancia y perdida van desde el día de inflexión hasta el vencimiento
	raw, err := os.ReadFile("testdata/legacy-projection.json")
	if err != nil {
		t.Fatalf("no se pudo leer la tabla del legado: %v", err)
	}
	var cases []struct {
		Amount                      float64   `json:"amount"`
		MaturityTimeInDays          int       `json:"maturityTimeInDays"`
		MaturityRewardsPercentage   float64   `json:"maturityRewardsPercentage"`
		InflectionPoint             int       `json:"inflectionPoint"`
		InitialPenaltyPercentage    float64   `json:"initialPenaltyPercentage"`
		InflectionPenaltyPercentage float64   `json:"inflectionPenaltyPercentage"`
		Today                       time.Time `json:"today"`
		UtcDiff                     int       `json:"utcDiff"`
		Dates                       [3]string `json:"dates"`
		Ganancia                    []float64 `json:"ganancia"`
		Perdida                     []float64 `json:"perdida"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("tabla del legado inválida: %v", err)
	}
	if len(cases) < 30 {
		t.Fatalf("la tabla del legado tiene %d casos, se esperaban al menos 30", len(cases))
	}

	mismatches, days := 0, 0
	report := func(format string, args ...any) {
		mismatches++
		if mismatches <= 10 {
			t.Errorf(format, args...)
		}
	}
	for _, c := range cases {
		m, p := c.MaturityTimeInDays, c.InflectionPoint
		name := fmt.Sprintf("monto %v, M %d, R %v, P %d", c.Amount, m, c.MaturityRewardsPercentage, p)
		terms := projectionTerms(m, c.MaturityRewardsPercentage, p, c.InitialPenaltyPercentage, c.InflectionPenaltyPercentage)
		points := shared_domain.SimulateProjection(terms, c.Amount, c.Today, c.UtcDiff)
		if len(points) != m+1 || len(c.Ganancia) != m-p+1 || len(c.Perdida) != m-p+1 {
			report("%s: %d puntos, el legado da %d desde la inflexión", name, len(points), len(c.Ganancia))
			continue
		}

		if dates := [3]string{points[0].Date, points[p].Date, points[m].Date}; dates != c.Dates {
			report("%s: fechas de los días 0, P y M = %v, el legado da %v", name, dates, c.Dates)
		}
		for i, pt := range points {
			days++
			if pt.Inversion != c.Amount {
				report("%s, día %d: inversión %v, el legado da %v", name, i, pt.Inversion, c.Amount)
			}
			if i < p {
				if pt.Ganancia != nil || pt.Perdida != nil {
					report("%s, día %d: %s/%s antes de la inflexión", name, i, pointValue(pt.Ganancia), pointValue(pt.Perdida))
				}
				continue
			}
			g, l := c.Ganancia[i-p], c.Perdida[i-p]
			if pt.Ganancia == nil || pt.Perdida == nil || *pt.Ganancia != g || *pt.Perdida != l {
				report("%s, día %d: %s/%s, el legado da %v/%v", name, i, pointValue(pt.Ganancia), pointValue(pt.Perdida), g, l)
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("%d diferencias con el legado en %d casos y %d días", mismatches, len(cases), days)
	}
}
