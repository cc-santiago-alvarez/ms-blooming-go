//go:build integration

// Suite end-to-end: levanta el servicio completo (configuración, composition root, middlewares y handlers)
// en un subproceso contra un Mongo real con replica set, con ms-oauth, ms-notification y ms-exporter falsos,
// y recorre los flujos por HTTP. El servicio usa su propia base, que se borra al empezar y al terminar.
//
//	go test -tags=integration ./test/e2e/
//
// La URL sale de FA_MONGO_URL o, si no está en el entorno, de .env y .env.<ENV> en la raíz del
// repositorio. Sin Mongo disponible, los tests se saltan.
package e2e_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/codecraftkit/mongocc"
	"github.com/futurapps/ms-blooming-go/cmd/api/server"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// serverMarker hace que el binario de test levante el servicio en lugar de correr los tests.
const serverMarker = "BLM_E2E_SERVER"

// testDBName es la base del servicio en estos tests; nunca la de FA_MONGO_DBNAME.
const testDBName = "blooming_e2e"

// Secretos del servicio y de los servicios falsos.
const (
	apiSecret          = "e2e-api-secret"
	oauthSecret        = "e2e-oauth-secret"
	notificationSecret = "e2e-notification-secret"
	exporterSecret     = "e2e-exporter-secret"
)

// adminEmail es el destinatario del aviso de retiro al admin.
const adminEmail = "admin@futurapps.test"

const (
	// startTimeout es lo que se espera a que el servicio responda /_health.
	startTimeout = 30 * time.Second
	// stopTimeout es lo que se espera a que el servicio se apague tras el SIGTERM.
	stopTimeout = 25 * time.Second
)

var (
	testDB     *mongocc.MongoQueries
	skipReason string
	// apiURL es la raíz del servicio, http://127.0.0.1:<puerto>.
	apiURL string
)

func TestMain(m *testing.M) {
	if os.Getenv(serverMarker) == "1" {
		server.InitServer().Run()
		os.Exit(0)
	}
	os.Exit(run(m))
}

// run arranca los servicios falsos, prepara la base y levanta el servicio, corre los tests y después
// apaga y borra todo.
func run(m *testing.M) int {
	for _, id := range fixtureIDs {
		if !shared_domain.IsValidID(id) {
			fmt.Fprintf(os.Stderr, "el ID de fixture %q no es Meteor ni UUID v4\n", id)
			return 1
		}
	}

	startFakes()
	defer closeFakes()

	url, err := mongoURL()
	if err != nil {
		skipReason = err.Error()
		return m.Run()
	}
	client, err := connect(url)
	if err != nil {
		skipReason = err.Error()
		return m.Run()
	}
	testDB = mongocc.NewFromClient(client, testDBName, nil)
	defer dropDatabase(client)

	if err := resetDatabase(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	svc, err := startService(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	code := m.Run()
	if err := svc.stop(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	if code != 0 {
		fmt.Fprintf(os.Stderr, "log del servicio:\n%s", svc.log.String())
	}
	return code
}

// connect abre un cliente propio y exige un replica set.
func connect(url string) (*mongo.Client, error) {
	if url == "" {
		return nil, errors.New("FA_MONGO_URL no está en el entorno ni en .env")
	}

	client, err := mongo.Connect(options.Client().
		ApplyURI(url).
		SetServerSelectionTimeout(5 * time.Second))
	if err != nil {
		// el error de parseo puede incluir la URL con credenciales
		return nil, errors.New("FA_MONGO_URL no es una URL de Mongo válida")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var hello struct {
		SetName string `bson:"setName"`
	}
	if err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&hello); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("sin respuesta de Mongo: %w", err)
	}
	if hello.SetName == "" {
		_ = client.Disconnect(context.Background())
		return nil, errors.New("el Mongo de pruebas no es un replica set y las transacciones lo requieren")
	}
	return client, nil
}

// mongoURL lee FA_MONGO_URL del entorno o, si no está, de .env y .env.<ENV> en la raíz del
// repositorio, sin modificar el entorno del proceso.
func mongoURL() (string, error) {
	if url := strings.TrimSpace(os.Getenv("FA_MONGO_URL")); url != "" {
		return url, nil
	}
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	vars, err := godotenv.Read(filepath.Join(root, ".env"))
	if err != nil {
		return "", nil
	}
	if env := strings.TrimSpace(vars["ENV"]); env != "" {
		if overrides, err := godotenv.Read(filepath.Join(root, ".env."+env)); err == nil {
			maps.Copy(vars, overrides)
		}
	}
	return strings.TrimSpace(vars["FA_MONGO_URL"]), nil
}

// repoRoot sube desde el directorio de trabajo hasta el que tiene go.mod.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no se encontró go.mod")
		}
		dir = parent
	}
}

// resetDatabase borra la base de pruebas y crea las colecciones vacías; los índices los crea el servicio
// al arrancar.
func resetDatabase() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := testDB.Database().Drop(ctx); err != nil {
		return fmt.Errorf("no se pudo borrar la base %s: %w", testDBName, err)
	}
	for _, name := range collections {
		if err := testDB.Database().CreateCollection(ctx, name); err != nil {
			return fmt.Errorf("no se pudo crear %s: %w", name, err)
		}
	}
	return nil
}

// dropDatabase borra la base de pruebas y cierra el cliente.
func dropDatabase(client *mongo.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := testDB.Database().Drop(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "no se pudo borrar la base %s: %v\n", testDBName, err)
	}
	_ = client.Disconnect(ctx)
}

// service es el servicio corriendo en un subproceso.
type service struct {
	cmd *exec.Cmd
	// dir es el directorio de trabajo, con los .env vacíos.
	dir string
	// log junta stdout y stderr; se lee después de que el proceso terminó.
	log  bytes.Buffer
	done chan struct{}
	err  error
}

// startService levanta el servicio con la configuración de serviceEnv y espera a que responda /_health.
func startService(mongoURL string) (*service, error) {
	dir, err := os.MkdirTemp("", "blooming-e2e-")
	if err != nil {
		return nil, err
	}
	// la configuración exige los dos archivos; las variables van en el entorno
	for _, name := range []string{".env", ".env.testing"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			_ = os.RemoveAll(dir)
			return nil, err
		}
	}
	port, err := freePort()
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	exe, err := os.Executable()
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}

	s := &service{cmd: exec.Command(exe), dir: dir, done: make(chan struct{})}
	s.cmd.Dir = dir
	s.cmd.Env = append([]string{"PATH=" + os.Getenv("PATH"), serverMarker + "=1"}, serviceEnv(port, mongoURL)...)
	s.cmd.Stdout, s.cmd.Stderr = &s.log, &s.log
	if err := s.cmd.Start(); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("no se pudo levantar el servicio: %w", err)
	}
	go func() {
		s.err = s.cmd.Wait()
		close(s.done)
	}()

	apiURL = "http://127.0.0.1:" + port
	if err := s.waitHealthy(); err != nil {
		_ = s.cmd.Process.Kill()
		<-s.done
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("%w\nlog del servicio:\n%s", err, s.log.String())
	}
	return s, nil
}

// serviceEnv es la configuración del servicio: su propia base y los servicios falsos.
func serviceEnv(port, mongoURL string) []string {
	vars := map[string]string{
		"ENV":                    "testing",
		"API_NAME":               "blooming-e2e",
		"API_PORT":               port,
		"API_SECRET":             apiSecret,
		"CORS_ORIGINS":           "http://localhost",
		"FA_MONGO_URL":           mongoURL,
		"FA_MONGO_DBNAME":        testDBName,
		"OAUTH_URL":              oauth.URL + "/api/v1",
		"OAUTH_SECRET":           oauthSecret,
		"MS_NOTIFICATION_URL":    notifications.URL + "/api/v1",
		"MS_NOTIFICATION_SECRET": notificationSecret,
		"MS_EXPORTER_URL":        exporter.URL,
		"MS_EXPORTER_SECRET":     exporterSecret,
		"BLOOMING_ADMIN_EMAILS":  adminEmail,
	}
	env := make([]string, 0, len(vars))
	for k, v := range vars {
		env = append(env, k+"="+v)
	}
	return env
}

// waitHealthy espera a que el servicio responda 200 en /_health.
func (s *service) waitHealthy() error {
	deadline := time.Now().Add(startTimeout)
	for time.Now().Before(deadline) {
		select {
		case <-s.done:
			return fmt.Errorf("el servicio terminó al arrancar: %v", s.err)
		case <-time.After(50 * time.Millisecond):
		}
		if resp, err := http.Get(apiURL + "/_health"); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
	}
	return fmt.Errorf("el servicio no respondió 200 en /_health en %s", startTimeout)
}

// stop apaga el servicio con SIGTERM, como el orquestador, y exige que termine con código 0.
func (s *service) stop() error {
	defer os.RemoveAll(s.dir)
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("no se pudo mandar SIGTERM al servicio: %w", err)
	}
	select {
	case <-s.done:
		if s.err != nil {
			return fmt.Errorf("el servicio terminó con error: %w", s.err)
		}
		return nil
	case <-time.After(stopTimeout):
		_ = s.cmd.Process.Kill()
		<-s.done
		return fmt.Errorf("el servicio no se apagó en %s", stopTimeout)
	}
}

// freePort devuelve un puerto TCP libre de 127.0.0.1.
func freePort() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer ln.Close()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	return port, err
}
