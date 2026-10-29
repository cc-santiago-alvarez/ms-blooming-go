package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/codecraftkit/mongocc"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondHistory_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/infrastructure/mongocc"
	shared_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/mongocc"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// applyOptions configuran -apply.
type applyOptions struct {
	only   map[string]bool
	runAt  time.Time
	pause  time.Duration
	backup *backupWriter
}

// applyResult es lo que hizo una normalización.
type applyResult struct {
	Fix     string
	Skipped bool
	Planned int
	Applied int
	// Changed son los documentos que cambiaron entre la auditoría y la escritura: quedan para la próxima corrida.
	Changed []string
	Failed  []docIssue
}

// applier escribe las normalizaciones. touched son los documentos que ya escribió la corrida.
type applier struct {
	db      *mongocc.MongoQueries
	tx      *shared_mongocc_repository.TransactionRunner
	history *bondHistory_mongocc_repository.BondHistoryMongoccRepository
	opts    applyOptions
	touched map[string]bool
}

// applyAll aplica las normalizaciones elegidas en orden: numéricos, isPrivateBond y bonos vendidos. Se detiene
// solo si no puede escribir el respaldo o se cancela el contexto; lo ya aplicado queda.
func applyAll(ctx context.Context, db *mongocc.MongoQueries, a *audit, opts applyOptions) ([]applyResult, error) {
	ap := &applier{
		db:      db,
		tx:      &shared_mongocc_repository.TransactionRunner{DB: db},
		history: &bondHistory_mongocc_repository.BondHistoryMongoccRepository{DB: db},
		opts:    opts,
		touched: map[string]bool{},
	}
	steps := []struct {
		fix string
		run func() (applyResult, error)
	}{
		{fixNumeric, func() (applyResult, error) { return ap.applyNumeric(ctx, a.Numeric.Plans) }},
		{fixPrivate, func() (applyResult, error) { return ap.applyPrivate(ctx, a.Private.Auto) }},
		{fixSold, func() (applyResult, error) { return ap.applySold(ctx, a.Sold.Fixable) }},
	}
	results := []applyResult{}
	for _, step := range steps {
		if !opts.only[step.fix] {
			results = append(results, applyResult{Fix: step.fix, Skipped: true})
			continue
		}
		res, err := step.run()
		results = append(results, res)
		if err != nil {
			return results, err
		}
	}
	return results, nil
}

// applyOne guarda el respaldo del cambio y aplica el update con su guarda. Con historial, el update y el
// historial van en una transacción, y el historial solo se registra si la guarda encontró el documento. Solo
// devuelve error si no se puede seguir; el resultado del documento queda en res.
func (ap *applier) applyOne(ctx context.Context, res *applyResult, change backupChange, filter, update bson.M, h *bondHistory_domain.BondHistory) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ap.opts.backup.Write(change); err != nil {
		return fmt.Errorf("no se pudo escribir el respaldo: %w", err)
	}

	matched := false
	write := func(c context.Context) error {
		matched = false
		r, err := ap.db.UpdateOne(c, change.Collection, filter, update, nil)
		if err != nil {
			return err
		}
		if r.MatchedCount == 0 {
			return nil
		}
		matched = true
		if h == nil {
			return nil
		}
		return ap.history.InsertOne(c, h)
	}
	var err error
	if h == nil {
		err = write(ctx)
	} else {
		err = ap.tx.RunInTransaction(ctx, write)
	}

	id := idString(change.ID)
	switch {
	case err != nil:
		res.Failed = append(res.Failed, docIssue{Collection: change.Collection, ID: id, Detail: redactQuoted(err.Error())})
	case matched:
		res.Applied++
		ap.touched[docKey(change.Collection, change.ID)] = true
	default:
		res.Changed = append(res.Changed, id)
	}
	return ap.wait(ctx)
}

// lastUpdatedAt es el updatedAt del documento antes del cambio: el que leyó la auditoría o, si la corrida ya lo
// escribió, runAt. Así el respaldo de dos cambios del mismo documento se revierte en cadena.
func (ap *applier) lastUpdatedAt(collection string, id, audited bson.RawValue) bson.RawValue {
	if !ap.touched[docKey(collection, id)] {
		return audited
	}
	t, data, err := bson.MarshalValue(ap.opts.runAt)
	if err != nil {
		return audited
	}
	return bson.RawValue{Type: t, Value: data}
}

// docKey identifica un documento de una colección.
func docKey(collection string, id bson.RawValue) string {
	return collection + "/" + idString(id)
}

// wait hace la pausa entre escrituras, si se pidió.
func (ap *applier) wait(ctx context.Context) error {
	if ap.opts.pause <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(ap.opts.pause)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// systemHistory arma un registro del historial a nombre del sistema en la agencia, con las mismas reglas que
// el historial del API. Se arma antes de escribir, para que su _id quede en el respaldo.
func systemHistory(p bondHistory_domain.AddParams, agencyID string) (*bondHistory_domain.BondHistory, error) {
	p.Normalize()
	if err := p.Validation(); err != nil {
		return nil, err
	}
	if agencyID == "" {
		return nil, errors.New("sin agencia para el historial de sistema")
	}
	return bondHistory_domain.NewBondHistory(p, "", agencyID), nil
}

// exactMatch es la condición de que el campo tenga ese valor y ese tipo BSON.
func exactMatch(rv bson.RawValue) bson.M {
	if rv.Type == bson.TypeNull {
		return bson.M{"$type": "null"}
	}
	return bson.M{"$eq": rv, "$type": typeAlias(rv.Type)}
}
