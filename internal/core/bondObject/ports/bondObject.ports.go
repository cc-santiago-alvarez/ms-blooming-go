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

	// List devuelve los productos no borrados del alcance de agencia de la sesión, por name. agencyIDs solo
	// filtra si la agencia de la sesión es la principal. Un agencyID inválido es ErrInvalidID.
	List(ctx context.Context, agencyIDs []string) ([]*bondObject_domain.BondObject, error)
	// ListGroupedByType agrupa por tipo los productos del alcance de la sesión; agencyIDs solo filtra si
	// la agencia de la sesión es la principal. Un agencyID inválido es ErrInvalidID.
	ListGroupedByType(ctx context.Context, agencyIDs []string) ([]BondObjectGroup, error)
	// GetByID devuelve el producto no borrado del alcance de agencia de la sesión con los perfiles públicos
	// de sus autorizados, o ErrBondObjectNotFound. Un id inválido es ErrInvalidID.
	GetByID(ctx context.Context, id string) (*bondObject_domain.BondObjectDetail, error)
	// GetByIDIncludingRemoved devuelve el producto aunque esté borrado, o ErrBondObjectNotFound. Un id
	// inválido es ErrInvalidID.
	GetByIDIncludingRemoved(ctx context.Context, id string) (*bondObject_domain.BondObject, error)
	// Create normaliza y valida p, guarda el producto y devuelve el creado. Fuera de la agencia principal, p
	// solo puede tener la agencia de la sesión; si no, es ErrForbidden. Un bondTypeId inexistente, borrado o
	// fuera del alcance de agencia es ErrBondObjectInvalidBondType.
	Create(ctx context.Context, p bondObject_domain.CreateParams) (*bondObject_domain.BondObject, error)
	// Update reemplaza los campos editables del producto del alcance de agencia de la sesión y lo devuelve
	// actualizado, o ErrBondObjectNotFound. Fuera de la agencia principal, el producto y p solo pueden tener
	// la agencia de la sesión; si no, es ErrForbidden. Un bondTypeId inexistente, borrado o fuera del
	// alcance de agencia es ErrBondObjectInvalidBondType.
	Update(ctx context.Context, id string, p bondObject_domain.UpdateParams) (*bondObject_domain.BondObject, error)
	// UpdateKeys cambia la clave de los productos no borrados de la lista; fuera de la agencia principal,
	// solo la de los que son únicamente de la agencia de la sesión.
	UpdateKeys(ctx context.Context, p bondObject_domain.UpdateKeysParams) (bondObject_domain.UpdateKeysResult, error)
	// Remove borra de forma lógica el producto del alcance de agencia de la sesión, o devuelve
	// ErrBondObjectNotFound. Fuera de la agencia principal, si también es de otras agencias devuelve
	// ErrForbidden. Un id inválido es ErrInvalidID.
	Remove(ctx context.Context, id string) error
	// ListInvestors devuelve los usuarios con bonos vigentes del producto, en el alcance de la sesión. Un
	// id inválido es ErrInvalidID.
	ListInvestors(ctx context.Context, id string) ([]bondObject_domain.BondObjectInvestor, error)
	// ListByType devuelve los productos activos del tipo que ve la sesión, con su fecha de maduración. Un
	// bondTypeId inválido es ErrInvalidID.
	ListByType(ctx context.Context, bondTypeID string) ([]bondObject_domain.BondObjectForInvestor, error)
	// MostPurchased devuelve los productos más comprados en la agencia de la sesión que ella puede ver;
	// un privado donde no está autorizada no aparece ni se reemplaza.
	MostPurchased(ctx context.Context) ([]*bondObject_domain.BondObject, error)
	// Latest devuelve los últimos productos publicados que ve la sesión.
	Latest(ctx context.Context) ([]*bondObject_domain.BondObject, error)
	// ChartPreview normaliza y valida los términos de un producto sin guardar y devuelve sus guías y, con
	// simulation, su simulación diaria.
	ChartPreview(ctx context.Context, p bondObject_domain.ChartPreviewParams) (bondObject_domain.BondObjectChart, error)
	// ChartByID devuelve las guías del producto y, con simulation, su simulación diaria. Si la sesión no
	// lo ve, es ErrBondObjectNotFound; un admin ve cualquiera no borrado de su alcance de agencia. Un id
	// inválido es ErrInvalidID.
	ChartByID(ctx context.Context, id string, simulation bool) (bondObject_domain.BondObjectChart, error)
	// FindVisibleForPurchase devuelve el producto no borrado, de cualquier agencia, o nil si no existe.
	// Un id inválido es ErrInvalidID.
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
	// UpdateKeys cambia key y updatedAt de los productos no borrados de ids. agencyID nil no filtra por
	// agencia; si no, solo cambia los productos que son únicamente de esa agencia.
	UpdateKeys(ctx context.Context, ids []string, key string, agencyID *string) (bondObject_domain.UpdateKeysResult, error)
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
