package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
)

// expectation es una diferencia con el legado que documentan las specs.
type expectation struct {
	id     string
	reason string
}

// Diferencias documentadas que el arnés reconoce.
var (
	expectTruncatedTerms = expectation{"O10", "el legado trunca con parseInt el monto o los porcentajes con decimales"}
	expectTruncatedRate  = expectation{"B14", "el legado trunca con parseInt la tasa con decimales"}
	expectMaturedProfit  = expectation{"B1", "desde el vencimiento Go informa la ganancia completa y el legado, la penalizada"}
	expectMaturedStatus  = expectation{"B37", "Go cotiza un bono madurado como vencido aunque no haya llegado su día en la zona"}

	expectations = []expectation{expectTruncatedTerms, expectTruncatedRate, expectMaturedProfit, expectMaturedStatus}
)

// Campos que se comparan en cada respuesta.
var (
	simFields      = []string{"inversion", "ganancia", "perdida"}
	preSaleFields  = []string{"inversion", "interest", "penaltyPercentage", "profit", "atMoment", "amount", "totalDaysPassed"}
	feedFields     = []string{"inversion", "ganancia", "perdida", "atMoment", "perdidaAtMoment", "bulletClass", "gananciaFinal"}
	progressFields = []string{"growing", "grew", "initialDateFormatted", "maturityDateFormatted"}
	accruedFields  = []string{"daysPassed", "acumulateAmount"}

	// moneyFields son los montos de la cotización que cambian con la tasa.
	moneyFields = []string{"profit", "atMoment", "amount"}
)

// Respuestas de los dos lados para un bono que todavía no se puede retirar.
const (
	withdrawCode  = -2030
	withdrawError = "ERR_INVALID_WITHDRAW_DATE"
)

// result es el resultado de una comparación: los casos iguales, los que solo tienen diferencias documentadas y
// los que tienen alguna sin documentar.
type result struct {
	name            string
	cases           int
	equal           int
	expectedCases   int
	expected        map[string]int
	expectedDiffs   []diff
	unexpectedCases int
	unexpected      []diff
}

func newResult(name string) *result {
	return &result{name: name, expected: map[string]int{}}
}

// add registra un caso con sus diferencias; explain devuelve la diferencia documentada que explica cada una, o
// nil.
func (r *result) add(diffs []diff, explain func(diff) *expectation) {
	r.cases++
	if len(diffs) == 0 {
		r.equal++
		return
	}

	ids := map[string]bool{}
	var explained, unexplained []diff
	for _, d := range diffs {
		if explain != nil {
			if e := explain(d); e != nil {
				d.Expected = e.id
				ids[e.id] = true
				explained = append(explained, d)
				continue
			}
		}
		unexplained = append(unexplained, d)
	}
	r.expectedDiffs = append(r.expectedDiffs, explained...)
	if len(unexplained) > 0 {
		r.unexpectedCases++
		r.unexpected = append(r.unexpected, unexplained...)
		return
	}
	r.expectedCases++
	for id := range ids {
		r.expected[id]++
	}
}

// runEnv son los clientes y las sesiones de la corrida.
type runEnv struct {
	meteor     *ddpClient
	http       *httpClient
	sessions   *sessions
	adminToken string
}

// check es una de las comparaciones de la corrida.
type check struct {
	name string
	run  func(ctx context.Context, env *runEnv, s sample) (*result, error)
}

var checks = []check{
	{name: "sim", run: checkSimulation},
	{name: "presale", run: checkPreSale},
	{name: "feed", run: checkBoundFeed},
	{name: "list", run: checkListing},
}

// legacyFailure es un error del legado: un Response.error o un método que falló.
type legacyFailure struct {
	code    any
	message string
}

// goFailure es una respuesta del API Go con un status distinto de 200.
type goFailure struct {
	status  int
	message string
}

// chartResponse son las guías y los puntos de una gráfica.
type chartResponse struct {
	Guides []map[string]any `json:"guides"`
	Data   []map[string]any `json:"data"`
}

// callLegacy llama a method de Meteor con la sesión de token y decodifica su respuesta, ya sin envoltorio, en
// out. Un método que falla vuelve como legacyFailure; un error es de la conexión y corta la corrida.
func (env *runEnv) callLegacy(ctx context.Context, method string, params map[string]any, token string, out any) (*legacyFailure, error) {
	params["extraData"] = token
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	raw, err := env.meteor.call(ctx, method, params)
	var me *methodError
	if errors.As(err, &me) {
		return &legacyFailure{code: me.Code, message: me.Error()}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	v, failure, err := legacyValue(raw)
	if err != nil || failure != nil {
		return failure, err
	}
	return nil, remarshal(v, out)
}

// legacyValue desenvuelve una respuesta del legado: con Response.ok devuelve su data, con Response.error la
// falla y, sin envoltorio, la respuesta misma.
func legacyValue(raw json.RawMessage) (any, *legacyFailure, error) {
	if len(raw) == 0 {
		return nil, nil, nil
	}
	var v any
	if err := decodeJSON(raw, &v); err != nil {
		return nil, nil, err
	}
	m, ok := v.(map[string]any)
	if !ok {
		return v, nil, nil
	}
	success, wrapped := m["success"].(bool)
	switch {
	case !wrapped:
		return v, nil, nil
	case !success:
		return nil, &legacyFailure{code: m["code"], message: fmt.Sprint(m["message"])}, nil
	default:
		return m["data"], nil, nil
	}
}

// remarshal pasa v, decodificado como any, a out.
func remarshal(v any, out any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return decodeJSON(data, out)
}

// getGo hace GET a path del API Go con la sesión de token y decodifica la respuesta en out. Otro status que
// 200 vuelve como goFailure; un error es de la conexión y corta la corrida.
func (env *runEnv) getGo(ctx context.Context, path, token string, out any) (*goFailure, error) {
	res, err := env.http.goGet(ctx, path, token)
	if err != nil {
		return nil, fmt.Errorf("API Go: %w", err)
	}
	if res.status != http.StatusOK {
		return &goFailure{status: res.status, message: apiErrorMessage(res.body)}, nil
	}
	return nil, decodeJSON(res.body, out)
}

// failureDiff es la diferencia de un caso en el que alguno de los dos lados respondió con error.
func failureDiff(caseID string, legacy *legacyFailure, goSide *goFailure) diff {
	d := diff{Case: caseID, Field: "respuesta", Legacy: "ok", Go: "ok"}
	if legacy != nil {
		d.Legacy = fmt.Sprintf("error %v: %s", legacy.code, legacy.message)
	}
	if goSide != nil {
		d.Go = fmt.Sprintf("error %d: %s", goSide.status, goSide.message)
	}
	return d
}

// checkSimulation compara la simulación de cada producto: getDataChart con los términos guardados, como lo
// pide la vista previa, contra la gráfica del API Go con simulation=true.
func checkSimulation(ctx context.Context, env *runEnv, s sample) (*result, error) {
	r := newResult("sim")
	for _, o := range s.objects {
		caseID := "bondObject " + o.ID
		var legacy, goChart chartResponse
		lf, err := env.callLegacy(ctx, "bondObject.getDataChart",
			map[string]any{"preview": true, "simulation": true, "bondObject": o.Terms}, env.adminToken, &legacy)
		if err != nil {
			return nil, err
		}
		gf, err := env.getGo(ctx, "/bond-objects/"+url.PathEscape(o.ID)+"/chart?simulation=true", env.adminToken, &goChart)
		if err != nil {
			return nil, err
		}
		if lf != nil || gf != nil {
			r.add([]diff{failureDiff(caseID, lf, gf)}, nil)
			continue
		}

		diffs := compareSeries(caseID, legacy.Data, goChart.Data, simFields)
		diffs = append(diffs, compareGuides(caseID, legacy.Guides, goChart.Guides)...)
		r.add(diffs, simExplain(o.Terms))
	}
	return r, nil
}

// simExplain explica las diferencias en los puntos de la simulación de un producto con monto o porcentajes con
// decimales. Las guías no dependen de esos valores.
func simExplain(t storedTerms) func(diff) *expectation {
	truncated := hasDecimals(t.Amount, t.MaturityRewardsPercentage,
		t.EarlyCashTerm.InitialPenaltyPercentage, t.EarlyCashTerm.InflectionPenaltyPercentage)
	return func(d diff) *expectation {
		if truncated && slices.Contains(simFields, d.Field) {
			return &expectTruncatedTerms
		}
		return nil
	}
}

// checkPreSale compara la cotización de cada bono de la muestra, con la sesión del admin. Si los dos lados
// rechazan la fecha de retiro, el caso es igual.
func checkPreSale(ctx context.Context, env *runEnv, s sample) (*result, error) {
	r := newResult("presale")
	for _, b := range s.bounds {
		caseID := "bondBound " + b.ID
		var legacy, goQuote map[string]any
		lf, err := env.callLegacy(ctx, "bondBound.preSale",
			map[string]any{"bondBoundId": b.ID, "raw": true}, env.adminToken, &legacy)
		if err != nil {
			return nil, err
		}
		gf, err := env.getGo(ctx, "/bond-bounds/"+url.PathEscape(b.ID)+"/pre-sale", env.adminToken, &goQuote)
		if err != nil {
			return nil, err
		}

		switch {
		case lf != nil && gf != nil && bothRejectWithdraw(lf, gf):
			r.add(nil, nil)
		case lf != nil || gf != nil:
			r.add([]diff{failureDiff(caseID, lf, gf)}, nil)
		default:
			r.add(compareFields(caseID, "", legacy, goQuote, preSaleFields), preSaleExplain(b, legacy))
		}
	}
	return r, nil
}

// bothRejectWithdraw dice si el legado y Go rechazaron la cotización porque todavía no se puede retirar.
func bothRejectWithdraw(legacy *legacyFailure, goSide *goFailure) bool {
	code, ok := num(legacy.code)
	return ok && code == withdrawCode &&
		goSide.status == http.StatusUnprocessableEntity && goSide.message == withdrawError
}

// preSaleExplain explica las diferencias de la cotización de b: un bono madurado antes de su día, la ganancia
// desde el vencimiento y los montos de una tasa con decimales.
func preSaleExplain(b boundDoc, legacy map[string]any) func(diff) *expectation {
	days, _ := num(legacy["totalDaysPassed"])
	maturity, _ := num(b.Terms.MaturityTimeInDays)
	maturity = math.Trunc(maturity)
	rateTruncated := hasDecimals(b.Terms.MaturityRewardsPercentage)
	return func(d diff) *expectation {
		switch {
		case b.StatusID == "2" && days < maturity:
			return &expectMaturedStatus
		case days >= maturity && d.Field == "profit":
			return &expectMaturedProfit
		case rateTruncated && slices.Contains(moneyFields, d.Field):
			return &expectTruncatedRate
		}
		return nil
	}
}

// checkBoundFeed compara la evolución de cada bono de la muestra: el feed de ms-feed contra la gráfica del API
// Go con la sesión del admin.
func checkBoundFeed(ctx context.Context, env *runEnv, s sample) (*result, error) {
	r := newResult("feed")
	for _, b := range s.bounds {
		caseID := "bondBound " + b.ID
		res, err := env.http.feedGet(ctx, "/bound/"+url.PathEscape(b.ID)+"/feed")
		if err != nil {
			return nil, fmt.Errorf("ms-feed: %w", err)
		}
		var legacy []map[string]any
		var lf *legacyFailure
		if res.status != http.StatusOK {
			lf = &legacyFailure{code: res.status, message: string(res.body)}
		} else if err := decodeJSON(res.body, &legacy); err != nil {
			return nil, fmt.Errorf("ms-feed, bono %s: %w", b.ID, err)
		}
		var goChart chartResponse
		gf, err := env.getGo(ctx, "/bond-bounds/"+url.PathEscape(b.ID)+"/chart", env.adminToken, &goChart)
		if err != nil {
			return nil, err
		}
		if lf != nil || gf != nil {
			r.add([]diff{failureDiff(caseID, lf, gf)}, nil)
			continue
		}
		r.add(compareSeries(caseID, legacy, goChart.Data, feedFields), feedExplain(b.Terms))
	}
	return r, nil
}

// feedExplain explica las diferencias en la evolución de un bono con porcentajes con decimales.
func feedExplain(t storedTerms) func(diff) *expectation {
	truncated := hasDecimals(t.MaturityRewardsPercentage,
		t.EarlyCashTerm.InitialPenaltyPercentage, t.EarlyCashTerm.InflectionPenaltyPercentage)
	return func(d diff) *expectation {
		if truncated && d.Field != "date" && d.Field != "bulletClass" {
			return &expectTruncatedTerms
		}
		return nil
	}
}

// checkListing compara los derivados del listado de cada dueño de la muestra, con su propia sesión: el estado
// de crecimiento y las fechas del listado del usuario, y los días y el monto acumulados del detalle de cada
// bono, contra los bonos iniciados y madurados del usuario en el API Go.
func checkListing(ctx context.Context, env *runEnv, s sample) (*result, error) {
	r := newResult("list")
	for _, owner := range s.owners {
		token, err := env.sessions.create(ctx, owner)
		if err != nil {
			return nil, err
		}
		goItems, gf, err := env.listMine(ctx, token)
		if err != nil {
			return nil, err
		}
		var legacyItems []map[string]any
		lf, err := env.callLegacy(ctx, "bondBound.listByUserlogged",
			map[string]any{"statusId": []string{"1", "2"}}, token, &legacyItems)
		if err != nil {
			return nil, err
		}
		if lf != nil || gf != nil {
			r.add([]diff{failureDiff("usuario "+owner, lf, gf)}, nil)
			continue
		}

		legacyByID := make(map[string]map[string]any, len(legacyItems))
		for _, item := range legacyItems {
			legacyByID[fmt.Sprint(item["_id"])] = item
		}
		for _, item := range goItems {
			id := fmt.Sprint(item["_id"])
			caseID := "bondBound " + id
			legacy, ok := legacyByID[id]
			if !ok {
				r.add([]diff{{Case: caseID, Field: "listado", Go: "presente"}}, nil)
				continue
			}
			delete(legacyByID, id)

			diffs := compareFields(caseID, "", legacy, item, progressFields)
			var details map[string]any
			df, err := env.callLegacy(ctx, "bondBound.getDetails", map[string]any{"bondBoundId": id}, token, &details)
			if err != nil {
				return nil, err
			}
			if df != nil {
				diffs = append(diffs, failureDiff(caseID, df, nil))
			} else {
				diffs = append(diffs, compareFields(caseID, "", details, item, accruedFields)...)
			}
			r.add(diffs, nil)
		}

		missing := make([]string, 0, len(legacyByID))
		for id := range legacyByID {
			missing = append(missing, id)
		}
		sort.Strings(missing)
		for _, id := range missing {
			r.add([]diff{{Case: "bondBound " + id, Field: "listado", Legacy: "presente"}}, nil)
		}
	}
	return r, nil
}

// listMine devuelve los bonos iniciados y madurados del usuario de token, recorriendo todas las páginas.
func (env *runEnv) listMine(ctx context.Context, token string) ([]map[string]any, *goFailure, error) {
	var items []map[string]any
	for page := 1; ; page++ {
		var res struct {
			Items      []map[string]any `json:"items"`
			TotalPages int              `json:"totalPages"`
		}
		path := fmt.Sprintf("/bond-bounds/me?statusId=1&statusId=2&page=%d&limit=100", page)
		gf, err := env.getGo(ctx, path, token, &res)
		if err != nil || gf != nil {
			return nil, gf, err
		}
		items = append(items, res.Items...)
		if page >= res.TotalPages {
			return items, nil, nil
		}
	}
}

// selectChecks devuelve las comparaciones de names, en el orden de checks.
func selectChecks(names []string) ([]check, error) {
	wanted := map[string]bool{}
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			wanted[n] = true
		}
	}
	var selected []check
	for _, c := range checks {
		if wanted[c.name] {
			selected = append(selected, c)
			delete(wanted, c.name)
		}
	}
	if len(wanted) > 0 || len(selected) == 0 {
		return nil, fmt.Errorf("-only acepta sim, presale, feed y list")
	}
	return selected, nil
}
