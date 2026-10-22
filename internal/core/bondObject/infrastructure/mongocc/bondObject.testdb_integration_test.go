//go:build integration

// Tests de integración del repositorio de bondObject y de las agregaciones sobre bondBound contra un
// Mongo real. Usan su propia base, que se borra al empezar y al terminar.
//
//	go test -tags=integration ./internal/core/bondObject/infrastructure/mongocc/
//
// La URL sale de FA_MONGO_URL o, si no está en el entorno, de .env y .env.<ENV> en la raíz del
// repositorio. Sin Mongo disponible, los tests se saltan.
package bondObject_mongocc_repository_test

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
	bondObject_application "github.com/futurapps/ms-blooming-go/internal/core/bondObject/application"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	bondObject_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondObject/infrastructure/mongocc"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// testDBName es la base de estos tests; nunca la de FA_MONGO_DBNAME.
const testDBName = "blooming_it_bondobject_mongocc"

// Colecciones que leen o escriben los adapters.
const (
	objectCollection  = "bondObject"
	typeCollection    = "bondType"
	boundCollection   = "bondBound"
	profileCollection = "profile"
)

var collections = []string{objectCollection, typeCollection, boundCollection, profileCollection}

// IDs de los fixtures: Meteor (17 alfanuméricos) y UUID v4, con referencias cruzadas mixtas.
const (
	agencyA      = "tDr6NwSsDe4Lnccp8"
	agencyB      = "0b6f8c2e-3d4a-4e5b-9c6d-7e8f9a0b1c2d"
	typeID       = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"
	legacyTypeID = "kQ3xT9mPz2LwA7bNc"
	otherTypeID  = "6a5b4c3d-2e1f-4a0b-8c9d-7e6f5a4b3c2d"
	userID       = "Hn4Wq8Ls2Dx6Vb9Ty"
	user2ID      = "8d7c6b5a-4f3e-4d2c-a1b0-9f8e7d6c5b4a"

	// productos
	meteor1 = "Bd9Xc4Vn7Mz2Lk5Jh"
	meteor2 = "Rv6Tb1Yn5Um9Io3Pa"
	meteor3 = "Zp5Rk1Mf7Gc3Qs8Ju"
	meteor4 = "Kq7Zt2RmX9pLw4NcB"
	uuid1   = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"
	uuid2   = "1b2c3d4e-5f6a-4b7c-9d8e-0f1a2b3c4d5e"
	uuid3   = "5c4b3a29-1807-4f6e-9d5c-4b3a29180716"
	uuid4   = "9e8d7c6b-5a4f-4e3d-b2c1-0a9b8c7d6e5f"
)

var fixtureIDs = []string{
	agencyA, agencyB, typeID, legacyTypeID, otherTypeID, userID, user2ID,
	meteor1, meteor2, meteor3, meteor4, uuid1, uuid2, uuid3, uuid4,
}

// fixtureAt es la fecha de alta y edición de los fixtures.
var fixtureAt = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

// investorUtcDiff es la diferencia horaria de la sesión de los inversionistas.
const investorUtcDiff = -5

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

// connect abre un cliente propio, con el registro de comandos, y comprueba que Mongo responda.
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
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("sin respuesta de Mongo: %w", err)
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

// resetDatabase borra la base de pruebas y crea las colecciones vacías.
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

// setup vacía las colecciones y el registro de comandos y devuelve el repositorio. Sin Mongo, salta el
// test.
func setup(t *testing.T) *bondObject_mongocc_repository.BondObjectMongoccRepository {
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
	return &bondObject_mongocc_repository.BondObjectMongoccRepository{DB: testDB}
}

// stats es el adapter de estadísticas sobre bondBound.
func stats() *bondObject_mongocc_repository.BondBoundStatsMongoccRepository {
	return &bondObject_mongocc_repository.BondBoundStatsMongoccRepository{DB: testDB}
}

// newApp arma los casos de uso de bondObject sobre el repositorio y el adapter de estadísticas reales.
func newApp() *bondObject_application.BondObjectApplication {
	return &bondObject_application.BondObjectApplication{
		Repository:     &bondObject_mongocc_repository.BondObjectMongoccRepository{DB: testDB},
		BondBoundStats: stats(),
	}
}

// investorSession es un context con la sesión de un inversionista de agencyID, en UTC-5.
func investorSession(userID, agencyID string) context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID:   userID,
		AgencyID: agencyID,
		UtcDiff:  session_domain.OptionalUtcDiff{Hours: investorUtcDiff, Valid: true},
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
	command    bson.Raw
}

func (r *commandRecorder) monitor() *event.CommandMonitor {
	return &event.CommandMonitor{
		Started: func(_ context.Context, e *event.CommandStartedEvent) {
			collection, _ := e.Command.Lookup(e.CommandName).StringValueOK()
			r.mu.Lock()
			defer r.mu.Unlock()
			r.started = append(r.started, recordedCommand{
				name:       e.CommandName,
				collection: collection,
				command:    append(bson.Raw(nil), e.Command...),
			})
		},
	}
}

func (r *commandRecorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.started = nil
}

// sent devuelve los comandos name que se mandaron sobre collection.
func (r *commandRecorder) sent(name, collection string) []bson.Raw {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []bson.Raw
	for _, c := range r.started {
		if c.name == name && c.collection == collection {
			out = append(out, c.command)
		}
	}
	return out
}

// earlyCashTermDoc son los términos de retiro anticipado de los fixtures.
var earlyCashTermDoc = bson.M{
	"initialPenaltyPercentage":    100.0,
	"inflectionPoint":             90,
	"inflectionPenaltyPercentage": 75.0,
	"blockWithdrawals":            false,
	"initialDayForWithdraw":       30,
}

// objectDoc arma un producto público y activo con la forma que dejaba Meteor; extra agrega o pisa campos.
func objectDoc(id, name, bondTypeID string, agencyID any, extra bson.M) bson.M {
	doc := bson.M{
		"_id":                       id,
		"name":                      name,
		"description":               "Producto " + name,
		"agencyId":                  agencyID,
		"key":                       "BP12",
		"bondTypeId":                bondTypeID,
		"amount":                    1000.0,
		"isPrivateBond":             false,
		"authorizedUserIds":         bson.A{},
		"maturityTimeInDays":        365,
		"maturityRewardsPercentage": 15.0,
		"active":                    true,
		"available":                 true,
		"saleInitDate":              nil,
		"saleEndDate":               nil,
		"earlyCashTerm":             earlyCashTermDoc,
		"createdAt":                 fixtureAt,
		"updatedAt":                 fixtureAt,
		"isRemove":                  false,
	}
	maps.Copy(doc, extra)
	return doc
}

// typeDoc arma un tipo activo de la agencia A; extra agrega o pisa campos.
func typeDoc(id, name string, extra bson.M) bson.M {
	doc := bson.M{
		"_id":         id,
		"name":        name,
		"description": "Tipo " + name,
		"agencyId":    bson.A{agencyA},
		"active":      true,
		"isRemove":    false,
		"createdAt":   fixtureAt,
		"updatedAt":   fixtureAt,
	}
	maps.Copy(doc, extra)
	return doc
}

// boundSeq numera los bonos de los fixtures.
var boundSeq int

// nextBoundID alterna _id Meteor y UUID v4 en los bonos de los fixtures.
func nextBoundID() string {
	boundSeq++
	if boundSeq%2 == 0 {
		return shared_domain.NewID()
	}
	return fmt.Sprintf("Bnd%014d", boundSeq)
}

// boundDoc arma un bono comprado con la forma de bondBound; extra agrega o pisa campos.
func boundDoc(id, objectID, userID, agencyID, statusID string, extra bson.M) bson.M {
	doc := bson.M{
		"_id":                       id,
		"bondObjectId":              objectID,
		"bondTypeId":                typeID,
		"key":                       "BP12-0001",
		"token":                     "tok-" + id,
		"userId":                    userID,
		"agencyId":                  agencyID,
		"statusId":                  statusID,
		"amount":                    1000.0,
		"maturityTimeInDays":        365,
		"maturityRewardsPercentage": 15.0,
		"earlyCashTerm":             earlyCashTermDoc,
		"initialDate":               fixtureAt,
		"maturityDate":              fixtureAt.AddDate(0, 0, 365),
		"createdAt":                 fixtureAt,
		"updatedAt":                 fixtureAt,
		"isRemove":                  false,
	}
	maps.Copy(doc, extra)
	return doc
}

// bounds arma n bonos del producto en estado iniciado, con _id Meteor y UUID alternados.
func bounds(n int, objectID, userID, agencyID string, extra bson.M) []bson.M {
	docs := make([]bson.M, n)
	for i := range docs {
		docs[i] = boundDoc(nextBoundID(), objectID, userID, agencyID, "1", extra)
	}
	return docs
}

// passwordHash es el hash que guarda profile y nunca debe salir de Mongo.
const passwordHash = "$2b$10$Qm9ub3NBbHRvcy5IYXNoLmRlLnBydWViYQ"

// profileDoc arma un perfil con la forma real de profile, password incluido.
func profileDoc(id, firstName string) bson.M {
	return bson.M{
		"_id":          id,
		"id":           "CC-" + firstName,
		"first_name":   firstName,
		"last_name":    "Gómez",
		"email":        strings.ToLower(firstName) + "@futurapps.com",
		"avatar":       "avatar-" + id,
		"address":      "Calle 10 # 20-30",
		"avatar_url":   "https://cdn.futurapps.com/avatars/" + id + ".png",
		"agencyId":     agencyA,
		"password":     passwordHash,
		"rolesIds":     bson.A{"r1", "r2"},
		"country_code": "CO",
		"createdAt":    fixtureAt,
		"updatedAt":    fixtureAt,
		"isRemove":     false,
	}
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

// rawDoc lee el documento tal como quedó en Mongo, sin pasar por el repositorio.
func rawDoc(t *testing.T, collection, id string) bson.M {
	t.Helper()
	var doc bson.M
	if err := testDB.FindOne(context.Background(), collection, bson.M{"_id": id}, nil).Decode(&doc); err != nil {
		t.Fatalf("no se pudo leer %s de %s: %v", id, collection, err)
	}
	return doc
}

// ids devuelve los _id de objects en orden.
func ids(objects []*bondObject_domain.BondObject) []string {
	out := make([]string, len(objects))
	for i, o := range objects {
		out[i] = o.ID
	}
	return out
}
