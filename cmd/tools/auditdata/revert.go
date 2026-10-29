package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/codecraftkit/mongocc"
	shared_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/mongocc"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// revertResult es lo que hizo, o haría, la reversión.
type revertResult struct {
	Reverted int
	// Skipped son los documentos que ya no tienen los valores del cambio: no se aplicó o cambiaron después.
	Skipped []string
	Failed  []docIssue
}

// revertBackup deshace los cambios de un respaldo del mismo entorno y la misma base, del último al primero.
// Cada documento vuelve a sus valores de antes solo si sigue con los que dejó el cambio, y su historial de
// sistema se borra en la misma transacción. Sin -apply solo cuenta lo que revertiría.
func revertBackup(ctx context.Context, db *mongocc.MongoQueries, cfg config, env string) (int, error) {
	header, changes, torn, err := readBackup(cfg.revert)
	if err != nil {
		return exitError, err
	}
	if name := db.Database().Name(); header.DB != name || header.Env != env {
		return exitError, fmt.Errorf("el respaldo es de %s/%s, no de %s/%s", header.Env, header.DB, env, name)
	}
	fmt.Printf("Respaldo de la corrida del %s: %s.\n", header.Run.UTC().Format(time.RFC3339), plural(len(changes), "cambio", "cambios"))
	if torn {
		fmt.Println("La última línea del respaldo quedó a medias: su cambio no se escribió en la base y se ignora.")
	}

	res, err := revertChanges(ctx, db, changes, cfg.apply, cfg.pause)
	printRevert(os.Stdout, res, cfg.apply, cfg.verbose)
	if err != nil {
		return exitError, err
	}
	if len(res.Skipped) > 0 || len(res.Failed) > 0 {
		return exitPending, nil
	}
	return exitOK, nil
}

// revertChanges revierte los cambios en orden inverso; sin apply solo cuenta cuáles se pueden revertir.
func revertChanges(ctx context.Context, db *mongocc.MongoQueries, changes []backupLine, apply bool, pause time.Duration) (revertResult, error) {
	res := revertResult{}
	tx := &shared_mongocc_repository.TransactionRunner{DB: db}
	ap := &applier{opts: applyOptions{pause: pause}}
	for i := len(changes) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		ch := changes[i]
		id := idString(ch.ID)
		filter, err := revertFilter(ch)
		if err != nil {
			res.Failed = append(res.Failed, docIssue{Collection: ch.Collection, ID: id, Detail: err.Error()})
			continue
		}

		if !apply {
			n, err := db.CountDocuments(ctx, ch.Collection, filter)
			switch {
			case err != nil:
				return res, err
			case n == 1:
				res.Reverted++
			default:
				res.Skipped = append(res.Skipped, id)
			}
			continue
		}

		matched := false
		err = tx.RunInTransaction(ctx, func(txCtx context.Context) error {
			matched = false
			r, err := db.UpdateOne(txCtx, ch.Collection, filter, revertUpdate(ch), nil)
			if err != nil || r.MatchedCount == 0 {
				return err
			}
			matched = true
			if ch.HistoryID == "" {
				return nil
			}
			_, err = db.DeleteOne(txCtx, bondHistoryCollection, bson.M{"_id": ch.HistoryID, "userId": "", "relatedId": id}, nil)
			return err
		})
		switch {
		case err != nil:
			res.Failed = append(res.Failed, docIssue{Collection: ch.Collection, ID: id, Detail: redactQuoted(err.Error())})
		case matched:
			res.Reverted++
		default:
			res.Skipped = append(res.Skipped, id)
		}
		if err := ap.wait(ctx); err != nil {
			return res, err
		}
	}
	return res, nil
}

// revertFilter es la condición de que el documento siga con los valores que dejó el cambio, con su tipo.
func revertFilter(ch backupLine) (bson.M, error) {
	filter := bson.M{"_id": ch.ID}
	elems, err := ch.After.Elements()
	if err != nil {
		return nil, fmt.Errorf("valores del cambio ilegibles: %w", err)
	}
	for _, e := range elems {
		filter[e.Key()] = exactMatch(e.Value())
	}
	return filter, nil
}

// revertUpdate vuelve los campos a sus valores de antes y quita los que no estaban.
func revertUpdate(ch backupLine) bson.M {
	set := bson.M{}
	if elems, err := ch.Before.Elements(); err == nil {
		for _, e := range elems {
			set[e.Key()] = e.Value()
		}
	}
	update := bson.M{}
	if len(set) > 0 {
		update["$set"] = set
	}
	if len(ch.Absent) > 0 {
		unset := bson.M{}
		for _, field := range ch.Absent {
			unset[field] = ""
		}
		update["$unset"] = unset
	}
	return update
}
