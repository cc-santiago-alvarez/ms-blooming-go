package api_config_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	api_config "github.com/futurapps/ms-blooming-go/cmd/api/config"
)

// childMarker hace que el binario de test cargue la configuración en lugar de correr los tests.
const childMarker = "BLM_CONFIG_CHILD"

// loaded es lo que imprime el subproceso cuando la configuración es válida.
type loaded struct {
	Env            string
	ApiName        string
	ApiPort        string
	DefaultUtcDiff int
	CorsOrigins    []string
}

func TestMain(m *testing.M) {
	if os.Getenv(childMarker) == "1" {
		env := api_config.InitConfig().Env
		_ = json.NewEncoder(os.Stdout).Encode(loaded{
			Env:            env.Env,
			ApiName:        env.ApiName,
			ApiPort:        env.ApiPort,
			DefaultUtcDiff: env.DefaultUtcDiff,
			CorsOrigins:    env.CorsOrigins,
		})
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// validEnv son las variables requeridas con valores que no deben aparecer en ningún log.
func validEnv() map[string]string {
	return map[string]string{
		"API_NAME":               "blooming-config-test",
		"API_PORT":               "25099",
		"API_SECRET":             "valor-api-secret-1",
		"CORS_ORIGINS":           "https://app.valor-cors.test",
		"FA_MONGO_URL":           "mongodb://usuario-valor:clave-valor@mongo.valor.test:27017",
		"FA_MONGO_DBNAME":        "valor-dbname",
		"OAUTH_URL":              "http://oauth.valor.test/api/v1",
		"OAUTH_SECRET":           "valor-oauth-secret-2",
		"MS_NOTIFICATION_URL":    "http://notification.valor.test/api/v1",
		"MS_NOTIFICATION_SECRET": "valor-notification-secret-3",
		"MS_EXPORTER_URL":        "http://exporter.valor.test",
		"MS_EXPORTER_SECRET":     "valor-exporter-secret-4",
	}
}

// envFile arma el contenido de un archivo .env.
func envFile(vars map[string]string) string {
	var b strings.Builder
	for _, k := range slices.Sorted(maps.Keys(vars)) {
		b.WriteString(k + "=" + vars[k] + "\n")
	}
	return b.String()
}

// testingFiles devuelve un .env con ENV=testing y las variables válidas, modificadas por change, y un
// .env.testing que no redefine nada requerido.
func testingFiles(change func(env map[string]string)) map[string]string {
	env := validEnv()
	env["ENV"] = "testing"
	if change != nil {
		change(env)
	}
	return map[string]string{
		".env":         envFile(env),
		".env.testing": envFile(map[string]string{"FA_MONGO_DEBUG": "false"}),
	}
}

type result struct {
	code   int
	stdout string
	stderr string
}

// runConfig carga la configuración en un subproceso, con un entorno vacío y el directorio de trabajo
// en un directorio temporal con files. Los nombres de dirs se crean como directorios: cargar uno de
// ellos falla con un error que nombra el archivo.
func runConfig(t *testing.T, files map[string]string, dirs ...string) result {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range dirs {
		if err := os.Mkdir(filepath.Join(dir, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), childMarker + "=1"}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	code := 0
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("no se pudo correr el subproceso: %v", err)
		}
		code = exitErr.ExitCode()
	}
	return result{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

// Una configuración inválida termina el proceso con código 1 y un log que nombra la variable sin
// mostrar ningún valor; sin un ENV válido no se carga ningún .env.<ENV>. CA-00.1
func TestInitConfig(t *testing.T) {
	t.Run("configuración válida", func(t *testing.T) {
		files := testingFiles(func(env map[string]string) { env["API_NAME"] = "nombre-de-env" })
		files[".env.testing"] = envFile(map[string]string{"API_NAME": "nombre-de-testing", "DEFAULT_UTC_DIFF": "-7"})

		r := runConfig(t, files)

		if r.code != 0 {
			t.Fatalf("salió con %d, se esperaba 0:\n%s", r.code, r.stderr)
		}
		var got loaded
		if err := json.Unmarshal([]byte(r.stdout), &got); err != nil {
			t.Fatalf("salida inválida %q: %v", r.stdout, err)
		}
		want := loaded{
			Env:            "testing",
			ApiName:        "nombre-de-testing",
			ApiPort:        "25099",
			DefaultUtcDiff: -7,
			CorsOrigins:    []string{"https://app.valor-cors.test"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("configuración cargada %+v, se esperaba %+v", got, want)
		}
	})

	without := func(keys ...string) func(map[string]string) {
		return func(env map[string]string) {
			for _, k := range keys {
				delete(env, k)
			}
		}
	}
	set := func(key, value string) func(map[string]string) {
		return func(env map[string]string) { env[key] = value }
	}
	noEnv := testingFiles(without("ENV"))
	delete(noEnv, ".env.testing")
	invalidEnv := testingFiles(set("ENV", "staging"))
	delete(invalidEnv, ".env.testing")
	redefined := testingFiles(nil)
	redefined[".env.testing"] = envFile(map[string]string{"ENV": "production"})
	noEnvFile := testingFiles(nil)
	delete(noEnvFile, ".env")
	noTestingFile := testingFiles(nil)
	delete(noTestingFile, ".env.testing")

	tests := []struct {
		name       string
		files      map[string]string
		dirs       []string
		wantNamed  []string
		badValue   string
		noEnvFiles bool
	}{
		{name: "falta API_SECRET", files: testingFiles(without("API_SECRET")), wantNamed: []string{"API_SECRET"}},
		{name: "faltan varias requeridas", files: testingFiles(without("API_SECRET", "OAUTH_URL", "MS_EXPORTER_SECRET")),
			wantNamed: []string{"API_SECRET", "OAUTH_URL", "MS_EXPORTER_SECRET"}},
		{name: ".env sin ENV", files: noEnv, dirs: []string{".env.", ".env.development", ".env.testing"},
			wantNamed: []string{"ENV"}, noEnvFiles: true},
		{name: "ENV inválido", files: invalidEnv, dirs: []string{".env.staging"},
			wantNamed: []string{"ENV"}, noEnvFiles: true},
		{name: ".env.testing redefine ENV", files: redefined, wantNamed: []string{".env.testing", "ENV"}},
		{name: "falta .env", files: noEnvFile, wantNamed: []string{".env"}},
		{name: "falta .env.testing", files: noTestingFile, wantNamed: []string{".env.testing"}},
		{name: "API_PORT no numérico", files: testingFiles(set("API_PORT", "puerto-xyz")), wantNamed: []string{"API_PORT"}, badValue: "puerto-xyz"},
		{name: "DEFAULT_UTC_DIFF fuera de rango", files: testingFiles(set("DEFAULT_UTC_DIFF", "99")), wantNamed: []string{"DEFAULT_UTC_DIFF"}, badValue: "99"},
		{name: "CORS_ORIGINS sin orígenes", files: testingFiles(set("CORS_ORIGINS", ",")), wantNamed: []string{"CORS_ORIGINS"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := runConfig(t, tt.files, tt.dirs...)

			if r.code != 1 {
				t.Fatalf("salió con %d, se esperaba 1; stderr:\n%s", r.code, r.stderr)
			}
			for _, name := range tt.wantNamed {
				if !strings.Contains(r.stderr, name) {
					t.Errorf("el log no nombra %s:\n%s", name, r.stderr)
				}
			}
			values := slices.Collect(maps.Values(validEnv()))
			if tt.badValue != "" {
				values = append(values, tt.badValue)
			}
			for _, v := range values {
				if strings.Contains(r.stderr, v) {
					t.Errorf("el log muestra el valor %q:\n%s", v, r.stderr)
				}
			}
			if tt.noEnvFiles && strings.Contains(r.stderr, "error al cargar") {
				t.Errorf("se intentó cargar un .env.<ENV>:\n%s", r.stderr)
			}
		})
	}
}
