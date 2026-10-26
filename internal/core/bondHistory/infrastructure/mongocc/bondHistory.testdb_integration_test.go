//go:build integration

// Tests de integración del historial contra un Mongo real con replica set, que las transacciones requieren:
// el registro con Add, los listados con su filtro, su orden y los registros de Meteor, y la lectura de
// perfiles en lote. Arman la aplicación con los adapters de Mongo, como el composition root. Usan su propia
// base, que se borra al empezar y al terminar.
//
//	go test -tags=integration ./internal/core/bondHistory/infrastructure/mongocc/
//
// La URL sale de FA_MONGO_URL o, si no está en el entorno, de .env y .env.<ENV> en la raíz del
// repositorio. Sin Mongo disponible, los tests se saltan.
package bondHistory_mongocc_repository_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/codecraftkit/mongocc"
	bondHistory_application "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/application"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondHistory_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/infrastructure/mongocc"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/mongocc"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// testDBName es la base de estos tests; nunca la de FA_MONGO_DBNAME.
const testDBName = "blooming_it_bondhistory_mongocc"

// Colecciones que leen o escriben el historial y su presentación.
const (
	historyCollection = "bondHistory"
	profileCollection = "profile"
)

var collections = []string{historyCollection, profileCollection}

// IDs de los fixtures: Meteor (17 alfanuméricos) y UUID v4.
const (
	agencyA = "tDr6NwSsDe4Lnccp8"
	agencyB = "0b6f8c2e-3d4a-4e5b-9c6d-7e8f9a0b1c2d"
	userID  = "Hn4Wq8Ls2Dx6Vb9Ty"
	adminID = "8d7c6b5a-4f3e-4d2c-a1b0-9f8e7d6c5b4a"

	objectID = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"
	boundID  = "5a6b7c8d-9e0f-4a1b-8c2d-3e4f5a6b7c8d"
	otherID  = "0e1f2a3b-4c5d-4e6f-8a7b-9c0d1e2f3a4b"
	// legacyBoundID es un bono comprado en Meteor.
	legacyBoundID = "Kp3Wq8Zr5Tn2Yx7Lm"
	// registros del historial escritos por Meteor
	meteorSoldID    = "Ms4Fv8Gb2Hn6Jm1Kq"
	meteorChangedID = "Mc7Rt3Yu9Io5Pa2Sd"
)

var fixtureIDs = []string{
	agencyA, agencyB, userID, adminID, objectID, boundID, otherID, legacyBoundID, meteorSoldID, meteorChangedID,
}

// fixtureAt es la fecha de alta de los perfiles y el primer registro de los historiales sembrados.
var fixtureAt = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

var (
	testDB     *mongocc.MongoQueries
	commands   = &commandRecorder{}
	skipReason string
)

func TestMain(m *testing.M) {
	for _, id := range fixtureIDs {
		if !shared_domain.IsValidID(id) {
			fmt.Fprintf(os.Stderr, "el ID de fixture %q no es Meteor ni UUID v4\n", id)
			os.Exit(1)
		}
	}

	client, err := connect()
	if err != nil {
		skipReason = err.Error()
		os.Exit(m.Run())
	}
	testDB = mongocc.NewFromClient(client, testDBName, nil)

	code := 1
	if err := resetDatabase(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		code = m.Run()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := testDB.Database().Drop(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "no se pudo borrar la base %s: %v\n", testDBName, err)
	}
	_ = client.Disconnect(ctx)
	cancel()
	os.Exit(code)
}

// connect abre un cliente propio, con el registro de comandos, y exige un replica set.
func connect() (*mongo.Client, error) {
	url, err := mongoURL()
	if err != nil {
		return nil, err
	}
	if url == "" {
		return nil, errors.New("FA_MONGO_URL no está en el entorno ni en .env")
	}

	client, err := mongo.Connect(options.Client().
		ApplyURI(url).
		SetServerSelectionTimeout(5 * time.Second).
		SetMonitor(commands.monitor()))
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

// resetDatabase borra la base de pruebas, crea las colecciones vacías y los índices del historial, como el
// composition root.
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
	if err := newRepo().EnsureIndexes(ctx); err != nil {
		return fmt.Errorf("no se pudieron crear los índices de %s: %w", historyCollection, err)
	}
	return nil
}

// setup vacía las colecciones, sin borrar sus índices, y el registro de comandos. Sin Mongo, salta el
// test.
func setup(t *testing.T) {
	t.Helper()
	if skipReason != "" {
		t.Skip("sin Mongo para integración: " + skipReason)
	}
	ctx := context.Background()
	for _, name := range collections {
		if _, err := testDB.GetCollection(name).DeleteMany(ctx, bson.M{}); err != nil {
			t.Fatalf("no se pudo vaciar %s: %v", name, err)
		}
	}
	commands.reset()
}

// newRepo es el repositorio del historial.
func newRepo() *bondHistory_mongocc_repository.BondHistoryMongoccRepository {
	return &bondHistory_mongocc_repository.BondHistoryMongoccRepository{DB: testDB}
}

// newApp arma los casos de uso del historial con los adapters reales, como el composition root.
func newApp() *bondHistory_application.BondHistoryApplication {
	return &bondHistory_application.BondHistoryApplication{
		Repository: newRepo(),
		Profiles:   &shared_mongocc_repository.ProfileMongoccRepository{DB: testDB},
	}
}

// txRunner es el runner de transacciones del composition root.
func txRunner() *shared_mongocc_repository.TransactionRunner {
	return &shared_mongocc_repository.TransactionRunner{DB: testDB}
}

// userSession es un context con la sesión de userID en agencyID, que no es la agencia principal.
func userSession(userID, agencyID string) context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{UserID: userID, AgencyID: agencyID})
}

// adminSession es un context con la sesión de un admin de Blooming en agencyID, que no es la agencia
// principal.
func adminSession(agencyID string) context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID:        adminID,
		AgencyID:      agencyID,
		AdminBlooming: true,
	})
}

// commandRecorder guarda los comandos que el cliente manda a Mongo.
type commandRecorder struct {
	mu      sync.Mutex
	started []recordedCommand
}

type recordedCommand struct {
	name       string
	collection string
}

func (r *commandRecorder) monitor() *event.CommandMonitor {
	return &event.CommandMonitor{
		Started: func(_ context.Context, e *event.CommandStartedEvent) {
			collection, _ := e.Command.Lookup(e.CommandName).StringValueOK()
			r.mu.Lock()
			defer r.mu.Unlock()
			r.started = append(r.started, recordedCommand{name: e.CommandName, collection: collection})
		},
	}
}

func (r *commandRecorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.started = nil
}

// count devuelve cuántos comandos name se mandaron sobre collection.
func (r *commandRecorder) count(name, collection string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, c := range r.started {
		if c.name == name && c.collection == collection {
			n++
		}
	}
	return n
}

// profileDoc arma un perfil con la forma real de profile.
func profileDoc(id, firstName, lastName string) bson.M {
	return bson.M{
		"_id":        id,
		"id":         "CC-" + firstName,
		"first_name": firstName,
		"last_name":  lastName,
		"email":      strings.ToLower(firstName) + "@futurapps.com",
		"agencyId":   agencyA,
		"password":   "$2b$10$Qm9ub3NBbHRvcy5IYXNoLmRlLnBydWViYQ",
		"createdAt":  fixtureAt,
		"updatedAt":  fixtureAt,
		"isRemove":   false,
	}
}

// msHistory guarda con el repositorio un registro de userID en agencyID creado en createdAt, como lo
// escribe el ms, y devuelve su _id.
func msHistory(t *testing.T, typeID bondHistory_domain.HistoryType, relatedID, agencyID, userID, text string, params []string, createdAt time.Time) string {
	t.Helper()
	h := bondHistory_domain.NewBondHistory(bondHistory_domain.AddParams{
		RelatedID: relatedID,
		TypeID:    typeID,
		Text:      text,
		Params:    params,
	}, userID, agencyID)
	h.CreatedAt, h.UpdatedAt = createdAt, createdAt
	if err := newRepo().InsertOne(context.Background(), h); err != nil {
		t.Fatalf("no se pudo guardar el registro de %s: %v", relatedID, err)
	}
	return h.ID
}

// meteorHistoryDoc arma un registro como lo dejaba Model.bondHistory.add con el hook de inserción de Meteor:
// _id de 17 caracteres y sin changes ni editedProperties; extra agrega o pisa campos.
func meteorHistoryDoc(id, relatedID, userID, text string, params []string, createdAt time.Time, extra bson.M) bson.M {
	doc := bson.M{
		"_id":       id,
		"relatedId": relatedID,
		"userId":    userID,
		"typeId":    string(bondHistory_domain.HistoryBondBound),
		"agencyId":  agencyA,
		"text":      text,
		"params":    params,
		"createdAt": createdAt,
		"updatedAt": createdAt,
		"isRemove":  false,
	}
	maps.Copy(doc, extra)
	return doc
}

// insert guarda los documentos crudos en collection.
func insert(t *testing.T, collection string, docs ...bson.M) {
	t.Helper()
	for _, doc := range docs {
		if _, err := testDB.InsertOne(context.Background(), collection, doc); err != nil {
			t.Fatalf("no se pudo guardar %v en %s: %v", doc["_id"], collection, err)
		}
	}
}

// rawDocs devuelve todos los documentos de collection tal como están guardados.
func rawDocs(t *testing.T, collection string) []bson.Raw {
	t.Helper()
	ctx := context.Background()
	cursor, err := testDB.Find(ctx, collection, bson.M{}, options.Find())
	if err != nil {
		t.Fatalf("no se pudo leer %s: %v", collection, err)
	}
	var docs []bson.Raw
	if err := cursor.All(ctx, &docs); err != nil {
		t.Fatalf("no se pudo decodificar %s: %v", collection, err)
	}
	return docs
}

func countDocs(t *testing.T, collection string) int64 {
	t.Helper()
	n, err := testDB.CountDocuments(context.Background(), collection, bson.M{})
	if err != nil {
		t.Fatalf("no se pudo contar %s: %v", collection, err)
	}
	return n
}

// entryIDs devuelve los _id de los registros, en orden.
func entryIDs(entries []bondHistory_domain.HistoryEntry) []string {
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.ID
	}
	return ids
}
