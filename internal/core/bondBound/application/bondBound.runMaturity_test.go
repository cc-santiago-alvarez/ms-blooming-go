package bondBound_application_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_notification "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/notification"
)

const (
	flexTypeID   = "8b7a6c5d-4e3f-4a2b-9c1d-0e9f8a7b6c5d"
	flexObjectID = "2d3e4f5a-6b7c-4d8e-9f0a-1b2c3d4e5f6a"
)

// maturityNow es el momento de las corridas de maduración.
var maturityNow = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

// maturityID es el _id del bono n; los ids se ordenan como n.
func maturityID(n int) string {
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", n)
}

// dueBound es el bono iniciado n de userID, del producto objectID y del tipo typeID, con la clave BP12-n, que
// vence days días después de maturityNow.
func dueBound(n int, userID string, days int) *bondBound_domain.BondBound {
	b := storedBound(maturityID(n))
	b.Key = fmt.Sprintf("BP12-%d", n)
	b.UserID = userID
	b.BondObjectID = objectID
	b.BondTypeID = typeID
	b.MaturityDate = maturityNow.AddDate(0, 0, days)
	return b
}

// maturityFakes arma las dependencias con los bonos dados, el producto "Bono Plus 12M" y el tipo "Renta fija".
func maturityFakes(bounds ...*bondBound_domain.BondBound) *fakes {
	f := newFakes(bounds...)
	f.objects.objects[objectID] = purchasableObject()
	f.types.types[typeID] = &bondType_domain.BondType{ID: typeID, Name: "Renta fija"}
	return f
}

// changedIDs devuelve los bonos de cada llamada a UpdateStatusIf, en orden.
func changedIDs(f *fakes) []string {
	ids := make([]string, len(f.repo.changes))
	for i, c := range f.repo.changes {
		ids[i] = c.id
	}
	return ids
}

// historyIDs devuelve el relatedId de cada historial registrado, en orden.
func historyIDs(f *fakes) []string {
	ids := make([]string, len(f.history.added))
	for i, h := range f.history.added {
		ids[i] = h.RelatedID
	}
	return ids
}

// Entre bonos vencidos ayer en estado vendido, desligado, borrados o con soldDate, y uno iniciado que vence
// mañana, solo madura el iniciado vencido: pasa a '2' con matured y maturedAt, y registra su historial. CA-10.3
func TestRunMaturityOnlyDueInitiatedBounds(t *testing.T) {
	yesterday := maturityNow.AddDate(0, 0, -1)
	a := dueBound(1, ownerID, -1)
	b := dueBound(2, ownerID, 1)
	c := dueBound(3, ownerID, -1)
	c.StatusID = bondBound_domain.StatusSold
	d := dueBound(4, ownerID, -1)
	d.StatusID = bondBound_domain.StatusUnbound
	e := dueBound(5, ownerID, -1)
	e.IsRemove = true
	sold := dueBound(6, ownerID, -1)
	sold.SoldDate = &yesterday
	f := maturityFakes(a, b, c, d, e, sold)

	result, err := f.app().RunMaturity(context.Background(), maturityNow)

	if err != nil {
		t.Fatalf("RunMaturity: %v", err)
	}
	if want := (bondBound_domain.MaturityResult{Matured: 1, Users: 1, Failed: 0}); result != want {
		t.Errorf("RunMaturity = %+v, se esperaba %+v", result, want)
	}
	wantQueries := []repoQuery{{method: "FindMaturable", now: maturityNow, limit: 500}}
	if !reflect.DeepEqual(f.repo.queries, wantQueries) {
		t.Errorf("consultas %+v, se esperaba %+v", f.repo.queries, wantQueries)
	}

	if len(f.repo.changes) != 1 {
		t.Fatalf("cambios de estado %+v, se esperaba solo el de A", f.repo.changes)
	}
	change := f.repo.changes[0]
	if change.id != a.ID || change.change.From != bondBound_domain.StatusInitiated || change.change.To != bondBound_domain.StatusFinalized ||
		change.change.MaturedAt == nil || !change.change.MaturedAt.Equal(maturityNow) ||
		change.change.SoldDate != nil || change.change.TookDate != nil {
		t.Errorf("cambio %+v, se esperaba A de '1' a '2' con maturedAt %s", change, maturityNow)
	}

	wantStatus := map[string]bondBound_domain.BondBoundStatus{
		a.ID: bondBound_domain.StatusFinalized, b.ID: bondBound_domain.StatusInitiated, c.ID: bondBound_domain.StatusSold,
		d.ID: bondBound_domain.StatusUnbound, e.ID: bondBound_domain.StatusInitiated, sold.ID: bondBound_domain.StatusInitiated,
	}
	for id, status := range wantStatus {
		stored := f.repo.bounds[id]
		if stored.StatusID != status || stored.Matured != (id == a.ID) {
			t.Errorf("bono %s en %q con matured %v, se esperaba %q con matured %v", id, stored.StatusID, stored.Matured, status, id == a.ID)
		}
	}

	wantHistory := []bondHistory_domain.AddParams{{
		RelatedID: a.ID,
		TypeID:    bondHistory_domain.HistoryBondBound,
		Text:      shared_domain.TextBondBoundStatusChanged,
		Params:    []string{"1", "2"},
	}}
	if !reflect.DeepEqual(f.history.added, wantHistory) {
		t.Errorf("historiales %+v, se esperaba %+v", f.history.added, wantHistory)
	}
}

// Con 501 bonos vencidos lee un lote de 500 y otro desde el último _id del primero, y maduran los 501. CA-10.3
func TestRunMaturityInBatches(t *testing.T) {
	bounds := make([]*bondBound_domain.BondBound, 501)
	for i := range bounds {
		bounds[i] = dueBound(i+1, ownerID, -1)
	}
	f := maturityFakes(bounds...)

	result, err := f.app().RunMaturity(context.Background(), maturityNow)

	if err != nil {
		t.Fatalf("RunMaturity: %v", err)
	}
	if want := (bondBound_domain.MaturityResult{Matured: 501, Users: 1, Failed: 0}); result != want {
		t.Errorf("RunMaturity = %+v, se esperaba %+v", result, want)
	}
	wantQueries := []repoQuery{
		{method: "FindMaturable", now: maturityNow, limit: 500},
		{method: "FindMaturable", now: maturityNow, limit: 500, afterID: maturityID(500)},
	}
	if !reflect.DeepEqual(f.repo.queries, wantQueries) {
		t.Errorf("consultas %+v, se esperaba %+v", f.repo.queries, wantQueries)
	}
	if len(f.repo.changes) != 501 || len(f.history.added) != 501 {
		t.Errorf("%d cambios de estado y %d historiales, se esperaban 501 de cada uno", len(f.repo.changes), len(f.history.added))
	}
}

// Un dueño con 3 bonos madurados y otro usuario con 1 reciben un aviso y un email cada uno, con una fila por
// bono (clave, tipo y nombre). El aviso va en la agencia del perfil o, sin perfil, en la del primer bono. Los
// tipos, los productos y los perfiles se leen una vez por corrida. CA-10.6
func TestRunMaturityNotifiesEachOwnerOnce(t *testing.T) {
	flex := dueBound(4, ownerID, -1)
	flex.BondTypeID = flexTypeID
	flex.BondObjectID = flexObjectID
	f := maturityFakes(dueBound(1, ownerID, -1), dueBound(2, otherUserID, -1), dueBound(3, ownerID, -1), flex)
	f.types.types[flexTypeID] = &bondType_domain.BondType{ID: flexTypeID, Name: "Renta variable"}
	f.objects.objects[flexObjectID] = &bondObject_domain.BondObject{ID: flexObjectID, Name: "Bono Flex 6M"}
	f.profiles.profiles[ownerID] = shared_domain.PublicProfile{ID: ownerID, FirstName: "Laura", LastName: "Gómez", AgencyID: agencyB}

	result, err := f.app().RunMaturity(context.Background(), maturityNow)

	if err != nil {
		t.Fatalf("RunMaturity: %v", err)
	}
	if want := (bondBound_domain.MaturityResult{Matured: 4, Users: 2, Failed: 0}); result != want {
		t.Errorf("RunMaturity = %+v, se esperaba %+v", result, want)
	}

	if len(f.notices.added) != 2 {
		t.Fatalf("se guardaron %d avisos, se esperaban 2: %+v", len(f.notices.added), f.notices.added)
	}
	wantNotices := []struct {
		userID   string
		agencyID string
		rows     [][]string
	}{
		{userID: ownerID, agencyID: agencyB, rows: [][]string{
			{"BP12-1", "Renta fija", "Bono Plus 12M"},
			{"BP12-3", "Renta fija", "Bono Plus 12M"},
			{"BP12-4", "Renta variable", "Bono Flex 6M"},
		}},
		{userID: otherUserID, agencyID: agencyA, rows: [][]string{{"BP12-2", "Renta fija", "Bono Plus 12M"}}},
	}
	for i, want := range wantNotices {
		n := f.notices.added[i]
		if n.Type != shared_domain.NoticeTypeBond || n.UserID != want.userID || n.AgencyID != want.agencyID || n.Title != "Bono Madurado" {
			t.Errorf("aviso %d de tipo %q para %s en %s con título %q, se esperaba el tipo 3 para %s en %s con \"Bono Madurado\"",
				i+1, n.Type, n.UserID, n.AgencyID, n.Title, want.userID, want.agencyID)
		}
		if rows := tableRows(t, n.Message); !reflect.DeepEqual(rows, want.rows) {
			t.Errorf("la tabla del aviso %d tiene %v, se esperaba %v", i+1, rows, want.rows)
		}
	}
	if !strings.Contains(f.notices.added[0].Message, "Estimado <strong>Laura Gómez</strong>") {
		t.Errorf("el aviso del dueño no saluda a Laura Gómez:\n%s", f.notices.added[0].Message)
	}

	sent := f.notifications.wait(t, 2)
	wantSent := []sentNotification{
		{group: "user-ids", name: "notify-blooming-bondMatured", params: map[string]any{
			"userIds": []string{ownerID},
			"bonds": []map[string]any{
				{"key": "BP12-1", "type": "Renta fija", "name": "Bono Plus 12M"},
				{"key": "BP12-3", "type": "Renta fija", "name": "Bono Plus 12M"},
				{"key": "BP12-4", "type": "Renta variable", "name": "Bono Flex 6M"},
			},
		}},
		{group: "user-ids", name: "notify-blooming-bondMatured", params: map[string]any{
			"userIds": []string{otherUserID},
			"bonds":   []map[string]any{{"key": "BP12-2", "type": "Renta fija", "name": "Bono Plus 12M"}},
		}},
	}
	if !reflect.DeepEqual(sent, wantSent) {
		t.Errorf("notificaciones\n%+v\nse esperaban\n%+v", sent, wantSent)
	}

	if want := [][]string{{typeID, flexTypeID}}; !reflect.DeepEqual(f.types.requested, want) {
		t.Errorf("se pidieron los tipos %v, se esperaba %v", f.types.requested, want)
	}
	if want := []string{objectID, flexObjectID}; !slices.Equal(f.objects.reads, want) {
		t.Errorf("se leyeron los productos %v, se esperaba %v", f.objects.reads, want)
	}
	if want := [][]string{{ownerID, otherUserID}}; !reflect.DeepEqual(f.profiles.requested, want) {
		t.Errorf("se pidieron los perfiles %v, se esperaba %v", f.profiles.requested, want)
	}
}

// Si ms-notification responde 500, los 3 bonos quedan madurados igual, el aviso se guarda y el email se
// intenta una sola vez. CA-10.6
func TestRunMaturityNotificationServiceFails(t *testing.T) {
	paths := make(chan string, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.Path
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	f := maturityFakes(dueBound(1, ownerID, -1), dueBound(2, ownerID, -1), dueBound(3, ownerID, -1))
	app := f.app()
	app.Notifications = shared_notification.NewNotificationClient(srv.URL, "s3cr3t-notif-valor")

	result, err := app.RunMaturity(context.Background(), maturityNow)

	if err != nil {
		t.Fatalf("RunMaturity = %v, se esperaba que los bonos maduraran", err)
	}
	if want := (bondBound_domain.MaturityResult{Matured: 3, Users: 1, Failed: 0}); result != want {
		t.Errorf("RunMaturity = %+v, se esperaba %+v", result, want)
	}
	for n := 1; n <= 3; n++ {
		if stored := f.repo.bounds[maturityID(n)]; stored.StatusID != bondBound_domain.StatusFinalized || !stored.Matured {
			t.Errorf("bono %d en %q con matured %v, se esperaba madurado", n, stored.StatusID, stored.Matured)
		}
	}
	if len(f.notices.added) != 1 {
		t.Errorf("se guardaron %d avisos, se esperaba 1", len(f.notices.added))
	}

	select {
	case p := <-paths:
		if p != "/notify/user-ids/notify-blooming-bondMatured" {
			t.Errorf("ms-notification recibió %s, se esperaba /notify/user-ids/notify-blooming-bondMatured", p)
		}
	case <-time.After(notifyTimeout):
		t.Fatal("ms-notification no recibió el email de maduración")
	}
	if len(paths) != 0 {
		t.Errorf("ms-notification recibió %d llamadas de más", len(paths))
	}
}

// Si UpdateStatusIf falla para 1 de 5 bonos, los otros 4 maduran y se avisan, el fallido queda iniciado y sin
// historial, el log nombra el bono y el resumen informa failed=1, y el lock se libera. CA-10.7
func TestRunMaturityContinuesAfterFailedBound(t *testing.T) {
	bounds := make([]*bondBound_domain.BondBound, 5)
	for i := range bounds {
		bounds[i] = dueBound(i+1, ownerID, -1)
	}
	failed := maturityID(3)
	f := maturityFakes(bounds...)
	f.repo.statusErrs = map[string]error{failed: errors.New("mongo no responde")}
	logs := captureLog(t)

	result, err := f.app().RunMaturity(context.Background(), maturityNow)

	if err != nil {
		t.Fatalf("RunMaturity = %v, se esperaba que la corrida siguiera", err)
	}
	if want := (bondBound_domain.MaturityResult{Matured: 4, Users: 1, Failed: 1}); result != want {
		t.Errorf("RunMaturity = %+v, se esperaba %+v", result, want)
	}
	all := []string{maturityID(1), maturityID(2), failed, maturityID(4), maturityID(5)}
	if got := changedIDs(f); !slices.Equal(got, all) {
		t.Errorf("se intentó cambiar %v, se esperaba %v", got, all)
	}
	for _, id := range all {
		want := bondBound_domain.StatusFinalized
		if id == failed {
			want = bondBound_domain.StatusInitiated
		}
		if stored := f.repo.bounds[id]; stored.StatusID != want || stored.Matured != (id != failed) {
			t.Errorf("bono %s en %q con matured %v, se esperaba %q", id, stored.StatusID, stored.Matured, want)
		}
	}
	matured := []string{maturityID(1), maturityID(2), maturityID(4), maturityID(5)}
	if got := historyIDs(f); !slices.Equal(got, matured) {
		t.Errorf("historiales de %v, se esperaba %v", got, matured)
	}

	if len(f.notices.added) != 1 {
		t.Fatalf("se guardaron %d avisos, se esperaba 1", len(f.notices.added))
	}
	var keys []string
	for _, row := range tableRows(t, f.notices.added[0].Message) {
		keys = append(keys, row[0])
	}
	if want := []string{"BP12-1", "BP12-2", "BP12-4", "BP12-5"}; !slices.Equal(keys, want) {
		t.Errorf("el aviso lista %v, se esperaba %v", keys, want)
	}

	for _, line := range []string{
		"no se pudo madurar el bono " + failed + ": mongo no responde",
		"maduración: matured=4 users=1 failed=1 ",
	} {
		if !strings.Contains(logs.String(), line) {
			t.Errorf("el log no contiene %q:\n%s", line, logs)
		}
	}

	if want := []lockCall{{name: "bond-maturity", ttl: 10 * time.Minute}}; !slices.Equal(f.lock.acquired, want) {
		t.Errorf("se tomó el lock %+v, se esperaba %+v", f.lock.acquired, want)
	}
	if want := []string{"bond-maturity"}; !slices.Equal(f.lock.released, want) {
		t.Errorf("se liberó el lock %v, se esperaba %v", f.lock.released, want)
	}
}
