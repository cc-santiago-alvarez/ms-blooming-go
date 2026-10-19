package bondHistory_application

import (
	bondHistory_ports "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/ports"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

// BondHistoryApplication implementa los casos de uso de bondHistory.
type BondHistoryApplication struct {
	Repository bondHistory_ports.BondHistoryMongoccRepositoryPorts
	Profiles   shared_ports.ProfileReader
}

var _ bondHistory_ports.BondHistoryApplicationPorts = (*BondHistoryApplication)(nil)
