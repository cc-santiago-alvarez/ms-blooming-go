package main

import (
	"context"
	"fmt"

	"github.com/codecraftkit/mongocc"
	bondUserSetting_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/infrastructure/mongocc"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// duplicateKey es una clave que usa más de un bono, con sus bonos en orden de alta.
type duplicateKey struct {
	Key    string
	Bounds []docIssue
}

// settingsAudit son los settings vivos que rompen el índice único de userId o que Go no encuentra.
type settingsAudit struct {
	// Groups son los settings vivos de cada usuario que tiene más de uno, del que conserva el dedup al resto.
	Groups [][]string
	// NullUser son los settings vivos sin userId.
	NullUser []string
	// NonString son los settings vivos con un userId que no es string.
	NonString []string
}

// collectDuplicateKeys busca las claves que usa más de un bono, incluidos los borrados.
func collectDuplicateKeys(ctx context.Context, db *mongocc.MongoQueries, a *audit) error {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"key": bson.M{"$type": "string"}}},
		bson.M{"$sort": bson.D{{Key: "createdAt", Value: 1}, {Key: "_id", Value: 1}}},
		bson.M{"$group": bson.M{
			"_id": "$key",
			"bounds": bson.M{"$push": bson.M{
				"_id":      "$_id",
				"statusId": "$statusId",
				"isRemove": "$isRemove",
			}},
			"count": bson.M{"$sum": 1},
		}},
		bson.M{"$match": bson.M{"count": bson.M{"$gt": 1}}},
		bson.M{"$sort": bson.D{{Key: "_id", Value: 1}}},
	}
	cursor, err := db.Aggregate(ctx, bondBoundCollection, pipeline, options.Aggregate().SetAllowDiskUse(true))
	if err != nil {
		return fmt.Errorf("claves repetidas: %w", err)
	}
	var groups []struct {
		Key    string `bson:"_id"`
		Bounds []struct {
			ID       bson.RawValue `bson:"_id"`
			StatusID bson.RawValue `bson:"statusId"`
			IsRemove bson.RawValue `bson:"isRemove"`
		} `bson:"bounds"`
	}
	if err := cursor.All(ctx, &groups); err != nil {
		return fmt.Errorf("claves repetidas: %w", err)
	}
	for _, g := range groups {
		d := duplicateKey{Key: g.Key}
		for _, b := range g.Bounds {
			d.Bounds = append(d.Bounds, docIssue{
				Collection: bondBoundCollection,
				ID:         idString(b.ID),
				Detail:     fmt.Sprintf("statusId %s, isRemove %s", valueText(b.StatusID), valueText(b.IsRemove)),
			})
		}
		a.Keys = append(a.Keys, d)
	}
	return nil
}

// collectSettings busca los usuarios con más de un setting vivo, agrupados como los agrupa el índice único
// (null y la falta de userId van juntos), y los settings vivos sin userId o con un userId que no es string.
func collectSettings(ctx context.Context, db *mongocc.MongoQueries, a *audit) error {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"isRemove": false}},
		bson.M{"$sort": bondUserSetting_mongocc_repository.LatestFirst()},
		bson.M{"$group": bson.M{
			"_id":   "$userId",
			"ids":   bson.M{"$push": "$_id"},
			"count": bson.M{"$sum": 1},
		}},
		bson.M{"$match": bson.M{"count": bson.M{"$gt": 1}}},
		bson.M{"$sort": bson.D{{Key: "_id", Value: 1}}},
	}
	cursor, err := db.Aggregate(ctx, bondUserSettingCollection, pipeline, nil)
	if err != nil {
		return fmt.Errorf("settings duplicados: %w", err)
	}
	var groups []struct {
		IDs []bson.RawValue `bson:"ids"`
	}
	if err := cursor.All(ctx, &groups); err != nil {
		return fmt.Errorf("settings duplicados: %w", err)
	}
	for _, g := range groups {
		ids := make([]string, len(g.IDs))
		for i, id := range g.IDs {
			ids[i] = idString(id)
		}
		a.Settings.Groups = append(a.Settings.Groups, ids)
	}

	coll := db.GetCollection(bondUserSettingCollection)
	opts := byID().SetProjection(bson.M{"_id": 1})
	err = scan(ctx, coll, bson.M{"isRemove": false, "userId": nil}, opts, func(c *mongo.Cursor) {
		a.Settings.NullUser = append(a.Settings.NullUser, idString(c.Current.Lookup("_id")))
	})
	if err != nil {
		return err
	}
	nonString := bson.M{"isRemove": false, "userId": bson.M{"$ne": nil, "$not": bson.M{"$type": "string"}}}
	return scan(ctx, coll, nonString, opts, func(c *mongo.Cursor) {
		a.Settings.NonString = append(a.Settings.NonString, idString(c.Current.Lookup("_id")))
	})
}

// valueText muestra un valor corto de un campo que no es personal: el string, el booleano o el tipo.
func valueText(rv bson.RawValue) string {
	if s, ok := rv.StringValueOK(); ok {
		return fmt.Sprintf("%q", s)
	}
	if b, ok := rv.BooleanOK(); ok {
		return fmt.Sprint(b)
	}
	return typeAlias(rv.Type)
}
