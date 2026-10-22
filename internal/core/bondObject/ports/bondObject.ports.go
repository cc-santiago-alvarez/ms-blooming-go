package bondObject_ports

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	bondType_ports "github.com/futurapps/ms-blooming-go/internal/core/bondType/ports"
)

// BondObjectGroup son los productos de un tipo; BondType es nil si el tipo no existe o está borrado.
type BondObjectGroup struct {
	BondTypeID string                         `json:"_id" bson:"_id"`
	BondType   *bondType_domain.BondType      `json:"bondType" bson:"bondType"`
	List       []bondObject_domain.BondObject `json:"list" bson:"list"`
}

// BondObjectApplicationPorts son los casos de uso de bondObject; también cuenta productos para bondType.
type BondObjectApplicationPorts interface {
	bondType_ports.BondObjectCounterPort

	// List devuelve los productos no borrados por name; con agencyIDs, solo los que tienen alguna de esas
	// agencias. Un agencyID inválido es ErrInvalidID.
	List(ctx context.Context, agencyIDs []string) ([]*bondObject_domain.BondObject, error)
	// ListGroupedByType agrupa por tipo los productos del alcance de la sesión; agencyIDs solo filtra si
	// la agencia de la sesión es la principal. Un agencyID inválido es ErrInvalidID.
	ListGroupedByType(ctx context.Context, agencyIDs []string) ([]BondObjectGroup, error)
	// GetByID devuelve el producto no borrado con los perfiles públicos de sus autorizados, o
	// ErrBondObjectNotFound. Un id inválido es ErrInvalidID.
	GetByID(ctx context.Context, id string) (*bondObject_domain.BondObjectDetail, error)
	// GetByIDIncludingRemoved devuelve el producto aunque esté borrado, o ErrBondObjectNotFound. Un id
	// inválido es ErrInvalidID.
	GetByIDIncludingRemoved(ctx context.Context, id string) (*bondObject_domain.BondObject, error)
	// Create normaliza y valida p, guarda el producto y devuelve el creado.
	Create(ctx context.Context, p bondObject_domain.CreateParams) (*bondObject_domain.BondObject, error)
	// Update reemplaza los campos editables y devuelve el producto actualizado, o ErrBondObjectNotFound.
	Update(ctx context.Context, id string, p bondObject_domain.UpdateParams) (*bondObject_domain.BondObject, error)
	// UpdateKeys cambia la clave de los productos no borrados de la lista.
	UpdateKeys(ctx context.Context, p bondObject_domain.UpdateKeysParams) (bondObject_domain.UpdateKeysResult, error)
	// Remove borra el producto de forma lógica, o devuelve ErrBondObjectNotFound.
	Remove(ctx context.Context, id string) error
	// ListInvestors devuelve los usuarios con bonos vigentes del producto, en el alcance de la sesión.
	ListInvestors(ctx context.Context, id string) ([]bondObject_domain.BondObjectInvestor, error)
	// ListByType devuelve los productos activos del tipo que ve la sesión, con su fecha de maduración.
	ListByType(ctx context.Context, bondTypeID string) ([]bondObject_domain.BondObjectForInvestor, error)
	// MostPurchased devuelve los productos más comprados en la agencia de la sesión que ella puede ver.
	MostPurchased(ctx context.Context) ([]*bondObject_domain.BondObject, error)
	// Latest devuelve los últimos productos publicados que ve la sesión.
	Latest(ctx context.Context) ([]*bondObject_domain.BondObject, error)
	// FindVisibleForPurchase devuelve el producto no borrado, de cualquier agencia, o nil si no existe.
	FindVisibleForPurchase(ctx context.Context, id string) (*bondObject_domain.BondObject, error)
}

// BondObjectMongoccRepositoryPorts es el repositorio de bondObject.
type BondObjectMongoccRepositoryPorts interface {
	// Find devuelve los productos no borrados por name; agencyIDs vacío no filtra por agencia.
	Find(ctx context.Context, agencyIDs []string) ([]*bondObject_domain.BondObject, error)
	// FindGroupedByType agrupa por bondTypeId los productos no borrados, con su tipo, en una sola
	// agregación; agencyIDs vacío no filtra por agencia.
	FindGroupedByType(ctx context.Context, agencyIDs []string) ([]BondObjectGroup, error)
	// FindOneByID devuelve nil, nil si el producto no existe o está borrado.
	FindOneByID(ctx context.Context, id string) (*bondObject_domain.BondObject, error)
	// FindOneByIDIncludingRemoved devuelve el producto aunque esté borrado, o nil, nil si no existe.
	FindOneByIDIncludingRemoved(ctx context.Context, id string) (*bondObject_domain.BondObject, error)
	// FindVisible devuelve los productos que ve un inversionista, por createdAt descendente.
	FindVisible(ctx context.Context, f bondObject_domain.VisibleFilter) ([]*bondObject_domain.BondObject, error)
	InsertOne(ctx context.Context, o *bondObject_domain.BondObject) error
	// UpdateOne guarda los campos editables y updatedAt de o; devuelve false si el producto ya no existe o
	// está borrado.
	UpdateOne(ctx context.Context, o *bondObject_domain.BondObject) (bool, error)
	// UpdateKeys cambia key y updatedAt de los productos no borrados de ids.
	UpdateKeys(ctx context.Context, ids []string, key string) (bondObject_domain.UpdateKeysResult, error)
	// SoftDelete marca el producto con isRemove true y actualiza updatedAt; devuelve false si no existe o
	// ya estaba borrado.
	SoftDelete(ctx context.Context, id string) (bool, error)
	// CountActiveByType cuenta, por tipo, los productos no borrados y activos que userID puede ver: los
	// privados solo si está autorizado. Un tipo sin productos puede no aparecer en el mapa.
	CountActiveByType(ctx context.Context, bondTypeIDs []string, userID string) (map[string]int, error)
	// ExistsNotRemovedByType indica si algún producto no borrado usa el tipo.
	ExistsNotRemovedByType(ctx context.Context, bondTypeID string) (bool, error)
	// EnsureIndexes crea los índices de la colección; es idempotente.
	EnsureIndexes(ctx context.Context) error
}

// BondBoundStatsPort agrega sobre bondBound en modo solo lectura.
type BondBoundStatsPort interface {
	// TopPurchasedObjectIDs devuelve los limit productos con más bonos no borrados de la agencia, de más
	// a menos compras.
	TopPurchasedObjectIDs(ctx context.Context, agencyID string, limit int) ([]string, error)
	// InvestorsByObject agrupa por usuario los bonos vigentes del producto, con su perfil; agencyID nil no
	// filtra por agencia.
	InvestorsByObject(ctx context.Context, objectID string, agencyID *string) ([]bondObject_domain.BondObjectInvestor, error)
}
