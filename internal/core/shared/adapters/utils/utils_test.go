package utils_adapters_test

import (
	"regexp"
	"testing"

	utils_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/adapters/utils"
)

var alnumPattern = regexp.MustCompile(`^[A-Za-z0-9]*$`)

// RandomAlnum da n caracteres alfanuméricos, sin repetir resultados ni sesgo visible. CA-00.37
func TestRandomAlnum(t *testing.T) {
	u := utils_adapters.NewUtils()

	for _, n := range []int{1, 16, 1000} {
		if s := u.RandomAlnum(n); len(s) != n || !alnumPattern.MatchString(s) {
			t.Errorf("RandomAlnum(%d) = %q, se esperaban %d alfanuméricos", n, s, n)
		}
	}
	for _, n := range []int{0, -1} {
		if s := u.RandomAlnum(n); s != "" {
			t.Errorf("RandomAlnum(%d) = %q, se esperaba vacío", n, s)
		}
	}

	t.Run("sin repetidos", func(t *testing.T) {
		seen := make(map[string]bool)
		for range 1000 {
			s := u.RandomAlnum(16)
			if seen[s] {
				t.Fatalf("RandomAlnum(16) repitió %q", s)
			}
			seen[s] = true
		}
	})

	t.Run("distribución", func(t *testing.T) {
		const n = 100000
		counts := make(map[rune]int)
		for _, c := range u.RandomAlnum(n) {
			counts[c]++
		}
		if len(counts) != 62 {
			t.Fatalf("aparecieron %d caracteres distintos, se esperaban 62", len(counts))
		}
		expected := n / 62
		for c, got := range counts {
			if got < expected/2 || got > expected*3/2 {
				t.Errorf("%q apareció %d veces, se esperaban cerca de %d", c, got, expected)
			}
		}
	})
}
