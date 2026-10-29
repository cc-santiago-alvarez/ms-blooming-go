package main

import (
	"context"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/codecraftkit/mongocc"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// maturityAudit es lo que madura la primera corrida de Go a la fecha de referencia.
type maturityAudit struct {
	Bonds  int
	Owners int
	// Dropped son los que ms-feed no madura: falta su profile, su bondType o su bondObject.
	Dropped       []docIssue
	DroppedOwners int
	// Undecodable son los que Go no decodifica: cada uno corta la corrida en su lote, y ni ese lote ni los
	// siguientes maduran.
	Undecodable []string
	ids         map[string]bool
}

// boundRefs son las referencias de un bono que ms-feed cruza antes de madurarlo.
type boundRefs struct {
	ID, UserID, BondTypeID, BondObjectID string
}

// cutoffFor es el corte de la maduración de Go: el inicio del día siguiente en UTC.
func cutoffFor(at time.Time) time.Time {
	return shared_domain.StartOfDay(at, 0).AddDate(0, 0, 1)
}

// collectMaturity cuenta los bonos que madura la primera corrida de Go, con el filtro de su maduración, y los
// que ms-feed nunca madura porque falta su profile, su bondType o su bondObject (no mira isRemove).
func collectMaturity(ctx context.Context, db *mongocc.MongoQueries, a *audit) error {
	filter := bson.M{
		"isRemove":     false,
		"statusId":     string(bondBound_domain.StatusInitiated),
		"maturityDate": bson.M{"$lt": a.Cutoff},
		"soldDate":     nil,
		"tookDate":     nil,
	}
	opts := byID().SetProjection(bson.M{"_id": 1, "userId": 1, "bondTypeId": 1, "bondObjectId": 1})
	refs := []boundRefs{}
	err := scan(ctx, db.GetCollection(bondBoundCollection), filter, opts, func(c *mongo.Cursor) {
		raw := c.Current
		refs = append(refs, boundRefs{
			ID:           idString(raw.Lookup("_id")),
			UserID:       stringOf(raw.Lookup("userId")),
			BondTypeID:   stringOf(raw.Lookup("bondTypeId")),
			BondObjectID: stringOf(raw.Lookup("bondObjectId")),
		})
	})
	if err != nil {
		return err
	}

	profiles, err := existingIDs(ctx, db, profileCollection, distinct(refs, func(r boundRefs) string { return r.UserID }))
	if err != nil {
		return err
	}
	types, err := existingIDs(ctx, db, bondTypeCollection, distinct(refs, func(r boundRefs) string { return r.BondTypeID }))
	if err != nil {
		return err
	}
	objects, err := existingIDs(ctx, db, bondObjectCollection, distinct(refs, func(r boundRefs) string { return r.BondObjectID }))
	if err != nil {
		return err
	}

	m := &a.Maturity
	m.ids = map[string]bool{}
	owners, droppedOwners := map[string]bool{}, map[string]bool{}
	for _, r := range refs {
		m.Bonds++
		m.ids[r.ID] = true
		owners[r.UserID] = true
		if missing := feedDrops(r, profiles, types, objects); len(missing) > 0 {
			m.Dropped = append(m.Dropped, docIssue{Collection: bondBoundCollection, ID: r.ID, Detail: strings.Join(missing, ", ")})
			droppedOwners[r.UserID] = true
		}
	}
	m.Owners, m.DroppedOwners = len(owners), len(droppedOwners)
	return nil
}

// feedDrops devuelve qué le falta al bono para pasar los cruces de ms-feed, o nada si los pasa.
func feedDrops(r boundRefs, profiles, types, objects map[string]bool) []string {
	missing := []string{}
	if !profiles[r.UserID] {
		missing = append(missing, "sin profile")
	}
	if !types[r.BondTypeID] {
		missing = append(missing, "sin bondType")
	}
	if !objects[r.BondObjectID] {
		missing = append(missing, "sin bondObject")
	}
	return missing
}

// distinct devuelve los valores no vacíos de field, sin repetir y ordenados.
func distinct(refs []boundRefs, field func(boundRefs) string) []string {
	seen := map[string]bool{}
	for _, r := range refs {
		if v := field(r); v != "" {
			seen[v] = true
		}
	}
	return slices.Sorted(maps.Keys(seen))
}

// crossUndecodable marca los bonos por madurar que Go no decodifica.
func (m *maturityAudit) crossUndecodable(blockers []blocker) {
	for _, b := range blockers {
		if b.Kind == blockUndecodable && b.Collection == bondBoundCollection && m.ids[b.ID] {
			m.Undecodable = append(m.Undecodable, b.ID)
		}
	}
}
