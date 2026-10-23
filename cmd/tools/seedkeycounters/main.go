// Siembra de los contadores de clave de bondBound y auditoría de claves repetidas. Informa las claves que ya
// están repetidas en bondBound y el contador que le corresponde a cada tipo: la cantidad de bonos del tipo,
// incluidos los borrados, como numeraba el legado. Con -apply guarda los contadores, sin bajar nunca uno que
// ya esté más alto. Hay que correrlo justo antes de habilitar la compra en el API.
//
//	go run ./cmd/tools/seedkeycounters          # solo informa
//	go run ./cmd/tools/seedkeycounters -apply   # además siembra los contadores
//
// Lee FA_MONGO_URL y FA_MONGO_DBNAME de los mismos archivos de entorno que el API.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/codecraftkit/mongocc"
	api_config "github.com/futurapps/ms-blooming-go/cmd/api/config"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Colecciones que lee y escribe la herramienta.
const (
	bondBoundCollection  = "bondBound"
	keyCounterCollection = "bondKeyCounter"
)

// boundRef es un bono que comparte su clave con otros. Los campos legacy se leen como any para no fallar
// con un documento mal formado.
type boundRef struct {
	ID         any       `bson:"_id"`
	BondTypeID any       `bson:"bondTypeId"`
	StatusID   any       `bson:"statusId"`
	IsRemove   any       `bson:"isRemove"`
	CreatedAt  time.Time `bson:"createdAt"`
}

// duplicateKey es una clave usada por más de un bono, con sus bonos en orden de alta.
type duplicateKey struct {
	Key    string     `bson:"_id"`
	Bounds []boundRef `bson:"bounds"`
}

// typeCount es la cantidad de bonos de un bondTypeId, que puede ser null o no ser string.
type typeCount struct {
	BondTypeID any `bson:"_id"`
	Count      int `bson:"count"`
}

// counterSeed es el contador que le corresponde a un tipo y el que tiene ahora, o nil si no existe.
type counterSeed struct {
	BondTypeID string
	Count      int
	Current    *int
}

func main() {
	apply := flag.Bool("apply", false, "guarda los contadores; sin él solo informa")
	flag.Parse()

	if err := run(*apply); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "seedkeycounters falló: %v\n", err)
		os.Exit(1)
	}
}

// run conecta con Mongo y ejecuta la auditoría y la siembra.
func run(apply bool) error {
	if err := api_config.LoadEnvFiles(); err != nil {
		return err
	}
	url := strings.TrimSpace(os.Getenv("FA_MONGO_URL"))
	dbName := strings.TrimSpace(os.Getenv("FA_MONGO_DBNAME"))
	if url == "" || dbName == "" {
		return errors.New("FA_MONGO_URL y FA_MONGO_DBNAME son requeridas")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := mongocc.Connect(url, dbName, nil)
	if err != nil {
		return err
	}
	defer func() { _ = db.Database().Client().Disconnect(context.Background()) }()

	return seed(ctx, db, apply)
}

// seed informa las claves repetidas y los contadores por tipo y, con apply, guarda los contadores.
func seed(ctx context.Context, db *mongocc.MongoQueries, apply bool) error {
	duplicates, err := findDuplicateKeys(ctx, db)
	if err != nil {
		return err
	}
	reportDuplicates(duplicates)

	seeds, skipped, err := planCounters(ctx, db)
	if err != nil {
		return err
	}
	reportCounters(seeds, skipped)

	if !apply {
		if len(seeds) > 0 {
			fmt.Println("Sin -apply no se modificó nada.")
		}
		return nil
	}

	created, raised, err := applyCounters(ctx, db, seeds)
	if err != nil {
		return err
	}
	fmt.Printf("Se crearon %d contadores y se subieron %d.\n", created, raised)
	return nil
}

// findDuplicateKeys devuelve las claves que usa más de un bono, incluidos los borrados.
func findDuplicateKeys(ctx context.Context, db *mongocc.MongoQueries) ([]duplicateKey, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"key": bson.M{"$type": "string"}}},
		bson.M{"$sort": bson.D{{Key: "createdAt", Value: 1}, {Key: "_id", Value: 1}}},
		bson.M{"$group": bson.M{
			"_id": "$key",
			"bounds": bson.M{"$push": bson.M{
				"_id":        "$_id",
				"bondTypeId": "$bondTypeId",
				"statusId":   "$statusId",
				"isRemove":   "$isRemove",
				"createdAt":  "$createdAt",
			}},
			"count": bson.M{"$sum": 1},
		}},
		bson.M{"$match": bson.M{"count": bson.M{"$gt": 1}}},
		bson.M{"$sort": bson.D{{Key: "_id", Value: 1}}},
	}

	cursor, err := db.Aggregate(ctx, bondBoundCollection, pipeline, options.Aggregate().SetAllowDiskUse(true))
	if err != nil {
		return nil, err
	}
	duplicates := []duplicateKey{}
	if err := cursor.All(ctx, &duplicates); err != nil {
		return nil, err
	}
	return duplicates, nil
}

// planCounters cuenta los bonos de cada tipo, incluidos los borrados, y le suma el contador actual. Devuelve
// también cuántos bonos no tienen un bondTypeId string, que no se numeran.
func planCounters(ctx context.Context, db *mongocc.MongoQueries) ([]counterSeed, int, error) {
	pipeline := bson.A{
		bson.M{"$group": bson.M{"_id": "$bondTypeId", "count": bson.M{"$sum": 1}}},
		bson.M{"$sort": bson.D{{Key: "_id", Value: 1}}},
	}
	cursor, err := db.Aggregate(ctx, bondBoundCollection, pipeline, nil)
	if err != nil {
		return nil, 0, err
	}
	counts := []typeCount{}
	if err := cursor.All(ctx, &counts); err != nil {
		return nil, 0, err
	}

	current, err := currentCounters(ctx, db)
	if err != nil {
		return nil, 0, err
	}

	seeds := []counterSeed{}
	skipped := 0
	for _, c := range counts {
		id, ok := c.BondTypeID.(string)
		if !ok || strings.TrimSpace(id) == "" {
			skipped += c.Count
			continue
		}
		s := counterSeed{BondTypeID: id, Count: c.Count}
		if seq, ok := current[id]; ok {
			s.Current = &seq
		}
		seeds = append(seeds, s)
	}
	return seeds, skipped, nil
}

// currentCounters devuelve el seq de cada contador que ya existe, por bondTypeId.
func currentCounters(ctx context.Context, db *mongocc.MongoQueries) (map[string]int, error) {
	cursor, err := db.Find(ctx, keyCounterCollection, bson.M{}, nil)
	if err != nil {
		return nil, err
	}
	var counters []struct {
		ID  string `bson:"_id"`
		Seq int    `bson:"seq"`
	}
	if err := cursor.All(ctx, &counters); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(counters))
	for _, c := range counters {
		out[c.ID] = c.Seq
	}
	return out, nil
}

// applyCounters guarda el contador de cada tipo sin bajar uno que ya esté más alto, y devuelve cuántos creó
// y cuántos subió.
func applyCounters(ctx context.Context, db *mongocc.MongoQueries, seeds []counterSeed) (int64, int64, error) {
	var created, raised int64
	opts := options.UpdateOne().SetUpsert(true)
	for _, s := range seeds {
		filter := bson.M{"_id": s.BondTypeID}
		update := bson.M{"$max": bson.M{"seq": s.Count}}
		res, err := db.UpdateOne(ctx, keyCounterCollection, filter, update, opts)
		if err != nil {
			return created, raised, fmt.Errorf("contador de %s: %w", s.BondTypeID, err)
		}
		created += res.UpsertedCount
		raised += res.ModifiedCount
	}
	return created, raised, nil
}

// reportDuplicates imprime cada clave repetida con sus bonos.
func reportDuplicates(duplicates []duplicateKey) {
	fmt.Println("Claves repetidas en bondBound:")
	for _, d := range duplicates {
		fmt.Printf("  %s: %d bonos\n", d.Key, len(d.Bounds))
		for _, b := range d.Bounds {
			fmt.Printf("    %v bondTypeId=%v statusId=%v isRemove=%v createdAt=%s\n",
				b.ID, b.BondTypeID, b.StatusID, b.IsRemove, b.CreatedAt.Format(time.RFC3339))
		}
	}
	fmt.Printf("%d claves repetidas.\n", len(duplicates))
}

// reportCounters imprime el contador actual y el que queda para cada tipo.
func reportCounters(seeds []counterSeed, skipped int) {
	fmt.Println("Contadores por tipo de bono:")
	for _, s := range seeds {
		switch {
		case s.Current == nil:
			fmt.Printf("  %s: %d bonos, sin contador → %d\n", s.BondTypeID, s.Count, s.Count)
		case *s.Current < s.Count:
			fmt.Printf("  %s: %d bonos, contador %d → %d\n", s.BondTypeID, s.Count, *s.Current, s.Count)
		default:
			fmt.Printf("  %s: %d bonos, contador %d se mantiene\n", s.BondTypeID, s.Count, *s.Current)
		}
	}
	fmt.Printf("%d tipos.\n", len(seeds))
	if skipped > 0 {
		fmt.Printf("%d bonos sin bondTypeId válido no se numeran.\n", skipped)
	}
}
