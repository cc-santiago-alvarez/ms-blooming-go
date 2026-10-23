package bondBound_domain

import (
	"fmt"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// bondBoundBSON lee un documento legacy: los numéricos copiados del producto pueden venir como string.
type bondBoundBSON struct {
	ID                        string                      `bson:"_id"`
	BondObjectID              string                      `bson:"bondObjectId"`
	BondTypeID                string                      `bson:"bondTypeId"`
	Key                       string                      `bson:"key"`
	KeySeq                    *int                        `bson:"keySeq"`
	Token                     string                      `bson:"token"`
	UserID                    string                      `bson:"userId"`
	AgencyID                  string                      `bson:"agencyId"`
	StatusID                  BondBoundStatus             `bson:"statusId"`
	Amount                    shared_domain.FlexNumber    `bson:"amount"`
	MaturityTimeInDays        shared_domain.FlexNumber    `bson:"maturityTimeInDays"`
	MaturityRewardsPercentage shared_domain.FlexNumber    `bson:"maturityRewardsPercentage"`
	EarlyCashTerm             shared_domain.EarlyCashTerm `bson:"earlyCashTerm"`
	InitialDate               time.Time                   `bson:"initialDate"`
	MaturityDate              time.Time                   `bson:"maturityDate"`
	SoldDate                  *time.Time                  `bson:"soldDate"`
	TookDate                  *time.Time                  `bson:"tookDate"`
	Matured                   bool                        `bson:"matured"`
	CreatedAt                 time.Time                   `bson:"createdAt"`
	UpdatedAt                 time.Time                   `bson:"updatedAt"`
	IsRemove                  bool                        `bson:"isRemove"`
}

// UnmarshalBSON decodifica el bono aceptando los formatos legacy. Los días se truncan a entero y un
// documento sin keySeq lo deja en nil.
func (b *BondBound) UnmarshalBSON(data []byte) error {
	var raw bondBoundBSON
	if err := bson.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("bondBound: %w", err)
	}

	*b = BondBound{
		ID:                        raw.ID,
		BondObjectID:              raw.BondObjectID,
		BondTypeID:                raw.BondTypeID,
		Key:                       raw.Key,
		KeySeq:                    raw.KeySeq,
		Token:                     raw.Token,
		UserID:                    raw.UserID,
		AgencyID:                  raw.AgencyID,
		StatusID:                  raw.StatusID,
		Amount:                    float64(raw.Amount),
		MaturityTimeInDays:        raw.MaturityTimeInDays.Int(),
		MaturityRewardsPercentage: float64(raw.MaturityRewardsPercentage),
		EarlyCashTerm:             raw.EarlyCashTerm,
		InitialDate:               raw.InitialDate,
		MaturityDate:              raw.MaturityDate,
		SoldDate:                  raw.SoldDate,
		TookDate:                  raw.TookDate,
		Matured:                   raw.Matured,
		CreatedAt:                 raw.CreatedAt,
		UpdatedAt:                 raw.UpdatedAt,
		IsRemove:                  raw.IsRemove,
	}
	return nil
}
