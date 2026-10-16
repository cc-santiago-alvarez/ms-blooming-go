//go:build integration

// Tests de integración de los adapters compartidos contra un Mongo real con replica set, que las
// transacciones requieren. Usan su propia base, que se borra al empezar y al terminar.
//
//	go test -tags=integration ./internal/core/shared/infrastructure/mongocc/
//
// La URL sale de FA_MONGO_URL o, si no está en el entorno, de .env y .env.<ENV> en la raíz del
// repositorio. Sin Mongo disponible, los tests se saltan.
package shared_mongocc_repository_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/codecraftkit/mongocc"
	utils_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/adapters/utils"
	shared_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/mongocc"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// testDBName es la base de estos tests; nunca la de FA_MONGO_DBNAME.
const testDBName = "blooming_it_shared_mongocc"

// txProbeCollection es una colección propia de los tests de transacción.
const txProbeCollection = "txProbe"

// movementCollections son las colecciones que escribe un retiro o un depósito.
var movementCollections = []string{"balanceDetail", "balanceHistory", "balanceOperation"}

// collections se crean antes de los tests para que ninguna transacción tenga que crearlas.
var collections = []string{
	"balanceAccount", "balanceDetail", "balanceHistory", "balanceOperation",
	"notice", "activity", "profile", "agency", "country", txProbeCollection,
}

var (
	testDB     *mongocc.MongoQueries
	commands   = &commandRecorder{}
	skipReason string
)

func TestMain(m *testing.M) {
	if err := checkFixtureIDs(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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

// setup vacía las colecciones y el registro de comandos. Sin Mongo, salta el test.
func setup(t *testing.T) *mongocc.MongoQueries {
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
	return testDB
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

// IDs de los fixtures: Meteor (17 alfanuméricos) y UUID v4, con referencias cruzadas mixtas.
const (
	meteorAgencyID     = "tDr6NwSsDe4Lnccp8"
	uuidAgencyID       = "0b6f8c2e-3d4a-4e5b-9c6d-7e8f9a0b1c2d"
	meteorAccountID    = "Kq7Zt2RmX9pLw4NcB"
	uuidAccountID      = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"
	meteorUserID       = "Hn4Wq8Ls2Dx6Vb9Ty"
	uuidUserID         = "8d7c6b5a-4f3e-4d2c-a1b0-9f8e7d6c5b4a"
	otherUserID        = "Zp5Rk1Mf7Gc3Qs8Ju"
	removedUserID      = "Rv6Tb1Yn5Um9Io3Pa"
	missingID          = "9e8d7c6b-5a4f-4e3d-b2c1-0a9b8c7d6e5f"
	meteorCountryID    = "oKDrhpKtbNs3degQe"
	uuidCountryID      = "6a5b4c3d-2e1f-4a0b-8c9d-7e6f5a4b3c2d"
	emptySymbolAgency  = "Ar8Kd2Lp5Qw9Zx3Cv"
	removedCountryAgcy = "Pe4Hj7Nm1Bv6Gt2Ys"
	removedAgencyID    = "Rm3Fd8Sa2Wq7Ek4Lo"
	noCountryAgencyID  = "5c4b3a29-1807-4f6e-9d5c-4b3a29180716"
	meteorBondID       = "Bd9Xc4Vn7Mz2Lk5Jh"
)

var fixtureIDs = []string{
	meteorAgencyID, uuidAgencyID, meteorAccountID, uuidAccountID, meteorUserID, uuidUserID,
	otherUserID, removedUserID, missingID, meteorCountryID, uuidCountryID, emptySymbolAgency,
	removedCountryAgcy, removedAgencyID, noCountryAgencyID, meteorBondID,
}

var (
	meteorIDPattern = regexp.MustCompile(`^[A-Za-z0-9]{17}$`)
	uuidV4Pattern   = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
)

// checkFixtureIDs verifica que cada ID de fixture sea Meteor o UUID v4.
func checkFixtureIDs() error {
	for _, id := range fixtureIDs {
		if !meteorIDPattern.MatchString(id) && !uuidV4Pattern.MatchString(id) {
			return fmt.Errorf("el ID de fixture %q no es Meteor ni UUID v4", id)
		}
	}
	return nil
}

// assertUUIDv4 falla si id no es un UUID v4.
func assertUUIDv4(t *testing.T, id string) {
	t.Helper()
	if !uuidV4Pattern.MatchString(id) {
		t.Fatalf("_id %q no es un UUID v4", id)
	}
}

// accountFixture es una cuenta de saldo. legacyInt guarda los montos como int32, igual que los
// documentos que creó Meteor.
type accountFixture struct {
	name      string
	id        string
	userID    string
	agencyID  string
	legacyInt bool
}

// accountFixtures cubren los dos formatos de _id, con dueño y agencia del otro formato.
var accountFixtures = []accountFixture{
	{name: "cuenta Meteor con dueño UUID", id: meteorAccountID, userID: uuidUserID, agencyID: meteorAgencyID, legacyInt: true},
	{name: "cuenta UUID con dueño Meteor", id: uuidAccountID, userID: meteorUserID, agencyID: uuidAgencyID},
}

// accountState son los valores de la cuenta que cambian entre tests.
type accountState struct {
	statusID  string
	available float64
	global    float64
	deferred  float64
	isRemove  bool
}

// activeAccount es una cuenta activa con saldo para retirar.
var activeAccount = accountState{statusID: "1", available: 1000, global: 1500, deferred: 200}

const accountNumber = "4000123412341234"

// accountCreatedAt es la fecha de creación y de última actualización de las cuentas de prueba.
var accountCreatedAt = time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

// insertAccount guarda la cuenta con todos los campos de balanceAccount.
func insertAccount(t *testing.T, db *mongocc.MongoQueries, a accountFixture, s accountState) {
	t.Helper()
	num := func(v float64) any {
		if a.legacyInt {
			return int32(v)
		}
		return v
	}
	doc := bson.M{
		"_id":                a.id,
		"userId":             a.userID,
		"agencyId":           a.agencyID,
		"number":             accountNumber,
		"statusId":           s.statusID,
		"typeId":             "1",
		"available":          num(s.available),
		"blocked":            num(0),
		"deferred":           num(s.deferred),
		"average":            num(0),
		"global":             num(s.global),
		"initialAmount":      num(0),
		"initialMonthAmount": num(0),
		"openedBy":           otherUserID,
		"createdAt":          accountCreatedAt,
		"updatedAt":          accountCreatedAt,
		"isRemove":           s.isRemove,
	}
	if _, err := db.InsertOne(context.Background(), "balanceAccount", doc); err != nil {
		t.Fatalf("no se pudo guardar la cuenta %s: %v", a.id, err)
	}
}

// storedAccount son los campos de balanceAccount que revisan los tests.
type storedAccount struct {
	Available float64   `bson:"available"`
	Global    float64   `bson:"global"`
	Deferred  float64   `bson:"deferred"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

func readAccount(t *testing.T, db *mongocc.MongoQueries, id string) storedAccount {
	t.Helper()
	var account storedAccount
	if err := db.FindOne(context.Background(), "balanceAccount", bson.M{"_id": id}, nil).Decode(&account); err != nil {
		t.Fatalf("no se pudo leer la cuenta %s: %v", id, err)
	}
	return account
}

// rawDoc devuelve el documento tal como está guardado.
func rawDoc(t *testing.T, db *mongocc.MongoQueries, collection, id string) bson.Raw {
	t.Helper()
	raw, err := db.FindOne(context.Background(), collection, bson.M{"_id": id}, nil).Raw()
	if err != nil {
		t.Fatalf("no se pudo leer %s/%s: %v", collection, id, err)
	}
	return raw
}

// assertDocUnchanged falla si el documento cambió desde before.
func assertDocUnchanged(t *testing.T, db *mongocc.MongoQueries, collection, id string, before bson.Raw) {
	t.Helper()
	if after := rawDoc(t, db, collection, id); !bytes.Equal(before, after) {
		t.Fatalf("%s/%s cambió:\n antes   %s\n después %s", collection, id, before, after)
	}
}

// assertFieldType falla si key no está en el documento o no es del tipo esperado.
func assertFieldType(t *testing.T, raw bson.Raw, key string, want bson.Type) {
	t.Helper()
	v, err := raw.LookupErr(key)
	if err != nil {
		t.Fatalf("el documento no tiene %q: %s", key, raw)
	}
	if v.Type != want {
		t.Fatalf("%q es de tipo %s, se esperaba %s", key, v.Type, want)
	}
}

// findAll decodifica todos los documentos de collection.
func findAll[T any](t *testing.T, db *mongocc.MongoQueries, collection string) []T {
	t.Helper()
	ctx := context.Background()
	cursor, err := db.Find(ctx, collection, bson.M{}, options.Find())
	if err != nil {
		t.Fatalf("no se pudo leer %s: %v", collection, err)
	}
	var out []T
	if err := cursor.All(ctx, &out); err != nil {
		t.Fatalf("no se pudo decodificar %s: %v", collection, err)
	}
	return out
}

func countDocs(t *testing.T, db *mongocc.MongoQueries, collection string) int64 {
	t.Helper()
	n, err := db.CountDocuments(context.Background(), collection, bson.M{})
	if err != nil {
		t.Fatalf("no se pudo contar %s: %v", collection, err)
	}
	return n
}

// assertNoMovements falla si quedó escrito algún detail, history u operation.
func assertNoMovements(t *testing.T, db *mongocc.MongoQueries) {
	t.Helper()
	for _, c := range movementCollections {
		if n := countDocs(t, db, c); n != 0 {
			t.Fatalf("%s tiene %d documentos, se esperaba 0", c, n)
		}
	}
}

// newBalanceRepo arma el BalancePort igual que el composition root.
func newBalanceRepo(db *mongocc.MongoQueries) *shared_mongocc_repository.BalanceMongoccRepository {
	return &shared_mongocc_repository.BalanceMongoccRepository{
		DB:    db,
		Tx:    &shared_mongocc_repository.TransactionRunner{DB: db},
		Utils: utils_adapters.NewUtils(),
	}
}
