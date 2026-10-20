package bondObject_domain

import (
	"fmt"
	"strings"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// bondObjectBSON lee un documento legacy: los numéricos pueden venir como string y las fechas de venta
// como string ISO.
type bondObjectBSON struct {
	ID                        string                      `bson:"_id"`
	Name                      string                      `bson:"name"`
	Description               string                      `bson:"description"`
	AgencyID                  shared_domain.AgencyIDs     `bson:"agencyId"`
	Key                       string                      `bson:"key"`
	BondTypeID                string                      `bson:"bondTypeId"`
	Amount                    shared_domain.FlexNumber    `bson:"amount"`
	IsPrivateBond             bool                        `bson:"isPrivateBond"`
	AuthorizedUserIDs         []string                    `bson:"authorizedUserIds"`
	MaturityTimeInDays        shared_domain.FlexNumber    `bson:"maturityTimeInDays"`
	MaturityRewardsPercentage shared_domain.FlexNumber    `bson:"maturityRewardsPercentage"`
	Active                    bool                        `bson:"active"`
	Available                 bool                        `bson:"available"`
	SaleInitDate              legacyTime                  `bson:"saleInitDate"`
	SaleEndDate               legacyTime                  `bson:"saleEndDate"`
	EarlyCashTerm             shared_domain.EarlyCashTerm `bson:"earlyCashTerm"`
	CreatedAt                 time.Time                   `bson:"createdAt"`
	UpdatedAt                 time.Time                   `bson:"updatedAt"`
	IsRemove                  bool                        `bson:"isRemove"`
}

// UnmarshalBSON decodifica el producto aceptando los formatos legacy. Los días se truncan a entero y
// authorizedUserIds null queda en [].
func (o *BondObject) UnmarshalBSON(data []byte) error {
	var raw bondObjectBSON
	if err := bson.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("bondObject: %w", err)
	}

	authorized := raw.AuthorizedUserIDs
	if authorized == nil {
		authorized = []string{}
	}
	*o = BondObject{
		ID:                        raw.ID,
		Name:                      raw.Name,
		Description:               raw.Description,
		AgencyID:                  raw.AgencyID,
		Key:                       raw.Key,
		BondTypeID:                raw.BondTypeID,
		Amount:                    float64(raw.Amount),
		IsPrivateBond:             raw.IsPrivateBond,
		AuthorizedUserIDs:         authorized,
		MaturityTimeInDays:        raw.MaturityTimeInDays.Int(),
		MaturityRewardsPercentage: float64(raw.MaturityRewardsPercentage),
		Active:                    raw.Active,
		Available:                 raw.Available,
		SaleInitDate:              raw.SaleInitDate.value,
		SaleEndDate:               raw.SaleEndDate.value,
		EarlyCashTerm:             raw.EarlyCashTerm,
		CreatedAt:                 raw.CreatedAt,
		UpdatedAt:                 raw.UpdatedAt,
		IsRemove:                  raw.IsRemove,
	}
	return nil
}

// legacyTime es una fecha opcional de un documento legacy, guardada como Date o como string.
type legacyTime struct {
	value *time.Time
}

// legacyDateLayouts son los formatos de string aceptados: ISO de JavaScript y solo fecha.
var legacyDateLayouts = []string{time.RFC3339, "2006-01-02"}

// UnmarshalBSONValue acepta Date y string; null, undefined y el string vacío quedan en nil. Un string que
// no es fecha u otro tipo es un error.
func (lt *legacyTime) UnmarshalBSONValue(t byte, data []byte) error {
	rv := bson.RawValue{Type: bson.Type(t), Value: data}

	switch rv.Type {
	case bson.TypeDateTime:
		if ms, ok := rv.DateTimeOK(); ok {
			v := time.UnixMilli(ms).UTC()
			lt.value = &v
			return nil
		}
	case bson.TypeString:
		if s, ok := rv.StringValueOK(); ok {
			return lt.parse(s)
		}
	case bson.TypeNull, bson.TypeUndefined:
		lt.value = nil
		return nil
	default:
		return fmt.Errorf("fecha legacy: tipo BSON %s no soportado", rv.Type)
	}
	return fmt.Errorf("fecha legacy: valor BSON %s malformado", rv.Type)
}

// parse lee una fecha en string; el string vacío es nil.
func (lt *legacyTime) parse(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		lt.value = nil
		return nil
	}
	for _, layout := range legacyDateLayouts {
		if v, err := time.Parse(layout, s); err == nil {
			v = v.UTC()
			lt.value = &v
			return nil
		}
	}
	return fmt.Errorf("fecha legacy: %q no es una fecha", s)
}
