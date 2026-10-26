//go:build integration

// Tests de integración del repositorio de bondBound y del contador de claves contra un Mongo real con
// replica set, que las transacciones requieren. Usan su propia base, que se borra al empezar y al terminar.
//
//	go test -tags=integration ./internal/core/bondBound/infrastructure/mongocc/
//
// La URL sale de FA_MONGO_URL o, si no está en el entorno, de .env y .env.<ENV> en la raíz del
// repositorio. Sin Mongo disponible, los tests se saltan.
package bondBound_mongocc_repository_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/codecraftkit/mongocc"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondBound/infrastructure/mongocc"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/mongocc"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// testDBName es la base de estos tests; nunca la de FA_MONGO_DBNAME.
const testDBName = "blooming_it_bondbound_mongocc"

// Colecciones que leen o escriben los adapters; las tres últimas solo las leen los lookups.
const (
	boundCollection   = "bondBound"
	counterCollection = "bondKeyCounter"
	profileCollection = "profile"
	objectCollection  = "bondObject"
	typeCollection    = "bondType"
)

// collections se crean antes de los tests para que ninguna transacción tenga que crearlas.
var collections = []string{boundCollection, counterCollection, profileCollection, objectCollection, typeCollection}

// IDs de los fixtures: Meteor (17 alfanuméricos) y UUID v4, con referencias cruzadas mixtas.
const (
	agencyA        = "tDr6NwSsDe4Lnccp8"
	agencyB        = "0b6f8c2e-3d4a-4e5b-9c6d-7e8f9a0b1c2d"
	typeID         = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"
	legacyTypeID   = "kQ3xT9mPz2LwA7bNc"
	objectID       = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"
	legacyObjectID = "Bd9Xc4Vn7Mz2Lk5Jh"
	userID         = "Hn4Wq8Ls2Dx6Vb9Ty"
	user2ID        = "8d7c6b5a-4f3e-4d2c-a1b0-9f8e7d6c5b4a"
	boundID        = "1b2c3d4e-5f6a-4b7c-9d8e-0f1a2b3c4d5e"
	legacyBoundID  = "Rv6Tb1Yn5Um9Io3Pa"
	legacyBound2ID = "Zp5Rk1Mf7Gc3Qs8Ju"
	missingID      = "9e8d7c6b-5a4f-4e3d-b2c1-0a9b8c7d6e5f"
)

var fixtureIDs = []string{
	agencyA, agencyB, typeID, legacyTypeID, objectID, legacyObjectID, userID, user2ID,
	boundID, legacyBoundID, legacyBound2ID, missingID,
}

// fixtureAt es la fecha de compra de los fixtures.
var fixtureAt = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

var (
	testDB     *mongocc.MongoQueries
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

// setup vacía las colecciones y devuelve el repositorio; los índices quedan. Sin Mongo, salta el test.
func setup(t *testing.T) *bondBound_mongocc_repository.BondBoundMongoccRepository {
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
	return &bondBound_mongocc_repository.BondBoundMongoccRepository{DB: testDB}
}

// keyCounter es el contador de claves sobre la base de pruebas.
func keyCounter() *bondBound_mongocc_repository.KeyCounterMongoccRepository {
	return &bondBound_mongocc_repository.KeyCounterMongoccRepository{DB: testDB}
}

// txRunner es el runner de transacciones del composition root.
func txRunner() *shared_mongocc_repository.TransactionRunner {
	return &shared_mongocc_repository.TransactionRunner{DB: testDB}
}

// earlyCashTermDoc son los términos de retiro anticipado de los fixtures.
var earlyCashTermDoc = bson.M{
	"initialPenaltyPercentage":    100.0,
	"inflectionPoint":             90,
	"inflectionPenaltyPercentage": 75.0,
	"blockWithdrawals":            false,
	"initialDayForWithdraw":       30,
}

// boundDoc arma un bono iniciado con la forma que dejaba Meteor, sin keySeq; extra agrega o pisa campos.
func boundDoc(id, bondTypeID, key string, extra bson.M) bson.M {
	doc := bson.M{
		"_id":                       id,
		"bondObjectId":              objectID,
		"bondTypeId":                bondTypeID,
		"key":                       key,
		"token":                     "Ab3dE6",
		"userId":                    userID,
		"agencyId":                  agencyA,
		"statusId":                  "1",
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

// boundAt arma un bono con boundDoc creado e iniciado minutes minutos después de fixtureAt, para fijar el
// orden de los listados.
func boundAt(id string, minutes int, extra bson.M) bson.M {
	at := fixtureAt.Add(time.Duration(minutes) * time.Minute)
	doc := boundDoc(id, typeID, "BP12-"+fmt.Sprint(minutes), bson.M{"createdAt": at, "initialDate": at})
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

// objectDoc arma un producto activo de la agencia A con la forma que dejaba Meteor y dos usuarios
// autorizados; extra agrega o pisa campos.
func objectDoc(id, name, key, bondTypeID string, extra bson.M) bson.M {
	doc := bson.M{
		"_id":                       id,
		"name":                      name,
		"description":               "Producto " + name,
		"agencyId":                  bson.A{agencyA},
		"key":                       key,
		"bondTypeId":                bondTypeID,
		"amount":                    1000.0,
		"isPrivateBond":             true,
		"authorizedUserIds":         bson.A{userID, user2ID},
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

// profileDoc arma un perfil con la forma de Meteor: la cédula en id y el hash del password.
func profileDoc(id, cedula, firstName, lastName, email string) bson.M {
	return bson.M{
		"_id":        id,
		"id":         cedula,
		"first_name": firstName,
		"last_name":  lastName,
		"email":      email,
		"agencyId":   agencyA,
		"password":   "$2b$10$Qm9ub3NBbHRvcy5IYXNoLmRlLnBydWViYQ",
		"createdAt":  fixtureAt,
		"updatedAt":  fixtureAt,
		"isRemove":   false,
	}
}

// seedCatalog guarda los dos tipos, los dos productos con referencias cruzadas mixtas (el UUID apunta al tipo
// Meteor y el Meteor al tipo UUID) y los perfiles de userID y user2ID.
func seedCatalog(t *testing.T) {
	t.Helper()
	insert(t, typeCollection, typeDoc(typeID, "Plus", nil), typeDoc(legacyTypeID, "Clásico", nil))
	insert(t, objectCollection,
		objectDoc(objectID, "Bono Plus 12M", "BP12", legacyTypeID, nil),
		objectDoc(legacyObjectID, "Bono Clásico 6M", "BC06", typeID, nil))
	insert(t, profileCollection,
		profileDoc(userID, "1020304050", "Carlos", "Ramírez", "carlos@futurapps.com"),
		profileDoc(user2ID, "5040302010", "Lucía", "Torres", "lucia@futurapps.com"))
}

// ids devuelve el _id de cada documento, en el orden recibido.
func ids(docs ...bson.M) []string {
	out := make([]string, len(docs))
	for i, doc := range docs {
		out[i] = doc["_id"].(string)
	}
	return out
}

// agencyScope es el alcance de una sesión de agencyID que no es la principal.
func agencyScope(id string) *string {
	return &id
}

// adminRowIDs devuelve el _id del bono de cada fila del listado de administración.
func adminRowIDs(rows []bondBound_ports.BondBoundAdminRow) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = row.Bound.ID
	}
	return out
}

// withObjectIDs devuelve el _id del bono de cada fila con producto.
func withObjectIDs(rows []bondBound_ports.BondBoundWithObject) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = row.Bound.ID
	}
	return out
}

// boundIDs devuelve el _id de cada bono.
func boundIDs(bounds []*bondBound_domain.BondBound) []string {
	out := make([]string, len(bounds))
	for i, b := range bounds {
		out[i] = b.ID
	}
	return out
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
func rawDoc(t *testing.T, collection, id string) bson.Raw {
	t.Helper()
	raw, err := testDB.FindOne(context.Background(), collection, bson.M{"_id": id}, nil).Raw()
	if err != nil {
		t.Fatalf("no se pudo leer %s de %s: %v", id, collection, err)
	}
	return raw
}

func countDocs(t *testing.T, collection string) int64 {
	t.Helper()
	n, err := testDB.CountDocuments(context.Background(), collection, bson.M{})
	if err != nil {
		t.Fatalf("no se pudo contar %s: %v", collection, err)
	}
	return n
}

// seedCounter deja el contador del tipo en seq, como la siembra.
func seedCounter(t *testing.T, bondTypeID string, seq int) {
	t.Helper()
	insert(t, counterCollection, bson.M{"_id": bondTypeID, "seq": seq})
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
