package main

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/codecraftkit/mongocc"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Colecciones que lee la auditoría. Solo escribe en las tres primeras, y solo con -apply.
const (
	bondObjectCollection      = "bondObject"
	bondBoundCollection       = "bondBound"
	bondHistoryCollection     = "bondHistory"
	bondTypeCollection        = "bondType"
	bondUserSettingCollection = "bondUserSetting"
	profileCollection         = "profile"
	balanceAccountCollection  = "balanceAccount"
)

// inChunk es la cantidad de ids por consulta $in.
const inChunk = 1000

// audit es el resultado de una pasada de solo lectura.
type audit struct {
	At       time.Time
	Cutoff   time.Time
	Keys     []duplicateKey
	Settings settingsAudit
	Numeric  numericAudit
	Private  privateAudit
	Sold     soldAudit
	Maturity maturityAudit
	Blockers []blocker
	// badProfiles son los perfiles que ya se informaron, para no contarlos dos veces.
	badProfiles map[string]bool
}

// docIssue es un documento con un hallazgo, sin datos personales.
type docIssue struct {
	Collection string
	ID         string
	Detail     string
}

// pending indica si queda algo por corregir: lo que normaliza -apply, los duplicados de settings, lo que va a
// revisión manual y los bloqueantes. Las claves repetidas, los bonos ya cobrados dos veces y la maduración
// solo se informan.
func (a *audit) pending() bool {
	return len(a.Settings.Groups) > 0 ||
		len(a.Numeric.Plans) > 0 || len(a.Numeric.NonCanonical) > 0 ||
		len(a.Private.Auto) > 0 || len(a.Private.Manual) > 0 ||
		len(a.Sold.Fixable) > 0 || len(a.Sold.Manual) > 0 ||
		len(a.Blockers) > 0
}

// collect corre todas las revisiones. No escribe nada.
func collect(ctx context.Context, db *mongocc.MongoQueries, at time.Time) (*audit, error) {
	a := &audit{At: at, Cutoff: cutoffFor(at), badProfiles: map[string]bool{}}
	steps := []func(context.Context, *mongocc.MongoQueries, *audit) error{
		collectDuplicateKeys,
		collectSettings,
		collectSold,
		collectMaturity,
		scanBondTypes,
		scanBondObjects,
		scanBondBounds,
		scanBondHistory,
		scanBondUserSettings,
		scanProfiles,
		scanOwnerProfiles,
		scanBalanceAccounts,
		collectIsRemoveOutsideBond,
	}
	for _, step := range steps {
		if err := step(ctx, db, a); err != nil {
			return nil, err
		}
	}
	a.Maturity.crossUndecodable(a.Blockers)
	return a, nil
}

// scan recorre los documentos uno por uno, para que uno que no decodifica no corte la pasada. visit recibe el
// cursor posicionado en cada documento: Current solo vale hasta el próximo, y Decode usa las opciones BSON de
// la colección.
func scan(ctx context.Context, coll *mongo.Collection, filter any, opts *options.FindOptionsBuilder, visit func(*mongo.Cursor)) error {
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return fmt.Errorf("%s: %w", coll.Name(), err)
	}
	defer func() { _ = cursor.Close(context.Background()) }()
	for cursor.Next(ctx) {
		visit(cursor)
	}
	if err := cursor.Err(); err != nil {
		return fmt.Errorf("%s: %w", coll.Name(), err)
	}
	return nil
}

// byID ordena por _id, para que la salida sea estable entre corridas.
func byID() *options.FindOptionsBuilder {
	return options.Find().SetSort(bson.D{{Key: "_id", Value: 1}})
}

// chunks parte ids en grupos de a lo sumo size.
func chunks(ids []string, size int) [][]string {
	out := [][]string{}
	for len(ids) > size {
		out = append(out, ids[:size])
		ids = ids[size:]
	}
	if len(ids) > 0 {
		out = append(out, ids)
	}
	return out
}

// existingIDs devuelve cuáles de ids existen en la colección, borrados incluidos.
func existingIDs(ctx context.Context, db *mongocc.MongoQueries, collection string, ids []string) (map[string]bool, error) {
	found := map[string]bool{}
	opts := options.Find().SetProjection(bson.M{"_id": 1})
	for _, chunk := range chunks(ids, inChunk) {
		err := scan(ctx, db.GetCollection(collection), bson.M{"_id": bson.M{"$in": chunk}}, opts, func(c *mongo.Cursor) {
			if id, ok := c.Current.Lookup("_id").StringValueOK(); ok {
				found[id] = true
			}
		})
		if err != nil {
			return nil, err
		}
	}
	return found, nil
}

// idString muestra un _id: el string tal cual o, si no es string, su valor BSON.
func idString(rv bson.RawValue) string {
	if s, ok := rv.StringValueOK(); ok {
		return s
	}
	if rv.Type == 0 {
		return "(sin _id)"
	}
	return rv.String()
}

// stringOf devuelve el valor si es un string; si no, "".
func stringOf(rv bson.RawValue) string {
	s, _ := rv.StringValueOK()
	return s
}

// cloneValue copia el valor, que de otro modo apunta al documento actual del cursor.
func cloneValue(rv bson.RawValue) bson.RawValue {
	return bson.RawValue{Type: rv.Type, Value: append([]byte(nil), rv.Value...)}
}

// isNullish indica si el valor falta, es null o es undefined.
func isNullish(rv bson.RawValue) bool {
	return rv.Type == 0 || rv.Type == bson.TypeNull || rv.Type == bson.TypeUndefined
}

// typeAlias es el nombre del tipo BSON como lo usa $type, o "ausente".
func typeAlias(t bson.Type) string {
	switch t {
	case 0:
		return "ausente"
	case bson.TypeDouble:
		return "double"
	case bson.TypeString:
		return "string"
	case bson.TypeEmbeddedDocument:
		return "object"
	case bson.TypeArray:
		return "array"
	case bson.TypeBinary:
		return "binData"
	case bson.TypeUndefined:
		return "undefined"
	case bson.TypeObjectID:
		return "objectId"
	case bson.TypeBoolean:
		return "bool"
	case bson.TypeDateTime:
		return "date"
	case bson.TypeNull:
		return "null"
	case bson.TypeInt32:
		return "int"
	case bson.TypeTimestamp:
		return "timestamp"
	case bson.TypeInt64:
		return "long"
	case bson.TypeDecimal128:
		return "decimal"
	}
	return t.String()
}

// quoted encuentra los textos entre comillas de un error.
var quoted = regexp.MustCompile(`"[^"]*"`)

// redactQuoted tapa los valores que algunos decoders citan en sus errores, para no mostrar datos guardados.
func redactQuoted(msg string) string {
	const maxLen = 300
	msg = quoted.ReplaceAllString(msg, `"…"`)
	if len(msg) > maxLen {
		msg = msg[:maxLen] + "…"
	}
	return msg
}
