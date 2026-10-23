package bondBound_application

import (
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
)

// BondBoundApplication implementa los casos de uso de bondBound.
type BondBoundApplication struct {
	Repository bondBound_ports.BondBoundMongoccRepositoryPorts
}
