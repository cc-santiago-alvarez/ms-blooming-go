//go:build integration

package bondBound_application_test

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	bondBound_application "github.com/futurapps/ms-blooming-go/internal/core/bondBound/application"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_notification "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/notification"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// noticeDateLayout es el formato de las fechas del aviso de venta.
const noticeDateLayout = "02/01/2006 3:4:5 pm"

// saleBoundDoc es un bono de ownerID del producto objectID con clave BP12-1, comprado hace days días y en
// status: 1000 a 100 días al 10 %, sin punto de inflexión, con 50 % de penalidad y retiro desde la compra.
// En UTC-5, el día 50 cotiza 1025 y desde el 100, 1100.
func saleBoundDoc(id string, status bondBound_domain.BondBoundStatus, days int) bson.M {
	initial := time.Now().UTC().AddDate(0, 0, -days).Truncate(time.Second)
	return bson.M{
		"_id":                       id,
		"bondObjectId":              objectID,
		"bondTypeId":                typeID,
		"key":                       "BP12-1",
		"keySeq":                    1,
		"token":                     "Ab3dE6",
		"userId":                    ownerID,
		"agencyId":                  agencyA,
		"statusId":                  string(status),
		"amount":                    1000.0,
		"maturityTimeInDays":        100,
		"maturityRewardsPercentage": 10.0,
		"earlyCashTerm": bson.M{
			"initialPenaltyPercentage":    100.0,
			"inflectionPoint":             0,
			"inflectionPenaltyPercentage": 50.0,
			"blockWithdrawals":            false,
			"initialDayForWithdraw":       0,
		},
		"initialDate":  initial,
		"maturityDate": initial.AddDate(0, 0, 100),
		"createdAt":    initial,
		"updatedAt":    initial,
		"isRemove":     false,
	}
}

// saleSetup guarda la cuenta de ownerID con 5000, seleccionada en su setting, el producto y el bono doc.
func saleSetup(t *testing.T, doc bson.M) {
	t.Helper()
	setupDB(t)
	insertAccount(t, accountID, ownerID, agencyA, 5000, false)
	insertSetting(t, ownerID, accountID)
	insertObject(t, objectID, typeID)
	insert(t, boundCollection, doc)
}

// saleApp arma la aplicación con los adapters de Mongo y devuelve también sus notificaciones registradas.
func saleApp() (*bondBound_application.BondBoundApplication, *fakeNotifications) {
	notifications := &fakeNotifications{}
	app := integrationApp(realBalance(), realKeyCounter())
	app.Notifications = notifications
	return app, notifications
}

// onlyBound devuelve el único bono guardado.
func onlyBound(t *testing.T) bondBound_domain.BondBound {
	t.Helper()
	bounds := findAll[bondBound_domain.BondBound](t, boundCollection)
	if len(bounds) != 1 {
		t.Fatalf("hay %d bonos, se esperaba 1", len(bounds))
	}
	return bounds[0]
}

// assertDateBetween verifica que la fecha name esté entre before y after, con la precisión de Mongo.
func assertDateBetween(t *testing.T, name string, at *time.Time, before, after time.Time) {
	t.Helper()
	if at == nil || at.Before(before.Truncate(time.Millisecond)) || at.After(after) {
		t.Errorf("%s %v, se esperaba entre %v y %v", name, at, before, after)
	}
}

// assertCredit verifica que haya un solo balanceDetail: el crédito de amount en la cuenta seleccionada.
func assertCredit(t *testing.T, amount float64, descriptionID, comment string) {
	t.Helper()
	details := findAll[shared_domain.BalanceDetail](t, detailCollection)
	if len(details) != 1 || details[0].Amount != amount || details[0].DescriptionID != descriptionID ||
		details[0].BalanceAccountID != accountID || details[0].UserID != ownerID || details[0].Comment != comment {
		t.Errorf("balanceDetail %+v, se esperaba uno de +%v con descriptionId %s y comment %q en %s",
			details, amount, descriptionID, comment, accountID)
	}
}

// gatedUserSettings usa el setting real, pero cada GetByUser espera a que lleguen parties lecturas: así las
// ventas concurrentes ya leyeron y cotizaron el bono iniciado antes de que una abra su transacción.
type gatedUserSettings struct {
	bondUserSetting_ports.BondUserSettingApplicationPorts

	parties int32
	arrived atomic.Int32
	open    chan struct{}
}

func newGatedUserSettings(settings bondUserSetting_ports.BondUserSettingApplicationPorts, parties int32) *gatedUserSettings {
	return &gatedUserSettings{BondUserSettingApplicationPorts: settings, parties: parties, open: make(chan struct{})}
}

// GetByUser espera a las demás ventas y lee el setting real.
func (g *gatedUserSettings) GetByUser(ctx context.Context, userID string) (*bondUserSetting_domain.BondUserSetting, error) {
	if g.arrived.Add(1) == g.parties {
		close(g.open)
	}
	select {
	case <-g.open:
	case <-time.After(gateTimeout):
		return nil, errors.New("las ventas concurrentes no llegaron juntas a la cuenta destino")
	}
	return g.BondUserSettingApplicationPorts.GetByUser(ctx, userID)
}

// concurrentSales lanza n ventas a la vez del bono de ownerID y espera a todas.
func concurrentSales(app *bondBound_application.BondBoundApplication, n int) ([]bondBound_domain.SaleResult, []error) {
	results := make([]bondBound_domain.SaleResult, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			results[i], errs[i] = app.Sale(withSession(ownerID, -5), boundID)
		})
	}
	wg.Wait()
	return results, errs
}

// Un bono propio iniciado, vendido el día 50, queda vendido con soldDate; se acredita +1025 con descriptionId 8
// en la cuenta seleccionada y la respuesta trae el disponible actualizado. CA-05.7
func TestSaleCommits(t *testing.T) {
	saleSetup(t, saleBoundDoc(boundID, bondBound_domain.StatusInitiated, 50))
	app, _ := saleApp()
	before := time.Now()

	result, err := app.Sale(withSession(ownerID, -5), boundID)

	after := time.Now()
	if err != nil {
		t.Fatalf("Sale: %v", err)
	}
	want := bondBound_domain.SaleResult{
		BondBoundID: boundID,
		StatusID:    bondBound_domain.StatusSold,
		Amount:      1025,
		Profit:      25,
		AccountID:   accountID,
		Available:   6025,
	}
	if result != want {
		t.Errorf("Sale devolvió %+v, se esperaba %+v", result, want)
	}
	b := onlyBound(t)
	if b.StatusID != bondBound_domain.StatusSold || b.TookDate != nil {
		t.Errorf("el bono quedó en %q con tookDate %v, se esperaba vendido sin tookDate", b.StatusID, b.TookDate)
	}
	assertDateBetween(t, "soldDate", b.SoldDate, before, after)
	assertCredit(t, 1025, shared_domain.BalanceDescriptionBondSaleMC, `Bono "BP12-1" vendido.`)
	if available := availableOf(t, accountID); available != 6025 {
		t.Errorf("la cuenta quedó en %v, se esperaba 6025", available)
	}
}

// Un bono propio madurado se canjea: queda canjeado con tookDate y se acredita el monto completo, 1100, con
// descriptionId 6. CA-05.8
func TestSaleTook(t *testing.T) {
	doc := saleBoundDoc(boundID, bondBound_domain.StatusFinalized, 100)
	doc["matured"] = true
	saleSetup(t, doc)
	app, _ := saleApp()
	before := time.Now()

	result, err := app.Sale(withSession(ownerID, -5), boundID)

	after := time.Now()
	if err != nil {
		t.Fatalf("Sale: %v", err)
	}
	want := bondBound_domain.SaleResult{
		BondBoundID: boundID,
		StatusID:    bondBound_domain.StatusTook,
		Amount:      1100,
		Profit:      100,
		AccountID:   accountID,
		Available:   6100,
	}
	if result != want {
		t.Errorf("Sale devolvió %+v, se esperaba %+v", result, want)
	}
	b := onlyBound(t)
	if b.StatusID != bondBound_domain.StatusTook || b.SoldDate != nil {
		t.Errorf("el bono quedó en %q con soldDate %v, se esperaba canjeado sin soldDate", b.StatusID, b.SoldDate)
	}
	assertDateBetween(t, "tookDate", b.TookDate, before, after)
	assertCredit(t, 1100, shared_domain.BalanceDescriptionBondMC, `Bono "BP12-1" canjeado.`)
	histories := findAll[bondHistory_domain.BondHistory](t, historyCollection)
	if len(histories) != 1 || !slices.Equal(histories[0].Params, []string{"2", "6"}) {
		t.Errorf("historial %+v, se esperaba el cambio de '2' a '6'", histories)
	}
}

// De dos ventas concurrentes del mismo bono, una se confirma y la otra es ErrBondAlreadySold, que responde
// 409: hay un solo crédito, la cuenta queda en 6025 y el historial tiene un solo cambio de estado. CA-05.13
func TestSaleConcurrent(t *testing.T) {
	saleSetup(t, saleBoundDoc(boundID, bondBound_domain.StatusInitiated, 50))
	app, _ := saleApp()
	app.UserSettings = newGatedUserSettings(app.UserSettings, 2)

	results, errs := concurrentSales(app, 2)

	var sold []bondBound_domain.SaleResult
	var failed []error
	for i, err := range errs {
		if err != nil {
			failed = append(failed, err)
		} else {
			sold = append(sold, results[i])
		}
	}
	if len(sold) != 1 || len(failed) != 1 {
		t.Fatalf("se confirmaron %d ventas y fallaron %d (%v), se esperaba una de cada una", len(sold), len(failed), errs)
	}
	if !errors.Is(failed[0], bondBound_domain.ErrBondAlreadySold) {
		t.Errorf("la venta rechazada falló con %v, se esperaba ErrBondAlreadySold", failed[0])
	}
	if status := bondBound_domain.MapToHTTPStatus(failed[0]); status != http.StatusConflict {
		t.Errorf("la venta rechazada respondería %d, se esperaba 409", status)
	}
	if sold[0].Available != 6025 {
		t.Errorf("la venta confirmada devolvió %v disponibles, se esperaba 6025", sold[0].Available)
	}
	if b := onlyBound(t); b.StatusID != bondBound_domain.StatusSold {
		t.Errorf("el bono quedó en %q, se esperaba vendido", b.StatusID)
	}
	assertCredit(t, 1025, shared_domain.BalanceDescriptionBondSaleMC, `Bono "BP12-1" vendido.`)
	if available, histories := availableOf(t, accountID), countDocs(t, historyCollection); available != 6025 || histories != 1 {
		t.Errorf("la cuenta quedó en %v con %d historiales, se esperaban 6025 y 1", available, histories)
	}
}

// La venta registra en bondHistory {typeId '2', text 'BOND_BOUND_STATUS_CHANGED', params ['1','5']}, con params
// como array de strings, y la consulta del gateway por ese texto y params de tamaño 2 lo encuentra. CA-05.14
func TestSaleHistoryForGateway(t *testing.T) {
	saleSetup(t, saleBoundDoc(boundID, bondBound_domain.StatusInitiated, 50))
	app, _ := saleApp()

	if _, err := app.Sale(withSession(ownerID, -5), boundID); err != nil {
		t.Fatalf("Sale: %v", err)
	}

	histories := findAll[bondHistory_domain.BondHistory](t, historyCollection)
	if len(histories) != 1 {
		t.Fatalf("hay %d bondHistory, se esperaba 1", len(histories))
	}
	raw := rawDoc(t, historyCollection, histories[0].ID)
	typeID, _ := raw.Lookup("typeId").StringValueOK()
	text, _ := raw.Lookup("text").StringValueOK()
	relatedID, _ := raw.Lookup("relatedId").StringValueOK()
	if typeID != "2" || text != "BOND_BOUND_STATUS_CHANGED" || relatedID != boundID {
		t.Errorf("typeId %q, text %q y relatedId %q, se esperaban \"2\", \"BOND_BOUND_STATUS_CHANGED\" y %q",
			typeID, text, relatedID, boundID)
	}
	array, ok := raw.Lookup("params").ArrayOK()
	if !ok {
		t.Fatalf("params no es un array: %v", raw.Lookup("params"))
	}
	values, err := array.Values()
	if err != nil {
		t.Fatalf("no se pudo leer params: %v", err)
	}
	var params []string
	for _, v := range values {
		s, ok := v.StringValueOK()
		if !ok {
			t.Fatalf("params tiene un %v, se esperaban strings: %v", v.Type, array)
		}
		params = append(params, s)
	}
	if !slices.Equal(params, []string{"1", "5"}) {
		t.Errorf("params %v, se esperaba [1 5]", params)
	}

	n, err := testDB.CountDocuments(context.Background(), historyCollection,
		bson.M{"text": "BOND_BOUND_STATUS_CHANGED", "params": bson.M{"$size": 2}})
	if err != nil || n != 1 {
		t.Errorf("la consulta del gateway encontró %d (%v), se esperaba 1", n, err)
	}
}

// El aviso y el email de la venta llevan la fecha de compra del bono, no la de hoy, y el disponible que quedó
// después del depósito, sumado una sola vez. Si ms-notification responde 500, la venta igual se confirma.
// CA-05.15
func TestSaleNotifications(t *testing.T) {
	t.Run("aviso y email", func(t *testing.T) {
		doc := saleBoundDoc(boundID, bondBound_domain.StatusInitiated, 50)
		saleSetup(t, doc)
		app, notifications := saleApp()

		if _, err := app.Sale(withSession(ownerID, -5), boundID); err != nil {
			t.Fatalf("Sale: %v", err)
		}

		purchased := doc["initialDate"].(time.Time).In(time.FixedZone("", -5*3600)).Format(noticeDateLayout)
		notices := findAll[shared_domain.NoticeRecord](t, noticeCollection)
		if len(notices) != 1 {
			t.Fatalf("hay %d avisos, se esperaba 1", len(notices))
		}
		n := notices[0]
		if n.TypeID != string(shared_domain.NoticeTypeBond) || n.UserID != ownerID || n.Title != "Bono BP12-1 vendido" {
			t.Errorf("aviso con typeId %q, userId %q y título %q, se esperaban \"3\", %q y \"Bono BP12-1 vendido\"",
				n.TypeID, n.UserID, n.Title, ownerID)
		}
		for _, part := range []string{
			"<strong>Bono:</strong> Bono Plus 12M <br>",
			"<strong>Fecha de compra:</strong> " + purchased + " <br>",
			"<strong>Ganancia:</strong> 25.00 <br>",
			"<strong>Monto:</strong> 1,025.00 <br>",
			"<strong>Disponible Actual:</strong> 6,025.00 <br>",
		} {
			if !strings.Contains(n.Message, part) {
				t.Errorf("el aviso no tiene %q:\n%s", part, n.Message)
			}
		}

		email := assertSentOnce(t, notifications.wait(t, 1), "notify-blooming-bondSold", "user-ids")
		if email["initialDate"] != purchased || email["credited"] != "1,025.00" || email["profit"] != "25.00" ||
			email["available"] != "6,025.00" {
			t.Errorf("bondSold con %v, se esperaban initialDate %q, credited 1,025.00, profit 25.00 y available 6,025.00",
				email, purchased)
		}
	})

	t.Run("ms-notification falla", func(t *testing.T) {
		saleSetup(t, saleBoundDoc(boundID, bondBound_domain.StatusInitiated, 50))
		paths := make(chan string, 10)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths <- r.URL.Path
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(srv.Close)
		app, _ := saleApp()
		app.Notifications = shared_notification.NewNotificationClient(srv.URL, "s3cr3t-notif-valor")

		result, err := app.Sale(withSession(ownerID, -5), boundID)

		if err != nil {
			t.Fatalf("Sale = %v, se esperaba que la venta se confirmara", err)
		}
		if result.StatusID != bondBound_domain.StatusSold || result.Available != 6025 ||
			onlyBound(t).StatusID != bondBound_domain.StatusSold || availableOf(t, accountID) != 6025 {
			t.Errorf("Sale devolvió %+v, se esperaba el bono vendido y la cuenta en 6025", result)
		}
		select {
		case p := <-paths:
			if p != "/notify/user-ids/notify-blooming-bondSold" {
				t.Errorf("ms-notification recibió %s, se esperaba /notify/user-ids/notify-blooming-bondSold", p)
			}
		case <-time.After(notifyTimeout):
			t.Fatal("ms-notification no recibió el email de la venta")
		}
	})
}

// Un bono comprado en Meteor, con _id de 17 caracteres, sin keySeq y con numéricos int32 o string, se vende:
// el documento conserva su _id y cambia de estado, y el bondHistory nuevo, con _id UUID, apunta al _id
// legacy. CA-05.16
func TestSaleLegacyBound(t *testing.T) {
	doc := saleBoundDoc(legacyBoundID, bondBound_domain.StatusInitiated, 50)
	delete(doc, "keySeq")
	maps.Copy(doc, bson.M{
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
	saleSetup(t, doc)
	app, _ := saleApp()

	result, err := app.Sale(withSession(ownerID, -5), legacyBoundID)

	if err != nil {
		t.Fatalf("Sale: %v", err)
	}
	if result.BondBoundID != legacyBoundID || result.StatusID != bondBound_domain.StatusSold || result.Amount != 1025 {
		t.Errorf("Sale devolvió %+v, se esperaba %s vendido por 1025", result, legacyBoundID)
	}
	if b := onlyBound(t); b.ID != legacyBoundID || b.StatusID != bondBound_domain.StatusSold || b.SoldDate == nil {
		t.Errorf("bono %s en %q con soldDate %v, se esperaba %s vendido con soldDate", b.ID, b.StatusID, b.SoldDate, legacyBoundID)
	}
	if _, err := rawDoc(t, boundCollection, legacyBoundID).LookupErr("keySeq"); err == nil {
		t.Error("la venta le agregó keySeq al bono legacy")
	}
	assertCredit(t, 1025, shared_domain.BalanceDescriptionBondSaleMC, `Bono "BP12-1" vendido.`)

	histories := findAll[bondHistory_domain.BondHistory](t, historyCollection)
	if len(histories) != 1 {
		t.Fatalf("hay %d bondHistory, se esperaba 1", len(histories))
	}
	if h := histories[0]; shared_domain.IDFormat(h.ID) != shared_domain.IDKindUUID || h.RelatedID != legacyBoundID {
		t.Errorf("bondHistory con _id %q y relatedId %q, se esperaban un UUID v4 y %q", h.ID, h.RelatedID, legacyBoundID)
	}
}
