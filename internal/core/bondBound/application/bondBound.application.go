package bondBound_application

import (
	"context"
	"errors"

	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	bondHistory_ports "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/ports"
	bondObject_ports "github.com/futurapps/ms-blooming-go/internal/core/bondObject/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	utils_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/adapters/utils"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

// errNoSession es un caso de uso de usuario llamado sin sesión; el middleware la garantiza, así que es un
// error interno.
var errNoSession = errors.New("bondBound: el contexto no tiene sesión de usuario")

// BondBoundApplication implementa los casos de uso de bondBound. Tx es el mismo que usa Balance, para que
// el cobro se sume a la transacción de la compra. BulkPurchaseMaxQuantity es el tope de bonos de una compra
// masiva.
type BondBoundApplication struct {
	Repository              bondBound_ports.BondBoundMongoccRepositoryPorts
	KeyCounter              bondBound_ports.KeyCounterPort
	BondObjects             bondObject_ports.BondObjectApplicationPorts
	Balance                 shared_ports.BalancePort
	History                 bondHistory_ports.BondHistoryPort
	Tx                      shared_ports.TransactionRunnerPort
	Utils                   utils_adapters.UtilsPorts
	BulkPurchaseMaxQuantity int
}

var _ bondBound_ports.BondBoundApplicationPorts = (*BondBoundApplication)(nil)

// sessionUser devuelve la sesión del context, o errNoSession si no hay o no tiene usuario.
func sessionUser(ctx context.Context) (*session_domain.UserSession, error) {
	session, ok := session_domain.GetUserSession(ctx)
	if !ok || session.UserID == "" {
		return nil, errNoSession
	}
	return session, nil
}
