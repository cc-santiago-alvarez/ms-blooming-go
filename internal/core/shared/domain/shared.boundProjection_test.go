package shared_domain_test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// boundInitialDate es el 1 de enero a las 10:00 en UTC-5.
var boundInitialDate = time.Date(2026, 1, 1, 15, 0, 0, 0, time.UTC)

// num devuelve un puntero al valor, para los campos opcionales de los puntos esperados.
func num(v float64) *float64 {
	return &v
}

// checkBoundPoint compara un día de la evolución con el esperado y muestra los dos en JSON.
func checkBoundPoint(t *testing.T, day int, got, want shared_domain.BoundChartPoint) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		g, _ := json.Marshal(got)
		w, _ := json.Marshal(want)
		t.Errorf("día %d = %s\nse esperaba %s", day, g, w)
	}
}

// boundValues devuelve los valores numéricos del punto por clave JSON.
func boundValues(p shared_domain.BoundChartPoint) map[string]*float64 {
	return map[string]*float64{
		"inversion":       p.Inversion,
		"ganancia":        p.Ganancia,
		"perdida":         p.Perdida,
		"atMoment":        p.AtMoment,
		"perdidaAtMoment": p.PerdidaAtMoment,
		"gananciaFinal":   p.GananciaFinal,
	}
}

// Hasta hoy cada día lleva lo acumulado con y sin penalidad, y hasta la inflexión también la ganancia
// igual al monto; hoy lleva los valores del momento, y después va lo proyectado hasta la ganancia final.
// CA-03.11
func TestBoundProjectionUntilToday(t *testing.T) {
	points := shared_domain.BoundProjection(projectionTerms(10, 10, 5, 100, 50), 1000, boundInitialDate,
		boundInitialDate.AddDate(0, 0, 7), -5)

	var want []shared_domain.BoundChartPoint
	for i := 0; i <= 5; i++ {
		want = append(want, shared_domain.BoundChartPoint{
			Date: fmt.Sprintf("2026-01-%02d", 1+i), Inversion: num(1000), Ganancia: num(1000), Perdida: num(1000),
		})
	}
	want = append(want,
		shared_domain.BoundChartPoint{Date: "2026-01-07", Inversion: num(1020), Perdida: num(1010)},
		shared_domain.BoundChartPoint{Date: "2026-01-08", Inversion: num(1040), Perdida: num(1020), AtMoment: num(1040),
			PerdidaAtMoment: num(1020), BulletClass: "lastBullet"},
		shared_domain.BoundChartPoint{Date: "2026-01-09", Ganancia: num(1060)},
		shared_domain.BoundChartPoint{Date: "2026-01-10", Ganancia: num(1080)},
		shared_domain.BoundChartPoint{Date: "2026-01-11", Ganancia: num(1100), GananciaFinal: num(1100)},
	)
	if len(points) != len(want) {
		t.Fatalf("la evolución tiene %d puntos, se esperaban %d", len(points), len(want))
	}
	for i := range want {
		checkBoundPoint(t, i, points[i], want[i])
	}
}

// Sin punto de inflexión la pérdida usa la penalidad de la inflexión y no la inicial, que la dejaría en
// el monto. CA-03.12
func TestBoundProjectionWithoutInflection(t *testing.T) {
	points := shared_domain.BoundProjection(projectionTerms(10, 10, 0, 100, 50), 1000, boundInitialDate,
		boundInitialDate.AddDate(0, 0, 2), -5)
	if len(points) != 11 {
		t.Fatalf("la evolución tiene %d puntos, se esperaban 11", len(points))
	}
	checkBoundPoint(t, 1, points[1], shared_domain.BoundChartPoint{Date: "2026-01-02", Inversion: num(1010), Perdida: num(1005)})
}

// Un bono vencido lleva lo acumulado en todos los días y no tiene punto de hoy; el último día cierra con
// la ganancia final. CA-03.13
func TestBoundProjectionMatured(t *testing.T) {
	points := shared_domain.BoundProjection(projectionTerms(10, 10, 5, 100, 50), 1000, boundInitialDate,
		boundInitialDate.AddDate(0, 0, 15), -5)
	if len(points) != 11 {
		t.Fatalf("la evolución tiene %d puntos, se esperaban 11", len(points))
	}
	for i, p := range points {
		if p.Inversion == nil || p.Perdida == nil {
			t.Errorf("el día %d no tiene inversión o pérdida", i)
		}
		if p.AtMoment != nil || p.PerdidaAtMoment != nil || p.BulletClass != "" {
			t.Errorf("el día %d tiene valores del momento: %s/%s %q", i, pointValue(p.AtMoment), pointValue(p.PerdidaAtMoment), p.BulletClass)
		}
	}
	checkBoundPoint(t, 10, points[10], shared_domain.BoundChartPoint{
		Date: "2026-01-11", Inversion: num(1100), Perdida: num(1050), GananciaFinal: num(1100),
	})
}

// Monto y tasa con decimales no se truncan: cada valor coincide con la fórmula en float y ninguno con el
// cálculo sobre 1000 y 7 %; en JSON todos son números con 2 decimales, y el día 0 es el de la compra en
// la zona utcDiff. CA-03.14
func TestBoundProjectionKeepsDecimals(t *testing.T) {
	amount, rate, pf := 1000.50, 7.5, 62.5
	m, p := 365, 30
	initialDate := time.Date(2026, 3, 10, 3, 0, 0, 0, time.UTC)
	now := initialDate.AddDate(0, 0, 100)
	points := shared_domain.BoundProjection(projectionTerms(m, rate, p, 100, pf), amount, initialDate, now, -5)
	truncated := shared_domain.BoundProjection(projectionTerms(m, 7, p, 100, pf), 1000, initialDate, now, -5)
	if len(points) != m+1 || len(truncated) != m+1 {
		t.Fatalf("la evolución tiene %d y %d puntos, se esperaban %d", len(points), len(truncated), m+1)
	}
	if points[0].Date != "2026-03-09" {
		t.Errorf("el día 0 es %q, se esperaba \"2026-03-09\"", points[0].Date)
	}

	daily := rate / float64(m-p) / 100
	current := 0.0
	for i, pt := range points {
		if i > p {
			// la conversión explícita evita FMA, igual que la implementación
			current += float64(amount * daily)
		}
		valor := shared_domain.Round2(current + amount)
		perdida := shared_domain.Round2(math.Abs(float64(current*(pf/100))-current) + amount)
		trunc := boundValues(truncated[i])
		for key, v := range boundValues(pt) {
			if v == nil {
				continue
			}
			want := valor
			if key == "perdida" || key == "perdidaAtMoment" {
				want = perdida
			}
			if *v != want {
				t.Errorf("día %d, %s = %v, se esperaba %v", i, key, *v, want)
			}
			if tv := trunc[key]; tv != nil && *tv == *v {
				t.Errorf("día %d, %s = %v, igual al cálculo truncado", i, key, *v)
			}
		}
	}

	raw, err := json.Marshal(points)
	if err != nil {
		t.Fatalf("no se pudo serializar la evolución: %v", err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("JSON propio inválido: %v", err)
	}
	for i, pt := range decoded {
		for key, v := range pt {
			if key == "date" || key == "bulletClass" {
				continue
			}
			f, ok := v.(float64)
			if !ok {
				t.Errorf("día %d, %s = %#v, se esperaba un número", i, key, v)
				continue
			}
			if shared_domain.Round2(f) != f {
				t.Errorf("día %d, %s = %v, con más de 2 decimales", i, key, f)
			}
		}
	}
}

// La evolución da, día por día, las mismas fechas, claves y valores que el feed de bonos comprados de
// ms-feed con los porcentajes sin truncar, incluido el punto de hoy. CA-03.15
func TestBoundProjectionMatchesLegacy(t *testing.T) {
	// salida de boundController.feed (ms-feed, bound/index.js) evaluado con Node 20 y moment 2.30.1, con la
	// zona del proceso en utcDiff (Etc/GMT), moment() fijado en now y parseInt cambiado por Number en los
	// porcentajes; los valores se pasaron a número con toFixed(2), porque el legado los da como string o,
	// en inversion, sin redondear
	raw, err := os.ReadFile("testdata/legacy-bound-projection.json")
	if err != nil {
		t.Fatalf("no se pudo leer la tabla del legado: %v", err)
	}
	var cases []struct {
		Amount                      float64           `json:"amount"`
		MaturityTimeInDays          int               `json:"maturityTimeInDays"`
		MaturityRewardsPercentage   float64           `json:"maturityRewardsPercentage"`
		InflectionPoint             int               `json:"inflectionPoint"`
		InitialPenaltyPercentage    float64           `json:"initialPenaltyPercentage"`
		InflectionPenaltyPercentage float64           `json:"inflectionPenaltyPercentage"`
		InitialDate                 time.Time         `json:"initialDate"`
		Now                         time.Time         `json:"now"`
		UtcDiff                     int               `json:"utcDiff"`
		Data                        []json.RawMessage `json:"data"`
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
		m := c.MaturityTimeInDays
		name := fmt.Sprintf("monto %v, M %d, R %v, P %d, Pf %v, compra %s, hoy %s en %d", c.Amount, m,
			c.MaturityRewardsPercentage, c.InflectionPoint, c.InflectionPenaltyPercentage,
			c.InitialDate.Format(time.RFC3339), c.Now.Format(time.RFC3339), c.UtcDiff)
		terms := projectionTerms(m, c.MaturityRewardsPercentage, c.InflectionPoint, c.InitialPenaltyPercentage,
			c.InflectionPenaltyPercentage)
		points := shared_domain.BoundProjection(terms, c.Amount, c.InitialDate, c.Now, c.UtcDiff)
		if len(points) != m+1 || len(c.Data) != m+1 {
			report("%s: %d puntos, el legado da %d", name, len(points), len(c.Data))
			continue
		}
		for i, pt := range points {
			days++
			if got, ok := sameJSON(t, pt, c.Data[i]); !ok {
				report("%s, día %d: %s\nel legado da %s", name, i, got, c.Data[i])
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("%d diferencias con el legado en %d casos y %d días", mismatches, len(cases), days)
	}
}
