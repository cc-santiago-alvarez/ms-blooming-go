package bondBound_application_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	bondBound_application "github.com/futurapps/ms-blooming-go/internal/core/bondBound/application"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_notification "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/notification"
)

const (
	adminEmail = "admin@futurapps.com"

	// maskedAccount es el número de ownAccount enmascarado.
	maskedAccount = "****************0708"

	// adminDateLayout es el formato de la fecha del email de admin.
	adminDateLayout = "2006-01-02 15:04:05"
)

// cellPattern lee el contenido de una celda de la tabla del aviso de compra masiva.
var cellPattern = regexp.MustCompile(`<td[^>]*>([^<]*)</td>`)

// notifyApp arma una compra posible de ownerID, con el perfil "Laura Gómez" y un destinatario
// administrativo.
func notifyApp() (*fakes, *bondBound_application.BondBoundApplication) {
	f := purchaseFakes(ownAccount(), purchasableObject())
	f.profiles.profiles[ownerID] = shared_domain.PublicProfile{ID: ownerID, FirstName: "Laura", LastName: "Gómez"}
	app := f.app()
	app.AdminEmails = []string{adminEmail}
	return f, app
}

// namedSession es un context con la sesión de ownerID, con su token, en la agencia A, con nombre y en UTC-5.
func namedSession() context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID:    ownerID,
		Token:     "tok-" + ownerID,
		AgencyID:  agencyA,
		FirstName: "Sesión",
		LastName:  "Usuario",
		UtcDiff:   session_domain.OptionalUtcDiff{Hours: -5, Valid: true},
	})
}

// oneBondParams es la compra de un bono del producto con la cuenta.
var oneBondParams = bondBound_domain.PurchaseParams{AccountID: accountID, BondObjectID: objectID}

// assertSentOnce verifica que name se envió una sola vez al grupo y devuelve sus parámetros.
func assertSentOnce(t *testing.T, sent []sentNotification, name, group string) map[string]any {
	t.Helper()
	var found []sentNotification
	for _, n := range sent {
		if n.name == name {
			found = append(found, n)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s se envió %d veces, se esperaba una: %+v", name, len(found), sent)
	}
	if found[0].group != group {
		t.Errorf("%s se envió al grupo %q, se esperaba %q", name, found[0].group, group)
	}
	return found[0].params
}

// assertAdminDate verifica que la fecha del email de admin sea la de la compra en UTC-5 y la quita de params
// para comparar el resto.
func assertAdminDate(t *testing.T, params map[string]any, before, after time.Time) {
	t.Helper()
	raw, _ := params["date"].(string)
	date, err := time.ParseInLocation(adminDateLayout, raw, time.FixedZone("", -5*3600))
	if err != nil || date.Before(before.Truncate(time.Second)) || date.After(after) {
		t.Errorf("date %q, se esperaba la hora de la compra en UTC-5 con formato %s", raw, adminDateLayout)
	}
	delete(params, "date")
}

// Una compra individual guarda el aviso del bono (typeId 3) y el del retiro (typeId 1) con los textos del
// legado, y manda el email del bono y los del retiro al comprador y al área administrativa. CA-04.11
func TestPurchaseNotifications(t *testing.T) {
	f, app := notifyApp()
	before := time.Now()

	result, err := app.Purchase(namedSession(), oneBondParams)

	after := time.Now()
	if err != nil {
		t.Fatalf("Purchase: %v", err)
	}
	want := []shared_domain.Notice{
		{
			Type:     "3",
			UserID:   ownerID,
			AgencyID: agencyA,
			Title:    "Compra de Bono (BP12-1), por $ 1,000.00",
			Message: "Gracias, <b>Laura Gómez</b>, has adquirido el bono <b>BP12-1</b> por un monto total de $ 1,000.00<br/></br>" +
				"El sistema automaticamente le descontara el dinero de la cuenta <b>" + maskedAccount + "</b>. " +
				"Dicho dinero será descontado de su nómina.",
		},
		{
			Type:     "1",
			UserID:   ownerID,
			AgencyID: agencyA,
			Title:    "Nuevo Retiro",
			Message: "Señor(a) <strong>Laura Gómez</strong>, por medio de la presente le informamos que se ha realizado un " +
				"retiro de su cuenta número <strong>" + maskedAccount + "</strong> por un monto de <strong>$ 1,000.00</strong> " +
				"en nuestro sistema de BALANCE.",
		},
	}
	if !reflect.DeepEqual(f.notices.added, want) {
		t.Errorf("avisos\n%+v\nse esperaban\n%+v", f.notices.added, want)
	}

	sent := f.notifications.wait(t, 3)
	if len(sent) != 3 {
		t.Fatalf("se enviaron %d notificaciones, se esperaban 3: %+v", len(sent), sent)
	}
	bond := assertSentOnce(t, sent, "notify-blooming-bondPurchased", "user-ids")
	wantBond := map[string]any{
		"userIds":             []string{ownerID},
		"key":                 result.Key,
		"amount":              "1,000.00",
		"accountNumberMasked": maskedAccount,
	}
	if !reflect.DeepEqual(bond, wantBond) {
		t.Errorf("bondPurchased con %v, se esperaba %v", bond, wantBond)
	}
	user := assertSentOnce(t, sent, "notify-blooming-withdrawUser", "user-ids")
	wantUser := map[string]any{
		"userIds":             []string{ownerID},
		"accountNumberMasked": maskedAccount,
		"amount":              "1,000.00",
		"keys":                []string{"BP12-1"},
		"unitPrice":           "1,000.00",
	}
	if !reflect.DeepEqual(user, wantUser) {
		t.Errorf("withdrawUser con %v, se esperaba %v", user, wantUser)
	}
	admin := assertSentOnce(t, sent, "notify-blooming-withdrawAdmin", "to-raw-emails")
	assertAdminDate(t, admin, before, after)
	wantAdmin := map[string]any{
		"emails":              []string{adminEmail},
		"userName":            "Laura Gómez",
		"accountNumberMasked": maskedAccount,
		"amount":              "1,000.00",
		"references":          []string{"REF001"},
		"keys":                []string{"BP12-1"},
		"unitPrice":           "1,000.00",
	}
	if !reflect.DeepEqual(admin, wantAdmin) {
		t.Errorf("withdrawAdmin con %v, se esperaba %v", admin, wantAdmin)
	}
}

// Si ms-notification responde 500, la compra igual se confirma y responde el bono: las tres notificaciones
// se intentan y el fallo no llega al llamador. CA-04.11
func TestPurchaseNotificationServiceFails(t *testing.T) {
	paths := make(chan string, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.Path
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	f, app := notifyApp()
	app.Notifications = shared_notification.NewNotificationClient(srv.URL, "s3cr3t-notif-valor")

	result, err := app.Purchase(namedSession(), oneBondParams)

	if err != nil {
		t.Fatalf("Purchase = %v, se esperaba que la compra se confirmara", err)
	}
	if result.Key != "BP12-1" || result.AccountAvailable != 4000 || len(f.repo.inserted) != 1 || len(f.balance.withdrawals) != 1 {
		t.Errorf("Purchase devolvió %+v con %d bonos y %d retiros, se esperaba BP12-1 con 4000, un bono y un retiro",
			result, len(f.repo.inserted), len(f.balance.withdrawals))
	}
	var got []string
	for range 3 {
		select {
		case p := <-paths:
			got = append(got, p)
		case <-time.After(notifyTimeout):
			t.Fatalf("ms-notification recibió %v, se esperaban 3 notificaciones", got)
		}
	}
	slices.Sort(got)
	want := []string{
		"/notify/to-raw-emails/notify-blooming-withdrawAdmin",
		"/notify/user-ids/notify-blooming-bondPurchased",
		"/notify/user-ids/notify-blooming-withdrawUser",
	}
	if !slices.Equal(got, want) {
		t.Errorf("ms-notification recibió %v, se esperaba %v", got, want)
	}
}

// Si no se puede leer el perfil del comprador, la compra se confirma y los avisos y el email de admin usan
// el nombre de la sesión. CA-04.11
func TestPurchaseNotificationsProfileFails(t *testing.T) {
	f, app := notifyApp()
	f.profiles.err = errors.New("profile no responde")

	if _, err := app.Purchase(namedSession(), oneBondParams); err != nil {
		t.Fatalf("Purchase = %v, se esperaba que la compra se confirmara", err)
	}

	if len(f.notices.added) != 2 ||
		!strings.Contains(f.notices.added[0].Message, "<b>Sesión Usuario</b>") ||
		!strings.Contains(f.notices.added[1].Message, "<strong>Sesión Usuario</strong>") {
		t.Errorf("avisos %+v, se esperaba el nombre de la sesión", f.notices.added)
	}
	admin := assertSentOnce(t, f.notifications.wait(t, 3), "notify-blooming-withdrawAdmin", "to-raw-emails")
	if admin["userName"] != "Sesión Usuario" {
		t.Errorf("withdrawAdmin con userName %v, se esperaba el nombre de la sesión", admin["userName"])
	}
}

// Los avisos escapan el HTML del nombre del comprador. CA-04.11
func TestPurchaseNotificationsEscapeHTML(t *testing.T) {
	f, app := notifyApp()
	f.profiles.profiles[ownerID] = shared_domain.PublicProfile{ID: ownerID, FirstName: "Ana", LastName: "<i>Pérez</i> & Cía"}

	if _, err := app.Purchase(namedSession(), oneBondParams); err != nil {
		t.Fatalf("Purchase: %v", err)
	}

	escaped := "Ana &lt;i&gt;Pérez&lt;/i&gt; &amp; Cía"
	if len(f.notices.added) != 2 {
		t.Fatalf("se guardaron %d avisos, se esperaban 2", len(f.notices.added))
	}
	for _, n := range f.notices.added {
		if !strings.Contains(n.Message, escaped) || strings.Contains(n.Message, "<i>") {
			t.Errorf("aviso %q, se esperaba el nombre escapado %q", n.Message, escaped)
		}
	}
}

// Una compra masiva de 3 guarda un solo aviso de bono, con una fila por bono que lleva la referencia de su
// retiro, y un aviso de retiro con las 3 claves, y manda un solo email al área administrativa. CA-04.12
func TestBulkPurchaseNotifications(t *testing.T) {
	keys := []string{"BP12-1", "BP12-2", "BP12-3"}
	// referencias que fakeBalance da a cada retiro, en orden
	references := []string{"REF001", "REF002", "REF003"}
	f, app := notifyApp()
	before := time.Now()

	result, err := app.BulkPurchase(namedSession(), bondBound_domain.BulkPurchaseParams{PurchaseParams: oneBondParams, Quantity: 3})

	after := time.Now()
	if err != nil {
		t.Fatalf("BulkPurchase: %v", err)
	}
	if len(f.notices.added) != 2 {
		t.Fatalf("se guardaron %d avisos, se esperaban 2: %+v", len(f.notices.added), f.notices.added)
	}

	bondNotice := f.notices.added[0]
	if bondNotice.Type != shared_domain.NoticeTypeBond || bondNotice.Title != "Compra de 3 Bonos, por $ 3,000.00" {
		t.Errorf("aviso de bono de tipo %q con título %q, se esperaba el tipo 3 y \"Compra de 3 Bonos, por $ 3,000.00\"",
			bondNotice.Type, bondNotice.Title)
	}
	for _, part := range []string{
		"Felicidades <strong>Laura Gómez</strong>, has adquirido <strong>3</strong> bonos por un monto total de " +
			"$ <strong>3,000.00</strong> con un valor de $ <strong>1,000.00</strong> c/u.",
		"de la cuenta <strong>" + maskedAccount + "</strong>",
	} {
		if !strings.Contains(bondNotice.Message, part) {
			t.Errorf("el aviso de bono no contiene %q:\n%s", part, bondNotice.Message)
		}
	}
	rows := tableRows(t, bondNotice.Message)
	if len(rows) != 3 {
		t.Fatalf("la tabla tiene %d filas, se esperaban 3: %v", len(rows), rows)
	}
	for i, row := range rows {
		want := []string{strconv.Itoa(i + 1), keys[i], references[i], "$ 1,000.00"}
		if !slices.Equal(row, want) || result.Items[i].Reference != references[i] {
			t.Errorf("fila %d %v con el item %+v, se esperaba %v con la referencia del retiro %d",
				i+1, row, result.Items[i], want, i+1)
		}
	}

	balanceNotice := f.notices.added[1]
	wantBalance := "Señor(a) <strong>Laura Gómez</strong>, por medio de la presente le informamos que se ha realizado " +
		"el descuento automatico de su cuenta número <strong>" + maskedAccount + "</strong> por un monto de " +
		"<strong>$ 3,000.00</strong> por la compra de los siguentes bonos: <strong>BP12-1, BP12-2, BP12-3</strong>; " +
		"con un valor de <strong>$ 1,000.00</strong> c/u."
	if balanceNotice.Type != shared_domain.NoticeTypeBalance || balanceNotice.Title != "Nuevo Retiro" || balanceNotice.Message != wantBalance {
		t.Errorf("aviso de retiro %+v, se esperaba el tipo 1 con \"Nuevo Retiro\" y el mensaje\n%s", balanceNotice, wantBalance)
	}

	sent := f.notifications.wait(t, 3)
	if len(sent) != 3 {
		t.Fatalf("se enviaron %d notificaciones, se esperaban 3: %+v", len(sent), sent)
	}
	bulk := assertSentOnce(t, sent, "notify-blooming-bondBulkPurchased", "user-ids")
	wantBulk := map[string]any{
		"userIds":             []string{ownerID},
		"quantity":            3,
		"total":               "3,000.00",
		"unitPrice":           "1,000.00",
		"accountNumberMasked": maskedAccount,
		"rows": []map[string]any{
			{"key": keys[0], "reference": references[0], "amount": "1,000.00"},
			{"key": keys[1], "reference": references[1], "amount": "1,000.00"},
			{"key": keys[2], "reference": references[2], "amount": "1,000.00"},
		},
	}
	if !reflect.DeepEqual(bulk, wantBulk) {
		t.Errorf("bondBulkPurchased con %v, se esperaba %v", bulk, wantBulk)
	}
	user := assertSentOnce(t, sent, "notify-blooming-withdrawUser", "user-ids")
	if user["amount"] != "3,000.00" || !reflect.DeepEqual(user["keys"], keys) {
		t.Errorf("withdrawUser con %v, se esperaban 3,000.00 y las claves %v", user, keys)
	}
	admin := assertSentOnce(t, sent, "notify-blooming-withdrawAdmin", "to-raw-emails")
	assertAdminDate(t, admin, before, after)
	wantAdmin := map[string]any{
		"emails":              []string{adminEmail},
		"userName":            "Laura Gómez",
		"accountNumberMasked": maskedAccount,
		"amount":              "3,000.00",
		"references":          references,
		"keys":                keys,
		"unitPrice":           "1,000.00",
	}
	if !reflect.DeepEqual(admin, wantAdmin) {
		t.Errorf("withdrawAdmin con %v, se esperaba %v", admin, wantAdmin)
	}
}

// tableRows devuelve las celdas de cada fila del cuerpo de la tabla del aviso.
func tableRows(t *testing.T, message string) [][]string {
	t.Helper()
	_, rest, ok := strings.Cut(message, "<tbody>")
	body, _, closed := strings.Cut(rest, "</tbody>")
	if !ok || !closed {
		t.Fatalf("el aviso no tiene <tbody>:\n%s", message)
	}
	var rows [][]string
	for _, row := range strings.Split(body, "<tr>")[1:] {
		var cells []string
		for _, m := range cellPattern.FindAllStringSubmatch(row, -1) {
			cells = append(cells, m[1])
		}
		rows = append(rows, cells)
	}
	return rows
}
