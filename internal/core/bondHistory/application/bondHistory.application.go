package bondHistory_application

import (
	bondHistory_ports "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/ports"
)

// BondHistoryApplication implementa los casos de uso de bondHistory.
type BondHistoryApplication struct {
	Repository bondHistory_ports.BondHistoryMongoccRepositoryPorts
}

var _ bondHistory_ports.BondHistoryPort = (*BondHistoryApplication)(nil)
