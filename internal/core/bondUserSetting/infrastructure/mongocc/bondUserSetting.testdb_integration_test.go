//go:build integration

// Tests de integración de las preferencias del inversionista contra un Mongo real: el repositorio con su
// índice único parcial, los adapters de solo lectura sobre bondBound y balanceAccount, y el endpoint interno.
// Usan su propia base, que se borra al empezar y al terminar.
//
//	go test -tags=integration ./internal/core/bondUserSetting/infrastructure/mongocc/
//
// La URL sale de FA_MONGO_URL o, si no está en el entorno, de .env y .env.<ENV> en la raíz del
// repositorio. Sin Mongo disponible, los tests se saltan.
package bondUserSetting_mongocc_repository_test

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
	bondUserSetting_application "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/application"
	bondUserSetting_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/infrastructure/mongocc"
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
const testDBName = "blooming_it_bondusersetting_mongocc"

// Colecciones que leen o escriben los adapters.
const (
	settingCollection = "bondUserSetting"
	accountCollection = "balanceAccount"
	boundCollection   = "bondBound"
	profileCollection = "profile"
)

var collections = []string{settingCollection, accountCollection, boundCollection, profileCollection}

// IDs de los fixtures: Meteor (17 alfanuméricos) y UUID v4, con referencias cruzadas mixtas.
const (
	agencyA   = "tDr6NwSsDe4Lnccp8"
	agencyB   = "0b6f8c2e-3d4a-4e5b-9c6d-7e8f9a0b1c2d"
	userID    = "Hn4Wq8Ls2Dx6Vb9Ty"
	user2ID   = "8d7c6b5a-4f3e-4d2c-a1b0-9f8e7d6c5b4a"
	newUserID = "Zp5Rk1Mf7Gc3Qs8Ju"

	settingID = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"
	// legacySettingID es un setting creado por Meteor.
	legacySettingID = "Bd9Xc4Vn7Mz2Lk5Jh"

	// cuentas de userID: accountOld es la viva más antigua de la agencia A
	accountOld         = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"
	accountNew         = "Kq7Zt2RmX9pLw4NcB"
	accountRemoved     = "Rv6Tb1Yn5Um9Io3Pa"
	accountOtherAgency = "6a5b4c3d-2e1f-4a0b-8c9d-7e6f5a4b3c2d"
	// accountOfUser2 es la cuenta de user2ID.
	accountOfUser2 = "Pe4Hj7Nm1Bv6Gt2Ys"

	bondObjectID = "Rm3Fd8Sa2Wq7Ek4Lo"
)

var fixtureIDs = []string{
	agencyA, agencyB, userID, user2ID, newUserID, settingID, legacySettingID,
	accountOld, accountNew, accountRemoved, accountOtherAgency, accountOfUser2, bondObjectID,
}

// fixtureAt es la fecha de alta y edición de los settings y los bonos de los fixtures.
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

// resetDatabase borra la base de pruebas, crea las colecciones vacías y los índices de bondUserSetting,
// incluido el único parcial por userId.
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
	repo := &bondUserSetting_mongocc_repository.BondUserSettingMongoccRepository{DB: testDB}
	if err := repo.EnsureIndexes(ctx); err != nil {
		return fmt.Errorf("no se pudieron crear los índices de %s: %w", settingCollection, err)
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

// newApp arma los casos de uso de bondUserSetting con los adapters reales, como el composition root.
func newApp() *bondUserSetting_application.BondUserSettingApplication {
	return &bondUserSetting_application.BondUserSettingApplication{
		Repository:   &bondUserSetting_mongocc_repository.BondUserSettingMongoccRepository{DB: testDB},
		BondBounds:   &bondUserSetting_mongocc_repository.BoundSummaryMongoccRepository{DB: testDB},
		FirstAccount: &bondUserSetting_mongocc_repository.FirstAccountMongoccRepository{DB: testDB},
		Balance:      &shared_mongocc_repository.BalanceMongoccRepository{DB: testDB},
		Profiles:     &shared_mongocc_repository.ProfileMongoccRepository{DB: testDB},
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

// settingDoc arma un setting vivo con la forma que dejaba Meteor, sin cuenta ni bono; extra agrega o pisa
// campos.
func settingDoc(id, userID, agencyID string, extra bson.M) bson.M {
	doc := bson.M{
		"_id":                 id,
		"userId":              userID,
		"agencyId":            agencyID,
		"language":            "es",
		"selectedAccountId":   nil,
		"selectedBondBoundId": nil,
		"createdAt":           fixtureAt,
		"updatedAt":           fixtureAt,
		"isRemove":            false,
	}
	maps.Copy(doc, extra)
	return doc
}

// accountNumber es el número de las cuentas de los fixtures.
const accountNumber = "3466-01-123454821"

// accountDoc arma una cuenta activa con la forma de balanceAccount; extra agrega o pisa campos.
func accountDoc(id, userID, agencyID string, createdAt time.Time, extra bson.M) bson.M {
	doc := bson.M{
		"_id":                id,
		"userId":             userID,
		"agencyId":           agencyID,
		"number":             accountNumber,
		"statusId":           "1",
		"typeId":             "1",
		"available":          5000.0,
		"blocked":            0.0,
		"deferred":           0.0,
		"average":            0.0,
		"global":             5000.0,
		"initialAmount":      0.0,
		"initialMonthAmount": 0.0,
		"createdAt":          createdAt,
		"updatedAt":          createdAt,
		"isRemove":           false,
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

// boundDoc arma un bono comprado con la forma de bondBound. amount es any para guardar también los int32
// que dejó Meteor; extra agrega o pisa campos.
func boundDoc(userID, agencyID, statusID string, amount any, extra bson.M) bson.M {
	doc := bson.M{
		"_id":          nextBoundID(),
		"bondObjectId": bondObjectID,
		"userId":       userID,
		"agencyId":     agencyID,
		"statusId":     statusID,
		"amount":       amount,
		"createdAt":    fixtureAt,
		"updatedAt":    fixtureAt,
		"isRemove":     false,
	}
	maps.Copy(doc, extra)
	return doc
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

// rawDoc devuelve el documento tal como está guardado.
func rawDoc(t *testing.T, collection, id string) bson.Raw {
	t.Helper()
	raw, err := testDB.FindOne(context.Background(), collection, bson.M{"_id": id}, nil).Raw()
	if err != nil {
		t.Fatalf("no se pudo leer %s/%s: %v", collection, id, err)
	}
	return raw
}

// countSettings cuenta los settings que cumplen filter, vivos o borrados.
func countSettings(t *testing.T, filter bson.M) int64 {
	t.Helper()
	n, err := testDB.CountDocuments(context.Background(), settingCollection, filter)
	if err != nil {
		t.Fatalf("no se pudo contar %s: %v", settingCollection, err)
	}
	return n
}

// val devuelve el valor de p, o "<nil>", para los mensajes de error.
func val(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}
