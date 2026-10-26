package bondBound_ports

import (
	"context"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
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

// ProfileName es el nombre del dueño de un bono.
type ProfileName struct {
	FirstName string `json:"first_name" bson:"first_name"`
	LastName  string `json:"last_name" bson:"last_name"`
}

// BondObjectRef es el producto de un bono: su nombre y, si la consulta los trae, sus términos de retiro.
type BondObjectRef struct {
	Name          string                       `json:"name" bson:"name"`
	EarlyCashTerm *shared_domain.EarlyCashTerm `json:"earlyCashTerm,omitempty" bson:"earlyCashTerm"`
}

// BondTypeRef es el tipo de un bono: su nombre.
type BondTypeRef struct {
	Name string `json:"name" bson:"name"`
}

// BondBoundAdminRow es un bono del listado de administración con su dueño, su producto y su tipo, y el mes
// y el año de su compra en la zona de la sesión. Profile es nil si el dueño no tiene perfil.
type BondBoundAdminRow struct {
	Bound      bondBound_domain.BondBound `bson:"bound"`
	Profile    *ProfileName               `bson:"profile"`
	BondObject BondObjectRef              `bson:"bondObject"`
	BondType   BondTypeRef                `bson:"bondType"`
	Month      int                        `bson:"month"`
	Year       int                        `bson:"year"`
}

// BondBoundWithObject es un bono con su producto; BondObject es nil si el producto no existe.
type BondBoundWithObject struct {
	Bound      bondBound_domain.BondBound `bson:"bound"`
	BondObject *BondObjectRef             `bson:"bondObject"`
}

// BondBoundDetailRow es un bono con su producto, sin los usuarios autorizados, y su tipo.
type BondBoundDetailRow struct {
	Bound      bondBound_domain.BondBound   `bson:"bound"`
	BondObject bondObject_domain.BondObject `bson:"bondObject"`
	BondType   bondType_domain.BondType     `bson:"bondType"`
}

// TimelineItem es una compra en la línea de tiempo: el producto con la clave del bono, y su fecha inicial en
// la zona de la sesión como inicio y fin.
type TimelineItem struct {
	ID    string `json:"_id" bson:"_id"`
	Text  string `json:"text" bson:"text"`
	Start string `json:"start" bson:"start"`
	End   string `json:"end" bson:"end"`
}

// PurchasedReportRow es un bono del reporte de comprados como lo recibe ms-exporter, con el nombre de su
// estado. Profile no sale si el dueño no tiene perfil.
type PurchasedReportRow struct {
	Key                       string        `json:"key"`
	Status                    string        `json:"status"`
	Amount                    float64       `json:"amount"`
	MaturityTimeInDays        int           `json:"maturityTimeInDays"`
	InitialDate               time.Time     `json:"initialDate"`
	MaturityDate              time.Time     `json:"maturityDate"`
	MaturityRewardsPercentage float64       `json:"maturityRewardsPercentage"`
	Profile                   *ProfileName  `json:"profile,omitempty"`
	BondType                  BondTypeRef   `json:"bondType"`
	BondObject                BondObjectRef `json:"bondObject"`
	CreatedAt                 time.Time     `json:"createdAt"`
}

// SummaryReportRow es el total comprado por un inversionista como lo recibe ms-exporter; Profile no sale si
// no tiene perfil.
type SummaryReportRow struct {
	UserID  string       `json:"_id" bson:"_id"`
	Total   float64      `json:"total" bson:"total"`
	Profile *ProfileName `json:"profile,omitempty" bson:"profile"`
}

// ReportFile es un reporte generado: el nombre del archivo, con su extensión, y su contenido.
type ReportFile struct {
	Name    string
	Content []byte
}

// StatusCount es el KPI de bonos por estado: la suma de los montos, la cantidad y el nombre del estado por el
// que se filtró, o "" si no se filtró.
type StatusCount struct {
	Total  float64 `json:"total" bson:"total"`
	Count  int64   `json:"count" bson:"count"`
	Status string  `json:"status" bson:"-"`
}

// BondBoundAdminItem es un bono del listado de administración, con los días y el monto acumulados a hoy.
// Profile no sale si el dueño no tiene perfil.
type BondBoundAdminItem struct {
	ID                        string                     `json:"_id"`
	Key                       string                     `json:"key"`
	Status                    bondBound_domain.StatusRef `json:"status"`
	Amount                    float64                    `json:"amount"`
	MaturityTimeInDays        int                        `json:"maturityTimeInDays"`
	InitialDate               time.Time                  `json:"initialDate"`
	MaturityRewardsPercentage float64                    `json:"maturityRewardsPercentage"`
	Profile                   *ProfileName               `json:"profile,omitempty"`
	BondType                  BondTypeRef                `json:"bondType"`
	BondObject                BondObjectRef              `json:"bondObject"`
	CreatedAt                 time.Time                  `json:"createdAt"`
	Month                     int                        `json:"month"`
	Year                      int                        `json:"year"`
	DaysPassed                int                        `json:"daysPassed"`
	AcumulateAmount           float64                    `json:"acumulateAmount"`
}

// BondBoundWithProfile es un bono con el perfil público de su dueño; Profile no sale si el dueño no tiene
// perfil.
type BondBoundWithProfile struct {
	bondBound_domain.BondBound
	Profile *shared_domain.PublicProfile `json:"profile,omitempty"`
}

// BondBoundMineItem es un bono del usuario de la sesión, con los días y el monto acumulados, si está
// creciendo o ya creció, y sus fechas inicial y de vencimiento como "D MMM YYYY". BondObject no sale si el
// producto no existe, y una fecha formateada no sale si la fecha falta.
type BondBoundMineItem struct {
	ID                        string                           `json:"_id"`
	Key                       string                           `json:"key"`
	StatusID                  bondBound_domain.BondBoundStatus `json:"statusId"`
	BondObjectID              string                           `json:"bondObjectId"`
	Amount                    float64                          `json:"amount"`
	InitialDate               time.Time                        `json:"initialDate"`
	MaturityDate              time.Time                        `json:"maturityDate"`
	MaturityRewardsPercentage float64                          `json:"maturityRewardsPercentage"`
	MaturityTimeInDays        int                              `json:"maturityTimeInDays"`
	EarlyCashTerm             shared_domain.EarlyCashTerm      `json:"earlyCashTerm"`
	BondObject                *BondObjectRef                   `json:"bondObject,omitempty"`
	DaysPassed                int                              `json:"daysPassed"`
	AcumulateAmount           float64                          `json:"acumulateAmount"`
	Growing                   bool                             `json:"growing"`
	Grew                      bool                             `json:"grew"`
	InitialDateFormatted      string                           `json:"initialDateFormatted,omitempty"`
	MaturityDateFormatted     string                           `json:"maturityDateFormatted,omitempty"`
}

// BondObjectPublic es el producto de un bono sin sus usuarios autorizados.
type BondObjectPublic struct {
	ID                        string                      `json:"_id"`
	Name                      string                      `json:"name"`
	Description               string                      `json:"description"`
	AgencyID                  shared_domain.AgencyIDs     `json:"agencyId"`
	Key                       string                      `json:"key"`
	BondTypeID                string                      `json:"bondTypeId"`
	Amount                    float64                     `json:"amount"`
	IsPrivateBond             bool                        `json:"isPrivateBond"`
	MaturityTimeInDays        int                         `json:"maturityTimeInDays"`
	MaturityRewardsPercentage float64                     `json:"maturityRewardsPercentage"`
	Active                    bool                        `json:"active"`
	Available                 bool                        `json:"available"`
	SaleInitDate              *time.Time                  `json:"saleInitDate"`
	SaleEndDate               *time.Time                  `json:"saleEndDate"`
	EarlyCashTerm             shared_domain.EarlyCashTerm `json:"earlyCashTerm"`
	CreatedAt                 time.Time                   `json:"createdAt"`
	UpdatedAt                 time.Time                   `json:"updatedAt"`
	IsRemove                  bool                        `json:"isRemove"`
}

// BondBoundDetail es el detalle de un bono: el bono sin sus ids de estado, dueño, producto y tipo, más el
// producto público, el tipo, el estado con su nombre y los días y el monto acumulados a hoy.
type BondBoundDetail struct {
	ID                        string                      `json:"_id"`
	Key                       string                      `json:"key"`
	Token                     string                      `json:"token"`
	AgencyID                  string                      `json:"agencyId"`
	Amount                    float64                     `json:"amount"`
	MaturityTimeInDays        int                         `json:"maturityTimeInDays"`
	MaturityRewardsPercentage float64                     `json:"maturityRewardsPercentage"`
	EarlyCashTerm             shared_domain.EarlyCashTerm `json:"earlyCashTerm"`
	InitialDate               time.Time                   `json:"initialDate"`
	MaturityDate              time.Time                   `json:"maturityDate"`
	SoldDate                  *time.Time                  `json:"soldDate,omitempty"`
	TookDate                  *time.Time                  `json:"tookDate,omitempty"`
	Matured                   bool                        `json:"matured,omitempty"`
	CreatedAt                 time.Time                   `json:"createdAt"`
	UpdatedAt                 time.Time                   `json:"updatedAt"`
	IsRemove                  bool                        `json:"isRemove"`
	BondObject                BondObjectPublic            `json:"bondObject"`
	BondType                  bondType_domain.BondType    `json:"bondType"`
	Status                    bondBound_domain.StatusRef  `json:"status"`
	DaysPassed                int                         `json:"daysPassed"`
	AcumulateAmount           float64                     `json:"acumulateAmount"`
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
	// HistoryOf devuelve el historial del bono en la agencia de la sesión, del más reciente al más antiguo y
	// listo para mostrar. Si no existe es ErrBondBoundNotFound, y si no es del usuario ni es admin,
	// ErrForbidden. Un id inválido es ErrInvalidID.
	HistoryOf(ctx context.Context, id string) ([]bondHistory_domain.HistoryEntry, error)
	// List devuelve la página p de los bonos no borrados del alcance de la sesión que pasan f, por createdAt
	// descendente, con su dueño, su producto, su tipo y los días y el monto acumulados a hoy; las fechas usan
	// la zona de la sesión. Un filtro inválido es el error de su Validation.
	List(ctx context.Context, f bondBound_domain.ListFilters, p shared_domain.Pagination) (shared_domain.Paginated[BondBoundAdminItem], error)
	// ListByUserAndObject devuelve los bonos iniciados, madurados, bloqueados o canjeados del usuario en el
	// producto, en el alcance de la sesión. Sin bondObjectID es ErrBondObjectIDRequired, y un id inválido,
	// ErrInvalidID.
	ListByUserAndObject(ctx context.Context, userID, bondObjectID string) ([]*bondBound_domain.BondBound, error)
	// ListByObject devuelve los bonos no borrados del producto en el alcance de la sesión, con el perfil
	// público de cada dueño. Un id inválido es ErrInvalidID.
	ListByObject(ctx context.Context, bondObjectID string) ([]BondBoundWithProfile, error)
	// Remove borra el bono de forma lógica. Si no existe, ya estaba borrado o es de otra agencia que la de la
	// sesión, que no es la principal, es ErrBondBoundNotFound. Un id inválido es ErrInvalidID.
	Remove(ctx context.Context, id string) error
	// Statuses devuelve los seis estados de un bono en orden, con su nombre.
	Statuses() []bondBound_domain.StatusRef
	// ListMine devuelve la página p de los bonos no borrados del usuario de la sesión en su agencia, por
	// createdAt descendente, con su producto y los derivados a hoy en la zona de la sesión. Un estado
	// desconocido es ErrInvalidStatus.
	ListMine(ctx context.Context, f bondBound_domain.MineFilters, p shared_domain.Pagination) (shared_domain.Paginated[BondBoundMineItem], error)
	// ListRecent devuelve los cinco bonos iniciados o madurados más recientes del usuario de la sesión, de
	// cualquier agencia, con su producto y los derivados a hoy en la zona de la sesión.
	ListRecent(ctx context.Context) ([]BondBoundMineItem, error)
	// GetByID devuelve el bono no borrado si es del usuario de la sesión o la sesión es admin. Si no existe
	// es ErrBondBoundNotFound, y si es ajeno, ErrForbidden. Un id inválido es ErrInvalidID.
	GetByID(ctx context.Context, id string) (*bondBound_domain.BondBound, error)
	// GetDetails devuelve el detalle del bono con lo acumulado a hoy en la zona de la sesión. Si no existe,
	// le falta el producto o el tipo, o está fuera del alcance de agencia de la sesión, es
	// ErrBondBoundNotFound; si es ajeno y la sesión no es admin, ErrForbidden. Un id inválido es ErrInvalidID.
	GetDetails(ctx context.Context, id string) (*BondBoundDetail, error)
	// Timeline devuelve las compras de userID, o del usuario de la sesión si viene vacío, con las fechas en
	// la zona de la sesión. Las de otro usuario exigen admin; si no, ErrForbidden. Un id inválido es
	// ErrInvalidID.
	Timeline(ctx context.Context, userID string) ([]TimelineItem, error)
	// ReportPurchased genera el Excel de los bonos no borrados de la agencia de la sesión que pasan f, por
	// createdAt descendente, con la moneda de la agencia; el filtro de fecha y la hora del nombre usan la zona
	// de la sesión. Un filtro inválido es el error de su Validation, y si ms-exporter falla, ErrExportFailed.
	ReportPurchased(ctx context.Context, f bondBound_domain.ReportFilters) (ReportFile, error)
	// ReportSummary genera el Excel del total comprado por cada inversionista en los bonos no borrados de la
	// agencia de la sesión, del mayor al menor, con la moneda de la agencia; de f solo filtra el estado, y la
	// hora del nombre usa la zona de la sesión. Un filtro inválido es el error de su Validation, y si
	// ms-exporter falla, ErrExportFailed.
	ReportSummary(ctx context.Context, f bondBound_domain.ReportFilters) (ReportFile, error)
	// CountByStatus devuelve la suma de los montos y la cantidad de los bonos no borrados del alcance de la
	// sesión que pasan f, con el nombre del estado por el que se filtró, o "" si no se filtró; el filtro de
	// fecha usa la zona de la sesión. Un admin cuenta los de f.UserID o, sin él, los de todo el alcance; un no
	// admin solo los suyos, y pedir los de otro usuario es ErrForbidden. Un filtro inválido es el error de su
	// Validation.
	CountByStatus(ctx context.Context, f bondBound_domain.KpiFilters) (StatusCount, error)
	// RunMaturity pasa a madurados los bonos iniciados que vencieron hasta now, con el lock del job tomado, y
	// registra el historial del sistema de cada uno; después avisa a cada dueño una vez, con todos sus bonos
	// madurados. Si otra corrida tiene el lock es ErrJobLocked. Un bono que falla se loguea, cuenta en Failed
	// y la corrida sigue.
	RunMaturity(ctx context.Context, now time.Time) (bondBound_domain.MaturityResult, error)
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
	// FindMaturable devuelve hasta limit bonos no borrados e iniciados, sin soldDate ni tookDate, cuya
	// maturityDate es hasta now, por _id ascendente y con _id mayor que afterID. afterID vacío empieza por el
	// primero y limit 0 no limita.
	FindMaturable(ctx context.Context, now time.Time, limit int, afterID string) ([]*bondBound_domain.BondBound, error)
	// ListAdmin devuelve la página p, ya normalizada, de los bonos no borrados que pasan f, por createdAt
	// descendente, con su dueño, su producto y su tipo; los bonos sin producto o sin tipo no aparecen. El
	// filtro de fecha, el mes y el año usan la zona utcDiff. agencyID nil no filtra por agencia.
	ListAdmin(ctx context.Context, agencyID *string, f bondBound_domain.ListFilters, p shared_domain.Pagination, utcDiff int) (shared_domain.Paginated[BondBoundAdminRow], error)
	// ListByUserAndObject devuelve los bonos no borrados del usuario y el producto que están iniciados,
	// madurados, bloqueados o canjeados, por createdAt descendente. agencyID nil no filtra por agencia.
	ListByUserAndObject(ctx context.Context, userID, bondObjectID string, agencyID *string) ([]*bondBound_domain.BondBound, error)
	// ListMine devuelve la página p, ya normalizada, de los bonos no borrados del usuario en la agencia, por
	// createdAt descendente, con su producto; los bonos sin producto no aparecen. statusIDs vacío no filtra
	// por estado.
	ListMine(ctx context.Context, userID, agencyID string, statusIDs []bondBound_domain.BondBoundStatus, p shared_domain.Pagination) (shared_domain.Paginated[BondBoundWithObject], error)
	// ListRecent devuelve los limit bonos no borrados más recientes del usuario que están iniciados o
	// madurados, de cualquier agencia, con su producto y sus términos de retiro. limit 0 no limita.
	ListRecent(ctx context.Context, userID string, limit int) ([]BondBoundWithObject, error)
	// ListByObject devuelve los bonos no borrados del producto, por createdAt descendente. agencyID nil no
	// filtra por agencia.
	ListByObject(ctx context.Context, bondObjectID string, agencyID *string) ([]*bondBound_domain.BondBound, error)
	// FindDetails devuelve el bono con su producto y su tipo, o nil, nil si no existe, está borrado o le
	// falta el producto o el tipo. No filtra por dueño ni por agencia.
	FindDetails(ctx context.Context, id string) (*BondBoundDetailRow, error)
	// Timeline devuelve las compras no borradas del usuario por fecha inicial, con las fechas en la zona
	// utcDiff.
	Timeline(ctx context.Context, userID string, utcDiff int) ([]TimelineItem, error)
	// SoftDelete marca el bono con isRemove true y actualiza updatedAt; devuelve false si no existe o ya
	// estaba borrado.
	SoftDelete(ctx context.Context, id string) (bool, error)
	// ReportPurchasedRows devuelve los bonos no borrados de la agencia que pasan f, por createdAt descendente,
	// con su dueño, su producto, su tipo y el nombre de su estado; los bonos sin producto o sin tipo no
	// aparecen. El filtro de fecha usa la zona utcDiff.
	ReportPurchasedRows(ctx context.Context, agencyID string, f bondBound_domain.ReportFilters, utcDiff int) ([]PurchasedReportRow, error)
	// ReportSummaryRows devuelve el total comprado por cada usuario en los bonos no borrados de la agencia,
	// del mayor al menor, con su perfil. De f solo usa el estado.
	ReportSummaryRows(ctx context.Context, agencyID string, f bondBound_domain.ReportFilters) ([]SummaryReportRow, error)
	// CountByStatus devuelve la suma de los montos y la cantidad de los bonos no borrados que pasan f; Status
	// queda vacío. El filtro de fecha usa la zona utcDiff, y agencyID nil no filtra por agencia.
	CountByStatus(ctx context.Context, agencyID *string, f bondBound_domain.KpiFilters, utcDiff int) (StatusCount, error)
	// EnsureIndexes crea los índices de la colección; es idempotente.
	EnsureIndexes(ctx context.Context) error
}

// KeyCounterPort numera las claves de los bonos por tipo.
type KeyCounterPort interface {
	// Reserve reserva n números de clave consecutivos del tipo y devuelve el primero; si el contador no
	// existe, lo crea desde 0. Usa el ctx recibido: si la transacción revierte, el contador no avanza.
	Reserve(ctx context.Context, bondTypeID string, n int) (start int, err error)
}
