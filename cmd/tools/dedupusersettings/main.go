// Auditoría de settings duplicados en bondUserSetting: informa los usuarios con más de un setting vivo y,
// con -apply, borra de forma lógica todos menos el último modificado. Hay que correrlo antes de que el API
// cree el índice único de userId.
//
//	go run ./cmd/tools/dedupusersettings          # solo informa
//	go run ./cmd/tools/dedupusersettings -apply   # además borra los duplicados
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
	bondUserSetting_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/infrastructure/mongocc"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const collectionName = "bondUserSetting"

// settingRef es un setting vivo de un usuario con duplicados.
type settingRef struct {
	ID                string    `bson:"_id"`
	AgencyID          string    `bson:"agencyId"`
	SelectedAccountID *string   `bson:"selectedAccountId"`
	UpdatedAt         time.Time `bson:"updatedAt"`
}

// duplicateGroup son los settings vivos de un usuario, del último modificado al primero. userId puede ser
// null o un ObjectId legacy.
type duplicateGroup struct {
	UserID   any          `bson:"_id"`
	Settings []settingRef `bson:"settings"`
}

func main() {
	apply := flag.Bool("apply", false, "borra de forma lógica los duplicados; sin él solo informa")
	flag.Parse()

	if err := run(*apply); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "dedupusersettings falló: %v\n", err)
		os.Exit(1)
	}
}

// run conecta con Mongo, informa los duplicados y, con apply, los borra.
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

	groups, err := findDuplicates(ctx, db)
	if err != nil {
		return err
	}

	toRemove := report(groups)
	if len(toRemove) == 0 || !apply {
		if len(toRemove) > 0 {
			fmt.Println("Sin -apply no se modificó nada.")
		}
		return nil
	}

	filter := bson.M{"_id": bson.M{"$in": toRemove}, "isRemove": false}
	update := bson.M{"$set": bson.M{"isRemove": true, "updatedAt": shared_domain.Now()}}
	res, err := db.UpdateMany(ctx, collectionName, filter, update, nil)
	if err != nil {
		return err
	}
	fmt.Printf("Se borraron de forma lógica %d settings.\n", res.ModifiedCount)
	return nil
}

// findDuplicates agrupa por userId los settings vivos y devuelve los grupos con más de uno. null y la
// falta de userId quedan en el mismo grupo, igual que en el índice único.
func findDuplicates(ctx context.Context, db *mongocc.MongoQueries) ([]duplicateGroup, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"isRemove": false}},
		bson.M{"$sort": bondUserSetting_mongocc_repository.LatestFirst()},
		bson.M{"$group": bson.M{
			"_id": "$userId",
			"settings": bson.M{"$push": bson.M{
				"_id":               "$_id",
				"agencyId":          "$agencyId",
				"selectedAccountId": "$selectedAccountId",
				"updatedAt":         "$updatedAt",
			}},
			"count": bson.M{"$sum": 1},
		}},
		bson.M{"$match": bson.M{"count": bson.M{"$gt": 1}}},
		bson.M{"$sort": bson.D{{Key: "_id", Value: 1}}},
	}

	cursor, err := db.Aggregate(ctx, collectionName, pipeline, nil)
	if err != nil {
		return nil, err
	}
	groups := []duplicateGroup{}
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

// report imprime qué setting conserva cada usuario y cuáles se borran, y devuelve los _id a borrar.
func report(groups []duplicateGroup) []string {
	toRemove := []string{}
	for _, g := range groups {
		userID := g.UserID
		if userID == nil {
			userID = "null"
		}
		fmt.Printf("userId %v: %d settings vivos\n", userID, len(g.Settings))
		for i, s := range g.Settings {
			action := "borra   "
			if i == 0 {
				action = "conserva"
			} else {
				toRemove = append(toRemove, s.ID)
			}
			fmt.Printf("  %s %s agencyId=%s selectedAccountId=%s updatedAt=%s\n",
				action, s.ID, s.AgencyID, deref(s.SelectedAccountID), s.UpdatedAt.Format(time.RFC3339))
		}
	}
	fmt.Printf("%d usuarios con duplicados, %d settings a borrar.\n", len(groups), len(toRemove))
	return toRemove
}

// deref devuelve el valor de s, o "null".
func deref(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}
