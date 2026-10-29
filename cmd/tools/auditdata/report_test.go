package main

import (
	"bytes"
	"strings"
	"testing"
)

// Queda algo pendiente si hay algo que normaliza -apply, settings duplicados, revisión manual o bloqueantes; las
// claves repetidas, los bonos cobrados dos veces y la maduración solo se informan.
func TestPending(t *testing.T) {
	informative := &audit{
		Keys:     []duplicateKey{{Key: "B001-1731"}},
		Sold:     soldAudit{Paid: []soldBound{{}}},
		Maturity: maturityAudit{Bonds: 395, Dropped: []docIssue{{}}},
	}
	if informative.pending() {
		t.Error("lo que solo se informa no deja pendientes")
	}

	cases := map[string]*audit{
		"settings":        {Settings: settingsAudit{Groups: [][]string{{"a", "b"}}}},
		"numéricos":       {Numeric: numericAudit{Plans: []numericPlan{{}}}},
		"no canónicos":    {Numeric: numericAudit{NonCanonical: []docIssue{{}}}},
		"privados":        {Private: privateAudit{Auto: []privatePlan{{}}}},
		"privados a mano": {Private: privateAudit{Manual: []docIssue{{}}}},
		"vendidos":        {Sold: soldAudit{Fixable: []soldBound{{}}}},
		"vendidos a mano": {Sold: soldAudit{Manual: []docIssue{{}}}},
		"bloqueantes":     {Blockers: []blocker{{Kind: blockUndecodable}}},
	}
	for name, a := range cases {
		if !a.pending() {
			t.Errorf("%s: no quedó pendiente", name)
		}
	}
}

// La tabla de resumen es Markdown, con una fila por revisión y la columna de pendientes.
func TestPrintSummary(t *testing.T) {
	a := &audit{
		Keys: []duplicateKey{{Key: "B001-1731", Bounds: []docIssue{{}, {}}}},
		Sold: soldAudit{Fixable: make([]soldBound, 290)},
	}
	var buf bytes.Buffer
	printSummary(&buf, "Resumen", a)
	out := buf.String()
	for _, want := range []string{
		"## Resumen",
		"| Revisión | Hallazgos | Acción | Pendiente |",
		"| Claves repetidas (B8) | 1 clave, 2 bonos | Se informan; no se renumeran | no |",
		"| `'2'` con `soldDate` (B22) | 290 pasan a `'5'`, 0 manuales | `-apply` (vendidos); los manuales, a mano | sí |",
		"| Bloqueantes de Go | 0 | A mano | no |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en\n%s", want, out)
		}
	}
	if rows := strings.Count(out, "\n|"); rows != 10 {
		t.Errorf("filas de la tabla = %d, se esperaban 10 (cabecera, separador y 8 revisiones)", rows)
	}
}

// Los montos van con punto de miles y coma decimal.
func TestFormatAmount(t *testing.T) {
	cases := map[float64]string{
		0:          "0",
		999:        "999",
		1000:       "1.000",
		33600000:   "33.600.000",
		1037.39:    "1.037,39",
		-1500.5:    "-1.500,50",
		0.004:      "0",
		1234567.89: "1.234.567,89",
	}
	for v, want := range cases {
		if got := formatAmount(v); got != want {
			t.Errorf("formatAmount(%v) = %q, se esperaba %q", v, got, want)
		}
	}
}

// Sin -v se listan hasta veinte documentos y se avisa cuántos faltan.
func TestPrintIDsLimit(t *testing.T) {
	ids := make([]string, 25)
	for i := range ids {
		ids[i] = "id"
	}
	var buf bytes.Buffer
	printIDs(&buf, "  ", ids, false)
	if got := strings.Count(buf.String(), "id"); got != maxListed {
		t.Errorf("se listaron %d ids, se esperaban %d", got, maxListed)
	}
	if !strings.Contains(buf.String(), "… y 5 más") {
		t.Errorf("falta el aviso de los que no se listan:\n%s", buf.String())
	}

	buf.Reset()
	printIDs(&buf, "  ", ids, true)
	if got := strings.Count(buf.String(), "id"); got != 25 {
		t.Errorf("con -v se listaron %d ids, se esperaban 25", got)
	}
}

// Los bloqueantes de perfiles y cuentas no muestran el _id: se agrupan por colección y causa.
func TestCausesHideIDs(t *testing.T) {
	a := &audit{}
	a.block(blockUndecodable, profileCollection, "UserIdPersonaUno", "id: int")
	a.block(blockUndecodable, profileCollection, "UserIdPersonaDos", "id: int")
	a.block(blockUndecodable, bondBoundCollection, "Bd9Xc4Vn7Mz2Lk5Jh", "amount: string")

	var buf bytes.Buffer
	printAudit(&buf, a, true)
	out := buf.String()
	if strings.Contains(out, "UserIdPersona") {
		t.Errorf("la salida muestra el _id de un perfil:\n%s", out)
	}
	for _, want := range []string{"profile con id: int: 2", "bondBound Bd9Xc4Vn7Mz2Lk5Jh: amount: string"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en\n%s", want, out)
		}
	}
}
