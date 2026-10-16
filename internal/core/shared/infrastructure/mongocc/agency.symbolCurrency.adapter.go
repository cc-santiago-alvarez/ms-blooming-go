package shared_mongocc_repository

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// SymbolCurrency sigue agency → country y devuelve el símbolo de moneda. Si la agencia o el país no
// existen, están borrados o no tienen símbolo, devuelve "$".
func (repo *AgencyMongoccRepository) SymbolCurrency(ctx context.Context, agencyID string) (string, error) {
	var agency struct {
		CountryCode string `bson:"countryCode"`
	}
	err := repo.DB.FindOne(ctx, agencyCollection,
		bson.M{"_id": agencyID, "isRemove": false},
		options.FindOne().SetProjection(bson.M{"countryCode": 1}),
	).Decode(&agency)
	if errors.Is(err, mongo.ErrNoDocuments) || (err == nil && agency.CountryCode == "") {
		return defaultSymbolCurrency, nil
	}
	if err != nil {
		return "", err
	}

	var country struct {
		SymbolCurrency string `bson:"symbolCurrency"`
	}
	err = repo.DB.FindOne(ctx, countryCollection,
		bson.M{"code": agency.CountryCode, "isRemove": false},
		options.FindOne().SetProjection(bson.M{"symbolCurrency": 1}),
	).Decode(&country)
	if errors.Is(err, mongo.ErrNoDocuments) || (err == nil && country.SymbolCurrency == "") {
		return defaultSymbolCurrency, nil
	}
	if err != nil {
		return "", err
	}
	return country.SymbolCurrency, nil
}
