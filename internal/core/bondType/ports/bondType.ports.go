package bondType_ports

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
)

// BondTypeApplicationPorts son los casos de uso de bondType.
type BondTypeApplicationPorts interface {
	// List devuelve los tipos no borrados del alcance de agencia de la sesión, por name. agencyIDs solo
	// filtra si la agencia de la sesión es la principal. Un agencyID inválido es ErrInvalidID.
	List(ctx context.Context, agencyIDs []string) ([]*bondType_domain.BondType, error)
	// ListActive devuelve los tipos activos de la agencia de la sesión, con la cantidad de bonos que el
	// usuario puede ver.
	ListActive(ctx context.Context) ([]bondType_domain.ActiveBondType, error)
	// GetByID devuelve el tipo no borrado del alcance de agencia de la sesión, o ErrBondTypeNotFound.
	GetByID(ctx context.Context, id string) (*bondType_domain.BondType, error)
	// GetManyByIDs devuelve los tipos no borrados que existan entre ids; los demás no aparecen.
	GetManyByIDs(ctx context.Context, ids []string) ([]*bondType_domain.BondType, error)
	// Create normaliza y valida p, y devuelve el tipo creado. Fuera de la agencia principal, p solo puede
	// tener la agencia de la sesión; si no, es ErrForbidden.
	Create(ctx context.Context, p bondType_domain.CreateParams) (*bondType_domain.BondType, error)
	// Update reemplaza los campos editables del tipo del alcance de agencia de la sesión y lo devuelve
	// actualizado, o ErrBondTypeNotFound. Fuera de la agencia principal, el tipo y p solo pueden tener la
	// agencia de la sesión; si no, es ErrForbidden.
	Update(ctx context.Context, id string, p bondType_domain.UpdateParams) (*bondType_domain.BondType, error)
	// Remove borra de forma lógica el tipo del alcance de agencia de la sesión, o devuelve
	// ErrBondTypeNotFound. Fuera de la agencia principal, si también es de otras agencias devuelve
	// ErrForbidden; si tiene bondObjects no borrados, ErrBondTypeHasBondObjects.
	Remove(ctx context.Context, id string) error
}

// BondTypeMongoccRepositoryPorts es el repositorio de bondType. Ninguna consulta devuelve tipos borrados.
type BondTypeMongoccRepositoryPorts interface {
	// Find devuelve los tipos por name ascendente. agencyIDs vacío no filtra por agencia; onlyActive deja
	// solo los activos.
	Find(ctx context.Context, agencyIDs []string, onlyActive bool) ([]*bondType_domain.BondType, error)
	// FindOneByID devuelve nil, nil si el tipo no existe o está borrado.
	FindOneByID(ctx context.Context, id string) (*bondType_domain.BondType, error)
	// FindManyByIDs devuelve los tipos de ids que existan; los demás no aparecen.
	FindManyByIDs(ctx context.Context, ids []string) ([]*bondType_domain.BondType, error)
	InsertOne(ctx context.Context, t *bondType_domain.BondType) error
	// UpdateOne guarda name, description, active, agencyId y updatedAt de t; devuelve false si el tipo ya
	// no existe o está borrado.
	UpdateOne(ctx context.Context, t *bondType_domain.BondType) (bool, error)
	// SoftDelete marca el tipo con isRemove true y actualiza updatedAt.
	SoftDelete(ctx context.Context, id string) error
	// EnsureIndexes crea los índices de la colección; es idempotente.
	EnsureIndexes(ctx context.Context) error
}

// BondObjectCounterPort cuenta los bondObjects de cada tipo; lo implementa la aplicación de bondObject.
type BondObjectCounterPort interface {
	// CountActiveByType cuenta, por tipo, los bondObjects no borrados y activos que userID puede ver: los
	// privados solo si está autorizado. Un tipo sin bonos puede no aparecer en el mapa.
	CountActiveByType(ctx context.Context, bondTypeIDs []string, userID string) (map[string]int, error)
	// ExistsNotRemovedByType indica si algún bondObject no borrado usa el tipo.
	ExistsNotRemovedByType(ctx context.Context, bondTypeID string) (bool, error)
}
