package bondType_domain

import (
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// BondType es un tipo de bono del catálogo.
type BondType struct {
	ID          string                  `json:"_id" bson:"_id"`
	AgencyID    shared_domain.AgencyIDs `json:"agencyId" bson:"agencyId"`
	Name        string                  `json:"name" bson:"name"`
	Description string                  `json:"description" bson:"description"`
	Active      bool                    `json:"active" bson:"active"`
	CreatedAt   time.Time               `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time               `json:"updatedAt" bson:"updatedAt"`
	IsRemove    bool                    `json:"isRemove" bson:"isRemove"`
}

// NewBondType arma el tipo nuevo de un alta ya normalizada.
func NewBondType(p CreateParams) *BondType {
	now := shared_domain.Now()
	return &BondType{
		ID:          shared_domain.NewID(),
		AgencyID:    p.AgencyID,
		Name:        p.Name,
		Description: p.Description,
		Active:      p.Active,
		CreatedAt:   now,
		UpdatedAt:   now,
		IsRemove:    false,
	}
}

// Apply reemplaza los campos editables con una edición ya normalizada y actualiza updatedAt.
func (t *BondType) Apply(p UpdateParams) {
	t.Name = p.Name
	t.Description = p.Description
	t.Active = p.Active
	t.AgencyID = p.AgencyID
	t.UpdatedAt = shared_domain.Now()
}
