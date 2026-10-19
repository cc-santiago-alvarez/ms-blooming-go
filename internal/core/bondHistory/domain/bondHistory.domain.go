package bondHistory_domain

import (
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// ChangeEntry es el valor anterior y el nuevo de un campo editado.
type ChangeEntry struct {
	Old any `json:"old" bson:"old"`
	New any `json:"new" bson:"new"`
}

// BondHistory es un registro del historial de un producto o de un bono comprado. Text es la clave del
// catálogo, que se formatea con Params al mostrarlo.
type BondHistory struct {
	ID               string                 `json:"_id" bson:"_id"`
	RelatedID        string                 `json:"relatedId" bson:"relatedId"`
	UserID           string                 `json:"-" bson:"userId"`
	TypeID           HistoryType            `json:"typeId" bson:"typeId"`
	AgencyID         string                 `json:"agencyId" bson:"agencyId"`
	Text             string                 `json:"text" bson:"text"`
	Params           []string               `json:"-" bson:"params"`
	Changes          map[string]ChangeEntry `json:"changes,omitempty" bson:"changes,omitempty"`
	EditedProperties []string               `json:"editedProperties,omitempty" bson:"editedProperties,omitempty"`
	CreatedAt        time.Time              `json:"createdAt" bson:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt" bson:"updatedAt"`
	IsRemove         bool                   `json:"isRemove" bson:"isRemove"`
}

// NewBondHistory arma el registro nuevo de un evento ya normalizado. userID vacío es un registro del
// sistema.
func NewBondHistory(p AddParams, userID, agencyID string) *BondHistory {
	now := shared_domain.Now()
	return &BondHistory{
		ID:               shared_domain.NewID(),
		RelatedID:        p.RelatedID,
		UserID:           userID,
		TypeID:           p.TypeID,
		AgencyID:         agencyID,
		Text:             p.Text,
		Params:           p.Params,
		Changes:          p.Changes,
		EditedProperties: p.EditedProperties,
		CreatedAt:        now,
		UpdatedAt:        now,
		IsRemove:         false,
	}
}
