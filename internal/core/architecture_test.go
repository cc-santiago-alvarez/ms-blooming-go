package core_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// corePrefix es la ruta de import de internal/core.
const corePrefix = "github.com/futurapps/ms-blooming-go/internal/core/"

// sharedDomain es el único dominio que cualquier otro puede importar.
const sharedDomain = "shared"

// Ningún paquete *_domain importa otro *_domain, salvo shared_domain. CA-00.27
func TestDomainPackagesAreIndependent(t *testing.T) {
	problems, _, err := domainRuleProblems(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range problems {
		t.Error(p)
	}
}

// Un *_domain que importa otro *_domain, directo, por un subpaquete o desde un test, hace fallar la
// regla con un mensaje que nombra los dos paquetes y el archivo. CA-00.27
func TestDomainRuleDetectsForbiddenImport(t *testing.T) {
	problems, checked, err := domainRuleProblems(filepath.Join("testdata", "forbidden"))
	if err != nil {
		t.Fatal(err)
	}
	if checked != 3 {
		t.Fatalf("se revisaron %d paquetes de dominio, se esperaban 3", checked)
	}
	want := []string{
		"bondBound_domain importa bondObject_domain (bondBound/domain/bondBound.go)",
		"bondBound_domain importa bondObject_domain (bondBound/domain/bondBound_test.go)",
		"bondBound_domain importa bondObject_domain (bondBound/domain/sub/sub.go)",
	}
	if len(problems) != len(want) {
		t.Fatalf("se encontraron %d violaciones, se esperaban %d:\n%s", len(problems), len(want), strings.Join(problems, "\n"))
	}
	for i, w := range want {
		if !strings.HasPrefix(problems[i], w) {
			t.Errorf("violación %d = %q, se esperaba %q", i, problems[i], w)
		}
	}
}

// Si encuentra menos de 2 paquetes de dominio, la regla falla: no estaría revisando nada. CA-00.27
func TestDomainRuleRequiresDomains(t *testing.T) {
	problems, checked, err := domainRuleProblems(filepath.Join("testdata", "single"))
	if err != nil {
		t.Fatal(err)
	}
	if checked != 1 || len(problems) != 1 || !strings.Contains(problems[0], "se revisaron 1 paquetes de dominio") {
		t.Fatalf("con un solo dominio: %d revisados y problemas %q", checked, problems)
	}
}

// domainRuleProblems devuelve las violaciones de la regla en root, más un problema si encontró menos
// de 2 paquetes de dominio: el layout cambió y la regla no estaría revisando nada.
func domainRuleProblems(root string) ([]string, int, error) {
	violations, checked, err := domainImportViolations(root)
	if err != nil {
		return nil, 0, err
	}
	if checked < 2 {
		violations = append(violations, fmt.Sprintf(
			"se revisaron %d paquetes de dominio y se esperaban al menos 2: ¿cambió el layout de internal/core?", checked))
	}
	return violations, checked, nil
}

// domainImportViolations revisa los imports de cada root/<d>/domain/, con sus subpaquetes y tests.
// Devuelve un mensaje por cada import de otro *_domain distinto de shared_domain y la cantidad de
// paquetes de dominio revisados.
func domainImportViolations(root string) ([]string, int, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, 0, err
	}

	var violations []string
	checked := 0
	fset := token.NewFileSet()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		domainDir := filepath.Join(root, name, "domain")
		if info, err := os.Stat(domainDir); err != nil || !info.IsDir() {
			continue
		}
		checked++

		err := filepath.WalkDir(domainDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}

			// Solo los imports: los cuerpos no hacen falta.
			file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, spec := range file.Imports {
				importPath, _ := strconv.Unquote(spec.Path.Value)
				other, ok := importedDomain(importPath)
				if !ok || other == name || other == sharedDomain {
					continue
				}
				rel, _ := filepath.Rel(root, path)
				violations = append(violations, fmt.Sprintf(
					"%s_domain importa %s_domain (%s). Lo que cruza dominios va en application o en ports.",
					name, other, rel,
				))
			}
			return nil
		})
		if err != nil {
			return nil, 0, err
		}
	}
	return violations, checked, nil
}

// importedDomain devuelve <x> si importPath es internal/core/<x>/domain o un subpaquete suyo.
func importedDomain(importPath string) (string, bool) {
	rest, ok := strings.CutPrefix(importPath, corePrefix)
	if !ok {
		return "", false
	}
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 2 || parts[1] != "domain" {
		return "", false
	}
	return parts[0], true
}
