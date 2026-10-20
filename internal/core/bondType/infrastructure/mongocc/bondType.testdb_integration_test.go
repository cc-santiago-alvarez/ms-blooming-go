//go:build integration

// Tests de integración del repositorio de bondType contra un Mongo real. Usan su propia base, que se
// borra al empezar y al terminar.
//
//	go test -tags=integration ./internal/core/bondType/infrastructure/mongocc/
//
// La URL sale de FA_MONGO_URL o, si no está en el entorno, de .env y .env.<ENV> en la raíz del
// repositorio. Sin Mongo disponible, los tests se saltan.
package bondType_mongocc_repository_test

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
	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	bondType_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondType/infrastructure/mongocc"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// testDBName es la base de estos tests; nunca la de FA_MONGO_DBNAME.
const testDBName = "blooming_it_bondtype_mongocc"

// collection es la colección del repositorio.
const collection = "bondType"

// IDs de los fixtures: Meteor (17 alfanuméricos) y UUID v4, con referencias cruzadas mixtas.
const (
	agencyA      = "tDr6NwSsDe4Lnccp8"
	agencyB      = "0b6f8c2e-3d4a-4e5b-9c6d-7e8f9a0b1c2d"
	t1ID         = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"
	t2ID         = "Hn4Wq8Ls2Dx6Vb9Ty"
	t3ID         = "Rv6Tb1Yn5Um9Io3Pa"
	inactiveID   = "8d7c6b5a-4f3e-4d2c-a1b0-9f8e7d6c5b4a"
	legacyTypeID = "kQ3xT9mPz2LwA7bNc"
	missingID    = "9e8d7c6b-5a4f-4e3d-b2c1-0a9b8c7d6e5f"
)

var fixtureIDs = []string{agencyA, agencyB, t1ID, t2ID, t3ID, inactiveID, legacyTypeID, missingID}

// fixtureAt es la fecha de alta y edición de los fixtures.
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

// connect abre un cliente propio y comprueba que Mongo responda.
func connect() (*mongo.Client, error) {
	url, err := mongoURL()
	if err != nil {
		return nil, err
	}
	if url == "" {
		return nil, errors.New("FA_MONGO_URL no está en el entorno ni en .env")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(url).SetServerSelectionTimeout(5 * time.Second))
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

// resetDatabase borra la base de pruebas y crea la colección vacía.
func resetDatabase() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := testDB.Database().Drop(ctx); err != nil {
		return fmt.Errorf("no se pudo borrar la base %s: %w", testDBName, err)
	}
	if err := testDB.Database().CreateCollection(ctx, collection); err != nil {
		return fmt.Errorf("no se pudo crear %s: %w", collection, err)
	}
	return nil
}

// setup vacía la colección y devuelve el repositorio. Sin Mongo, salta el test.
func setup(t *testing.T) *bondType_mongocc_repository.BondTypeMongoccRepository {
	t.Helper()
	if skipReason != "" {
		t.Skip("sin Mongo para integración: " + skipReason)
	}
	if _, err := testDB.GetCollection(collection).DeleteMany(context.Background(), bson.M{}); err != nil {
		t.Fatalf("no se pudo vaciar %s: %v", collection, err)
	}
	return &bondType_mongocc_repository.BondTypeMongoccRepository{DB: testDB}
}

// legacyType arma un documento con la forma que dejaba Meteor; extra agrega o pisa campos.
func legacyType(id, name string, agencyID any, extra bson.M) bson.M {
	doc := bson.M{
		"_id":         id,
		"name":        name,
		"description": "Tipo " + name,
		"agencyId":    agencyID,
		"active":      true,
		"isRemove":    false,
		"createdAt":   fixtureAt,
		"updatedAt":   fixtureAt,
	}
	maps.Copy(doc, extra)
	return doc
}

// insertTypes guarda los documentos crudos en bondType.
func insertTypes(t *testing.T, docs ...bson.M) {
	t.Helper()
	for _, doc := range docs {
		if _, err := testDB.InsertOne(context.Background(), collection, doc); err != nil {
			t.Fatalf("no se pudo guardar %v: %v", doc["_id"], err)
		}
	}
}

// rawDoc lee el documento tal como quedó en Mongo, sin pasar por el repositorio.
func rawDoc(t *testing.T, id string) bson.M {
	t.Helper()
	var doc bson.M
	if err := testDB.FindOne(context.Background(), collection, bson.M{"_id": id}, nil).Decode(&doc); err != nil {
		t.Fatalf("no se pudo leer %s: %v", id, err)
	}
	return doc
}

// ids devuelve los _id de types en orden.
func ids(types []*bondType_domain.BondType) []string {
	out := make([]string, len(types))
	for i, bt := range types {
		out[i] = bt.ID
	}
	return out
}
