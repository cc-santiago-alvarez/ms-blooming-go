package api_config

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultUtcDiff                 = -5
	defaultBulkPurchaseMaxQuantity = 100
)

// allowedEnvs son los valores válidos de ENV; cada uno tiene su archivo .env.<ENV>.
var allowedEnvs = []string{"development", "testing", "production"}

// requiredEnvs son las variables sin las que el servicio no arranca.
var requiredEnvs = []string{
	"API_NAME",
	"API_PORT",
	"API_SECRET",
	"CORS_ORIGINS",
	"FA_MONGO_URL",
	"FA_MONGO_DBNAME",
	"OAUTH_URL",
	"OAUTH_SECRET",
	"MS_NOTIFICATION_URL",
	"MS_NOTIFICATION_SECRET",
	"MS_EXPORTER_URL",
	"MS_EXPORTER_SECRET",
}

type cfg struct {
	Env *ApiEnv
}

func InitConfig() *cfg {
	c := &cfg{}
	c.LoadEnvs()
	return c
}

// LoadEnvs carga los archivos de entorno y valida las variables. Si falta un archivo o una
// variable requerida, o un valor es inválido, termina el proceso.
func (c *cfg) LoadEnvs() {
	if err := LoadEnvFiles(); err != nil {
		log.Fatalf("[config] %v", err)
	}

	apiEnv, err := newApiEnv(os.Getenv)
	if err != nil {
		log.Fatalf("[config] configuración inválida: %v", err)
	}
	c.Env = apiEnv
}

// LoadEnvFiles carga .env y luego .env.<ENV>, que pisa sus valores, desde el directorio de trabajo.
// La usan el API y el binario de healthcheck, para leer los mismos archivos con la misma precedencia.
func LoadEnvFiles() error {
	if err := godotenv.Load(".env"); err != nil {
		return fmt.Errorf("error al cargar .env: %w", err)
	}

	env := strings.TrimSpace(os.Getenv("ENV"))
	if err := checkEnvName(env); err != nil {
		return fmt.Errorf("configuración inválida: %w", err)
	}

	envFile := ".env." + env
	if _, err := os.Stat(envFile); err == nil {
		if err := godotenv.Overload(envFile); err != nil {
			return fmt.Errorf("error al cargar %s: %w", envFile, err)
		}
	} else if os.IsNotExist(err) {
		return fmt.Errorf("no existe el archivo de entorno %s", envFile)
	} else {
		return fmt.Errorf("error al verificar el archivo de entorno %s: %w", envFile, err)
	}

	// El entorno lo decide .env; el archivo ya elegido no puede cambiarlo.
	if strings.TrimSpace(os.Getenv("ENV")) != env {
		return fmt.Errorf("%s no puede redefinir ENV", envFile)
	}
	return nil
}

// newApiEnv construye y valida ApiEnv. Reporta todos los problemas en un solo error, sin valores.
func newApiEnv(getenv func(string) string) (*ApiEnv, error) {
	get := func(key string) string { return strings.TrimSpace(getenv(key)) }

	var problems []string

	if err := checkEnvName(get("ENV")); err != nil {
		problems = append(problems, err.Error())
	}

	var missing []string
	for _, key := range requiredEnvs {
		if get(key) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		problems = append(problems, "faltan variables requeridas: "+strings.Join(missing, ", "))
	}

	env := &ApiEnv{
		Env: get("ENV"),

		ApiName:   get("API_NAME"),
		ApiPort:   get("API_PORT"),
		ApiSecret: get("API_SECRET"),
		ApiDebug:  get("API_DEBUG") == "true",

		CorsOrigins: splitList(get("CORS_ORIGINS")),

		FuturappsMongoUrl:    get("FA_MONGO_URL"),
		FuturappsMongoDbName: get("FA_MONGO_DBNAME"),
		FuturappsMongoDebug:  get("FA_MONGO_DEBUG") == "true",

		OAuthURL:    get("OAUTH_URL"),
		OAuthSecret: get("OAUTH_SECRET"),

		NotificationServiceURL: get("MS_NOTIFICATION_URL"),
		NotificationSecret:     get("MS_NOTIFICATION_SECRET"),

		ExporterServiceURL: get("MS_EXPORTER_URL"),
		ExporterSecret:     get("MS_EXPORTER_SECRET"),

		BloomingAdminEmails: splitList(get("BLOOMING_ADMIN_EMAILS")),
		BloomingTrading:     get("BLOOMING_TRADING") == "true",
	}

	// Un valor como "," no está vacío pero no declara ningún origen.
	if get("CORS_ORIGINS") != "" && len(env.CorsOrigins) == 0 {
		problems = append(problems, "CORS_ORIGINS no contiene ningún origen")
	}

	var err error
	if _, err = parseInt(get, "API_PORT", 0, 1, 65535); err != nil {
		problems = append(problems, err.Error())
	}
	if env.DefaultUtcDiff, err = parseInt(get, "DEFAULT_UTC_DIFF", defaultUtcDiff, -12, 14); err != nil {
		problems = append(problems, err.Error())
	}
	if env.BulkPurchaseMaxQuantity, err = parseInt(get, "BULK_PURCHASE_MAX_QUANTITY", defaultBulkPurchaseMaxQuantity, 1, math.MaxInt); err != nil {
		problems = append(problems, err.Error())
	}

	if len(problems) > 0 {
		return nil, errors.New(strings.Join(problems, "; "))
	}
	return env, nil
}

// checkEnvName verifica que ENV esté declarada y sea uno de allowedEnvs.
func checkEnvName(env string) error {
	expected := strings.Join(allowedEnvs, ", ")
	if env == "" {
		return fmt.Errorf("falta la variable requerida ENV (%s)", expected)
	}
	if !slices.Contains(allowedEnvs, env) {
		return fmt.Errorf("ENV tiene un valor inválido, se espera uno de: %s", expected)
	}
	return nil
}

// parseInt lee un entero dentro de [min, max]. Si la variable está vacía devuelve def sin error.
func parseInt(get func(string) string, key string, def, min, max int) (int, error) {
	v := get(key)
	if v == "" {
		return def, nil
	}

	n, err := strconv.Atoi(v)
	if err == nil && n >= min && n <= max {
		return n, nil
	}

	expected := fmt.Sprintf("un entero entre %d y %d", min, max)
	if max == math.MaxInt {
		expected = fmt.Sprintf("un entero mayor o igual a %d", min)
	}
	return def, fmt.Errorf("%s tiene un valor inválido, se espera %s", key, expected)
}

// splitList separa una lista por comas, sin espacios ni elementos vacíos.
func splitList(v string) []string {
	var items []string
	for item := range strings.SplitSeq(v, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}
