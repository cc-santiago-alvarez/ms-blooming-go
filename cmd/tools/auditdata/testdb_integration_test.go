//go:build integration

// Tests de integración de la auditoría contra un Mongo real con replica set, que las transacciones de -apply
// requieren. Usan su propia base, que se borra al empezar y al terminar.
//
//	go test -tags=integration ./cmd/tools/auditdata/
//
// La URL sale de FA_MONGO_URL o, si no está en el entorno, de .env y .env.<ENV> en la raíz del repositorio. Sin
// Mongo disponible, los tests se saltan.
package main

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
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// testDBName es la base de estos tests; nunca la de FA_MONGO_DBNAME.
const testDBName = "blooming_it_auditdata"

// collections se crean antes de los tests para que ninguna transacción tenga que crearlas.
var collections = []string{
	bondTypeCollection, bondObjectCollection, bondBoundCollection, bondHistoryCollection,
	bondUserSettingCollection, profileCollection, balanceAccountCollection,
}

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
	missingID      = "9e8d7c6b-5a4f-4e3d-b2c1-0a9b8c7d6e5f"
)

var fixtureIDs = []string{agencyA, agencyB, typeID, legacyTypeID, objectID, legacyObjectID, userID, user2ID, missingID}

// fixtureAt es la fecha de compra de los fixtures; los bonos vencen un año después.
var fixtureAt = time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC)

// referenceAt es la fecha de referencia de la maduración en los tests: los bonos de fixtureAt ya vencieron.
var referenceAt = time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

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

	client, err := connectTestDB()
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

// connectTestDB abre un cliente propio y exige un replica set.
func connectTestDB() (*mongo.Client, error) {
	url, err := mongoURL()
	if err != nil {
		return nil, err
	}
	if url == "" {
		return nil, errors.New("FA_MONGO_URL no está en el entorno ni en .env")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, replicaSet, err := connect(ctx, url)
	if err != nil {
		return nil, err
	}
	if replicaSet == "" {
		_ = client.Disconnect(context.Background())
		return nil, errors.New("el Mongo de pruebas no es un replica set y las transacciones lo requieren")
	}
	return client, nil
}

// mongoURL lee FA_MONGO_URL del entorno o, si no está, de .env y .env.<ENV> en la raíz del repositorio, sin
// modificar el entorno del proceso.
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

// setup vacía las colecciones. Sin Mongo, salta el test.
func setup(t *testing.T) {
	t.Helper()
	if skipReason != "" {
		t.Skip("sin Mongo para integración: " + skipReason)
	}
	for _, name := range collections {
		if _, err := testDB.GetCollection(name).DeleteMany(context.Background(), bson.M{}); err != nil {
			t.Fatalf("no se pudo vaciar %s: %v", name, err)
		}
	}
}

// meteorID arma un _id de Meteor de 17 caracteres a partir de un prefijo de letras.
func meteorID(prefix string) string {
	id := prefix + strings.Repeat("x", 17)
	return id[:17]
}

// earlyCashTermDoc son los términos de retiro anticipado de los fixtures.
func earlyCashTermDoc() bson.D {
	return bson.D{
		{Key: "initialPenaltyPercentage", Value: int32(100)},
		{Key: "inflectionPoint", Value: int32(90)},
		{Key: "inflectionPenaltyPercentage", Value: int32(75)},
		{Key: "blockWithdrawals", Value: false},
		{Key: "initialDayForWithdraw", Value: int32(30)},
	}
}

// typeDoc arma un tipo activo de la agencia A, con la forma de Meteor.
func typeDoc(id string, extra bson.M) bson.M {
	doc := bson.M{
		"_id": id, "name": "Tipo", "description": "Tipo de prueba", "agencyId": bson.A{agencyA},
		"active": true, "createdAt": fixtureAt, "updatedAt": fixtureAt, "isRemove": false,
	}
	maps.Copy(doc, extra)
	return doc
}

// objectDoc arma un producto activo y público de la agencia A, con la forma de Meteor.
func objectDoc(id, bondTypeID string, extra bson.M) bson.M {
	doc := bson.M{
		"_id": id, "name": "Producto", "description": "Producto de prueba", "agencyId": bson.A{agencyA},
		"key": "BP12", "bondTypeId": bondTypeID, "amount": int32(1000), "isPrivateBond": false,
		"authorizedUserIds": bson.A{}, "maturityTimeInDays": int32(365), "maturityRewardsPercentage": int32(15),
		"active": true, "available": true, "saleInitDate": nil, "saleEndDate": nil,
		"earlyCashTerm": earlyCashTermDoc(), "createdAt": fixtureAt, "updatedAt": fixtureAt, "isRemove": false,
	}
	maps.Copy(doc, extra)
	return doc
}

// boundDoc arma un bono iniciado de userID, comprado en fixtureAt, con la forma de Meteor.
func boundDoc(id string, extra bson.M) bson.M {
	doc := bson.M{
		"_id": id, "bondObjectId": objectID, "bondTypeId": legacyTypeID, "key": "BP12-" + id[:4],
		"token": "Ab3dE6", "userId": userID, "agencyId": agencyA, "statusId": "1",
		"amount": int32(1000), "maturityTimeInDays": int32(365), "maturityRewardsPercentage": int32(15),
		"earlyCashTerm": earlyCashTermDoc(), "initialDate": fixtureAt, "maturityDate": fixtureAt.AddDate(1, 0, 0),
		"createdAt": fixtureAt, "updatedAt": fixtureAt, "isRemove": false,
	}
	maps.Copy(doc, extra)
	return doc
}

// soldExtra son los campos de un bono vendido y madurado después por la regla vieja de ms-feed.
func soldExtra(extra bson.M) bson.M {
	doc := bson.M{"statusId": "2", "soldDate": fixtureAt.AddDate(0, 3, 0), "updatedAt": fixtureAt.AddDate(0, 3, 0)}
	maps.Copy(doc, extra)
	return doc
}

// saleHistoryDoc es el cambio de estado que escribe la venta de Meteor, a nombre del dueño.
func saleHistoryDoc(relatedID string) bson.M {
	return bson.M{
		"_id": shared_domain.NewID(), "relatedId": relatedID, "userId": userID, "typeId": "2", "agencyId": agencyA,
		"text": shared_domain.TextBondBoundStatusChanged, "params": bson.A{"1", "5"},
		"createdAt": fixtureAt.AddDate(0, 3, 0), "updatedAt": fixtureAt.AddDate(0, 3, 0), "isRemove": false,
	}
}

// settingDoc arma un setting vivo de Meteor.
func settingDoc(id string, user any, updatedAt time.Time, extra bson.M) bson.M {
	doc := bson.M{
		"_id": id, "userId": user, "agencyId": agencyA, "language": "es", "selectedAccountId": nil,
		"selectedBondBoundId": nil, "createdAt": fixtureAt, "updatedAt": updatedAt, "isRemove": false,
	}
	maps.Copy(doc, extra)
	return doc
}

// profileDoc arma un perfil vivo con la forma de Meteor.
func profileDoc(id string, extra bson.M) bson.M {
	doc := bson.M{
		"_id": id, "id": "1020304050", "first_name": "Nombre", "last_name": "Apellido", "email": "prueba@example.com",
		"agencyId": agencyA, "password": "hash", "createdAt": fixtureAt, "updatedAt": fixtureAt, "isRemove": false,
	}
	maps.Copy(doc, extra)
	return doc
}

// accountDoc arma una cuenta viva de userID.
func accountDoc(id string, extra bson.M) bson.M {
	doc := bson.M{
		"_id": id, "userId": userID, "agencyId": agencyA, "statusId": "1", "available": 5000.0,
		"number": "1234567890123456", "createdAt": fixtureAt, "updatedAt": fixtureAt, "isRemove": false,
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

// rawDocument lee el documento tal como quedó en Mongo.
func rawDocument(t *testing.T, collection, id string) bson.Raw {
	t.Helper()
	raw, err := testDB.FindOne(context.Background(), collection, bson.M{"_id": id}, nil).Raw()
	if err != nil {
		t.Fatalf("no se pudo leer %s de %s: %v", id, collection, err)
	}
	return raw
}

// snapshot lee todos los documentos de todas las colecciones, para comprobar que nada cambió.
func snapshot(t *testing.T) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, name := range collections {
		cursor, err := testDB.Find(context.Background(), name, bson.M{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
		if err != nil {
			t.Fatal(err)
		}
		for cursor.Next(context.Background()) {
			out[name] = append(out[name], cursor.Current.String())
		}
		if err := cursor.Err(); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

// historyOf devuelve los registros de sistema (userId "") del historial de relatedID.
func historyOf(t *testing.T, relatedID string) []bson.Raw {
	t.Helper()
	cursor, err := testDB.Find(context.Background(), bondHistoryCollection, bson.M{"relatedId": relatedID, "userId": ""}, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := []bson.Raw{}
	for cursor.Next(context.Background()) {
		out = append(out, append(bson.Raw(nil), cursor.Current...))
	}
	return out
}

// mustCollect corre la auditoría sobre la base de pruebas.
func mustCollect(t *testing.T) *audit {
	t.Helper()
	a, err := collect(context.Background(), testDB, referenceAt)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	return a
}

// mustApply aplica las normalizaciones de only con un respaldo nuevo y devuelve los resultados por normalización.
func mustApply(t *testing.T, a *audit, runAt time.Time, backupPath string, only ...string) map[string]applyResult {
	t.Helper()
	selected := map[string]bool{}
	for _, fix := range only {
		selected[fix] = true
	}
	backup := newBackupWriter(backupPath, backupHeader{Kind: lineRun, Run: runAt, DB: testDBName, Only: only})
	defer func() { _ = backup.Close() }()
	results, err := applyAll(context.Background(), testDB, a, applyOptions{only: selected, runAt: runAt, backup: backup})
	if err != nil {
		t.Fatalf("applyAll: %v", err)
	}
	out := map[string]applyResult{}
	for _, r := range results {
		out[r.Fix] = r
	}
	return out
}

// issueIDs devuelve el _id de cada hallazgo.
func issueIDs(issues []docIssue) []string {
	out := make([]string, len(issues))
	for i, is := range issues {
		out[i] = is.ID
	}
	return out
}
