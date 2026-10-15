package shared_domain

import "time"

// NoticeCategoryInfo es la categoría de todos los avisos del ms.
const NoticeCategoryInfo = "1"

// NoticeRecord es un documento de notice. Unread en null marca el aviso como no leído.
type NoticeRecord struct {
	ID         string    `json:"_id" bson:"_id"`
	TypeID     string    `json:"typeId" bson:"typeId"`
	UserID     string    `json:"userId" bson:"userId"`
	AgencyID   string    `json:"agencyId" bson:"agencyId"`
	Message    string    `json:"message" bson:"message"`
	Title      string    `json:"title" bson:"title"`
	CategoryID string    `json:"categoryId" bson:"categoryId"`
	Unread     *bool     `json:"unread" bson:"unread"`
	CreatedAt  time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt" bson:"updatedAt"`
	IsRemove   bool      `json:"isRemove" bson:"isRemove"`
}

// NewNoticeRecord arma el documento nuevo de un aviso.
func NewNoticeRecord(n Notice) *NoticeRecord {
	now := Now()
	return &NoticeRecord{
		ID:         NewID(),
		TypeID:     string(n.Type),
		UserID:     n.UserID,
		AgencyID:   n.AgencyID,
		Message:    n.Message,
		Title:      n.Title,
		CategoryID: NoticeCategoryInfo,
		Unread:     nil,
		CreatedAt:  now,
		UpdatedAt:  now,
		IsRemove:   false,
	}
}
