package bondBound_application

import (
	"context"
	"errors"

	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	bondHistory_ports "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/ports"
	bondObject_ports "github.com/futurapps/ms-blooming-go/internal/core/bondObject/ports"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	utils_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/adapters/utils"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

// errNoSession es un caso de uso de usuario llamado sin sesión; el middleware la garantiza, así que es un
// error interno.
var errNoSession = errors.New("bondBound: el contexto no tiene sesión de usuario")

// BondBoundApplication implementa los casos de uso de bondBound. Tx es el mismo que usa Balance, para que
// el cobro y el depósito se sumen a la transacción de la compra y de la venta. UserSettings da la cuenta
// destino de las ventas. BulkPurchaseMaxQuantity es el tope de bonos de una compra masiva y AdminEmails, los
// destinatarios del aviso de retiro al área administrativa.
type BondBoundApplication struct {
	Repository              bondBound_ports.BondBoundMongoccRepositoryPorts
	KeyCounter              bondBound_ports.KeyCounterPort
	BondObjects             bondObject_ports.BondObjectApplicationPorts
	UserSettings            bondUserSetting_ports.BondUserSettingApplicationPorts
	Balance                 shared_ports.BalancePort
	History                 bondHistory_ports.BondHistoryPort
	Tx                      shared_ports.TransactionRunnerPort
	Utils                   utils_adapters.UtilsPorts
	Notices                 shared_ports.NoticePort
	Notifications           shared_ports.NotificationPort
	Profiles                shared_ports.ProfileReader
	BulkPurchaseMaxQuantity int
	AdminEmails             []string
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
