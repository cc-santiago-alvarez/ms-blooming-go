package main

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/codecraftkit/mongocc"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// soldAudit son los bonos madurados o canjeados que ya tenían fecha de venta.
type soldAudit struct {
	// Fixable son los '2' con soldDate y sin tookDate, con la venta en el historial: -apply los pasa a '5'.
	Fixable []soldBound
	// Manual son los '2' con soldDate que -apply no toca.
	Manual []docIssue
	// Paid son los '6' con soldDate y tookDate: se cobraron dos veces.
	Paid []soldBound
	// AuditBonds y AuditOwners son lo que cuenta la auditoría de doble cobro del gateway: bonos vivos con las dos
	// fechas, de cualquier estado, de dueños con perfil.
	AuditBonds  int
	AuditOwners int
}

// soldBound es un bono con fecha de venta. UserID solo sirve para contar dueños: nunca se muestra.
type soldBound struct {
	ID        bson.RawValue
	UserID    string
	AgencyID  string
	Amount    float64
	SoldDate  time.Time
	TookDate  *time.Time
	IsRemove  bool
	UpdatedAt bson.RawValue
}

// soldFields es lo que se lee de cada bono vendido.
var soldFields = bson.M{
	"_id": 1, "userId": 1, "agencyId": 1, "amount": 1, "soldDate": 1, "tookDate": 1, "isRemove": 1, "updatedAt": 1,
}

// collectSold busca los '2' con soldDate y los separa en los que -apply pasa a '5' y los que van a revisión
// manual, y busca los '6' con soldDate y tookDate.
func collectSold(ctx context.Context, db *mongocc.MongoQueries, a *audit) error {
	matured, err := findSold(ctx, db, bson.M{
		"statusId": string(bondBound_domain.StatusFinalized),
		"soldDate": bson.M{"$type": "date"},
	})
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(matured))
	for _, b := range matured {
		if id, ok := b.ID.StringValueOK(); ok {
			ids = append(ids, id)
		}
	}
	evidence, err := saleEvidence(ctx, db, ids)
	if err != nil {
		return err
	}
	for _, b := range matured {
		id, isString := b.ID.StringValueOK()
		reason := ""
		switch {
		case b.TookDate != nil:
			reason = "también tiene tookDate"
		case !isString:
			reason = "_id que no es string"
		case !evidence[id]:
			reason = "el historial no registra la venta: puede no haberse cobrado"
		case b.AgencyID == "":
			reason = "sin agencyId para el historial"
		}
		if reason != "" {
			a.Sold.Manual = append(a.Sold.Manual, docIssue{Collection: bondBoundCollection, ID: idString(b.ID), Detail: reason})
			continue
		}
		a.Sold.Fixable = append(a.Sold.Fixable, b)
	}

	a.Sold.Paid, err = findSold(ctx, db, bson.M{
		"statusId": string(bondBound_domain.StatusTook),
		"soldDate": bson.M{"$type": "date"},
		"tookDate": bson.M{"$type": "date"},
	})
	if err != nil {
		return err
	}
	a.Sold.AuditBonds, a.Sold.AuditOwners, err = countDoubleProcess(ctx, db)
	return err
}

// countDoubleProcess cuenta lo que devuelve la auditoría de doble cobro del gateway, con su mismo filtro y su
// mismo cruce con profile: bonos y dueños.
func countDoubleProcess(ctx context.Context, db *mongocc.MongoQueries) (int, int, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"isRemove": false, "soldDate": bson.M{"$exists": true}, "tookDate": bson.M{"$exists": true}}},
		bson.M{"$group": bson.M{"_id": "$userId", "count": bson.M{"$sum": 1}}},
		bson.M{"$lookup": bson.M{
			"from":     profileCollection,
			"let":      bson.M{"userId": "$_id"},
			"pipeline": bson.A{bson.M{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$_id", "$$userId"}}}}, bson.M{"$project": bson.M{"_id": 1}}},
			"as":       "profile",
		}},
		bson.M{"$unwind": "$profile"},
		bson.M{"$group": bson.M{"_id": nil, "bonds": bson.M{"$sum": "$count"}, "owners": bson.M{"$sum": 1}}},
	}
	cursor, err := db.Aggregate(ctx, bondBoundCollection, pipeline, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("auditoría de doble cobro: %w", err)
	}
	var totals []struct {
		Bonds  int `bson:"bonds"`
		Owners int `bson:"owners"`
	}
	if err := cursor.All(ctx, &totals); err != nil {
		return 0, 0, fmt.Errorf("auditoría de doble cobro: %w", err)
	}
	if len(totals) == 0 {
		return 0, 0, nil
	}
	return totals[0].Bonds, totals[0].Owners, nil
}

// findSold lee los bonos del filtro, incluidos los borrados: el canje de Meteor los busca solo por _id.
func findSold(ctx context.Context, db *mongocc.MongoQueries, filter bson.M) ([]soldBound, error) {
	out := []soldBound{}
	opts := byID().SetProjection(soldFields)
	err := scan(ctx, db.GetCollection(bondBoundCollection), filter, opts, func(c *mongo.Cursor) {
		raw := c.Current
		b := soldBound{
			ID:        cloneValue(raw.Lookup("_id")),
			UserID:    stringOf(raw.Lookup("userId")),
			AgencyID:  stringOf(raw.Lookup("agencyId")),
			Amount:    amountOf(raw.Lookup("amount")),
			UpdatedAt: cloneValue(raw.Lookup("updatedAt")),
		}
		if ms, ok := raw.Lookup("soldDate").DateTimeOK(); ok {
			b.SoldDate = time.UnixMilli(ms).UTC()
		}
		if took := raw.Lookup("tookDate"); !isNullish(took) {
			t := time.Time{}
			if ms, ok := took.DateTimeOK(); ok {
				t = time.UnixMilli(ms).UTC()
			}
			b.TookDate = &t
		}
		b.IsRemove, _ = raw.Lookup("isRemove").BooleanOK()
		out = append(out, b)
	})
	return out, err
}

// saleEvidence devuelve los bonos de ids cuyo historial registra la venta: un cambio de estado hacia '5'. La
// venta de Meteor escribe ese registro después de acreditar el monto.
func saleEvidence(ctx context.Context, db *mongocc.MongoQueries, ids []string) (map[string]bool, error) {
	found := map[string]bool{}
	opts := options.Find().SetProjection(bson.M{"relatedId": 1})
	for _, chunk := range chunks(ids, inChunk) {
		filter := bson.M{
			"relatedId": bson.M{"$in": chunk},
			"text":      shared_domain.TextBondBoundStatusChanged,
			"params.1":  string(bondBound_domain.StatusSold),
		}
		err := scan(ctx, db.GetCollection(bondHistoryCollection), filter, opts, func(c *mongo.Cursor) {
			found[stringOf(c.Current.Lookup("relatedId"))] = true
		})
		if err != nil {
			return nil, err
		}
	}
	return found, nil
}

// amountOf lee el monto como lo lee Go; si no se puede leer, cuenta 0.
func amountOf(rv bson.RawValue) float64 {
	var n shared_domain.FlexNumber
	if err := n.UnmarshalBSONValue(byte(rv.Type), rv.Value); err != nil {
		return 0
	}
	return float64(n)
}

// applySold pasa a '5' cada bono madurado que ya estaba vendido, con un cambio de estado de sistema en la misma
// transacción. El update exige que el bono siga en '2', con soldDate y sin tookDate. No mueve saldo: el bono se
// cobró con la venta.
func (ap *applier) applySold(ctx context.Context, bounds []soldBound) (applyResult, error) {
	res := applyResult{Fix: fixSold, Planned: len(bounds)}
	from, to := string(bondBound_domain.StatusFinalized), string(bondBound_domain.StatusSold)
	for _, b := range bounds {
		id := stringOf(b.ID)
		h, err := systemHistory(bondHistory_domain.AddParams{
			RelatedID: id,
			TypeID:    bondHistory_domain.HistoryBondBound,
			Text:      shared_domain.TextBondBoundStatusChanged,
			Params:    []string{from, to},
		}, b.AgencyID)
		if err != nil {
			res.Failed = append(res.Failed, docIssue{Collection: bondBoundCollection, ID: id, Detail: err.Error()})
			continue
		}
		change := backupChange{
			Kind: lineChange, Fix: fixSold, Collection: bondBoundCollection, ID: b.ID,
			Before:    bson.D{{Key: "statusId", Value: from}},
			After:     bson.D{{Key: "statusId", Value: to}},
			HistoryID: h.ID,
		}
		change.withUpdatedAt(ap.lastUpdatedAt(bondBoundCollection, b.ID, b.UpdatedAt), ap.opts.runAt)
		filter := bson.M{"_id": b.ID, "statusId": from, "soldDate": bson.M{"$type": "date"}, "tookDate": nil}
		update := bson.M{"$set": bson.M{"statusId": to, "updatedAt": ap.opts.runAt}}
		if err := ap.applyOne(ctx, &res, change, filter, update, h); err != nil {
			return res, err
		}
	}
	return res, nil
}

// soldSummary son la cantidad de bonos, de dueños y la suma de los montos.
func soldSummary(bounds []soldBound) (bonds, owners int, total float64) {
	seen := map[string]bool{}
	for _, b := range bounds {
		seen[b.UserID] = true
		total += b.Amount
	}
	return len(bounds), len(seen), total
}

// byYear agrupa los bonos por el año de la fecha que devuelve date.
func byYear(bounds []soldBound, date func(soldBound) time.Time) []yearRow {
	rows := map[int]*yearRow{}
	for _, b := range bounds {
		y := date(b).Year()
		if rows[y] == nil {
			rows[y] = &yearRow{Year: y}
		}
		rows[y].Bonds++
		rows[y].Total += b.Amount
	}
	out := make([]yearRow, 0, len(rows))
	for _, y := range slices.Sorted(maps.Keys(rows)) {
		out = append(out, *rows[y])
	}
	return out
}

// yearRow son los bonos y la suma de montos de un año.
type yearRow struct {
	Year  int
	Bonds int
	Total float64
}

// String muestra la fila del año.
func (r yearRow) String() string {
	return fmt.Sprintf("%d: %d bonos, Σ amount %s", r.Year, r.Bonds, formatAmount(r.Total))
}
