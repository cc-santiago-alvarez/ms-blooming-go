package bondBound_application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	bondType_ports "github.com/futurapps/ms-blooming-go/internal/core/bondType/ports"
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

// repoQuery es una consulta al repositorio: el método y los argumentos con que se llamó.
type repoQuery struct {
	method        string
	id            string
	agencyID      *string
	userID        string
	bondObjectID  string
	filters       bondBound_domain.ListFilters
	reportFilters bondBound_domain.ReportFilters
	kpiFilters    bondBound_domain.KpiFilters
	statusIDs     []bondBound_domain.BondBoundStatus
	page          shared_domain.Pagination
	utcDiff       int
	limit         int
	now           time.Time
	afterID       string
}

// fakeRepository guarda los bonos en memoria y registra las lecturas, los inserts, los cambios de estado, las
// consultas y las bajas. Los listados responden lo configurado; el filtro de cada consulta es del adapter.
type fakeRepository struct {
	bondBound_ports.BondBoundMongoccRepositoryPorts

	bounds map[string]*bondBound_domain.BondBound
	err    error

	adminPage     shared_domain.Paginated[bondBound_ports.BondBoundAdminRow]
	minePage      shared_domain.Paginated[bondBound_ports.BondBoundWithObject]
	recent        []bondBound_ports.BondBoundWithObject
	listed        []*bondBound_domain.BondBound
	timeline      []bondBound_ports.TimelineItem
	detailObject  bondObject_domain.BondObject
	detailType    bondType_domain.BondType
	purchasedRows []bondBound_ports.PurchasedReportRow
	summaryRows   []bondBound_ports.SummaryReportRow
	statusCount   bondBound_ports.StatusCount
	// deleteLost simula que otra baja borró el bono entre la lectura y la escritura.
	deleteLost bool
	// statusErrs es el error que UpdateStatusIf devuelve para cada bono, sin cambiarlo.
	statusErrs map[string]error

	reads    []string
	inserted []bondBound_domain.BondBound
	changes  []statusChange
	queries  []repoQuery
	deleted  []string
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

// UpdateStatusIf devuelve el error de statusErrs del bono, si tiene; si no, aplica c solo si el bono existe,
// no está borrado, sigue en c.From y no tiene soldDate ni tookDate, y devuelve si lo cambió. Con maturedAt
// marca el bono como madurado.
func (r *fakeRepository) UpdateStatusIf(_ context.Context, id string, c bondBound_ports.StatusChange) (bool, error) {
	r.changes = append(r.changes, statusChange{id: id, change: c})
	if err := r.statusErrs[id]; err != nil {
		return false, err
	}
	b, ok := r.bounds[id]
	if !ok || b.IsRemove || b.StatusID != c.From || b.SoldDate != nil || b.TookDate != nil {
		return false, nil
	}
	b.StatusID = c.To
	b.SoldDate, b.TookDate = c.SoldDate, c.TookDate
	if c.MaturedAt != nil {
		b.Matured = true
	}
	return true, nil
}

// FindMaturable filtra los bonos en memoria igual que el adapter: no borrados, iniciados, sin soldDate ni
// tookDate y vencidos hasta now, con _id mayor que afterID, por _id ascendente y hasta limit; devuelve copias.
func (r *fakeRepository) FindMaturable(_ context.Context, now time.Time, limit int, afterID string) ([]*bondBound_domain.BondBound, error) {
	r.queries = append(r.queries, repoQuery{method: "FindMaturable", now: now, limit: limit, afterID: afterID})
	if r.err != nil {
		return nil, r.err
	}
	var found []*bondBound_domain.BondBound
	for _, b := range r.bounds {
		if b.IsRemove || b.StatusID != bondBound_domain.StatusInitiated || b.SoldDate != nil || b.TookDate != nil ||
			b.MaturityDate.After(now) || b.ID <= afterID {
			continue
		}
		copied := *b
		found = append(found, &copied)
	}
	slices.SortFunc(found, func(a, b *bondBound_domain.BondBound) int { return strings.Compare(a.ID, b.ID) })
	if limit > 0 && len(found) > limit {
		found = found[:limit]
	}
	return found, nil
}

func (r *fakeRepository) ListAdmin(_ context.Context, agencyID *string, f bondBound_domain.ListFilters, p shared_domain.Pagination, utcDiff int) (shared_domain.Paginated[bondBound_ports.BondBoundAdminRow], error) {
	r.queries = append(r.queries, repoQuery{method: "ListAdmin", agencyID: agencyID, filters: f, page: p, utcDiff: utcDiff})
	if r.err != nil {
		return shared_domain.Paginated[bondBound_ports.BondBoundAdminRow]{}, r.err
	}
	return r.adminPage, nil
}

func (r *fakeRepository) ListByUserAndObject(_ context.Context, userID, bondObjectID string, agencyID *string) ([]*bondBound_domain.BondBound, error) {
	r.queries = append(r.queries, repoQuery{method: "ListByUserAndObject", userID: userID, bondObjectID: bondObjectID, agencyID: agencyID})
	if r.err != nil {
		return nil, r.err
	}
	return r.listed, nil
}

func (r *fakeRepository) ListMine(_ context.Context, userID, agencyID string, statusIDs []bondBound_domain.BondBoundStatus, p shared_domain.Pagination) (shared_domain.Paginated[bondBound_ports.BondBoundWithObject], error) {
	r.queries = append(r.queries, repoQuery{method: "ListMine", userID: userID, agencyID: &agencyID, statusIDs: statusIDs, page: p})
	if r.err != nil {
		return shared_domain.Paginated[bondBound_ports.BondBoundWithObject]{}, r.err
	}
	return r.minePage, nil
}

func (r *fakeRepository) ListRecent(_ context.Context, userID string, limit int) ([]bondBound_ports.BondBoundWithObject, error) {
	r.queries = append(r.queries, repoQuery{method: "ListRecent", userID: userID, limit: limit})
	if r.err != nil {
		return nil, r.err
	}
	return r.recent, nil
}

func (r *fakeRepository) ListByObject(_ context.Context, bondObjectID string, agencyID *string) ([]*bondBound_domain.BondBound, error) {
	r.queries = append(r.queries, repoQuery{method: "ListByObject", bondObjectID: bondObjectID, agencyID: agencyID})
	if r.err != nil {
		return nil, r.err
	}
	return r.listed, nil
}

// FindDetails devuelve err si está puesto, nil si el bono no existe o está borrado, y si no una copia con el
// producto y el tipo configurados.
func (r *fakeRepository) FindDetails(_ context.Context, id string) (*bondBound_ports.BondBoundDetailRow, error) {
	r.queries = append(r.queries, repoQuery{method: "FindDetails", id: id})
	if r.err != nil {
		return nil, r.err
	}
	b, ok := r.bounds[id]
	if !ok || b.IsRemove {
		return nil, nil
	}
	return &bondBound_ports.BondBoundDetailRow{Bound: *b, BondObject: r.detailObject, BondType: r.detailType}, nil
}

func (r *fakeRepository) Timeline(_ context.Context, userID string, utcDiff int) ([]bondBound_ports.TimelineItem, error) {
	r.queries = append(r.queries, repoQuery{method: "Timeline", userID: userID, utcDiff: utcDiff})
	if r.err != nil {
		return nil, r.err
	}
	return r.timeline, nil
}

func (r *fakeRepository) ReportPurchasedRows(_ context.Context, agencyID string, f bondBound_domain.ReportFilters, utcDiff int) ([]bondBound_ports.PurchasedReportRow, error) {
	r.queries = append(r.queries, repoQuery{method: "ReportPurchasedRows", agencyID: &agencyID, reportFilters: f, utcDiff: utcDiff})
	if r.err != nil {
		return nil, r.err
	}
	return r.purchasedRows, nil
}

func (r *fakeRepository) ReportSummaryRows(_ context.Context, agencyID string, f bondBound_domain.ReportFilters) ([]bondBound_ports.SummaryReportRow, error) {
	r.queries = append(r.queries, repoQuery{method: "ReportSummaryRows", agencyID: &agencyID, reportFilters: f})
	if r.err != nil {
		return nil, r.err
	}
	return r.summaryRows, nil
}

func (r *fakeRepository) CountByStatus(_ context.Context, agencyID *string, f bondBound_domain.KpiFilters, utcDiff int) (bondBound_ports.StatusCount, error) {
	r.queries = append(r.queries, repoQuery{method: "CountByStatus", agencyID: agencyID, kpiFilters: f, utcDiff: utcDiff})
	if r.err != nil {
		return bondBound_ports.StatusCount{}, r.err
	}
	return r.statusCount, nil
}

// SoftDelete marca el bono como borrado; devuelve false si no existe, ya estaba borrado o deleteLost está
// puesto.
func (r *fakeRepository) SoftDelete(_ context.Context, id string) (bool, error) {
	r.deleted = append(r.deleted, id)
	b, ok := r.bounds[id]
	if !ok || b.IsRemove || r.deleteLost {
		return false, nil
	}
	b.IsRemove = true
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

// fakeBondTypes conoce los tipos de types y registra los ids de cada consulta.
type fakeBondTypes struct {
	bondType_ports.BondTypeApplicationPorts

	types map[string]*bondType_domain.BondType

	requested [][]string
}

// GetManyByIDs devuelve, en el orden pedido, copias de los tipos que conoce.
func (bt *fakeBondTypes) GetManyByIDs(_ context.Context, ids []string) ([]*bondType_domain.BondType, error) {
	bt.requested = append(bt.requested, ids)
	var found []*bondType_domain.BondType
	for _, id := range ids {
		if t, ok := bt.types[id]; ok {
			copied := *t
			found = append(found, &copied)
		}
	}
	return found, nil
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

// historyQuery es una consulta del historial: el tipo y el relatedId.
type historyQuery struct {
	typeID    bondHistory_domain.HistoryType
	relatedID string
}

// fakeHistory registra los eventos y las consultas, y responde entries a cada consulta.
type fakeHistory struct {
	bondHistory_ports.BondHistoryApplicationPorts

	entries []bondHistory_domain.HistoryEntry

	added  []bondHistory_domain.AddParams
	listed []historyQuery
}

func (h *fakeHistory) Add(_ context.Context, p bondHistory_domain.AddParams) error {
	h.added = append(h.added, p)
	return nil
}

func (h *fakeHistory) ListByRelated(_ context.Context, typeID bondHistory_domain.HistoryType, relatedID string) ([]bondHistory_domain.HistoryEntry, error) {
	h.listed = append(h.listed, historyQuery{typeID: typeID, relatedID: relatedID})
	return h.entries, nil
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

// lockCall es una toma del lock: el nombre y la duración pedidos.
type lockCall struct {
	name string
	ttl  time.Duration
}

// fakeJobLock siempre da el lock y registra las tomas y las liberaciones.
type fakeJobLock struct {
	shared_ports.JobLockPort

	acquired []lockCall
	released []string
}

func (l *fakeJobLock) Acquire(_ context.Context, name string, ttl time.Duration) (bool, error) {
	l.acquired = append(l.acquired, lockCall{name: name, ttl: ttl})
	return true, nil
}

func (l *fakeJobLock) Release(_ context.Context, name string) {
	l.released = append(l.released, name)
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

// fakeProfiles responde los perfiles conocidos, o err, y registra los ids de cada consulta; sin perfiles los
// avisos usan el nombre de la sesión.
type fakeProfiles struct {
	shared_ports.ProfileReader

	profiles map[string]shared_domain.PublicProfile
	err      error

	requested [][]string
}

// GetPublic devuelve, en el orden pedido, los perfiles que conoce.
func (p *fakeProfiles) GetPublic(_ context.Context, ids ...string) ([]shared_domain.PublicProfile, error) {
	p.requested = append(p.requested, ids)
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

// fakeAgencies responde el símbolo de moneda configurado, o err, y registra las agencias que se leyeron.
type fakeAgencies struct {
	symbol string
	err    error

	read []string
}

func (a *fakeAgencies) SymbolCurrency(_ context.Context, agencyID string) (string, error) {
	a.read = append(a.read, agencyID)
	if a.err != nil {
		return "", a.err
	}
	return a.symbol, nil
}

// exportCall es un reporte pedido a ms-exporter, con el payload como JSON.
type exportCall struct {
	report  string
	payload []byte
}

// fakeExporter responde el archivo configurado, o err, y registra cada reporte pedido con el payload tal como
// lo enviaría el cliente real.
type fakeExporter struct {
	file []byte
	err  error

	calls []exportCall
}

func (e *fakeExporter) Excel(_ context.Context, report string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	e.calls = append(e.calls, exportCall{report: report, payload: body})
	if e.err != nil {
		return nil, e.err
	}
	return e.file, nil
}

// fakes son las dependencias de la aplicación.
type fakes struct {
	repo          *fakeRepository
	keys          *fakeKeyCounter
	objects       *fakeBondObjects
	types         *fakeBondTypes
	userSettings  *fakeUserSettings
	balance       *fakeBalance
	history       *fakeHistory
	tx            *fakeTx
	notices       *fakeNotices
	notifications *fakeNotifications
	profiles      *fakeProfiles
	agencies      *fakeAgencies
	exporter      *fakeExporter
	lock          *fakeJobLock
}

// newFakes arma las dependencias con los bonos dados, sin cuentas, productos, tipos ni settings. La moneda de
// las agencias es "Pesos $" y ms-exporter responde un archivo de dos bytes.
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
		types:         &fakeBondTypes{types: map[string]*bondType_domain.BondType{}},
		userSettings:  &fakeUserSettings{settings: map[string]*bondUserSetting_domain.BondUserSetting{}},
		balance:       &fakeBalance{accounts: map[string]*shared_domain.BalanceAccount{}},
		history:       &fakeHistory{},
		tx:            &fakeTx{},
		notices:       &fakeNotices{},
		notifications: &fakeNotifications{},
		profiles:      &fakeProfiles{profiles: map[string]shared_domain.PublicProfile{}},
		agencies:      &fakeAgencies{symbol: "Pesos $"},
		exporter:      &fakeExporter{file: []byte{80, 75}},
		lock:          &fakeJobLock{},
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
		BondTypes:               f.types,
		UserSettings:            f.userSettings,
		Balance:                 f.balance,
		History:                 f.history,
		Tx:                      f.tx,
		Utils:                   fakeUtils{},
		Notices:                 f.notices,
		Notifications:           f.notifications,
		Profiles:                f.profiles,
		Agencies:                f.agencies,
		Exporter:                f.exporter,
		JobLock:                 f.lock,
		BulkPurchaseMaxQuantity: maxQuantity,
	}
}

// captureLog redirige el log estándar a un buffer mientras dura el test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(previous) })
	return &buf
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

// withSession es un context con la sesión de userID, con su token, en la agencia A y la zona utcDiff.
func withSession(userID string, utcDiff int) context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID:   userID,
		Token:    "tok-" + userID,
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

// withMainAdmin es un context con la sesión de un admin de Blooming en la agencia principal A, que ve todas
// las agencias, en UTC-5.
func withMainAdmin() context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID:        adminID,
		AgencyID:      agencyA,
		Agency:        session_domain.SessionAgency{ID: agencyA, IsMain: true},
		AdminBlooming: true,
		UtcDiff:       session_domain.OptionalUtcDiff{Hours: -5, Valid: true},
	})
}

// agency es el alcance de agencia que recibe el repositorio.
func agency(id string) *string {
	return &id
}

// assertNoQueries verifica que el caso de uso no leyó, consultó ni borró nada en el repositorio.
func assertNoQueries(t *testing.T, f *fakes) {
	t.Helper()
	if len(f.repo.reads) != 0 || len(f.repo.queries) != 0 || len(f.repo.deleted) != 0 {
		t.Errorf("el repositorio recibió lecturas %v, consultas %+v y bajas %v, se esperaba ninguna",
			f.repo.reads, f.repo.queries, f.repo.deleted)
	}
}

// assertExported verifica que se leyó una vez la moneda de la agencia A y se pidió una vez report a
// ms-exporter, con el body {currency, bondBounds} del legado: la moneda y rows.
func assertExported(t *testing.T, f *fakes, report string, rows any) {
	t.Helper()
	if !slices.Equal(f.agencies.read, []string{agencyA}) {
		t.Errorf("se leyó la moneda de %v, se esperaba [%s]", f.agencies.read, agencyA)
	}
	if len(f.exporter.calls) != 1 || f.exporter.calls[0].report != report {
		t.Fatalf("ms-exporter recibió %+v, se esperaba una llamada a %s", f.exporter.calls, report)
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(f.exporter.calls[0].payload, &body); err != nil {
		t.Fatalf("el payload no es un objeto JSON: %v", err)
	}
	wantRows, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("json.Marshal(rows): %v", err)
	}
	if len(body) != 2 || string(body["currency"]) != `"Pesos $"` || string(body["bondBounds"]) != string(wantRows) {
		t.Errorf("payload %s, se esperaba {\"currency\":\"Pesos $\",\"bondBounds\":%s}", f.exporter.calls[0].payload, wantRows)
	}
}

// assertNotExported verifica que no se leyó la moneda ni se pidió nada a ms-exporter.
func assertNotExported(t *testing.T, f *fakes) {
	t.Helper()
	if len(f.agencies.read) != 0 || len(f.exporter.calls) != 0 {
		t.Errorf("se leyó la moneda de %v y ms-exporter recibió %+v, se esperaba nada", f.agencies.read, f.exporter.calls)
	}
}

// assertReportName verifica que name es prefix, una hora YYYY-MM-DD_HHmmss entre before y after en la zona
// utcDiff, y .xlsx.
func assertReportName(t *testing.T, name, prefix string, utcDiff int, before, after time.Time) {
	t.Helper()
	stamp, hasPrefix := strings.CutPrefix(name, prefix)
	stamp, hasExt := strings.CutSuffix(stamp, ".xlsx")
	at, err := time.ParseInLocation("2006-01-02_150405", stamp, time.FixedZone("", utcDiff*3600))
	if !hasPrefix || !hasExt || err != nil || at.Before(before.Truncate(time.Second)) || at.After(after) {
		t.Errorf("nombre %q, se esperaba %s<hora entre %s y %s en UTC%+d>.xlsx", name, prefix,
			before.Format(time.RFC3339), after.Format(time.RFC3339), utcDiff)
	}
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
