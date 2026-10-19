package bondHistory_ports

import (
	"context"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
)

// BondHistoryPort registra eventos en el historial; es lo único de bondHistory que usan los otros
// dominios. El usuario y la agencia salen del ctx, y el registro usa ese ctx: dentro de una transacción
// se suma a ella.
type BondHistoryPort interface {
	Add(ctx context.Context, p bondHistory_domain.AddParams) error
}

// BondHistoryApplicationPorts son los casos de uso de bondHistory.
type BondHistoryApplicationPorts interface {
	BondHistoryPort
	// ListByRelated devuelve el historial de un producto o de un bono comprado, del más reciente al más
	// antiguo y listo para mostrar. No valida permisos: lo hace quien llama.
	ListByRelated(ctx context.Context, typeID bondHistory_domain.HistoryType, relatedID string) ([]bondHistory_domain.HistoryEntry, error)
}

// BondHistoryMongoccRepositoryPorts es el repositorio de bondHistory.
type BondHistoryMongoccRepositoryPorts interface {
	InsertOne(ctx context.Context, h *bondHistory_domain.BondHistory) error
	// FindByRelated devuelve los registros no borrados de relatedID con ese tipo, por createdAt
	// descendente. agencyID nil no filtra por agencia.
	FindByRelated(ctx context.Context, typeID bondHistory_domain.HistoryType, relatedID string, agencyID *string) ([]*bondHistory_domain.BondHistory, error)
	// EnsureIndexes crea los índices de la colección; es idempotente.
	EnsureIndexes(ctx context.Context) error
}
