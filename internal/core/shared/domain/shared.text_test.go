package shared_domain_test

import (
	"encoding/json"
	"os"
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// firstDiff devuelve un fragmento de a y de b alrededor de su primera diferencia.
func firstDiff(a, b string) (string, string) {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	from := max(0, i-20)
	return a[from:min(len(a), i+20)], b[from:min(len(b), i+20)]
}

// Text devuelve la plantilla literal del legado para cada clave, y la clave misma si no existe.
// CA-00.38
func TestTextMatchesLegacy(t *testing.T) {
	// plantillas que da backend/server/lib/text.js del legado, evaluado con Node
	raw, err := os.ReadFile("testdata/legacy-texts.json")
	if err != nil {
		t.Fatalf("no se pudo leer el catálogo del legado: %v", err)
	}
	var legacy map[string]string
	if err := json.Unmarshal(raw, &legacy); err != nil {
		t.Fatalf("catálogo del legado inválido: %v", err)
	}
	if len(legacy) != 22 {
		t.Fatalf("el catálogo del legado tiene %d claves, se esperaban 22", len(legacy))
	}

	for key, want := range legacy {
		t.Run(key, func(t *testing.T) {
			if got := shared_domain.Text(key); got != want {
				g, w := firstDiff(got, want)
				t.Fatalf("Text(%s) difiere del legado:\n got  …%q…\n want …%q…", key, g, w)
			}
		})
	}

	t.Run("clave desconocida", func(t *testing.T) {
		for _, key := range []string{"NO_EXISTE", ""} {
			if got := shared_domain.Text(key); got != key {
				t.Errorf("Text(%q) = %q, se esperaba la clave", key, got)
			}
		}
	})
}
