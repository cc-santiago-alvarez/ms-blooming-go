package shared_domain_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// guidesBuilder arma las guías a partir de una fecha base.
type guidesBuilder func(shared_domain.ProjectionTerms, time.Time, int) []shared_domain.ChartGuide

// guideTerms arma los términos que usan las guías.
func guideTerms(m, p, w int) shared_domain.ProjectionTerms {
	return shared_domain.ProjectionTerms{
		MaturityTimeInDays:        m,
		MaturityRewardsPercentage: 15,
		EarlyCashTerm: shared_domain.EarlyCashTerm{
			InitialPenaltyPercentage:    100,
			InflectionPoint:             p,
			InflectionPenaltyPercentage: 75,
			InitialDayForWithdraw:       w,
		},
	}
}

// sameJSON serializa got y lo compara con el JSON del legado sin importar el orden de las claves.
func sameJSON(t *testing.T, got any, legacy json.RawMessage) (string, bool) {
	t.Helper()
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("no se pudo serializar %v: %v", got, err)
	}
	var g, w any
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatalf("JSON propio inválido: %v", err)
	}
	if err := json.Unmarshal(legacy, &w); err != nil {
		t.Fatalf("JSON del legado inválido: %v", err)
	}
	return string(b), reflect.DeepEqual(g, w)
}

// Sin punto de inflexión ni día de retiro hay una sola guía, sin grosor ni texto, y lineThickness 0
// sale en el JSON. CA-03.4
func TestChartGuidesWithoutInflection(t *testing.T) {
	base := time.Date(2026, 10, 13, 15, 0, 0, 0, time.UTC)
	want := `[{"date":"2026-10-13","lineColor":"#CC0000","lineAlpha":1,"fillAlpha":0.2,"fillColor":"#CC0000",` +
		`"dashLength":10,"inside":true,"labelRotation":90,"lineThickness":0,"label":""}]`
	tests := []struct {
		name  string
		build guidesBuilder
	}{
		{name: "bondObject", build: shared_domain.BuildObjectGuides},
		{name: "bondBound", build: shared_domain.BuildBoundGuides},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.build(guideTerms(365, 0, 0), base, -5))
			if err != nil {
				t.Fatalf("no se pudieron serializar las guías: %v", err)
			}
			if string(got) != want {
				t.Fatalf("guías = %s\nse esperaba %s", got, want)
			}
		})
	}
}

// La zona de retiro del bondObject va de hoy + initialDayForWithdraw al día anterior al vencimiento,
// contando los días en la zona utcDiff. CA-03.5
func TestBuildObjectGuidesWithdrawZone(t *testing.T) {
	tests := []struct {
		name    string
		at      string
		utcDiff int
		date    string
		toDate  string
	}{
		{name: "mediodía del 13 en UTC-5", at: "2026-10-13T15:00:00Z", utcDiff: -5, date: "2026-11-12", toDate: "2027-10-12"},
		{name: "madrugada UTC del 14, todavía 13 en UTC-5", at: "2026-10-14T03:00:00Z", utcDiff: -5, date: "2026-11-12", toDate: "2027-10-12"},
		{name: "el mismo instante en UTC ya es 14", at: "2026-10-14T03:00:00Z", utcDiff: 0, date: "2026-11-13", toDate: "2027-10-13"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guides := shared_domain.BuildObjectGuides(guideTerms(365, 0, 30), mustParse(t, tt.at), tt.utcDiff)
			if len(guides) != 2 {
				t.Fatalf("hay %d guías, se esperaban 2", len(guides))
			}
			zone := guides[1]
			if zone.Date != tt.date || zone.ToDate != tt.toDate {
				t.Errorf("zona de retiro %s → %s, se esperaba %s → %s", zone.Date, zone.ToDate, tt.date, tt.toDate)
			}
			if zone.FillColor != "#4ec74c" || zone.Position != "top" || zone.LabelRotation != 0 ||
				zone.LineThickness != nil || zone.Label != "Zona de Retiro" {
				t.Errorf("zona de retiro con formato inesperado: %+v", zone)
			}
		})
	}
}

// Las guías del bondObject y del bondBound son las mismas que arman getDataChart y getChartGuides del
// legado; las del bondBound cuentan los días desde la compra en la zona de la sesión. CA-03.4, CA-03.5,
// CA-03.8
func TestChartGuidesMatchLegacy(t *testing.T) {
	// salida de getDataChart (bond.object.js) y getChartGuides (bond.bound.js) evaluados con Node y
	// moment; getChartGuides corrió con el servidor en America/Bogota, que equivale a utcDiff -5
	raw, err := os.ReadFile("testdata/legacy-guides.json")
	if err != nil {
		t.Fatalf("no se pudo leer la tabla del legado: %v", err)
	}
	var cases []struct {
		MaturityTimeInDays    int `json:"maturityTimeInDays"`
		InflectionPoint       int `json:"inflectionPoint"`
		InitialDayForWithdraw int `json:"initialDayForWithdraw"`
		Object                struct {
			Today   time.Time       `json:"today"`
			UtcDiff int             `json:"utcDiff"`
			Guides  json.RawMessage `json:"guides"`
		} `json:"object"`
		Bound []struct {
			CreatedAt time.Time       `json:"createdAt"`
			UtcDiff   int             `json:"utcDiff"`
			Guides    json.RawMessage `json:"guides"`
		} `json:"bound"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("tabla del legado inválida: %v", err)
	}
	if len(cases) < 40 {
		t.Fatalf("la tabla del legado tiene %d casos, se esperaban al menos 40", len(cases))
	}

	t.Run("bondObject", func(t *testing.T) {
		mismatches := 0
		for _, c := range cases {
			terms := guideTerms(c.MaturityTimeInDays, c.InflectionPoint, c.InitialDayForWithdraw)
			got, ok := sameJSON(t, shared_domain.BuildObjectGuides(terms, c.Object.Today, c.Object.UtcDiff), c.Object.Guides)
			if !ok {
				mismatches++
				if mismatches <= 10 {
					t.Errorf("M %d, P %d, W %d, hoy %s en %d: %s\nel legado da %s", c.MaturityTimeInDays, c.InflectionPoint,
						c.InitialDayForWithdraw, c.Object.Today.Format(time.RFC3339), c.Object.UtcDiff, got, c.Object.Guides)
				}
			}
		}
		if mismatches > 0 {
			t.Errorf("%d de %d casos difieren del legado", mismatches, len(cases))
		}
	})

	t.Run("bondBound", func(t *testing.T) {
		mismatches, total := 0, 0
		for _, c := range cases {
			terms := guideTerms(c.MaturityTimeInDays, c.InflectionPoint, c.InitialDayForWithdraw)
			for _, b := range c.Bound {
				total++
				got, ok := sameJSON(t, shared_domain.BuildBoundGuides(terms, b.CreatedAt, b.UtcDiff), b.Guides)
				if !ok {
					mismatches++
					if mismatches <= 10 {
						t.Errorf("M %d, P %d, W %d, compra %s en %d: %s\nel legado da %s", c.MaturityTimeInDays, c.InflectionPoint,
							c.InitialDayForWithdraw, b.CreatedAt.Format(time.RFC3339), b.UtcDiff, got, b.Guides)
					}
				}
			}
		}
		if total == 0 {
			t.Fatal("la tabla del legado no tiene guías de bondBound")
		}
		if mismatches > 0 {
			t.Errorf("%d de %d casos difieren del legado", mismatches, total)
		}
	})
}
