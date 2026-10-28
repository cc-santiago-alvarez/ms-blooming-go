// Arnés de paridad entre el legado y el API Go. Sobre la misma base llama a los métodos de Meteor por DDP, al
// feed de ms-feed y a los endpoints Go, y compara las fórmulas: la simulación de cada producto (getDataChart),
// la cotización de venta (preSale) y la evolución (/bound/:id/feed) de una muestra de bonos, y los derivados del
// listado de una muestra de dueños. Separa las diferencias que documentan las specs de las demás y termina con
// código 1 si encuentra alguna sin documentar, o con 2 si la corrida cruzó la medianoche de la zona.
//
//	go run ./cmd/tools/parity -db parity
//	go run ./cmd/tools/parity -db parity -only presale,feed -bounds 50 -v
//
// Antes de correrlo:
//   - La base (-db) tiene que ser una copia: el arnés crea sesiones temporales en ella, y Meteor escribe al
//     arrancar.
//   - Meteor (backend en origin/main), ms-feed (origin/main), ms-oauth y el API Go tienen que estar levantados
//     contra esa copia. Meteor va sin MS_FEED_URL, para que su cron de maduración no llame a ms-feed.
//   - Meteor y ms-feed tienen que correr con la zona del proceso igual a -utc-diff (TZ=Etc/GMT+5 para -5),
//     porque calculan algunas fechas con la zona del servidor y Go, con la de la sesión.
//
// Las sesiones son del admin (-admin, o el primer admin de Blooming de la agencia principal en la zona, que ve
// los bonos de todas las agencias) y de los dueños de la muestra, solo de usuarios cuyo perfil y agencia están
// en la zona: ms-oauth toma la zona del perfil y Meteor, de la agencia. Se borran al terminar.
//
// Lee FA_MONGO_URL, FA_MONGO_DBNAME, API_SECRET y API_PORT de los mismos archivos de entorno que el API; -db
// pisa FA_MONGO_DBNAME.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/codecraftkit/mongocc"
	api_config "github.com/futurapps/ms-blooming-go/cmd/api/config"
)

// runConfig son los parámetros de la corrida.
type runConfig struct {
	db        string
	admin     string
	utcDiff   int
	meteorURL string
	feedURL   string
	goURL     string
	bounds    int
	users     int
	only      []string
	verbose   bool
}

func main() {
	var cfg runConfig
	var only string
	flag.StringVar(&cfg.db, "db", "", "base de la corrida, una copia; pisa FA_MONGO_DBNAME")
	flag.StringVar(&cfg.admin, "admin", "", "userId del admin de Blooming; sin él se elige el primero de la agencia principal en la zona")
	flag.IntVar(&cfg.utcDiff, "utc-diff", -5, "zona de la corrida en horas; Meteor y ms-feed tienen que correr con esa TZ")
	flag.StringVar(&cfg.meteorURL, "meteor", "ws://localhost:3000/websocket", "websocket DDP de Meteor")
	flag.StringVar(&cfg.feedURL, "feed", "http://localhost:8090", "URL base de ms-feed")
	flag.StringVar(&cfg.goURL, "go", "", "URL base del API Go, con /api/v1 (por defecto http://localhost:$API_PORT/api/v1)")
	flag.IntVar(&cfg.bounds, "bounds", 200, "cantidad máxima de bonos de la muestra")
	flag.IntVar(&cfg.users, "users", 20, "cantidad máxima de dueños cuyos listados se comparan")
	flag.StringVar(&only, "only", "sim,presale,feed,list", "comparaciones a correr, separadas por coma")
	flag.BoolVar(&cfg.verbose, "v", false, "lista todas las diferencias, también las documentadas")
	flag.Parse()
	cfg.only = strings.Split(only, ",")

	code, err := run(cfg)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "parity falló: %v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}

// run conecta con la base y con los servicios, corre las comparaciones y devuelve el código de salida.
func run(cfg runConfig) (int, error) {
	if err := api_config.LoadEnvFiles(); err != nil {
		return 0, err
	}
	mongoURL := strings.TrimSpace(os.Getenv("FA_MONGO_URL"))
	if cfg.db == "" {
		cfg.db = strings.TrimSpace(os.Getenv("FA_MONGO_DBNAME"))
	}
	secret := strings.TrimSpace(os.Getenv("API_SECRET"))
	if cfg.goURL == "" {
		cfg.goURL = "http://localhost:" + strings.TrimSpace(os.Getenv("API_PORT")) + "/api/v1"
	}
	if mongoURL == "" || cfg.db == "" || secret == "" {
		return 0, errors.New("FA_MONGO_URL, FA_MONGO_DBNAME (o -db) y API_SECRET son requeridas")
	}
	if cfg.bounds < 1 || cfg.users < 0 {
		return 0, errors.New("-bounds tiene que ser 1 o más y -users, 0 o más")
	}
	selected, err := selectChecks(cfg.only)
	if err != nil {
		return 0, err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := mongocc.Connect(mongoURL, cfg.db, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = db.Database().Client().Disconnect(context.Background()) }()

	day := today(cfg.utcDiff)
	s, err := loadSample(ctx, db, cfg)
	if err != nil {
		return 0, err
	}

	sess := &sessions{db: db}
	defer func() {
		if err := sess.removeAll(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "no se pudieron borrar las sesiones temporales: %v\n", err)
		}
	}()
	adminToken, err := sess.create(ctx, s.admin)
	if err != nil {
		return 0, err
	}

	dialCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	meteor, err := dialDDP(dialCtx, cfg.meteorURL)
	cancel()
	if err != nil {
		return 0, fmt.Errorf("Meteor en %s: %w", cfg.meteorURL, err)
	}
	defer func() { _ = meteor.close() }()

	env := &runEnv{
		meteor: meteor,
		http: &httpClient{
			client:  &http.Client{},
			goURL:   strings.TrimRight(cfg.goURL, "/"),
			feedURL: strings.TrimRight(cfg.feedURL, "/"),
			secret:  secret,
		},
		sessions:   sess,
		adminToken: adminToken,
	}

	fmt.Printf("Base %s, zona %d, día %s. Admin %s; %d productos, %d bonos y %d dueños.\n",
		cfg.db, cfg.utcDiff, day, s.admin, len(s.objects), len(s.bounds), len(s.owners))

	results := make([]*result, 0, len(selected))
	for _, c := range selected {
		r, err := c.run(ctx, env, s)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", c.name, err)
		}
		results = append(results, r)
	}
	unexpected := report(os.Stdout, results, cfg.verbose)

	if today(cfg.utcDiff) != day {
		fmt.Println("\nLa corrida cruzó la medianoche de la zona: hay que repetirla.")
		return 2, nil
	}
	if unexpected > 0 {
		return 1, nil
	}
	return 0, nil
}

// today es la fecha actual en la zona utcDiff, como YYYY-MM-DD.
func today(utcDiff int) string {
	return time.Now().In(time.FixedZone("", utcDiff*3600)).Format(time.DateOnly)
}
