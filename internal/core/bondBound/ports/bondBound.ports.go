package bondBound_ports

import (
	"context"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// StatusChange pasa un bono de From a To. Las fechas nil no se escriben; MaturedAt también escribe
// matured: true.
type StatusChange struct {
	From      bondBound_domain.BondBoundStatus
	To        bondBound_domain.BondBoundStatus
	SoldDate  *time.Time
	TookDate  *time.Time
	MaturedAt *time.Time
}

// BondBoundApplicationPorts son los casos de uso de bondBound.
type BondBoundApplicationPorts interface {
	// Purchase compra un bono del producto con una cuenta del usuario de la sesión: valida y, en una
	// transacción, crea el bono, cobra y registra el historial; después notifica. Devuelve el bono creado y
	// el saldo que le queda a la cuenta.
	Purchase(ctx context.Context, p bondBound_domain.PurchaseParams) (bondBound_domain.PurchaseResult, error)
	// BulkPurchase compra quantity bonos del producto como Purchase, todos en una sola transacción.
	BulkPurchase(ctx context.Context, p bondBound_domain.BulkPurchaseParams) (bondBound_domain.BulkPurchaseResult, error)
	// ChartGuides devuelve las guías del bono a partir de su createdAt, en la zona de la sesión. Si no
	// existe es ErrBondBoundNotFound, y si no es del usuario ni es admin, ErrForbidden. Un id inválido es
	// ErrInvalidID.
	ChartGuides(ctx context.Context, id string) (bondBound_domain.BondBoundChart, error)
	// PreSale cotiza la venta anticipada o el canje del bono a hoy, en la zona de la sesión. Si no existe es
	// ErrBondBoundNotFound; si no es del usuario ni es admin, ErrForbidden; si su estado no se puede vender,
	// el error de SaleTransitionFor, y si la cotización falla, el de QuoteSale. Un id inválido es
	// ErrInvalidID.
	PreSale(ctx context.Context, id string) (bondBound_domain.SaleProjection, error)
	// Sale vende el bono iniciado o canjea el madurado del usuario de la sesión: en una transacción cambia
	// su estado, acredita la cotización en la cuenta destino y registra el historial. Si no existe es
	// ErrBondBoundNotFound y si es ajeno, ErrForbidden, aunque la sesión sea admin. Si su estado no se puede
	// vender es el error de SaleTransitionFor, y si la cotización falla, el de QuoteSale. Sin cuenta destino
	// es ErrUserSettingAccountRequired, y si no existe o no es suya, ErrAccountNotFound. Si otra venta lo
	// cambió antes, ErrBondAlreadySold. Un id inválido es ErrInvalidID.
	Sale(ctx context.Context, id string) (bondBound_domain.SaleResult, error)
}

// BondBoundMongoccRepositoryPorts es el repositorio de bondBound. Las escrituras usan el ctx recibido, así
// que dentro de una transacción se suman a ella.
type BondBoundMongoccRepositoryPorts interface {
	InsertMany(ctx context.Context, bounds []*bondBound_domain.BondBound) error
	// FindOneByID devuelve nil, nil si el bono no existe o está borrado.
	FindOneByID(ctx context.Context, id string) (*bondBound_domain.BondBound, error)
	// UpdateStatusIf aplica c y actualiza updatedAt solo si el bono no está borrado, sigue en c.From y no
	// tiene soldDate ni tookDate; devuelve false si no modificó nada.
	UpdateStatusIf(ctx context.Context, id string, c StatusChange) (bool, error)
	// EnsureIndexes crea los índices de la colección; es idempotente.
	EnsureIndexes(ctx context.Context) error
}

// KeyCounterPort numera las claves de los bonos por tipo.
type KeyCounterPort interface {
	// Reserve reserva n números de clave consecutivos del tipo y devuelve el primero; si el contador no
	// existe, lo crea desde 0. Usa el ctx recibido: si la transacción revierte, el contador no avanza.
	Reserve(ctx context.Context, bondTypeID string, n int) (start int, err error)
}
