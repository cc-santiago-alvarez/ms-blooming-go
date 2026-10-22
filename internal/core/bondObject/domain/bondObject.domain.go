package bondObject_domain

import (
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// BondObject es un producto del catálogo: precio, plazo, rendimiento, términos de retiro anticipado,
// ventana de venta y si es privado.
type BondObject struct {
	ID                        string                      `json:"_id" bson:"_id"`
	Name                      string                      `json:"name" bson:"name"`
	Description               string                      `json:"description" bson:"description"`
	AgencyID                  shared_domain.AgencyIDs     `json:"agencyId" bson:"agencyId"`
	Key                       string                      `json:"key" bson:"key"`
	BondTypeID                string                      `json:"bondTypeId" bson:"bondTypeId"`
	Amount                    float64                     `json:"amount" bson:"amount"`
	IsPrivateBond             bool                        `json:"isPrivateBond" bson:"isPrivateBond"`
	AuthorizedUserIDs         []string                    `json:"authorizedUserIds" bson:"authorizedUserIds"`
	MaturityTimeInDays        int                         `json:"maturityTimeInDays" bson:"maturityTimeInDays"`
	MaturityRewardsPercentage float64                     `json:"maturityRewardsPercentage" bson:"maturityRewardsPercentage"`
	Active                    bool                        `json:"active" bson:"active"`
	Available                 bool                        `json:"available" bson:"available"`
	SaleInitDate              *time.Time                  `json:"saleInitDate" bson:"saleInitDate"`
	SaleEndDate               *time.Time                  `json:"saleEndDate" bson:"saleEndDate"`
	EarlyCashTerm             shared_domain.EarlyCashTerm `json:"earlyCashTerm" bson:"earlyCashTerm"`
	CreatedAt                 time.Time                   `json:"createdAt" bson:"createdAt"`
	UpdatedAt                 time.Time                   `json:"updatedAt" bson:"updatedAt"`
	IsRemove                  bool                        `json:"isRemove" bson:"isRemove"`
}

// NewBondObject arma el producto nuevo de un alta ya normalizada.
func NewBondObject(p CreateParams) *BondObject {
	now := shared_domain.Now()
	o := &BondObject{
		ID:        shared_domain.NewID(),
		CreatedAt: now,
		UpdatedAt: now,
		IsRemove:  false,
	}
	o.setEditable(p)
	return o
}

// ProjectionTerms devuelve los términos del producto que usan la simulación y las guías.
func (o BondObject) ProjectionTerms() shared_domain.ProjectionTerms {
	return shared_domain.ProjectionTerms{
		MaturityTimeInDays:        o.MaturityTimeInDays,
		MaturityRewardsPercentage: o.MaturityRewardsPercentage,
		EarlyCashTerm:             o.EarlyCashTerm,
	}
}

// Apply reemplaza los campos editables con una edición ya normalizada y actualiza updatedAt.
func (o *BondObject) Apply(p UpdateParams) {
	o.setEditable(p)
	o.UpdatedAt = shared_domain.Now()
}

// setEditable copia los campos que reciben el alta y la edición.
func (o *BondObject) setEditable(p CreateParams) {
	o.Name = p.Name
	o.Description = p.Description
	o.AgencyID = p.AgencyID
	o.Key = p.Key
	o.BondTypeID = p.BondTypeID
	o.Amount = p.Amount
	o.IsPrivateBond = p.IsPrivateBond
	o.AuthorizedUserIDs = p.AuthorizedUserIDs
	o.MaturityTimeInDays = p.MaturityTimeInDays
	o.MaturityRewardsPercentage = p.MaturityRewardsPercentage
	o.Active = p.Active
	o.Available = p.Available
	o.SaleInitDate = p.SaleInitDate
	o.SaleEndDate = p.SaleEndDate
	o.EarlyCashTerm = p.EarlyCashTerm
}
