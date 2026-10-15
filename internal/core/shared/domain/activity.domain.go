package shared_domain

import "time"

// Módulo de Blooming en activity.
const (
	ActivityModuleBloomingID   = "1"
	ActivityModuleBloomingName = "Blooming"
)

// ActivityRecord es un documento de activity.
type ActivityRecord struct {
	ID          string         `json:"_id" bson:"_id"`
	Description string         `json:"description" bson:"description"`
	ModuleID    string         `json:"moduleId" bson:"moduleId"`
	Name        string         `json:"name" bson:"name"`
	RelatedID   string         `json:"relatedId" bson:"relatedId"`
	AgencyID    string         `json:"agencyId" bson:"agencyId"`
	UserID      string         `json:"userId" bson:"userId"`
	Params      map[string]any `json:"params" bson:"params"`
	CreatedAt   time.Time      `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt" bson:"updatedAt"`
	IsRemove    bool           `json:"isRemove" bson:"isRemove"`
}

// NewActivityRecord arma el documento nuevo de una actividad; sin params guarda {}.
func NewActivityRecord(a Activity) *ActivityRecord {
	params := a.Params
	if params == nil {
		params = map[string]any{}
	}
	now := Now()
	return &ActivityRecord{
		ID:          NewID(),
		Description: a.Description,
		ModuleID:    ActivityModuleBloomingID,
		Name:        ActivityModuleBloomingName,
		RelatedID:   a.RelatedID,
		AgencyID:    a.AgencyID,
		UserID:      a.UserID,
		Params:      params,
		CreatedAt:   now,
		UpdatedAt:   now,
		IsRemove:    false,
	}
}
