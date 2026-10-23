//go:build integration

// Tests de integración de la compra contra un Mongo real con replica set, que las transacciones requieren.
// Arman la aplicación con los adapters de Mongo, como el composition root, y solo reemplazan la
// notificación por HTTP. Usan su propia base, que se borra al empezar y al terminar.
//
//	go test -tags=integration ./internal/core/bondBound/application/
//
// La URL sale de FA_MONGO_URL o, si no está en el entorno, de .env y .env.<ENV> en la raíz del
// repositorio. Sin Mongo disponible, los tests se saltan.
package bondBound_application_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/codecraftkit/mongocc"
	bondBound_application "github.com/futurapps/ms-blooming-go/internal/core/bondBound/application"
	bondBound_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondBound/infrastructure/mongocc"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	bondHistory_application "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/application"
	bondHistory_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/infrastructure/mongocc"
	bondObject_application "github.com/futurapps/ms-blooming-go/internal/core/bondObject/application"
	bondObject_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondObject/infrastructure/mongocc"
	utils_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/adapters/utils"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/mongocc"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// testDBName es la base de estos tests; nunca la de FA_MONGO_DBNAME.
const testDBName = "blooming_it_bondbound_application"

// Colecciones que lee o escribe la compra.
const (
	boundCollection   = "bondBound"
	counterCollection = "bondKeyCounter"
	historyCollection = "bondHistory"
	objectCollection  = "bondObject"
	accountCollection = "balanceAccount"
	detailCollection  = "balanceDetail"
	noticeCollection  = "notice"
)

// movementCollections son las colecciones que escribe un retiro.
var movementCollections = []string{detailCollection, "balanceHistory", "balanceOperation"}

// collections se crean antes de los tests para que ninguna transacción tenga que crearlas.
var collections = []string{
	boundCollection, counterCollection, historyCollection, objectCollection, accountCollection,
	detailCollection, "balanceHistory", "balanceOperation", noticeCollection, "profile",
}

// IDs legacy de los fixtures, Meteor de 17 caracteres; los UUID salen de bondBound.fakes_test.go.
const (
	legacyAccountID = "Kq7Zt2RmX9pLw4NcB"
	legacyObjectID  = "Bd9Xc4Vn7Mz2Lk5Jh"
	legacyTypeID    = "Tp4Hs8Nc2Vb6Qw3Er"
)

// gateTimeout es lo máximo que una compra espera a la otra en gatedKeyCounter.
const gateTimeout = 10 * time.Second

// fixtureAt es la fecha de alta de las cuentas y los productos.
var fixtureAt = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

var (
	testDB     *mongocc.MongoQueries
	skipReason string
)

func TestMain(m *testing.M) {
	for _, id := range []string{ownerID, agencyA, accountID, objectID, typeID, legacyAccountID, legacyObjectID, legacyTypeID} {
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

// connect abre un cliente propio y exige un replica set.
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

// resetDatabase borra la base de pruebas, crea las colecciones vacías y los índices de bondBound y
// bondHistory, como el composition root.
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
	if err := (&bondBound_mongocc_repository.BondBoundMongoccRepository{DB: testDB}).EnsureIndexes(ctx); err != nil {
		return fmt.Errorf("no se pudieron crear los índices de bondBound: %w", err)
	}
	if err := (&bondHistory_mongocc_repository.BondHistoryMongoccRepository{DB: testDB}).EnsureIndexes(ctx); err != nil {
		return fmt.Errorf("no se pudieron crear los índices de bondHistory: %w", err)
	}
	return nil
}

// setupDB vacía las colecciones; los índices quedan. Sin Mongo, salta el test.
func setupDB(t *testing.T) {
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
}

// txRunner es el runner de transacciones del composition root; no guarda estado.
func txRunner() *shared_mongocc_repository.TransactionRunner {
	return &shared_mongocc_repository.TransactionRunner{DB: testDB}
}

// realBalance es el BalancePort de Mongo.
func realBalance() shared_ports.BalancePort {
	return &shared_mongocc_repository.BalanceMongoccRepository{DB: testDB, Tx: txRunner(), Utils: utils_adapters.NewUtils()}
}

// realKeyCounter es el contador de claves de Mongo.
func realKeyCounter() bondBound_ports.KeyCounterPort {
	return &bondBound_mongocc_repository.KeyCounterMongoccRepository{DB: testDB}
}

// integrationApp arma la aplicación de bondBound con los adapters de Mongo, salvo el saldo y el contador,
// que recibe para poder envolverlos, y la notificación, que es un fake.
func integrationApp(balance shared_ports.BalancePort, keys bondBound_ports.KeyCounterPort) *bondBound_application.BondBoundApplication {
	profiles := &shared_mongocc_repository.ProfileMongoccRepository{DB: testDB}
	return &bondBound_application.BondBoundApplication{
		Repository: &bondBound_mongocc_repository.BondBoundMongoccRepository{DB: testDB},
		KeyCounter: keys,
		BondObjects: &bondObject_application.BondObjectApplication{
			Repository: &bondObject_mongocc_repository.BondObjectMongoccRepository{DB: testDB},
		},
		Balance: balance,
		History: &bondHistory_application.BondHistoryApplication{
			Repository: &bondHistory_mongocc_repository.BondHistoryMongoccRepository{DB: testDB},
			Profiles:   profiles,
		},
		Tx:                      txRunner(),
		Utils:                   utils_adapters.NewUtils(),
		Notices:                 &shared_mongocc_repository.NoticeMongoccRepository{DB: testDB},
		Notifications:           &fakeNotifications{},
		Profiles:                profiles,
		BulkPurchaseMaxQuantity: maxQuantity,
	}
}

// errWithdraw es la falla que simula failingBalance.
var errWithdraw = errors.New("falla simulada del retiro")

// failingBalance usa el BalancePort real, pero el retiro número failOn devuelve errWithdraw.
type failingBalance struct {
	shared_ports.BalancePort

	failOn    int
	withdraws int
}

func (b *failingBalance) Withdraw(ctx context.Context, p shared_domain.WithdrawParams) (*shared_domain.BalanceMovement, error) {
	b.withdraws++
	if b.withdraws == b.failOn {
		return nil, errWithdraw
	}
	return b.BalancePort.Withdraw(ctx, p)
}

// gatedKeyCounter usa el contador real, pero cada Reserve espera a que lleguen parties reservas: así las
// compras concurrentes ya validaron y abrieron su transacción antes de que una confirme.
type gatedKeyCounter struct {
	bondBound_ports.KeyCounterPort

	parties int32
	arrived atomic.Int32
	open    chan struct{}
}

func newGatedKeyCounter(parties int32) *gatedKeyCounter {
	return &gatedKeyCounter{KeyCounterPort: realKeyCounter(), parties: parties, open: make(chan struct{})}
}

// Reserve espera a las demás compras y reserva con el contador real; los reintentos de la transacción
// ya no esperan.
func (g *gatedKeyCounter) Reserve(ctx context.Context, bondTypeID string, n int) (int, error) {
	if g.arrived.Add(1) == g.parties {
		close(g.open)
	}
	select {
	case <-g.open:
	case <-time.After(gateTimeout):
		return 0, errors.New("las compras concurrentes no llegaron juntas a la reserva")
	}
	return g.KeyCounterPort.Reserve(ctx, bondTypeID, n)
}

// insertAccount guarda una cuenta activa con la forma de balanceAccount. legacyInt guarda los montos como
// int32, igual que los documentos que creó Meteor.
func insertAccount(t *testing.T, id, userID, agencyID string, available float64, legacyInt bool) {
	t.Helper()
	num := func(v float64) any {
		if legacyInt {
			return int32(v)
		}
		return v
	}
	insert(t, accountCollection, bson.M{
		"_id":                id,
		"userId":             userID,
		"agencyId":           agencyID,
		"number":             "4000123412341234",
		"statusId":           shared_domain.BalanceAccountStatusActive,
		"typeId":             "1",
		"available":          num(available),
		"blocked":            num(0),
		"deferred":           num(0),
		"average":            num(0),
		"global":             num(available),
		"initialAmount":      num(0),
		"initialMonthAmount": num(0),
		"openedBy":           userID,
		"createdAt":          fixtureAt,
		"updatedAt":          fixtureAt,
		"isRemove":           false,
	})
}

// insertObject guarda un producto público y activo de la agencia A, de 1000 y 365 días, con clave BP12 y la
// forma que dejaba Meteor.
func insertObject(t *testing.T, id, bondTypeID string) {
	t.Helper()
	insert(t, objectCollection, bson.M{
		"_id":                       id,
		"name":                      "Bono Plus 12M",
		"description":               "Renta fija 12 meses",
		"agencyId":                  bson.A{agencyA},
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
		"earlyCashTerm": bson.M{
			"initialPenaltyPercentage":    100.0,
			"inflectionPoint":             90,
			"inflectionPenaltyPercentage": 75.0,
			"blockWithdrawals":            false,
			"initialDayForWithdraw":       30,
		},
		"createdAt": fixtureAt,
		"updatedAt": fixtureAt,
		"isRemove":  false,
	})
}

// seedCounter deja el contador del tipo en seq, como la siembra.
func seedCounter(t *testing.T, bondTypeID string, seq int) {
	t.Helper()
	insert(t, counterCollection, bson.M{"_id": bondTypeID, "seq": seq})
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

// rawDoc lee el documento tal como quedó en Mongo.
func rawDoc(t *testing.T, collection, id string) bson.Raw {
	t.Helper()
	raw, err := testDB.FindOne(context.Background(), collection, bson.M{"_id": id}, nil).Raw()
	if err != nil {
		t.Fatalf("no se pudo leer %s de %s: %v", id, collection, err)
	}
	return raw
}

// findAll decodifica todos los documentos de collection.
func findAll[T any](t *testing.T, collection string) []T {
	t.Helper()
	ctx := context.Background()
	cursor, err := testDB.Find(ctx, collection, bson.M{}, options.Find())
	if err != nil {
		t.Fatalf("no se pudo leer %s: %v", collection, err)
	}
	var out []T
	if err := cursor.All(ctx, &out); err != nil {
		t.Fatalf("no se pudo decodificar %s: %v", collection, err)
	}
	return out
}

func countDocs(t *testing.T, collection string) int64 {
	t.Helper()
	n, err := testDB.CountDocuments(context.Background(), collection, bson.M{})
	if err != nil {
		t.Fatalf("no se pudo contar %s: %v", collection, err)
	}
	return n
}

// availableOf devuelve el disponible guardado de la cuenta.
func availableOf(t *testing.T, id string) float64 {
	t.Helper()
	var account struct {
		Available float64 `bson:"available"`
	}
	if err := testDB.FindOne(context.Background(), accountCollection, bson.M{"_id": id}, nil).Decode(&account); err != nil {
		t.Fatalf("no se pudo leer la cuenta %s: %v", id, err)
	}
	return account.Available
}

// counterSeq devuelve el seq guardado del tipo; ok es false si el contador no existe.
func counterSeq(t *testing.T, bondTypeID string) (seq int, ok bool) {
	t.Helper()
	var c struct {
		Seq int `bson:"seq"`
	}
	err := testDB.FindOne(context.Background(), counterCollection, bson.M{"_id": bondTypeID}, nil).Decode(&c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return 0, false
	}
	if err != nil {
		t.Fatalf("no se pudo leer el contador de %s: %v", bondTypeID, err)
	}
	return c.Seq, true
}
