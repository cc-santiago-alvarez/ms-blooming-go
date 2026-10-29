package main

import (
	"fmt"
	"io"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

// maxListed es la cantidad de documentos que se listan por hallazgo sin -v.
const maxListed = 20

// printAudit muestra cada revisión con sus cantidades y los documentos que encontró.
func printAudit(w io.Writer, a *audit, verbose bool) {
	keyBonds := 0
	for _, k := range a.Keys {
		keyBonds += len(k.Bounds)
	}
	section(w, "Claves de bono repetidas (B8)")
	fmt.Fprintf(w, "%s, %s. No se renumeran.\n", plural(len(a.Keys), "clave repetida", "claves repetidas"), plural(keyBonds, "bono", "bonos"))
	for i, k := range a.Keys {
		if !verbose && i == maxListed {
			fmt.Fprintf(w, "  … y %d claves más (-v las lista)\n", len(a.Keys)-maxListed)
			break
		}
		fmt.Fprintf(w, "  %s: %s\n", k.Key, plural(len(k.Bounds), "bono", "bonos"))
		printIssues(w, "    ", k.Bounds, true)
	}

	section(w, "Settings duplicados (U3)")
	fmt.Fprintf(w, "%s con más de un setting vivo. Se corrigen con dedupusersettings -apply, que conserva el primero de cada grupo.\n",
		plural(len(a.Settings.Groups), "usuario", "usuarios"))
	for i, g := range a.Settings.Groups {
		if !verbose && i == maxListed {
			fmt.Fprintf(w, "  … y %d usuarios más (-v los lista)\n", len(a.Settings.Groups)-maxListed)
			break
		}
		fmt.Fprintf(w, "  conserva %s; borraría %s\n", g[0], strings.Join(g[1:], ", "))
	}
	fmt.Fprintf(w, "Settings vivos sin userId: %d.\n", len(a.Settings.NullUser))
	printIDs(w, "  ", a.Settings.NullUser, verbose)
	fmt.Fprintf(w, "Settings vivos con un userId que no es string: %d.\n", len(a.Settings.NonString))
	printIDs(w, "  ", a.Settings.NonString, verbose)

	section(w, "Numéricos guardados como string")
	fmt.Fprintln(w, "Tipos por campo:")
	for _, field := range slices.Sorted(maps.Keys(a.Numeric.Census)) {
		fmt.Fprintf(w, "  %s: %s\n", field, census(a.Numeric.Census[field]))
	}
	fmt.Fprintf(w, "Documentos con strings canónicos que -apply guarda como número: bondObject %d, bondBound %d.\n",
		a.Numeric.count(bondObjectCollection), a.Numeric.count(bondBoundCollection))
	fmt.Fprintf(w, "Strings no canónicos, para revisión manual: %d.\n", len(a.Numeric.NonCanonical))
	printIssues(w, "  ", a.Numeric.NonCanonical, verbose)

	section(w, "isPrivateBond que no es booleano (O11)")
	fmt.Fprintf(w, "Productos que -apply guarda como booleano, con el valor con el que ya los leen Go y Meteor: %d.\n", len(a.Private.Auto))
	auto := make([]docIssue, len(a.Private.Auto))
	for i, p := range a.Private.Auto {
		auto[i] = docIssue{Collection: bondObjectCollection, ID: idString(p.ID), Detail: fmt.Sprintf("%s → %v", p.Detail, p.Target)}
	}
	printIssues(w, "  ", auto, verbose)
	fmt.Fprintf(w, "Para revisión manual: %d.\n", len(a.Private.Manual))
	printIssues(w, "  ", a.Private.Manual, verbose)
	fmt.Fprintf(w, "Productos sin el campo, que Go y Meteor leen como públicos y no se tocan: %d.\n", a.Private.Missing)
	fmt.Fprintf(w, "Productos privados sin una lista en authorizedUserIds, cuyo tipo Meteor no puede listar: %d.\n", len(a.Private.NoAuthorized))
	printIssues(w, "  ", a.Private.NoAuthorized, verbose)

	section(w, "Bonos madurados que ya estaban vendidos (B22)")
	bonds, owners, total := soldSummary(a.Sold.Fixable)
	removed := 0
	for _, b := range a.Sold.Fixable {
		if b.IsRemove {
			removed++
		}
	}
	fmt.Fprintf(w, "Bonos '2' con soldDate que -apply pasa a '5': %d, de %s, Σ amount %s (borrados: %d). Meteor todavía los paga al canjearlos.\n",
		bonds, plural(owners, "dueño", "dueños"), formatAmount(total), removed)
	printYears(w, "Por año de soldDate:", byYear(a.Sold.Fixable, func(b soldBound) time.Time { return b.SoldDate }))
	printIDs(w, "  ", soldIDs(a.Sold.Fixable), verbose)
	fmt.Fprintf(w, "Para revisión manual: %d.\n", len(a.Sold.Manual))
	printIssues(w, "  ", a.Sold.Manual, verbose)

	section(w, "Bonos vendidos y canjeados: cobrados dos veces")
	bonds, owners, total = soldSummary(a.Sold.Paid)
	removed = 0
	for _, b := range a.Sold.Paid {
		if b.IsRemove {
			removed++
		}
	}
	fmt.Fprintf(w, "Bonos '6' con soldDate y tookDate: %d, de %s, Σ amount %s (borrados: %d). Solo se informan, para contabilidad.\n",
		bonds, plural(owners, "dueño", "dueños"), formatAmount(total), removed)
	fmt.Fprintf(w, "Lo que cuenta B_BondBound_Audit_doubleProcess del gateway, la línea base del monitoreo: %s de %s con perfil.\n",
		plural(a.Sold.AuditBonds, "bono", "bonos"), plural(a.Sold.AuditOwners, "dueño", "dueños"))
	printYears(w, "Por año de tookDate:", byYear(a.Sold.Paid, func(b soldBound) time.Time {
		if b.TookDate == nil {
			return time.Time{}
		}
		return *b.TookDate
	}))
	printIDs(w, "  ", soldIDs(a.Sold.Paid), verbose)

	section(w, "Primera corrida de la maduración de Go (B36)")
	fmt.Fprintf(w, "Referencia %s, corte %s. Bonos '1' vencidos: %d, de %s.\n",
		a.At.Format(time.RFC3339), a.Cutoff.Format(time.RFC3339), a.Maturity.Bonds, plural(a.Maturity.Owners, "dueño", "dueños"))
	fmt.Fprintf(w, "Que ms-feed nunca madura, porque falta su profile, su bondType o su bondObject: %d, de %s.\n",
		len(a.Maturity.Dropped), plural(a.Maturity.DroppedOwners, "dueño", "dueños"))
	printIssues(w, "  ", a.Maturity.Dropped, verbose)
	fmt.Fprintf(w, "Que Go no decodifica, y cortan cada corrida en su lote (no maduran ni ese lote ni los siguientes): %d.\n", len(a.Maturity.Undecodable))
	printIDs(w, "  ", a.Maturity.Undecodable, verbose)

	section(w, "Bloqueantes de Go")
	fmt.Fprintf(w, "Documentos que Go no puede leer u operar: %d.\n", len(a.Blockers))
	for _, kind := range blockerKinds {
		issues := []docIssue{}
		for _, b := range a.Blockers {
			if b.Kind == kind {
				issues = append(issues, b.docIssue)
			}
		}
		if len(issues) == 0 {
			continue
		}
		fmt.Fprintf(w, "  %s: %d\n", kind, len(issues))
		printIssues(w, "    ", identified(issues), verbose)
		for _, line := range causes(issues) {
			fmt.Fprintf(w, "    %s\n", line)
		}
	}
}

// identified devuelve los hallazgos que se muestran con su _id.
func identified(issues []docIssue) []docIssue {
	out := []docIssue{}
	for _, is := range issues {
		if is.ID != "" {
			out = append(out, is)
		}
	}
	return out
}

// causes agrupa los hallazgos sin _id por colección y causa, con su cantidad.
func causes(issues []docIssue) []string {
	counts := map[string]int{}
	for _, is := range issues {
		if is.ID == "" {
			counts[is.Collection+" con "+is.Detail]++
		}
	}
	out := []string{}
	for _, key := range slices.Sorted(maps.Keys(counts)) {
		out = append(out, fmt.Sprintf("%s: %d", key, counts[key]))
	}
	return out
}

// printSummary muestra la tabla de resumen, en Markdown para poder copiarla al informe.
func printSummary(w io.Writer, title string, a *audit) {
	keyBonds := 0
	for _, k := range a.Keys {
		keyBonds += len(k.Bounds)
	}
	paid, paidOwners, _ := soldSummary(a.Sold.Paid)
	rows := [][]string{
		{"Claves repetidas (B8)", plural(len(a.Keys), "clave", "claves") + ", " + plural(keyBonds, "bono", "bonos"), "Se informan; no se renumeran", "no"},
		{"Settings duplicados (U3)", plural(len(a.Settings.Groups), "usuario", "usuarios"), "`dedupusersettings -apply`", yesNo(len(a.Settings.Groups) > 0)},
		{"Numéricos como string", plural(len(a.Numeric.Plans), "documento canónico", "documentos canónicos") + ", " + plural(len(a.Numeric.NonCanonical), "string no canónico", "strings no canónicos"),
			"`-apply` (numericos); los no canónicos, a mano", yesNo(len(a.Numeric.Plans)+len(a.Numeric.NonCanonical) > 0)},
		{"`isPrivateBond` no booleano (O11)", fmt.Sprintf("%d automáticos, %d manuales", len(a.Private.Auto), len(a.Private.Manual)),
			"`-apply` (privados); los manuales, a mano", yesNo(len(a.Private.Auto)+len(a.Private.Manual) > 0)},
		{"`'2'` con `soldDate` (B22)", fmt.Sprintf("%d pasan a `'5'`, %d manuales", len(a.Sold.Fixable), len(a.Sold.Manual)),
			"`-apply` (vendidos); los manuales, a mano", yesNo(len(a.Sold.Fixable)+len(a.Sold.Manual) > 0)},
		{"`'6'` cobrados dos veces", fmt.Sprintf("%s, %s; la auditoría del gateway cuenta %d", plural(paid, "bono", "bonos"), plural(paidOwners, "dueño", "dueños"), a.Sold.AuditBonds),
			"Se informan a contabilidad", "no"},
		{"Primera corrida de Go (B36)", fmt.Sprintf("%s de %s; ms-feed no madura %d", plural(a.Maturity.Bonds, "bono", "bonos"), plural(a.Maturity.Owners, "dueño", "dueños"), len(a.Maturity.Dropped)),
			"Los madura el corte de la maduración", "no"},
		{"Bloqueantes de Go", fmt.Sprint(len(a.Blockers)), "A mano", yesNo(len(a.Blockers) > 0)},
	}
	fmt.Fprintf(w, "\n## %s\n\n| Revisión | Hallazgos | Acción | Pendiente |\n|---|---|---|---|\n", title)
	for _, r := range rows {
		fmt.Fprintf(w, "| %s |\n", strings.Join(r, " | "))
	}
}

// printApply muestra lo que hizo cada normalización y dónde quedó el respaldo.
func printApply(w io.Writer, results []applyResult, backup *backupWriter, verbose bool) {
	section(w, "Aplicación")
	for _, r := range results {
		if r.Skipped {
			fmt.Fprintf(w, "%s: omitida por -only.\n", r.Fix)
			continue
		}
		fmt.Fprintf(w, "%s: %d planeados, %d aplicados, %d cambiaron durante la corrida, %d fallaron.\n",
			r.Fix, r.Planned, r.Applied, len(r.Changed), len(r.Failed))
		printIDs(w, "  cambiaron: ", r.Changed, verbose)
		printIssues(w, "  falló ", r.Failed, verbose)
	}
	if backup.changes == 0 {
		fmt.Fprintln(w, "No hubo cambios: no se creó el respaldo.")
		return
	}
	fmt.Fprintf(w, "Respaldo: %s (%d cambios). Para deshacerlos: -revert %s -apply -confirm <ENV>/<base>.\n",
		backup.path, backup.changes, backup.path)
}

// printRevert muestra lo que hizo, o haría sin apply, la reversión.
func printRevert(w io.Writer, res revertResult, apply, verbose bool) {
	verb := "Se revertirían"
	if apply {
		verb = "Revertidos"
	}
	fmt.Fprintf(w, "%s: %d.\n", verb, res.Reverted)
	fmt.Fprintf(w, "Omitidos porque ya no tienen los valores del cambio (no se aplicó o cambiaron después): %d.\n", len(res.Skipped))
	printIDs(w, "  ", res.Skipped, verbose)
	fmt.Fprintf(w, "Fallaron: %d.\n", len(res.Failed))
	printIssues(w, "  ", res.Failed, verbose)
	if !apply {
		fmt.Fprintln(w, "Sin -apply no se modificó nada.")
	}
}

// section abre una sección de la salida.
func section(w io.Writer, title string) {
	fmt.Fprintf(w, "\n== %s ==\n", title)
}

// printIssues lista los documentos; sin verbose, hasta maxListed.
func printIssues(w io.Writer, indent string, issues []docIssue, verbose bool) {
	for i, is := range issues {
		if !verbose && i == maxListed {
			fmt.Fprintf(w, "%s… y %d más (-v los lista)\n", indent, len(issues)-maxListed)
			return
		}
		fmt.Fprintf(w, "%s%s %s: %s\n", indent, is.Collection, is.ID, is.Detail)
	}
}

// printIDs lista ids en líneas de diez; sin verbose, hasta maxListed.
func printIDs(w io.Writer, indent string, ids []string, verbose bool) {
	shown := ids
	if !verbose && len(ids) > maxListed {
		shown = ids[:maxListed]
	}
	for start := 0; start < len(shown); start += 10 {
		end := min(start+10, len(shown))
		fmt.Fprintf(w, "%s%s\n", indent, strings.Join(shown[start:end], " "))
	}
	if len(shown) < len(ids) {
		fmt.Fprintf(w, "%s… y %d más (-v los lista)\n", indent, len(ids)-len(shown))
	}
}

// printYears muestra los bonos y montos por año.
func printYears(w io.Writer, title string, rows []yearRow) {
	if len(rows) == 0 {
		return
	}
	fmt.Fprintln(w, title)
	for _, r := range rows {
		fmt.Fprintf(w, "  %s\n", r)
	}
}

// soldIDs devuelve el _id de cada bono.
func soldIDs(bounds []soldBound) []string {
	out := make([]string, len(bounds))
	for i, b := range bounds {
		out[i] = idString(b.ID)
	}
	return out
}

// census muestra los tipos de un campo con su cantidad, del más común al menos común.
func census(types map[string]int) string {
	names := slices.Sorted(maps.Keys(types))
	slices.SortStableFunc(names, func(x, y string) int { return types[y] - types[x] })
	parts := make([]string, len(names))
	for i, name := range names {
		parts[i] = fmt.Sprintf("%s %d", name, types[name])
	}
	return strings.Join(parts, ", ")
}

// plural muestra una cantidad con el sustantivo en singular o plural.
func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// yesNo muestra un booleano de la tabla.
func yesNo(b bool) string {
	if b {
		return "sí"
	}
	return "no"
}

// formatAmount muestra un monto con punto de miles y, si tiene, coma decimal: 33.600.000 o 1.037,39.
func formatAmount(v float64) string {
	sign := ""
	if v < 0 {
		sign, v = "-", -v
	}
	cents := int64(math.Round(v * 100))
	digits := strconv.FormatInt(cents/100, 10)
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	if frac := cents % 100; frac != 0 {
		fmt.Fprintf(&b, ",%02d", frac)
	}
	return sign + b.String()
}
