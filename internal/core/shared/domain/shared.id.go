package shared_domain

import (
	"regexp"

	"github.com/google/uuid"
)

// IDKind es el formato de un _id.
type IDKind string

const (
	IDKindUUID    IDKind = "uuid"
	IDKindMeteor  IDKind = "meteor"
	IDKindInvalid IDKind = "invalid"
)

var (
	uuidV4Pattern   = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	meteorIDPattern = regexp.MustCompile(`^[A-Za-z0-9]{17}$`)
)

// NewID genera el _id de un documento nuevo (UUID v4).
func NewID() string {
	return uuid.New().String()
}

// IDFormat devuelve el formato del id: UUID v4, Meteor de 17 caracteres o inválido.
func IDFormat(id string) IDKind {
	switch {
	case uuidV4Pattern.MatchString(id):
		return IDKindUUID
	case meteorIDPattern.MatchString(id):
		return IDKindMeteor
	}
	return IDKindInvalid
}

// IsValidID indica si el id es UUID v4 o Meteor.
func IsValidID(id string) bool {
	return IDFormat(id) != IDKindInvalid
}
