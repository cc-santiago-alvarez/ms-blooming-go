//go:build integration

package bondBound_application_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/codecraftkit/nexus"
	jobs_handlers "github.com/futurapps/ms-blooming-go/cmd/api/handlers/jobs"
	bondBound_application "github.com/futurapps/ms-blooming-go/internal/core/bondBound/application"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondType_application "github.com/futurapps/ms-blooming-go/internal/core/bondType/application"
	bondType_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondType/infrastructure/mongocc"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/mongocc"
	middlewares_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/middlewares"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Colecciones de la maduración que no están en collections.
const (
	lockCollection = "bondJobLock"
	typeCollection = "bondType"
)

const (
	// maturityLockID es el _id del lock de la maduración.
	maturityLockID = "bond-maturity"

	// jobsSecret es el API_SECRET del servidor del endpoint.
	jobsSecret = "s3cr3t-jobs-valor"
	// maturityRunPath es la ruta del endpoint interno de la maduración.
	maturityRunPath = "/api/v1/internal/jobs/bond-maturity/run"
)

// maturitySetup vacía las colecciones, también el lock y los tipos, y guarda el producto y su tipo.
func maturitySetup(t *testing.T) {
	t.Helper()
	setupDB(t)
	for _, name := range []string{lockCollection, typeCollection} {
		if _, err := testDB.GetCollection(name).DeleteMany(context.Background(), bson.M{}); err != nil {
			t.Fatalf("no se pudo vaciar %s: %v", name, err)
		}
	}
	insertObject(t, objectID, typeID)
	insert(t, typeCollection, bson.M{
		"_id":         typeID,
		"agencyId":    bson.A{agencyA},
		"name":        "Renta fija",
		"description": "Bonos de renta fija",
		"active":      true,
		"createdAt":   fixtureAt,
		"updatedAt":   fixtureAt,
		"isRemove":    false,
	})
}

// maturityBoundDoc es el bono iniciado id de userID con la clave BP12-seq, comprado hace days días con el plazo
// de 100 de saleBoundDoc: con 101 venció ayer y con 98 vence pasado mañana, fuera del corte aunque la corrida
// cruce la medianoche UTC.
func maturityBoundDoc(id string, seq int, userID string, days int) bson.M {
	doc := saleBoundDoc(id, bondBound_domain.StatusInitiated, days)
	maps.Copy(doc, bson.M{"key": fmt.Sprintf("BP12-%d", seq), "keySeq": seq, "userId": userID})
	return doc
}

// maturityApp arma la aplicación con los adapters de Mongo, los tipos y el lock a nombre de owner, y devuelve
// también sus notificaciones registradas.
func maturityApp(owner string) (*bondBound_application.BondBoundApplication, *fakeNotifications) {
	notifications := &fakeNotifications{}
	app := integrationApp(realBalance(), realKeyCounter())
	app.BondTypes = &bondType_application.BondTypeApplication{
		Repository: &bondType_mongocc_repository.BondTypeMongoccRepository{DB: testDB},
	}
	app.JobLock = &shared_mongocc_repository.JobLockMongoccRepository{DB: testDB, Owner: owner}
	app.Notifications = notifications
	return app, notifications
}

// gatedMaturable usa el repositorio real, pero su primer FindMaturable espera a que se cierre release: así la
// réplica que tomó el lock lo retiene mientras la otra intenta tomarlo.
type gatedMaturable struct {
	bondBound_ports.BondBoundMongoccRepositoryPorts

	release <-chan struct{}
	calls   atomic.Int32
}

func (g *gatedMaturable) FindMaturable(ctx context.Context, cutoff time.Time, limit int, afterID string) ([]*bondBound_domain.BondBound, error) {
	if g.calls.Add(1) == 1 {
		select {
		case <-g.release:
		case <-time.After(gateTimeout):
			return nil, errors.New("la compuerta de la réplica no se abrió a tiempo")
		}
	}
	return g.BondBoundMongoccRepositoryPorts.FindMaturable(ctx, cutoff, limit, afterID)
}

// maturityOutcome es lo que devolvió la corrida de una réplica.
type maturityOutcome struct {
	replica int
	result  bondBound_domain.MaturityResult
	err     error
}

// awaitOutcome espera la próxima corrida terminada; falla el test si no llega.
func awaitOutcome(t *testing.T, outcomes <-chan maturityOutcome) maturityOutcome {
	t.Helper()
	select {
	case o := <-outcomes:
		return o
	case <-time.After(2 * gateTimeout):
		t.Fatal("la corrida de maduración no terminó")
		return maturityOutcome{}
	}
}

// jobLock es el lock de la maduración tal como quedó en Mongo.
type jobLock struct {
	Owner       string    `bson:"owner"`
	LockedUntil time.Time `bson:"lockedUntil"`
}

// insertJobLock guarda el lock de la maduración tomado por owner hasta until.
func insertJobLock(t *testing.T, owner string, until time.Time) {
	t.Helper()
	insert(t, lockCollection, bson.M{"_id": maturityLockID, "owner": owner, "lockedUntil": until, "updatedAt": until})
}

// jobLockDoc lee el lock de la maduración.
func jobLockDoc(t *testing.T) jobLock {
	t.Helper()
	var lock jobLock
	if err := testDB.FindOne(context.Background(), lockCollection, bson.M{"_id": maturityLockID}, nil).Decode(&lock); err != nil {
		t.Fatalf("no se pudo leer el lock %s: %v", maturityLockID, err)
	}
	return lock
}

// statusOf devuelve el statusId guardado del bono.
func statusOf(t *testing.T, id string) string {
	t.Helper()
	status, _ := rawDoc(t, boundCollection, id).Lookup("statusId").StringValueOK()
	return status
}

// historyGroup es un grupo de la auditoría del gateway: el usuario y el bono.
type historyGroup struct {
	UserID    string `bson:"userId"`
	RelatedID string `bson:"relatedId"`
}

// historyGroups corre las etapas $match y $group de B_BondBound_Audit_details (ms-gateway-gql) sobre
// bondHistory y devuelve cuántos cambios de estado tiene cada grupo.
func historyGroups(t *testing.T) map[historyGroup]int {
	t.Helper()
	ctx := context.Background()
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"text": "BOND_BOUND_STATUS_CHANGED", "params": bson.M{"$size": 2}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"userId": "$userId", "relatedId": "$relatedId"},
			"count": bson.M{"$sum": 1},
		}}},
	}
	cursor, err := testDB.GetCollection(historyCollection).Aggregate(ctx, pipeline)
	if err != nil {
		t.Fatalf("no se pudo agrupar %s: %v", historyCollection, err)
	}
	var rows []struct {
		Group historyGroup `bson:"_id"`
		Count int          `bson:"count"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		t.Fatalf("no se pudieron leer los grupos: %v", err)
	}
	groups := map[historyGroup]int{}
	for _, r := range rows {
		groups[r.Group] = r.Count
	}
	return groups
}

// stringArray devuelve el array de strings key del documento; falla el test si no lo es.
func stringArray(t *testing.T, raw bson.Raw, key string) []string {
	t.Helper()
	array, ok := raw.Lookup(key).ArrayOK()
	if !ok {
		t.Fatalf("%s no es un array: %v", key, raw.Lookup(key))
	}
	values, err := array.Values()
	if err != nil {
		t.Fatalf("no se pudo leer %s: %v", key, err)
	}
	var out []string
	for _, v := range values {
		s, ok := v.StringValueOK()
		if !ok {
			t.Fatalf("%s tiene un %v, se esperaban strings: %v", key, v.Type, array)
		}
		out = append(out, s)
	}
	return out
}

// rejectingVerifier rechaza toda sesión: el endpoint interno no la pide.
type rejectingVerifier struct{}

func (rejectingVerifier) Verify(context.Context, string) (*session_domain.UserSession, error) {
	return nil, errors.New("el endpoint interno no debe pedir sesión")
}

// maturityAPI arma el servidor del endpoint interno con los middlewares de secret y de sesión, como
// server.config.go.
func maturityAPI(app *bondBound_application.BondBoundApplication) http.Handler {
	server := &nexus.Server{
		ServerName: "blooming-test",
		Secret:     jobsSecret,
		Settings:   &nexus.Settings{PathPrefix: "/api/v1"},
	}
	server.Use(middlewares_adapters.ValidateApiSecret)
	server.Use(middlewares_adapters.VerifySession(rejectingVerifier{}))
	jobs_handlers.NewJobsHandler(server, app)
	return server.ApplyMiddlewares(server.PrepareEndpoints())
}

// De dos réplicas que corren a la vez, la que toma el lock madura los 3 bonos y la otra termina con
// ErrJobLocked sin leer ni escribir nada: un solo cambio y un solo historial por bono, un aviso, y el lock
// queda liberado. CA-10.2
func TestRunMaturityConcurrentReplicas(t *testing.T) {
	maturitySetup(t)
	for n := 1; n <= 3; n++ {
		insert(t, boundCollection, maturityBoundDoc(maturityID(n), n, ownerID, 101))
	}
	release := make(chan struct{})
	open := sync.OnceFunc(func() { close(release) })
	t.Cleanup(open)

	owners := []string{"replica-1", "replica-2"}
	gates := make([]*gatedMaturable, len(owners))
	outcomes := make(chan maturityOutcome, len(owners))
	for i, owner := range owners {
		app, _ := maturityApp(owner)
		gates[i] = &gatedMaturable{BondBoundMongoccRepositoryPorts: app.Repository, release: release}
		app.Repository = gates[i]
		go func() {
			result, err := app.RunMaturity(context.Background(), shared_domain.Now())
			outcomes <- maturityOutcome{replica: i, result: result, err: err}
		}()
	}

	// la que tomó el lock espera en la compuerta: la primera en terminar es la otra
	lost := awaitOutcome(t, outcomes)
	winner := 1 - lost.replica
	if !errors.Is(lost.err, bondBound_domain.ErrJobLocked) || lost.result != (bondBound_domain.MaturityResult{}) {
		t.Errorf("%s devolvió %+v, %v; se esperaba ErrJobLocked sin resultado", owners[lost.replica], lost.result, lost.err)
	}
	if calls := gates[lost.replica].calls.Load(); calls != 0 {
		t.Errorf("%s leyó %d lotes, se esperaba ninguno", owners[lost.replica], calls)
	}
	if lock := jobLockDoc(t); lock.Owner != owners[winner] {
		t.Errorf("con la corrida en curso el lock es de %q, se esperaba %q", lock.Owner, owners[winner])
	}

	open()
	won := awaitOutcome(t, outcomes)
	end := time.Now()

	if won.err != nil {
		t.Fatalf("%s: %v", owners[winner], won.err)
	}
	if want := (bondBound_domain.MaturityResult{Matured: 3, Users: 1, Failed: 0}); won.result != want {
		t.Errorf("%s devolvió %+v, se esperaba %+v", owners[winner], won.result, want)
	}
	for n := 1; n <= 3; n++ {
		if status := statusOf(t, maturityID(n)); status != "2" {
			t.Errorf("bono %d en %q, se esperaba madurado", n, status)
		}
	}
	if histories, notices := countDocs(t, historyCollection), countDocs(t, noticeCollection); histories != 3 || notices != 1 {
		t.Errorf("hay %d bondHistory y %d avisos, se esperaban 3 y 1", histories, notices)
	}
	if lock := jobLockDoc(t); lock.Owner != owners[winner] || lock.LockedUntil.After(end) {
		t.Errorf("lock de %q hasta %v, se esperaba liberado por %q antes de %v", lock.Owner, lock.LockedUntil, owners[winner], end)
	}
}

// Un lock vencido de otra réplica se vuelve a tomar y la corrida procesa; uno vigente la deja en
// ErrJobLocked sin tocar el bono, el historial ni el lock. CA-10.2
func TestRunMaturityLockState(t *testing.T) {
	tests := []struct {
		name          string
		heldFor       time.Duration
		wantErr       error
		wantResult    bondBound_domain.MaturityResult
		wantStatus    string
		wantHistories int64
		wantOwner     string
	}{
		{
			name:          "vencido",
			heldFor:       -time.Minute,
			wantResult:    bondBound_domain.MaturityResult{Matured: 1, Users: 1},
			wantStatus:    "2",
			wantHistories: 1,
			wantOwner:     "replica-1",
		},
		{
			name:       "vigente de otra réplica",
			heldFor:    5 * time.Minute,
			wantErr:    bondBound_domain.ErrJobLocked,
			wantStatus: "1",
			wantOwner:  "replica-2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maturitySetup(t)
			insert(t, boundCollection, maturityBoundDoc(boundID, 1, ownerID, 101))
			until := shared_domain.Now().Add(tt.heldFor).Truncate(time.Millisecond)
			insertJobLock(t, "replica-2", until)
			app, _ := maturityApp("replica-1")
			before := time.Now()

			result, err := app.RunMaturity(context.Background(), shared_domain.Now())

			after := time.Now()
			if !errors.Is(err, tt.wantErr) || result != tt.wantResult {
				t.Errorf("RunMaturity = %+v, %v; se esperaba %+v, %v", result, err, tt.wantResult, tt.wantErr)
			}
			if status := statusOf(t, boundID); status != tt.wantStatus {
				t.Errorf("bono en %q, se esperaba %q", status, tt.wantStatus)
			}
			if n := countDocs(t, historyCollection); n != tt.wantHistories {
				t.Errorf("hay %d bondHistory, se esperaban %d", n, tt.wantHistories)
			}
			lock := jobLockDoc(t)
			if lock.Owner != tt.wantOwner {
				t.Errorf("lock de %q, se esperaba %q", lock.Owner, tt.wantOwner)
			}
			if tt.wantErr != nil && !lock.LockedUntil.Equal(until) {
				t.Errorf("lock hasta %v, se esperaba sin cambios hasta %v", lock.LockedUntil, until)
			}
			if tt.wantErr == nil {
				assertDateBetween(t, "lockedUntil", &lock.LockedUntil, before, after)
			}
		})
	}
}

// Con now 2026-10-13T00:03Z, en Mongo madura el bono que vence el 13 a las 23:59:59 UTC, con maturedAt now,
// y el que vence a la medianoche siguiente queda iniciado. CA-10.10
func TestRunMaturityCutoffInMongo(t *testing.T) {
	maturitySetup(t)
	today := maturityBoundDoc(maturityID(1), 1, ownerID, 101)
	today["maturityDate"] = time.Date(2026, 10, 13, 23, 59, 59, 0, time.UTC)
	tomorrow := maturityBoundDoc(maturityID(2), 2, ownerID, 101)
	tomorrow["maturityDate"] = time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	insert(t, boundCollection, today, tomorrow)
	app, _ := maturityApp("replica-1")
	now := time.Date(2026, 10, 13, 0, 3, 0, 0, time.UTC)

	result, err := app.RunMaturity(context.Background(), now)

	if err != nil {
		t.Fatalf("RunMaturity: %v", err)
	}
	if want := (bondBound_domain.MaturityResult{Matured: 1, Users: 1, Failed: 0}); result != want {
		t.Errorf("RunMaturity = %+v, se esperaba %+v", result, want)
	}
	raw := rawDoc(t, boundCollection, maturityID(1))
	status, _ := raw.Lookup("statusId").StringValueOK()
	matured, _ := raw.Lookup("matured").BooleanOK()
	maturedAt, ok := raw.Lookup("maturedAt").TimeOK()
	if status != "2" || !matured || !ok || !maturedAt.Equal(now) {
		t.Errorf("el bono que vence hoy quedó en %q con matured %v y maturedAt %v, se esperaba madurado con maturedAt %s",
			status, matured, raw.Lookup("maturedAt"), now)
	}
	if status := statusOf(t, maturityID(2)); status != "1" {
		t.Errorf("el bono que vence a la medianoche quedó en %q, se esperaba iniciado", status)
	}
}

// Dos corridas seguidas: la segunda da matured 0, sin historiales, avisos ni emails nuevos. CA-10.4
func TestRunMaturityIdempotent(t *testing.T) {
	maturitySetup(t)
	insert(t, boundCollection,
		maturityBoundDoc(maturityID(1), 1, ownerID, 101),
		maturityBoundDoc(maturityID(2), 2, ownerID, 101),
		maturityBoundDoc(maturityID(3), 3, otherUserID, 101),
		maturityBoundDoc(maturityID(4), 4, ownerID, 98),
	)
	app, notifications := maturityApp("replica-1")

	first, err := app.RunMaturity(context.Background(), shared_domain.Now())
	if err != nil {
		t.Fatalf("primera corrida: %v", err)
	}
	if want := (bondBound_domain.MaturityResult{Matured: 3, Users: 2, Failed: 0}); first != want {
		t.Fatalf("la primera corrida devolvió %+v, se esperaba %+v", first, want)
	}
	notifications.wait(t, 2)

	second, err := app.RunMaturity(context.Background(), shared_domain.Now())

	if err != nil {
		t.Fatalf("segunda corrida: %v", err)
	}
	if second != (bondBound_domain.MaturityResult{}) {
		t.Errorf("la segunda corrida devolvió %+v, se esperaba matured 0", second)
	}
	if histories, notices := countDocs(t, historyCollection), countDocs(t, noticeCollection); histories != 3 || notices != 2 {
		t.Errorf("hay %d bondHistory y %d avisos, se esperaban los 3 y 2 de la primera corrida", histories, notices)
	}
	if sent := notifications.wait(t, 2); len(sent) != 2 {
		t.Errorf("se enviaron %d emails, se esperaban los 2 de la primera corrida", len(sent))
	}
	if status := statusOf(t, maturityID(4)); status != "1" {
		t.Errorf("el bono que vence pasado mañana quedó en %q, se esperaba iniciado", status)
	}
}

// El historial de la maduración es {typeId '2', text 'BOND_BOUND_STATUS_CHANGED', params ['1','2'], userId ""}
// en la agencia del bono. Agrupado por (userId, relatedId) como B_BondBound_Audit_details, queda en otro grupo
// que el canje del dueño sobre el mismo bono, y ningún grupo cuenta dos cambios. CA-10.5
func TestRunMaturitySystemHistory(t *testing.T) {
	maturitySetup(t)
	insertAccount(t, accountID, ownerID, agencyA, 5000, false)
	insertSetting(t, ownerID, accountID)
	insert(t, boundCollection, maturityBoundDoc(boundID, 1, ownerID, 101))
	app, _ := maturityApp("replica-1")

	if _, err := app.RunMaturity(context.Background(), shared_domain.Now()); err != nil {
		t.Fatalf("RunMaturity: %v", err)
	}

	histories := findAll[bondHistory_domain.BondHistory](t, historyCollection)
	if len(histories) != 1 {
		t.Fatalf("hay %d bondHistory, se esperaba 1", len(histories))
	}
	raw := rawDoc(t, historyCollection, histories[0].ID)
	typeID, _ := raw.Lookup("typeId").StringValueOK()
	text, _ := raw.Lookup("text").StringValueOK()
	relatedID, _ := raw.Lookup("relatedId").StringValueOK()
	agencyID, _ := raw.Lookup("agencyId").StringValueOK()
	userID, isString := raw.Lookup("userId").StringValueOK()
	if typeID != "2" || text != "BOND_BOUND_STATUS_CHANGED" || relatedID != boundID || agencyID != agencyA || !isString || userID != "" {
		t.Errorf("bondHistory %s, se esperaba typeId \"2\", text BOND_BOUND_STATUS_CHANGED, relatedId %s, agencyId %s y userId \"\"",
			raw, boundID, agencyA)
	}
	if params := stringArray(t, raw, "params"); !slices.Equal(params, []string{"1", "2"}) {
		t.Errorf("params %v, se esperaba [1 2]", params)
	}

	result, err := app.Sale(withSession(ownerID, -5), boundID)
	if err != nil {
		t.Fatalf("Sale: %v", err)
	}
	if result.StatusID != bondBound_domain.StatusTook {
		t.Fatalf("Sale dejó el bono en %q, se esperaba canjeado", result.StatusID)
	}

	want := map[historyGroup]int{
		{UserID: "", RelatedID: boundID}:      1,
		{UserID: ownerID, RelatedID: boundID}: 1,
	}
	if groups := historyGroups(t); !maps.Equal(groups, want) {
		t.Errorf("grupos de la auditoría %v, se esperaba %v", groups, want)
	}
}

// POST /internal/jobs/bond-maturity/run sin x-secret, o con uno inválido, responde 401 sin tocar nada; con
// x-secret y sin sesión responde 200 {matured, users, failed}; con el lock de otra réplica, 409
// ERR_JOB_LOCKED. CA-10.8
func TestRunMaturityEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		secret     string
		locked     bool
		status     int
		code       string
		body       string
		wantStatus string
	}{
		{name: "sin x-secret", status: http.StatusUnauthorized, code: "UNAUTHORIZED_INVALID_SECRET", wantStatus: "1"},
		{name: "x-secret inválido", secret: "otro-valor", status: http.StatusUnauthorized, code: "UNAUTHORIZED_INVALID_SECRET", wantStatus: "1"},
		{name: "con x-secret y sin sesión", secret: jobsSecret, status: http.StatusOK, body: `{"matured":1,"users":1,"failed":0}`, wantStatus: "2"},
		{name: "lock de otra réplica", secret: jobsSecret, locked: true, status: http.StatusConflict, code: "ERR_JOB_LOCKED", wantStatus: "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maturitySetup(t)
			insert(t, boundCollection, maturityBoundDoc(boundID, 1, ownerID, 101))
			if tt.locked {
				insertJobLock(t, "replica-2", shared_domain.Now().Add(5*time.Minute))
			}
			app, _ := maturityApp("replica-1")
			captureLog(t)
			req := httptest.NewRequest(http.MethodPost, maturityRunPath, nil)
			if tt.secret != "" {
				req.Header.Set("x-secret", tt.secret)
			}
			rec := httptest.NewRecorder()

			maturityAPI(app).ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("status %d, se esperaba %d (body %s)", rec.Code, tt.status, rec.Body)
			}
			if tt.body != "" && strings.TrimSpace(rec.Body.String()) != tt.body {
				t.Errorf("body %s, se esperaba %s", rec.Body, tt.body)
			}
			if tt.code != "" {
				var body nexus.ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Message != tt.code {
					t.Errorf("body %s, se esperaba el código %s", rec.Body, tt.code)
				}
			}
			if status := statusOf(t, boundID); status != tt.wantStatus {
				t.Errorf("bono en %q, se esperaba %q", status, tt.wantStatus)
			}
			switch {
			case tt.status == http.StatusUnauthorized:
				if n := countDocs(t, lockCollection); n != 0 {
					t.Errorf("hay %d locks, se esperaba que no se tomara", n)
				}
			case tt.locked:
				if lock := jobLockDoc(t); lock.Owner != "replica-2" {
					t.Errorf("lock de %q, se esperaba que siguiera de replica-2", lock.Owner)
				}
			}
		})
	}
}

// Un bono legacy de 17 caracteres, sin keySeq y con numéricos int32 o string, madura junto con uno nuevo:
// los dos pasan a '2' con matured y maturedAt, el legacy no gana keySeq, y su historial de sistema, con _id
// UUID, tiene relatedId igual al _id legacy. CA-10.9
func TestRunMaturityLegacyBound(t *testing.T) {
	maturitySetup(t)
	legacy := maturityBoundDoc(legacyBoundID, 7, ownerID, 101)
	delete(legacy, "keySeq")
	maps.Copy(legacy, bson.M{
		"amount":                    int32(1000),
		"maturityTimeInDays":        int32(100),
		"maturityRewardsPercentage": int32(10),
		"earlyCashTerm": bson.M{
			"initialPenaltyPercentage":    "100",
			"inflectionPoint":             "0",
			"inflectionPenaltyPercentage": "50",
			"blockWithdrawals":            false,
			"initialDayForWithdraw":       "0",
		},
	})
	insert(t, boundCollection, legacy, maturityBoundDoc(boundID, 8, ownerID, 101))
	app, _ := maturityApp("replica-1")
	before := time.Now()

	result, err := app.RunMaturity(context.Background(), shared_domain.Now())

	after := time.Now()
	if err != nil {
		t.Fatalf("RunMaturity: %v", err)
	}
	if want := (bondBound_domain.MaturityResult{Matured: 2, Users: 1, Failed: 0}); result != want {
		t.Errorf("RunMaturity = %+v, se esperaba %+v", result, want)
	}
	for _, id := range []string{legacyBoundID, boundID} {
		raw := rawDoc(t, boundCollection, id)
		status, _ := raw.Lookup("statusId").StringValueOK()
		matured, _ := raw.Lookup("matured").BooleanOK()
		maturedAt, ok := raw.Lookup("maturedAt").TimeOK()
		if status != "2" || !matured || !ok {
			t.Errorf("bono %s en %q con matured %v y maturedAt %v, se esperaba madurado", id, status, matured, raw.Lookup("maturedAt"))
			continue
		}
		assertDateBetween(t, "maturedAt de "+id, &maturedAt, before, after)
	}
	if _, err := rawDoc(t, boundCollection, legacyBoundID).LookupErr("keySeq"); err == nil {
		t.Error("la maduración le agregó keySeq al bono legacy")
	}

	histories := findAll[bondHistory_domain.BondHistory](t, historyCollection)
	var related []string
	for _, h := range histories {
		if shared_domain.IDFormat(h.ID) != shared_domain.IDKindUUID || h.UserID != "" {
			t.Errorf("bondHistory con _id %q y userId %q, se esperaba un UUID v4 del sistema", h.ID, h.UserID)
		}
		related = append(related, h.RelatedID)
	}
	slices.Sort(related)
	if want := []string{boundID, legacyBoundID}; !slices.Equal(related, want) {
		t.Errorf("bondHistory de %v, se esperaba uno por bono: %v", related, want)
	}
}
