package bondBound_application_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	bondBound_application "github.com/futurapps/ms-blooming-go/internal/core/bondBound/application"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondHistory_ports "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/ports"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	bondObject_ports "github.com/futurapps/ms-blooming-go/internal/core/bondObject/ports"
	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	utils_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/adapters/utils"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

const (
	agencyA     = "agencyAAAAAAAAAAA"
	agencyB     = "agencyBBBBBBBBBBB"
	ownerID     = "user1AAAAAAAAAAAA"
	otherUserID = "user2AAAAAAAAAAAA"
	adminID     = "adminAAAAAAAAAAAA"
	boundID     = "5a6b7c8d-9e0f-4a1b-8c2d-3e4f5a6b7c8d"
	missingID   = "0e1f2a3b-4c5d-4e6f-8a7b-9c0d1e2f3a4b"
	accountID   = "9d8c7b6a-5f4e-4d3c-8b2a-1f0e9d8c7b6a"
	objectID    = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"
	typeID      = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"

	// legacyBoundID es un _id Meteor de 17 caracteres.
	legacyBoundID = "Kp3Wq8Zr5Tn2Yx7Lm"

	// maxQuantity es el tope de la compra masiva en los tests.
	maxQuantity = 5
)

// boundCreatedAt es la fecha de compra de los bonos guardados.
var boundCreatedAt = time.Date(2026, 1, 1, 15, 0, 0, 0, time.UTC)

// statusChange es una llamada a UpdateStatusIf: el bono y el cambio pedido.
type statusChange struct {
	id     string
	change bondBound_ports.StatusChange
}

// fakeRepository guarda los bonos en memoria y registra las lecturas, los inserts y los cambios de estado.
type fakeRepository struct {
	bondBound_ports.BondBoundMongoccRepositoryPorts

	bounds map[string]*bondBound_domain.BondBound
	err    error

	reads    []string
	inserted []bondBound_domain.BondBound
	changes  []statusChange
}

// FindOneByID devuelve err si está puesto, nil si el bono no existe o está borrado, y si no una copia.
func (r *fakeRepository) FindOneByID(_ context.Context, id string) (*bondBound_domain.BondBound, error) {
	r.reads = append(r.reads, id)
	if r.err != nil {
		return nil, r.err
	}
	b, ok := r.bounds[id]
	if !ok || b.IsRemove {
		return nil, nil
	}
	found := *b
	return &found, nil
}

func (r *fakeRepository) InsertMany(_ context.Context, bounds []*bondBound_domain.BondBound) error {
	for _, b := range bounds {
		r.inserted = append(r.inserted, *b)
	}
	return nil
}

// UpdateStatusIf aplica c solo si el bono existe, no está borrado, sigue en c.From y no tiene soldDate ni
// tookDate; devuelve si lo cambió.
func (r *fakeRepository) UpdateStatusIf(_ context.Context, id string, c bondBound_ports.StatusChange) (bool, error) {
	r.changes = append(r.changes, statusChange{id: id, change: c})
	b, ok := r.bounds[id]
	if !ok || b.IsRemove || b.StatusID != c.From || b.SoldDate != nil || b.TookDate != nil {
		return false, nil
	}
	b.StatusID = c.To
	b.SoldDate, b.TookDate = c.SoldDate, c.TookDate
	return true, nil
}

// fakeKeyCounter numera las claves por tipo en memoria y registra cuántas reserva cada llamada.
type fakeKeyCounter struct {
	bondBound_ports.KeyCounterPort

	seq      map[string]int
	reserved []int
}

// Reserve avanza el contador del tipo en n y devuelve el primer número reservado.
func (k *fakeKeyCounter) Reserve(_ context.Context, bondTypeID string, n int) (int, error) {
	k.reserved = append(k.reserved, n)
	start := k.seq[bondTypeID] + 1
	k.seq[bondTypeID] += n
	return start, nil
}

// fakeBondObjects conoce los productos de objects y registra las consultas.
type fakeBondObjects struct {
	bondObject_ports.BondObjectApplicationPorts

	objects map[string]*bondObject_domain.BondObject
	reads   []string
}

// FindVisibleForPurchase devuelve una copia del producto, o nil si no existe.
func (o *fakeBondObjects) FindVisibleForPurchase(_ context.Context, id string) (*bondObject_domain.BondObject, error) {
	o.reads = append(o.reads, id)
	obj, ok := o.objects[id]
	if !ok {
		return nil, nil
	}
	found := *obj
	return &found, nil
}

// GetByIDIncludingRemoved devuelve una copia del producto, o ErrBondObjectNotFound si no existe.
func (o *fakeBondObjects) GetByIDIncludingRemoved(_ context.Context, id string) (*bondObject_domain.BondObject, error) {
	o.reads = append(o.reads, id)
	obj, ok := o.objects[id]
	if !ok {
		return nil, bondObject_domain.ErrBondObjectNotFound
	}
	found := *obj
	return &found, nil
}

// fakeUserSettings guarda los settings por usuario en memoria y registra las lecturas.
type fakeUserSettings struct {
	bondUserSetting_ports.BondUserSettingApplicationPorts

	settings map[string]*bondUserSetting_domain.BondUserSetting
	reads    []string
}

// GetByUser devuelve una copia del setting del usuario, o nil si no tiene.
func (s *fakeUserSettings) GetByUser(_ context.Context, userID string) (*bondUserSetting_domain.BondUserSetting, error) {
	s.reads = append(s.reads, userID)
	setting, ok := s.settings[userID]
	if !ok {
		return nil, nil
	}
	found := *setting
	return &found, nil
}

// accountRead es una lectura de cuenta: la cuenta pedida y el usuario con el que se pidió.
type accountRead struct {
	accountID string
	userID    string
}

// fakeBalance guarda las cuentas en memoria y registra las lecturas, los retiros y los depósitos.
type fakeBalance struct {
	shared_ports.BalancePort

	accounts map[string]*shared_domain.BalanceAccount

	reads       []accountRead
	withdrawals []shared_domain.WithdrawParams
	deposits    []shared_domain.DepositParams
}

// GetAccount devuelve una copia de la cuenta, o nil si no existe o no es de userID.
func (b *fakeBalance) GetAccount(_ context.Context, accountID, userID string) (*shared_domain.BalanceAccount, error) {
	b.reads = append(b.reads, accountRead{accountID: accountID, userID: userID})
	a, ok := b.accounts[accountID]
	if !ok || a.UserID != userID {
		return nil, nil
	}
	found := *a
	return &found, nil
}

// Withdraw descuenta el monto de la cuenta y devuelve el movimiento, con una referencia por retiro.
func (b *fakeBalance) Withdraw(_ context.Context, p shared_domain.WithdrawParams) (*shared_domain.BalanceMovement, error) {
	b.withdrawals = append(b.withdrawals, p)
	a := b.accounts[p.AccountID]
	a.Available -= p.Amount
	return &shared_domain.BalanceMovement{
		DetailID:  fmt.Sprintf("detail-%d", len(b.withdrawals)),
		Reference: fmt.Sprintf("REF%03d", len(b.withdrawals)),
		Amount:    -p.Amount,
		Balance:   a.Available,
		Available: a.Available,
	}, nil
}

// Deposit suma el monto a la cuenta y devuelve el movimiento, con una referencia por depósito.
func (b *fakeBalance) Deposit(_ context.Context, p shared_domain.DepositParams) (*shared_domain.BalanceMovement, error) {
	b.deposits = append(b.deposits, p)
	a := b.accounts[p.AccountID]
	a.Available += p.Amount
	return &shared_domain.BalanceMovement{
		DetailID:  fmt.Sprintf("deposit-%d", len(b.deposits)),
		Reference: fmt.Sprintf("DEP%03d", len(b.deposits)),
		Amount:    p.Amount,
		Balance:   a.Available,
		Available: a.Available,
	}, nil
}

// fakeHistory registra los eventos.
type fakeHistory struct {
	bondHistory_ports.BondHistoryPort

	added []bondHistory_domain.AddParams
}

func (h *fakeHistory) Add(_ context.Context, p bondHistory_domain.AddParams) error {
	h.added = append(h.added, p)
	return nil
}

// fakeTx corre fn con el mismo ctx y cuenta las transacciones.
type fakeTx struct {
	shared_ports.TransactionRunnerPort

	runs int
}

func (tx *fakeTx) RunInTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	tx.runs++
	return fn(ctx)
}

// fakeUtils genera siempre el mismo texto.
type fakeUtils struct {
	utils_adapters.UtilsPorts
}

func (fakeUtils) RandomAlnum(n int) string {
	return strings.Repeat("x", n)
}

// fakeNotices registra los avisos.
type fakeNotices struct {
	shared_ports.NoticePort

	added []shared_domain.Notice
}

func (n *fakeNotices) Add(_ context.Context, notice shared_domain.Notice) {
	n.added = append(n.added, notice)
}

// notifyTimeout es lo máximo que wait espera las notificaciones.
const notifyTimeout = 2 * time.Second

// sentNotification es una notificación enviada: el grupo de destinatarios, su nombre y sus parámetros.
type sentNotification struct {
	group  string
	name   string
	params map[string]any
}

// fakeNotifications registra cada notificación; Send corre en una goroutine.
type fakeNotifications struct {
	shared_ports.NotificationPort

	mu   sync.Mutex
	sent []sentNotification
}

func (n *fakeNotifications) Send(_ context.Context, group, notification string, params map[string]any) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, sentNotification{group: group, name: notification, params: params})
}

// wait espera a que haya al menos count notificaciones y devuelve una copia; falla el test si no llegan.
func (n *fakeNotifications) wait(t *testing.T, count int) []sentNotification {
	t.Helper()
	deadline := time.Now().Add(notifyTimeout)
	for {
		n.mu.Lock()
		sent := slices.Clone(n.sent)
		n.mu.Unlock()
		if len(sent) >= count {
			return sent
		}
		if time.Now().After(deadline) {
			t.Fatalf("llegaron %d notificaciones, se esperaban %d: %+v", len(sent), count, sent)
		}
		time.Sleep(time.Millisecond)
	}
}

// fakeProfiles responde los perfiles conocidos, o err; sin perfiles los avisos usan el nombre de la sesión.
type fakeProfiles struct {
	shared_ports.ProfileReader

	profiles map[string]shared_domain.PublicProfile
	err      error
}

// GetPublic devuelve, en el orden pedido, los perfiles que conoce.
func (p *fakeProfiles) GetPublic(_ context.Context, ids ...string) ([]shared_domain.PublicProfile, error) {
	if p.err != nil {
		return nil, p.err
	}
	out := []shared_domain.PublicProfile{}
	for _, id := range ids {
		if profile, ok := p.profiles[id]; ok {
			out = append(out, profile)
		}
	}
	return out, nil
}

// fakes son las dependencias de la aplicación.
type fakes struct {
	repo          *fakeRepository
	keys          *fakeKeyCounter
	objects       *fakeBondObjects
	userSettings  *fakeUserSettings
	balance       *fakeBalance
	history       *fakeHistory
	tx            *fakeTx
	notices       *fakeNotices
	notifications *fakeNotifications
	profiles      *fakeProfiles
}

// newFakes arma las dependencias con los bonos dados, sin cuentas, productos ni settings.
func newFakes(bounds ...*bondBound_domain.BondBound) *fakes {
	repo := &fakeRepository{bounds: map[string]*bondBound_domain.BondBound{}}
	for _, b := range bounds {
		stored := *b
		repo.bounds[b.ID] = &stored
	}
	return &fakes{
		repo:          repo,
		keys:          &fakeKeyCounter{seq: map[string]int{}},
		objects:       &fakeBondObjects{objects: map[string]*bondObject_domain.BondObject{}},
		userSettings:  &fakeUserSettings{settings: map[string]*bondUserSetting_domain.BondUserSetting{}},
		balance:       &fakeBalance{accounts: map[string]*shared_domain.BalanceAccount{}},
		history:       &fakeHistory{},
		tx:            &fakeTx{},
		notices:       &fakeNotices{},
		notifications: &fakeNotifications{},
		profiles:      &fakeProfiles{profiles: map[string]shared_domain.PublicProfile{}},
	}
}

// purchaseFakes arma las dependencias con la cuenta y el producto dados; nil es que no existe.
func purchaseFakes(account *shared_domain.BalanceAccount, object *bondObject_domain.BondObject) *fakes {
	f := newFakes()
	if account != nil {
		stored := *account
		f.balance.accounts[account.ID] = &stored
	}
	if object != nil {
		stored := *object
		f.objects.objects[object.ID] = &stored
	}
	return f
}

// app arma la aplicación con los fakes.
func (f *fakes) app() *bondBound_application.BondBoundApplication {
	return &bondBound_application.BondBoundApplication{
		Repository:              f.repo,
		KeyCounter:              f.keys,
		BondObjects:             f.objects,
		UserSettings:            f.userSettings,
		Balance:                 f.balance,
		History:                 f.history,
		Tx:                      f.tx,
		Utils:                   fakeUtils{},
		Notices:                 f.notices,
		Notifications:           f.notifications,
		Profiles:                f.profiles,
		BulkPurchaseMaxQuantity: maxQuantity,
	}
}

// assertNothingWritten verifica que el caso de uso no abrió la transacción ni escribió nada.
func assertNothingWritten(t *testing.T, f *fakes) {
	t.Helper()
	if f.tx.runs != 0 || len(f.keys.reserved) != 0 || len(f.repo.inserted) != 0 || len(f.repo.changes) != 0 ||
		len(f.balance.withdrawals) != 0 || len(f.balance.deposits) != 0 || len(f.history.added) != 0 || len(f.notices.added) != 0 {
		t.Errorf("se escribió: %d transacciones, reservas %v, %d bonos, cambios de estado %+v, retiros %+v, depósitos %+v, %d historiales y %d avisos",
			f.tx.runs, f.keys.reserved, len(f.repo.inserted), f.repo.changes, f.balance.withdrawals, f.balance.deposits,
			len(f.history.added), len(f.notices.added))
	}
}

// withSession es un context con la sesión de userID en la agencia A y la zona utcDiff.
func withSession(userID string, utcDiff int) context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID:   userID,
		AgencyID: agencyA,
		UtcDiff:  session_domain.OptionalUtcDiff{Hours: utcDiff, Valid: true},
	})
}

// withAdmin es un context con la sesión de un admin de Blooming en la agencia A, en UTC-5.
func withAdmin() context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID:        adminID,
		AgencyID:      agencyA,
		AdminBlooming: true,
		UtcDiff:       session_domain.OptionalUtcDiff{Hours: -5, Valid: true},
	})
}

// storedBound es un bono iniciado de ownerID, comprado el 1 de enero a las 15:00 UTC.
func storedBound(id string) *bondBound_domain.BondBound {
	return &bondBound_domain.BondBound{
		ID:                        id,
		Key:                       "BP12-1",
		UserID:                    ownerID,
		AgencyID:                  agencyA,
		StatusID:                  bondBound_domain.StatusInitiated,
		Amount:                    1000,
		MaturityTimeInDays:        365,
		MaturityRewardsPercentage: 15,
		EarlyCashTerm: shared_domain.EarlyCashTerm{
			InitialPenaltyPercentage:    100,
			InflectionPoint:             30,
			InflectionPenaltyPercentage: 75,
			InitialDayForWithdraw:       10,
		},
		InitialDate:  boundCreatedAt,
		MaturityDate: boundCreatedAt.AddDate(0, 0, 365),
		CreatedAt:    boundCreatedAt,
		UpdatedAt:    boundCreatedAt,
	}
}

// boundOnDay es el bono guardado de boundID, del producto objectID, comprado hace days días a esta misma hora;
// en cualquier zona de la sesión lleva days días.
func boundOnDay(days int) *bondBound_domain.BondBound {
	b := storedBound(boundID)
	b.BondObjectID = objectID
	b.InitialDate = time.Now().UTC().AddDate(0, 0, -days)
	b.MaturityDate = b.InitialDate.AddDate(0, 0, b.MaturityTimeInDays)
	return b
}

// ownAccount es una cuenta activa de ownerID en la agencia A con 5000 disponibles.
func ownAccount() *shared_domain.BalanceAccount {
	return &shared_domain.BalanceAccount{
		ID:        accountID,
		UserID:    ownerID,
		AgencyID:  agencyA,
		StatusID:  shared_domain.BalanceAccountStatusActive,
		Available: 5000,
		Number:    "0102030405060708",
	}
}

// purchasableObject es un producto activo y público de la agencia B, sin ventana de venta, de 1000 y 365
// días.
func purchasableObject() *bondObject_domain.BondObject {
	return &bondObject_domain.BondObject{
		ID:                        objectID,
		Name:                      "Bono Plus 12M",
		AgencyID:                  shared_domain.AgencyIDs{agencyB},
		Key:                       "BP12",
		BondTypeID:                typeID,
		Amount:                    1000,
		AuthorizedUserIDs:         []string{},
		MaturityTimeInDays:        365,
		MaturityRewardsPercentage: 15,
		Active:                    true,
		Available:                 true,
		EarlyCashTerm: shared_domain.EarlyCashTerm{
			InitialPenaltyPercentage:    100,
			InflectionPoint:             90,
			InflectionPenaltyPercentage: 75,
			InitialDayForWithdraw:       30,
		},
		CreatedAt: boundCreatedAt,
		UpdatedAt: boundCreatedAt,
	}
}
