package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/codecraftkit/mongocc"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Colecciones que lee el arnés; session es la única en la que escribe.
const (
	bondObjectCollection = "bondObject"
	bondBoundCollection  = "bondBound"
	profileCollection    = "profile"
	agencyCollection     = "agency"
	sessionCollection    = "session"
)

// zoneByCountry es la diferencia horaria por país con la que el legado y el API resuelven la de la sesión.
var zoneByCountry = map[string]int{"CO": -5, "VE": -4, "MX": -7, "US": -6}

// byID ordena por _id, para que la muestra sea estable entre corridas.
var byID = bson.D{{Key: "_id", Value: 1}}

// storedTerms son los términos de un producto o un bono tal como están guardados, con los numéricos legacy que
// pueden ser string: así se los manda al legado.
type storedTerms struct {
	Amount                    any                  `bson:"amount" json:"amount"`
	MaturityTimeInDays        any                  `bson:"maturityTimeInDays" json:"maturityTimeInDays"`
	MaturityRewardsPercentage any                  `bson:"maturityRewardsPercentage" json:"maturityRewardsPercentage"`
	EarlyCashTerm             storedTermsEarlyCash `bson:"earlyCashTerm" json:"earlyCashTerm"`
}

// storedTermsEarlyCash son las condiciones de retiro anticipado tal como están guardadas.
type storedTermsEarlyCash struct {
	InitialPenaltyPercentage    any `bson:"initialPenaltyPercentage" json:"initialPenaltyPercentage"`
	InflectionPoint             any `bson:"inflectionPoint" json:"inflectionPoint"`
	InflectionPenaltyPercentage any `bson:"inflectionPenaltyPercentage" json:"inflectionPenaltyPercentage"`
	InitialDayForWithdraw       any `bson:"initialDayForWithdraw" json:"initialDayForWithdraw"`
}

// objectDoc es un producto de la muestra.
type objectDoc struct {
	ID    string      `bson:"_id"`
	Terms storedTerms `bson:",inline"`
}

// boundDoc es un bono de la muestra.
type boundDoc struct {
	ID       string      `bson:"_id"`
	UserID   string      `bson:"userId"`
	StatusID string      `bson:"statusId"`
	Terms    storedTerms `bson:",inline"`
}

// profileDoc son los campos del perfil que deciden si un usuario sirve para la corrida.
type profileDoc struct {
	ID            string `bson:"_id"`
	AgencyID      string `bson:"agencyId"`
	CountryCode   string `bson:"country_code"`
	AdminBlooming bool   `bson:"adminBlooming"`
}

// sample son los casos de la corrida: el admin con el que se cotizan y grafican los bonos, los productos, los
// bonos y los dueños cuyos listados se comparan.
type sample struct {
	admin   string
	objects []objectDoc
	bounds  []boundDoc
	owners  []string
}

// loadSample arma la muestra: todos los productos no borrados, hasta cfg.bounds bonos iniciados o madurados sin
// venta ni canje, y hasta cfg.users dueños de esos bonos en la zona de la corrida.
func loadSample(ctx context.Context, db *mongocc.MongoQueries, cfg runConfig) (sample, error) {
	admin, err := pickAdmin(ctx, db, cfg.admin, cfg.utcDiff)
	if err != nil {
		return sample{}, err
	}
	objects, err := findAll[objectDoc](ctx, db, bondObjectCollection, bson.M{"isRemove": false},
		options.Find().SetSort(byID))
	if err != nil {
		return sample{}, err
	}
	bounds, err := findAll[boundDoc](ctx, db, bondBoundCollection, bson.M{
		"isRemove": false,
		"statusId": bson.M{"$in": bson.A{"1", "2"}},
		"soldDate": nil,
		"tookDate": nil,
	}, options.Find().SetSort(byID).SetLimit(int64(cfg.bounds)))
	if err != nil {
		return sample{}, err
	}
	owners, err := pickOwners(ctx, db, bounds, cfg.utcDiff, cfg.users)
	if err != nil {
		return sample{}, err
	}
	return sample{admin: admin, objects: objects, bounds: bounds, owners: owners}, nil
}

// pickAdmin devuelve id si es un admin de Blooming de la agencia principal en la zona, o, sin id, el primero
// que lo sea: un admin de otra agencia no ve los bonos de las demás.
func pickAdmin(ctx context.Context, db *mongocc.MongoQueries, id string, zone int) (string, error) {
	filter := bson.M{"adminBlooming": true, "isRemove": false, "status": true}
	if id != "" {
		filter["_id"] = id
	}
	profiles, err := findAll[profileDoc](ctx, db, profileCollection, filter, options.Find().SetSort(byID))
	if err != nil {
		return "", err
	}
	for _, p := range profiles {
		ok, err := inZone(ctx, db, p, zone)
		if err != nil {
			return "", err
		}
		if !ok {
			continue
		}
		main, err := isMainAgency(ctx, db, p.AgencyID)
		if err != nil {
			return "", err
		}
		if main {
			return p.ID, nil
		}
	}
	if id != "" {
		return "", fmt.Errorf("%s no es un admin de Blooming activo de la agencia principal con perfil y agencia en la zona %d", id, zone)
	}
	return "", fmt.Errorf("no hay un admin de Blooming activo de la agencia principal con perfil y agencia en la zona %d; pasar -admin", zone)
}

// isMainAgency indica si agencyID es la agencia principal: la que tiene main, que ms-oauth entrega como
// agency.isMain.
func isMainAgency(ctx context.Context, db *mongocc.MongoQueries, agencyID string) (bool, error) {
	n, err := db.CountDocuments(ctx, agencyCollection, bson.M{"_id": agencyID, "main": true})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// pickOwners devuelve hasta limit dueños distintos de bounds, en orden, activos y con perfil y agencia en la
// zona: ms-oauth rechaza la sesión de un usuario inactivo.
func pickOwners(ctx context.Context, db *mongocc.MongoQueries, bounds []boundDoc, zone, limit int) ([]string, error) {
	var owners []string
	seen := map[string]bool{}
	for _, b := range bounds {
		if len(owners) == limit {
			break
		}
		if seen[b.UserID] {
			continue
		}
		seen[b.UserID] = true

		var p profileDoc
		err := db.FindOne(ctx, profileCollection, bson.M{"_id": b.UserID, "isRemove": false, "status": true}, nil).Decode(&p)
		if errors.Is(err, mongo.ErrNoDocuments) {
			continue
		}
		if err != nil {
			return nil, err
		}
		ok, err := inZone(ctx, db, p, zone)
		if err != nil {
			return nil, err
		}
		if ok {
			owners = append(owners, b.UserID)
		}
	}
	return owners, nil
}

// inZone dice si el país del perfil y el de su agencia corresponden a zone: Meteor toma la zona de la sesión de
// la agencia y ms-oauth, del perfil, así que tienen que coincidir.
func inZone(ctx context.Context, db *mongocc.MongoQueries, p profileDoc, zone int) (bool, error) {
	if z, ok := zoneByCountry[strings.ToUpper(p.CountryCode)]; !ok || z != zone {
		return false, nil
	}
	var agency struct {
		CountryCode string `bson:"countryCode"`
	}
	err := db.FindOne(ctx, agencyCollection, bson.M{"_id": p.AgencyID}, nil).Decode(&agency)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	z, ok := zoneByCountry[strings.ToUpper(agency.CountryCode)]
	return ok && z == zone, nil
}

// findAll devuelve los documentos de collection que pasan filter.
func findAll[T any](ctx context.Context, db *mongocc.MongoQueries, collection string, filter any, opts *options.FindOptionsBuilder) ([]T, error) {
	cursor, err := db.Find(ctx, collection, filter, opts)
	if err != nil {
		return nil, err
	}
	var docs []T
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("%s: %w", collection, err)
	}
	return docs, nil
}

// sessions crea las sesiones temporales de la corrida. Meteor y ms-oauth, y por lo tanto el API Go, aceptan
// cualquier token que esté en la colección session.
type sessions struct {
	db  *mongocc.MongoQueries
	ids []string
}

// create guarda una sesión de userID y devuelve su token.
func (s *sessions) create(ctx context.Context, userID string) (string, error) {
	id, token := uuid.NewString(), uuid.NewString()
	now := time.Now().UTC()
	doc := bson.M{"_id": id, "userId": userID, "token": token, "isRemove": false, "createdAt": now, "updatedAt": now}
	if _, err := s.db.InsertOne(ctx, sessionCollection, doc); err != nil {
		return "", fmt.Errorf("no se pudo crear la sesión de %s: %w", userID, err)
	}
	s.ids = append(s.ids, id)
	return token, nil
}

// removeAll borra las sesiones creadas, aunque la corrida se haya cancelado.
func (s *sessions) removeAll() error {
	if len(s.ids) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := s.db.DeleteMany(ctx, sessionCollection, bson.M{"_id": bson.M{"$in": s.ids}}, nil)
	return err
}
