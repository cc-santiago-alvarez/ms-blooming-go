// Auditoría de datos previa al corte. Informa lo que Go no puede leer u operar y los datos legacy que hay que
// normalizar: claves de bono repetidas, settings duplicados, numéricos guardados como string, isPrivateBond que
// no es booleano, bonos madurados que ya estaban vendidos, bonos vendidos y canjeados, y lo que madura la
// primera corrida de Go. Con -apply normaliza los numéricos, isPrivateBond y los bonos madurados que ya
// estaban vendidos, que pasan a vendido; antes de cada escritura guarda un respaldo que -revert deshace.
//
//	go run ./cmd/tools/auditdata                                         # solo informa
//	go run ./cmd/tools/auditdata -at 2023-07-06                          # la maduración, a esa fecha
//	go run ./cmd/tools/auditdata -apply -confirm <ENV>/<base>            # además normaliza
//	go run ./cmd/tools/auditdata -apply -confirm <ENV>/<base> -only vendidos
//	go run ./cmd/tools/auditdata -revert <respaldo>                      # muestra qué revertiría
//	go run ./cmd/tools/auditdata -revert <respaldo> -apply -confirm <ENV>/<base>
//
// Lee ENV, FA_MONGO_URL y FA_MONGO_DBNAME de los mismos archivos de entorno que el API; -db pisa
// FA_MONGO_DBNAME. -confirm repite el entorno y la base, porque la base se llama igual en testing y en live. La
// salida no tiene datos personales: solo ids de bonos, productos y settings, y cantidades. Termina con 0 si no
// queda nada pendiente, con 2 si queda algo pendiente y con 1 si falla.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/codecraftkit/mongocc"
	api_config "github.com/futurapps/ms-blooming-go/cmd/api/config"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Códigos de salida.
const (
	exitOK      = 0
	exitError   = 1
	exitPending = 2
)

// Normalizaciones que aplica -apply, en el orden en que se aplican.
const (
	fixNumeric = "numericos"
	fixPrivate = "privados"
	fixSold    = "vendidos"
)

var allFixes = []string{fixNumeric, fixPrivate, fixSold}

// config son las opciones de la línea de comandos.
type config struct {
	apply   bool
	confirm string
	dbName  string
	at      time.Time
	only    map[string]bool
	backup  string
	revert  string
	pause   time.Duration
	verbose bool
}

func main() {
	cfg, err := parseConfig(os.Args[1:], shared_domain.Now())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "auditdata: %v\n", err)
		os.Exit(exitError)
	}

	code, err := run(cfg)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "auditdata falló: %v\n", err)
		os.Exit(exitError)
	}
	os.Exit(code)
}

// parseConfig lee las opciones; now es la fecha de referencia por defecto.
func parseConfig(args []string, now time.Time) (config, error) {
	fs := flag.NewFlagSet("auditdata", flag.ContinueOnError)
	apply := fs.Bool("apply", false, "aplica las normalizaciones (o la reversión con -revert); sin él solo informa")
	confirm := fs.String("confirm", "", "<ENV>/<base> de la conexión, por ejemplo testing/futurapps; obligatorio con -apply")
	dbName := fs.String("db", "", "base a auditar; pisa FA_MONGO_DBNAME")
	at := fs.String("at", "", "fecha de referencia de la maduración, RFC 3339 o AAAA-MM-DD en UTC; por defecto, ahora")
	only := fs.String("only", "", "normalizaciones de -apply separadas por coma: numericos, privados, vendidos; por defecto, todas")
	backup := fs.String("backup", "", "archivo del respaldo de -apply; por defecto tmp/auditdata/<base>-<fecha>.jsonl")
	revert := fs.String("revert", "", "revierte los cambios de un respaldo")
	pause := fs.Duration("pausa", 0, "pausa entre escrituras, para no saturar a Meteor")
	verbose := fs.Bool("v", false, "lista todos los ids; sin él, hasta 20 por hallazgo")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if fs.NArg() > 0 {
		return config{}, fmt.Errorf("argumentos de más: %v", fs.Args())
	}

	cfg := config{
		apply:   *apply,
		confirm: strings.TrimSpace(*confirm),
		dbName:  strings.TrimSpace(*dbName),
		backup:  strings.TrimSpace(*backup),
		revert:  strings.TrimSpace(*revert),
		pause:   *pause,
		verbose: *verbose,
	}

	var err error
	if cfg.at, err = parseAt(*at, now); err != nil {
		return config{}, err
	}
	if cfg.only, err = parseOnly(*only); err != nil {
		return config{}, err
	}
	switch {
	case cfg.apply && cfg.confirm == "":
		return config{}, errors.New("-apply exige -confirm con <ENV>/<base>")
	case cfg.revert != "" && (*only != "" || cfg.backup != ""):
		return config{}, errors.New("-revert no admite -only ni -backup")
	case cfg.pause < 0:
		return config{}, errors.New("-pausa no puede ser negativa")
	}
	return cfg, nil
}

// parseAt lee la fecha de referencia; vacía es now.
func parseAt(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return now.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.DateOnly, s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("-at %q no es una fecha RFC 3339 ni AAAA-MM-DD", s)
}

// parseOnly lee las normalizaciones de -only; vacío son todas.
func parseOnly(s string) (map[string]bool, error) {
	only := map[string]bool{}
	if strings.TrimSpace(s) == "" {
		for _, fix := range allFixes {
			only[fix] = true
		}
		return only, nil
	}
	for _, part := range strings.Split(s, ",") {
		fix := strings.TrimSpace(part)
		if !isFix(fix) {
			return nil, fmt.Errorf("-only: %q no es una normalización (numericos, privados, vendidos)", fix)
		}
		only[fix] = true
	}
	return only, nil
}

// isFix indica si name es una de las normalizaciones.
func isFix(name string) bool {
	for _, fix := range allFixes {
		if fix == name {
			return true
		}
	}
	return false
}

// run conecta con la base, audita y, con -apply, normaliza y vuelve a auditar. Devuelve el código de salida.
func run(cfg config) (int, error) {
	if err := api_config.LoadEnvFiles(); err != nil {
		return exitError, err
	}
	url := strings.TrimSpace(os.Getenv("FA_MONGO_URL"))
	dbName := strings.TrimSpace(os.Getenv("FA_MONGO_DBNAME"))
	if cfg.dbName != "" {
		dbName = cfg.dbName
	}
	if url == "" || dbName == "" {
		return exitError, errors.New("FA_MONGO_URL y FA_MONGO_DBNAME (o -db) son requeridas")
	}
	env := strings.TrimSpace(os.Getenv("ENV"))
	if cfg.apply {
		if err := checkConfirm(cfg.confirm, env, dbName); err != nil {
			return exitError, err
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, replicaSet, err := connect(ctx, url)
	if err != nil {
		return exitError, err
	}
	defer func() { _ = client.Disconnect(context.Background()) }()
	db := mongocc.NewFromClient(client, dbName, nil)

	rev, modified := build()
	fmt.Printf("Entorno %s · base %s · replica set %s · %s\n", env, dbName, orNone(replicaSet), buildText(rev, modified))
	if cfg.apply && replicaSet == "" {
		return exitError, errors.New("-apply necesita un replica set: las normalizaciones usan transacciones")
	}

	if cfg.revert != "" {
		return revertBackup(ctx, db, cfg, env)
	}

	a, err := collect(ctx, db, cfg.at)
	if err != nil {
		return exitError, err
	}
	printAudit(os.Stdout, a, cfg.verbose)
	printSummary(os.Stdout, "Resumen", a)
	if !cfg.apply {
		if a.pending() {
			fmt.Println("Sin -apply no se modificó nada.")
			return exitPending, nil
		}
		return exitOK, nil
	}

	runAt := shared_domain.Now().Truncate(time.Millisecond)
	path := cfg.backup
	if path == "" {
		path = filepath.Join("tmp", "auditdata", fmt.Sprintf("%s-%s.jsonl", dbName, runAt.Format("20060102T150405Z")))
	}
	header := backupHeader{
		Kind: lineRun, Run: runAt, Env: env, DB: dbName, ReplicaSet: replicaSet,
		Only: selectedFixes(cfg.only), Revision: rev, Modified: modified,
	}
	backup := newBackupWriter(path, header)
	defer func() { _ = backup.Close() }()

	results, err := applyAll(ctx, db, a, applyOptions{only: cfg.only, runAt: runAt, pause: cfg.pause, backup: backup})
	printApply(os.Stdout, results, backup, cfg.verbose)
	if err != nil {
		return exitError, err
	}

	after, err := collect(ctx, db, cfg.at)
	if err != nil {
		return exitError, err
	}
	printSummary(os.Stdout, "Después de aplicar", after)
	if after.pending() {
		return exitPending, nil
	}
	return exitOK, nil
}

// connect abre el cliente y devuelve el nombre del replica set, o "" si la base no es un replica set. El error
// de parseo se reemplaza porque puede traer la URL con credenciales; los del driver traen host y puerto, nunca
// credenciales.
func connect(ctx context.Context, url string) (*mongo.Client, string, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(url).SetServerSelectionTimeout(10 * time.Second))
	if err != nil {
		// el error de parseo puede incluir la URL con credenciales
		return nil, "", errors.New("FA_MONGO_URL no es una URL de Mongo válida")
	}
	var hello struct {
		SetName string `bson:"setName"`
	}
	if err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&hello); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, "", fmt.Errorf("sin respuesta de Mongo: %w", err)
	}
	return client, hello.SetName, nil
}

// checkConfirm exige que -confirm sea el entorno y la base de la conexión.
func checkConfirm(confirm, env, dbName string) error {
	if want := env + "/" + dbName; confirm != want {
		return fmt.Errorf("-confirm %q no coincide con la conexión %q", confirm, want)
	}
	return nil
}

// selectedFixes devuelve las normalizaciones elegidas, en el orden en que se aplican.
func selectedFixes(only map[string]bool) []string {
	out := []string{}
	for _, fix := range allFixes {
		if only[fix] {
			out = append(out, fix)
		}
	}
	return out
}

// orNone devuelve s, o "(ninguno)" si está vacío.
func orNone(s string) string {
	if s == "" {
		return "(ninguno)"
	}
	return s
}
