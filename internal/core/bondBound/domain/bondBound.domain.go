package bondBound_domain

import (
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// BondBound es un bono comprado: guarda una copia de los términos del producto al momento de la compra.
// KeySeq es el número de la clave dentro de su tipo; los bonos legacy no lo tienen.
type BondBound struct {
	ID                        string                      `json:"_id" bson:"_id"`
	BondObjectID              string                      `json:"bondObjectId" bson:"bondObjectId"`
	BondTypeID                string                      `json:"bondTypeId" bson:"bondTypeId"`
	Key                       string                      `json:"key" bson:"key"`
	KeySeq                    *int                        `json:"-" bson:"keySeq,omitempty"`
	Token                     string                      `json:"token" bson:"token"`
	UserID                    string                      `json:"userId" bson:"userId"`
	AgencyID                  string                      `json:"agencyId" bson:"agencyId"`
	StatusID                  BondBoundStatus             `json:"statusId" bson:"statusId"`
	Amount                    float64                     `json:"amount" bson:"amount"`
	MaturityTimeInDays        int                         `json:"maturityTimeInDays" bson:"maturityTimeInDays"`
	MaturityRewardsPercentage float64                     `json:"maturityRewardsPercentage" bson:"maturityRewardsPercentage"`
	EarlyCashTerm             shared_domain.EarlyCashTerm `json:"earlyCashTerm" bson:"earlyCashTerm"`
	InitialDate               time.Time                   `json:"initialDate" bson:"initialDate"`
	MaturityDate              time.Time                   `json:"maturityDate" bson:"maturityDate"`
	SoldDate                  *time.Time                  `json:"soldDate,omitempty" bson:"soldDate,omitempty"`
	TookDate                  *time.Time                  `json:"tookDate,omitempty" bson:"tookDate,omitempty"`
	Matured                   bool                        `json:"matured,omitempty" bson:"matured,omitempty"`
	CreatedAt                 time.Time                   `json:"createdAt" bson:"createdAt"`
	UpdatedAt                 time.Time                   `json:"updatedAt" bson:"updatedAt"`
	IsRemove                  bool                        `json:"isRemove" bson:"isRemove"`
}
