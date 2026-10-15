package shared_domain

import "github.com/google/uuid"

// NewID genera el _id de un documento nuevo (UUID v4).
func NewID() string {
	return uuid.New().String()
}
