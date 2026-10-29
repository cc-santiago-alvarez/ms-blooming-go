package main

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/codecraftkit/mongocc"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Tipos de bloqueante: datos que Go no puede leer u operar.
const (
	blockUndecodable   = "no decodifica"
	blockIsRemove      = "isRemove no es booleano"
	blockDateType      = "fecha que no es Date"
	blockStatus        = "statusId desconocido"
	blockSoldInitiated = "'1' con soldDate o tookDate"
	blockNoAgency      = "'1' sin agencyId"
	blockInvalidTerms  = "plazo menor o igual al punto de inflexión"
	blockBalance       = "saldo que no es número"
)

// blockerKinds es el orden en que se informan los bloqueantes.
var blockerKinds = []string{
	blockUndecodable, blockIsRemove, blockDateType, blockStatus, blockSoldInitiated, blockNoAgency, blockInvalidTerms,
	blockBalance,
}

// blocker es un documento que Go no puede leer u operar. En perfiles y cuentas no se guarda el _id, que es o
// apunta a un usuario: solo la causa.
type blocker struct {
	Kind string
	docIssue
}

// personalCollections son las colecciones cuyos _id no se muestran.
var personalCollections = map[string]bool{profileCollection: true, balanceAccountCollection: true}

// decodedKey encuentra el campo que no se pudo decodificar en un error del driver.
var decodedKey = regexp.MustCompile(`error decoding key ([^:\s]+)`)

// dateRule es una fecha que Go lee como time.Time; optional admite además null o la falta del campo.
type dateRule struct {
	Field    string
	Optional bool
}

// dateRules son las fechas de cada colección.
var dateRules = map[string][]dateRule{
	bondBoundCollection: {
		{Field: "initialDate"}, {Field: "maturityDate"}, {Field: "createdAt"}, {Field: "updatedAt"},
		{Field: "soldDate", Optional: true}, {Field: "tookDate", Optional: true},
	},
	bondObjectCollection:      {{Field: "createdAt"}, {Field: "updatedAt"}},
	bondTypeCollection:        {{Field: "createdAt"}, {Field: "updatedAt"}},
	bondHistoryCollection:     {{Field: "createdAt"}, {Field: "updatedAt"}},
	bondUserSettingCollection: {{Field: "createdAt"}, {Field: "updatedAt"}},
}

// publicProfileProjection es la proyección con la que Go lee los perfiles por id.
var publicProfileProjection = bson.M{
	"_id": 1, "id": 1, "first_name": 1, "last_name": 1, "email": 1, "avatar_url": 1, "agencyId": 1,
}

// ownerProfileProjection junta lo que leen del perfil del dueño los cruces de los listados, los reportes y los
// inversionistas.
var ownerProfileProjection = bson.M{
	"_id": 1, "id": 1, "first_name": 1, "last_name": 1, "email": 1, "avatar": 1, "address": 1, "avatar_url": 1,
}

// balanceAccountProjection es la proyección con la que Go lee las cuentas, más global, que el saldo también suma.
var balanceAccountProjection = bson.M{
	"_id": 1, "userId": 1, "agencyId": 1, "statusId": 1, "available": 1, "number": 1, "global": 1,
}

// investorStatuses son los estados de los bonos cuyo dueño aparece como inversionista del producto.
var investorStatuses = map[string]bool{"1": true, "2": true, "3": true, "6": true}

// block registra un bloqueante; en perfiles y cuentas, sin el _id.
func (a *audit) block(kind, collection, id, detail string) {
	if personalCollections[collection] {
		id = ""
	}
	a.Blockers = append(a.Blockers, blocker{Kind: kind, docIssue: docIssue{Collection: collection, ID: id, Detail: detail}})
}

// decode decodifica el documento actual con el tipo de Go y registra el error, si lo hay. En perfiles y cuentas
// registra el campo y su tipo, sin el error.
func (a *audit) decode(collection string, c *mongo.Cursor, v any) error {
	err := c.Decode(v)
	if err == nil {
		return nil
	}
	raw := c.Current
	detail := fmt.Sprintf("isRemove %s: %s", valueText(raw.Lookup("isRemove")), redactQuoted(err.Error()))
	if personalCollections[collection] {
		detail = decodeCause(err, raw)
	}
	a.block(blockUndecodable, collection, idString(raw.Lookup("_id")), detail)
	return err
}

// decodeCause es el campo que no se pudo decodificar y el tipo con el que está guardado.
func decodeCause(err error, raw bson.Raw) string {
	m := decodedKey.FindStringSubmatch(err.Error())
	if m == nil {
		return "documento ilegible"
	}
	return fmt.Sprintf("%s: %s", m[1], typeAlias(raw.Lookup(strings.Split(m[1], ".")...).Type))
}

// checkTypes revisa que isRemove sea booleano y que las fechas de la colección sean Date.
func (a *audit) checkTypes(collection string, raw bson.Raw) {
	id := idString(raw.Lookup("_id"))
	if t := raw.Lookup("isRemove").Type; t != bson.TypeBoolean {
		a.block(blockIsRemove, collection, id, typeAlias(t))
	}
	for _, rule := range dateRules[collection] {
		rv := raw.Lookup(rule.Field)
		if rv.Type == bson.TypeDateTime || (rule.Optional && isNullish(rv)) {
			continue
		}
		a.block(blockDateType, collection, id, fmt.Sprintf("%s: %s", rule.Field, typeAlias(rv.Type)))
	}
}

// checkTerms registra un producto o un bono cuyo plazo no supera al punto de inflexión: Go no lo puede cotizar
// para la venta ni el canje.
func (a *audit) checkTerms(collection, id string, days, inflection int, detail string) {
	if days-inflection <= 0 {
		a.block(blockInvalidTerms, collection, id, fmt.Sprintf("plazo %d, inflexión %d, %s", days, inflection, detail))
	}
}

// checkBoundState revisa el estado de un bono: que sea conocido y que un '1' vivo se pueda madurar.
func (a *audit) checkBoundState(raw bson.Raw) {
	id := idString(raw.Lookup("_id"))
	status, ok := raw.Lookup("statusId").StringValueOK()
	if !ok || !bondBound_domain.BondBoundStatus(status).IsValid() {
		a.block(blockStatus, bondBoundCollection, id, valueText(raw.Lookup("statusId")))
		return
	}
	removed, _ := raw.Lookup("isRemove").BooleanOK()
	if status != string(bondBound_domain.StatusInitiated) || removed {
		return
	}
	if !isNullish(raw.Lookup("soldDate")) || !isNullish(raw.Lookup("tookDate")) {
		a.block(blockSoldInitiated, bondBoundCollection, id, fmt.Sprintf("soldDate %s, tookDate %s",
			typeAlias(raw.Lookup("soldDate").Type), typeAlias(raw.Lookup("tookDate").Type)))
	}
	if stringOf(raw.Lookup("agencyId")) == "" {
		a.block(blockNoAgency, bondBoundCollection, id, typeAlias(raw.Lookup("agencyId").Type))
	}
}

// scanBondTypes decodifica cada tipo de bono.
func scanBondTypes(ctx context.Context, db *mongocc.MongoQueries, a *audit) error {
	return scan(ctx, db.GetCollection(bondTypeCollection), bson.M{}, byID(), func(c *mongo.Cursor) {
		var t bondType_domain.BondType
		_ = a.decode(bondTypeCollection, c, &t)
		a.checkTypes(bondTypeCollection, c.Current)
	})
}

// scanBondObjects decodifica cada producto y revisa sus numéricos, isPrivateBond y sus términos.
func scanBondObjects(ctx context.Context, db *mongocc.MongoQueries, a *audit) error {
	return scan(ctx, db.GetCollection(bondObjectCollection), bson.M{}, byID(), func(c *mongo.Cursor) {
		var o bondObject_domain.BondObject
		err := a.decode(bondObjectCollection, c, &o)
		a.checkTypes(bondObjectCollection, c.Current)
		a.Numeric.visit(bondObjectCollection, c.Current)
		a.Private.visit(c.Current, o, err)
		if err == nil && !o.IsRemove {
			a.checkTerms(bondObjectCollection, o.ID, o.MaturityTimeInDays, o.EarlyCashTerm.InflectionPoint,
				fmt.Sprintf("activo %v", o.Active))
		}
	})
}

// scanBondBounds decodifica cada bono y revisa su estado, sus numéricos y sus términos.
func scanBondBounds(ctx context.Context, db *mongocc.MongoQueries, a *audit) error {
	return scan(ctx, db.GetCollection(bondBoundCollection), bson.M{}, byID(), func(c *mongo.Cursor) {
		var b bondBound_domain.BondBound
		err := a.decode(bondBoundCollection, c, &b)
		a.checkTypes(bondBoundCollection, c.Current)
		a.checkBoundState(c.Current)
		a.Numeric.visit(bondBoundCollection, c.Current)
		operable := b.StatusID == bondBound_domain.StatusInitiated || b.StatusID == bondBound_domain.StatusFinalized
		if err == nil && !b.IsRemove && operable {
			a.checkTerms(bondBoundCollection, b.ID, b.MaturityTimeInDays, b.EarlyCashTerm.InflectionPoint,
				fmt.Sprintf("statusId %q", b.StatusID))
		}
	})
}

// scanBondHistory decodifica cada registro del historial.
func scanBondHistory(ctx context.Context, db *mongocc.MongoQueries, a *audit) error {
	return scan(ctx, db.GetCollection(bondHistoryCollection), bson.M{}, byID(), func(c *mongo.Cursor) {
		var h bondHistory_domain.BondHistory
		_ = a.decode(bondHistoryCollection, c, &h)
		a.checkTypes(bondHistoryCollection, c.Current)
	})
}

// scanBondUserSettings decodifica cada setting como lo lee su repositorio: un userId ObjectId legacy se lee
// como su hex.
func scanBondUserSettings(ctx context.Context, db *mongocc.MongoQueries, a *audit) error {
	opts := options.Collection().SetBSONOptions(&options.BSONOptions{ObjectIDAsHexString: true})
	coll := db.Database().Collection(bondUserSettingCollection, opts)
	return scan(ctx, coll, bson.M{}, byID(), func(c *mongo.Cursor) {
		var s bondUserSetting_domain.BondUserSetting
		_ = a.decode(bondUserSettingCollection, c, &s)
		a.checkTypes(bondUserSettingCollection, c.Current)
	})
}

// scanProfiles decodifica los perfiles vivos con la proyección pública con la que los lee Go por id. Guarda los
// que fallan, para no contarlos otra vez como dueños.
func scanProfiles(ctx context.Context, db *mongocc.MongoQueries, a *audit) error {
	opts := byID().SetProjection(publicProfileProjection)
	return scan(ctx, db.GetCollection(profileCollection), bson.M{"isRemove": false}, opts, func(c *mongo.Cursor) {
		var p shared_domain.PublicProfile
		if a.decode(profileCollection, c, &p) != nil {
			a.badProfiles[stringOf(c.Current.Lookup("_id"))] = true
		}
	})
}

// scanOwnerProfiles decodifica el perfil de cada dueño de bonos vivos como lo leen los cruces de Go, que no
// filtran isRemove: el nombre en los listados y reportes y, si tiene bonos '1', '2', '3' o '6', los datos de
// inversionista del producto.
func scanOwnerProfiles(ctx context.Context, db *mongocc.MongoQueries, a *audit) error {
	investors := map[string]bool{}
	opts := options.Find().SetProjection(bson.M{"userId": 1, "statusId": 1})
	err := scan(ctx, db.GetCollection(bondBoundCollection), bson.M{"isRemove": false}, opts, func(c *mongo.Cursor) {
		if owner := stringOf(c.Current.Lookup("userId")); owner != "" {
			investors[owner] = investors[owner] || investorStatuses[stringOf(c.Current.Lookup("statusId"))]
		}
	})
	if err != nil {
		return err
	}

	owners := slices.Sorted(maps.Keys(investors))
	opts = byID().SetProjection(ownerProfileProjection)
	for _, chunk := range chunks(owners, inChunk) {
		err := scan(ctx, db.GetCollection(profileCollection), bson.M{"_id": bson.M{"$in": chunk}}, opts, func(c *mongo.Cursor) {
			id := stringOf(c.Current.Lookup("_id"))
			if a.badProfiles[id] {
				return
			}
			var name bondBound_ports.ProfileName
			if a.decode(profileCollection, c, &name) != nil || !investors[id] {
				return
			}
			var investor bondObject_domain.InvestorProfile
			_ = a.decode(profileCollection, c, &investor)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// scanBalanceAccounts decodifica las cuentas vivas con la proyección con la que las lee Go, y revisa que el saldo
// se pueda sumar: Go mueve available y global con $inc, que falla sobre un valor que no es número.
func scanBalanceAccounts(ctx context.Context, db *mongocc.MongoQueries, a *audit) error {
	opts := byID().SetProjection(balanceAccountProjection)
	return scan(ctx, db.GetCollection(balanceAccountCollection), bson.M{"isRemove": false}, opts, func(c *mongo.Cursor) {
		var acc shared_domain.BalanceAccount
		if a.decode(balanceAccountCollection, c, &acc) != nil {
			return
		}
		for _, field := range []string{"available", "global"} {
			switch t := c.Current.Lookup(field).Type; t {
			case 0, bson.TypeInt32, bson.TypeInt64, bson.TypeDouble, bson.TypeDecimal128:
			default:
				a.block(blockBalance, balanceAccountCollection, "", fmt.Sprintf("%s: %s", field, typeAlias(t)))
			}
		}
	})
}

// collectIsRemoveOutsideBond busca perfiles y cuentas cuyo isRemove no es booleano: Go no los ve.
func collectIsRemoveOutsideBond(ctx context.Context, db *mongocc.MongoQueries, a *audit) error {
	filter := bson.M{"isRemove": bson.M{"$not": bson.M{"$type": "bool"}}}
	opts := byID().SetProjection(bson.M{"_id": 1, "isRemove": 1})
	for _, collection := range []string{profileCollection, balanceAccountCollection} {
		err := scan(ctx, db.GetCollection(collection), filter, opts, func(c *mongo.Cursor) {
			a.block(blockIsRemove, collection, "", "isRemove: "+typeAlias(c.Current.Lookup("isRemove").Type))
		})
		if err != nil {
			return err
		}
	}
	return nil
}
