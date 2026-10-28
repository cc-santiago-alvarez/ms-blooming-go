package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// maxListed es la cantidad de diferencias no documentadas que se listan por comparación sin -v.
const maxListed = 20

// report imprime el resumen de cada comparación y sus diferencias no documentadas; con verbose las lista
// todas, también las documentadas. Devuelve la cantidad de casos con diferencias no documentadas.
func report(w io.Writer, results []*result, verbose bool) int {
	total := 0
	seen := map[string]bool{}
	for _, r := range results {
		_, _ = fmt.Fprintf(w, "\n%s: %d casos, %d iguales, %d con diferencias documentadas%s, %d con diferencias no documentadas\n",
			r.name, r.cases, r.equal, r.expectedCases, formatCounts(r.expected), r.unexpectedCases)

		listed := r.unexpected
		if !verbose && len(listed) > maxListed {
			listed = listed[:maxListed]
		}
		for _, d := range listed {
			_, _ = fmt.Fprintf(w, "  no documentada: %s\n", d)
		}
		if rest := len(r.unexpected) - len(listed); rest > 0 {
			_, _ = fmt.Fprintf(w, "  … y %d diferencias más (-v las lista todas)\n", rest)
		}
		if verbose {
			for _, d := range r.expectedDiffs {
				_, _ = fmt.Fprintf(w, "  documentada (%s): %s\n", d.Expected, d)
			}
		}

		for id := range r.expected {
			seen[id] = true
		}
		total += r.unexpectedCases
	}

	if len(seen) > 0 {
		_, _ = fmt.Fprintln(w, "\nDiferencias documentadas:")
		for _, e := range expectations {
			if seen[e.id] {
				_, _ = fmt.Fprintf(w, "  %s: %s\n", e.id, e.reason)
			}
		}
	}
	return total
}

// formatCounts muestra los casos por diferencia documentada, como " (B1: 3, O10: 2)".
func formatCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}
	ids := make([]string, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%s: %d", id, counts[id])
	}
	return " (" + strings.Join(parts, ", ") + ")"
}
