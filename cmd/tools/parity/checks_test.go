package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// legacyValue saca la data de Response.ok, convierte Response.error en una falla y deja igual una respuesta
// sin envoltorio, como el objeto de getDataChart o el array de listByUserlogged.
func TestLegacyValue(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		want        any
		wantFailure *legacyFailure
	}{
		{name: "Response.ok", raw: `{"success":true,"code":200,"message":"ok","data":{"profit":5}}`,
			want: map[string]any{"profit": json.Number("5")}},
		{name: "Response.error", raw: `{"success":false,"code":-2030,"message":"El bono no puede ser vendido","stack":null}`,
			wantFailure: &legacyFailure{code: json.Number("-2030"), message: "El bono no puede ser vendido"}},
		{name: "objeto sin envoltorio", raw: `{"guides":[],"data":[]}`,
			want: map[string]any{"guides": []any{}, "data": []any{}}},
		{name: "array", raw: `[{"_id":"b1"}]`, want: []any{map[string]any{"_id": "b1"}}},
		{name: "sin resultado", raw: ``},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, failure, err := legacyValue(json.RawMessage(tt.raw))
			if err != nil {
				t.Fatalf("legacyValue: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) || !reflect.DeepEqual(failure, tt.wantFailure) {
				t.Errorf("legacyValue = %v, %+v; se esperaba %v, %+v", got, failure, tt.want, tt.wantFailure)
			}
		})
	}
}

// Un caso sin diferencias es igual; con todas explicadas cuenta en cada diferencia documentada que aparece;
// con alguna sin explicar es un caso no documentado, que lista solo esas.
func TestResultAdd(t *testing.T) {
	explainProfit := func(d diff) *expectation {
		if d.Field == "profit" {
			return &expectMaturedProfit
		}
		return nil
	}
	r := newResult("presale")

	r.add(nil, explainProfit)
	r.add([]diff{{Case: "b2", Field: "profit"}}, explainProfit)
	r.add([]diff{{Case: "b3", Field: "profit"}, {Case: "b3", Field: "amount"}}, explainProfit)

	if r.cases != 3 || r.equal != 1 || r.expectedCases != 1 || r.unexpectedCases != 1 {
		t.Errorf("casos %d, iguales %d, documentados %d, no documentados %d; se esperaba 3, 1, 1, 1",
			r.cases, r.equal, r.expectedCases, r.unexpectedCases)
	}
	if want := map[string]int{"B1": 1}; !reflect.DeepEqual(r.expected, want) {
		t.Errorf("por diferencia documentada %v, se esperaba %v", r.expected, want)
	}
	if want := []diff{{Case: "b3", Field: "amount"}}; !reflect.DeepEqual(r.unexpected, want) {
		t.Errorf("no documentadas %v, se esperaba %v", r.unexpected, want)
	}
	if len(r.expectedDiffs) != 2 || r.expectedDiffs[0].Expected != "B1" {
		t.Errorf("documentadas %v, se esperaban las 2 de profit con B1", r.expectedDiffs)
	}
}

// La cotización de un bono madurado antes de su día se explica entera; la ganancia, desde el día M; y los
// montos, si la tasa tiene decimales.
func TestPreSaleExplain(t *testing.T) {
	bound := func(status string, rate any) boundDoc {
		return boundDoc{ID: "b1", StatusID: status, Terms: storedTerms{MaturityTimeInDays: "100", MaturityRewardsPercentage: rate}}
	}
	tests := []struct {
		name  string
		bound boundDoc
		days  string
		field string
		want  *expectation
	}{
		{name: "madurado antes de su día", bound: bound("2", int32(10)), days: "99", field: "amount", want: &expectMaturedStatus},
		{name: "ganancia desde el vencimiento", bound: bound("1", int32(10)), days: "100", field: "profit", want: &expectMaturedProfit},
		{name: "monto desde el vencimiento", bound: bound("1", int32(10)), days: "100", field: "amount"},
		{name: "monto con tasa con decimales", bound: bound("1", 12.5), days: "50", field: "atMoment", want: &expectTruncatedRate},
		{name: "tasa con decimales no cambia los días", bound: bound("1", 12.5), days: "50", field: "totalDaysPassed"},
		{name: "ganancia antes del vencimiento", bound: bound("1", int32(10)), days: "50", field: "profit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			explain := preSaleExplain(tt.bound, map[string]any{"totalDaysPassed": json.Number(tt.days)})
			if got := explain(diff{Field: tt.field}); got != tt.want {
				t.Errorf("explicación %v, se esperaba %v", got, tt.want)
			}
		})
	}
}

// Con monto o porcentajes con decimales se explican los puntos de la simulación, no las guías.
func TestSimExplain(t *testing.T) {
	explain := simExplain(storedTerms{Amount: "1000.50", MaturityRewardsPercentage: int32(10)})
	if got := explain(diff{Field: "ganancia"}); got != &expectTruncatedTerms {
		t.Errorf("ganancia: %v, se esperaba la truncación de términos", got)
	}
	if got := explain(diff{Where: "guía 1", Field: "date"}); got != nil {
		t.Errorf("guía: %v, se esperaba sin explicación", got)
	}
	if got := simExplain(storedTerms{Amount: int32(1000)})(diff{Field: "ganancia"}); got != nil {
		t.Errorf("sin decimales: %v, se esperaba sin explicación", got)
	}
}

// Con porcentajes con decimales se explican los valores de la evolución, no las fechas ni el punto de hoy.
func TestFeedExplain(t *testing.T) {
	explain := feedExplain(storedTerms{Amount: int32(1000), EarlyCashTerm: storedTermsEarlyCash{InflectionPenaltyPercentage: "50.5"}})
	if got := explain(diff{Field: "perdida"}); got != &expectTruncatedTerms {
		t.Errorf("perdida: %v, se esperaba la truncación de términos", got)
	}
	for _, field := range []string{"date", "bulletClass"} {
		if got := explain(diff{Field: field}); got != nil {
			t.Errorf("%s: %v, se esperaba sin explicación", field, got)
		}
	}
}

// Solo es el mismo rechazo si el legado responde -2030 y Go, 422 ERR_INVALID_WITHDRAW_DATE.
func TestBothRejectWithdraw(t *testing.T) {
	legacy := &legacyFailure{code: json.Number("-2030")}
	if !bothRejectWithdraw(legacy, &goFailure{status: http.StatusUnprocessableEntity, message: "ERR_INVALID_WITHDRAW_DATE"}) {
		t.Error("-2030 y 422 ERR_INVALID_WITHDRAW_DATE son el mismo rechazo")
	}
	if bothRejectWithdraw(legacy, &goFailure{status: http.StatusConflict, message: "ERR_BOND_STATUS_NOT_SELLABLE"}) {
		t.Error("un 409 de Go no es el rechazo por fecha")
	}
	if bothRejectWithdraw(&legacyFailure{code: json.Number("500")}, &goFailure{status: http.StatusUnprocessableEntity, message: "ERR_INVALID_WITHDRAW_DATE"}) {
		t.Error("un 500 del legado no es el rechazo por fecha")
	}
}

// -only acepta las comparaciones en cualquier orden y las devuelve en el de la corrida; rechaza un nombre
// desconocido y una lista vacía.
func TestSelectChecks(t *testing.T) {
	got, err := selectChecks([]string{"presale", " sim"})
	if err != nil || len(got) != 2 || got[0].name != "sim" || got[1].name != "presale" {
		t.Errorf("selectChecks = %v, %v; se esperaba sim y presale", got, err)
	}
	for _, names := range [][]string{{"sim", "otra"}, {""}} {
		if _, err := selectChecks(names); err == nil {
			t.Errorf("selectChecks(%q) debería fallar", names)
		}
	}
}

// El reporte resume cada comparación, lista hasta 20 diferencias no documentadas, explica las documentadas que
// aparecieron y devuelve los casos no documentados.
func TestReport(t *testing.T) {
	r := newResult("feed")
	for i := 0; i < 25; i++ {
		r.add([]diff{{Case: "bondBound b", Field: "perdida"}}, nil)
	}
	r.add([]diff{{Case: "bondBound c", Field: "ganancia"}}, func(diff) *expectation { return &expectTruncatedTerms })

	var out bytes.Buffer
	got := report(&out, []*result{r}, false)

	if got != 25 {
		t.Errorf("report = %d, se esperaban 25 casos no documentados", got)
	}
	text := out.String()
	for _, want := range []string{
		"feed: 26 casos, 0 iguales, 1 con diferencias documentadas (O10: 1), 25 con diferencias no documentadas",
		"… y 5 diferencias más",
		"O10: " + expectTruncatedTerms.reason,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("el reporte no contiene %q:\n%s", want, text)
		}
	}
	if n := strings.Count(text, "no documentada:"); n != 20 {
		t.Errorf("se listaron %d diferencias no documentadas, se esperaban 20", n)
	}
}
