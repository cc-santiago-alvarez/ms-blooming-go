package bondUserSetting_domain

import (
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// DefaultLanguage es el idioma de un setting nuevo.
const DefaultLanguage = "es"

// BondUserSetting son las preferencias de un inversionista: idioma, bono seleccionado y la cuenta donde
// se acreditan las ventas y los canjes.
type BondUserSetting struct {
	ID                  string    `json:"_id" bson:"_id"`
	UserID              string    `json:"userId" bson:"userId"`
	AgencyID            string    `json:"agencyId" bson:"agencyId"`
	Language            string    `json:"language" bson:"language"`
	SelectedAccountID   *string   `json:"selectedAccountId" bson:"selectedAccountId"`
	SelectedBondBoundID *string   `json:"selectedBondBoundId" bson:"selectedBondBoundId"`
	CreatedAt           time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt" bson:"updatedAt"`
	IsRemove            bool      `json:"isRemove" bson:"isRemove"`
}

// NewBondUserSetting arma el setting por defecto de un usuario: idioma "es", la cuenta recibida (o nil) y
// sin bono seleccionado.
func NewBondUserSetting(userID, agencyID string, selectedAccountID *string) *BondUserSetting {
	now := shared_domain.Now()
	return &BondUserSetting{
		ID:                  shared_domain.NewID(),
		UserID:              userID,
		AgencyID:            agencyID,
		Language:            DefaultLanguage,
		SelectedAccountID:   selectedAccountID,
		SelectedBondBoundID: nil,
		CreatedAt:           now,
		UpdatedAt:           now,
		IsRemove:            false,
	}
}
