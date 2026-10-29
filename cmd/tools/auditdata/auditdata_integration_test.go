//go:build integration

package main

import (
	"bytes"
	"context"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// runAt es la hora de las corridas de -apply de los tests.
var runAt = time.Date(2026, 10, 29, 15, 4, 5, 123_000_000, time.UTC)

// sorted devuelve una copia ordenada.
func sorted(ids ...string) []string {
	return slices.Sorted(slices.Values(ids))
}

// details devuelve el detalle de cada hallazgo por _id.
func details(issues []docIssue) map[string]string {
	out := map[string]string{}
	for _, is := range issues {
		out[is.ID] = is.Detail
	}
	return out
}

// Un '2' con soldDate se pasa a '5' solo si el historial registra la venta y tiene agencia; si no, o si también
// tiene tookDate, va a revisión manual. Los borrados también cuentan, porque el canje de Meteor los busca por
// _id. Los '6' con las dos fechas se informan aparte.
func TestCollectSold(t *testing.T) {
	setup(t)
	s1, s2, s3 := meteorID("SoldOne"), shared_domain.NewID(), meteorID("SoldThree")
	s4, s5, s6 := shared_domain.NewID(), meteorID("SoldFive"), shared_domain.NewID()
	p1, p2 := meteorID("PaidOne"), shared_domain.NewID()
	tookAt := fixtureAt.AddDate(1, 0, 1)
	insert(t, bondBoundCollection,
		boundDoc(s1, soldExtra(nil)),
		boundDoc(s2, soldExtra(bson.M{"isRemove": true, "userId": user2ID})),
		boundDoc(s3, soldExtra(nil)),
		boundDoc(s4, soldExtra(bson.M{"agencyId": ""})),
		boundDoc(s5, soldExtra(bson.M{"tookDate": tookAt})),
		boundDoc(s6, bson.M{"statusId": "2"}),
		boundDoc(p1, bson.M{"statusId": "6", "soldDate": fixtureAt.AddDate(0, 3, 0), "tookDate": tookAt}),
		boundDoc(p2, bson.M{"statusId": "6", "tookDate": tookAt}),
	)
	insert(t, bondHistoryCollection, saleHistoryDoc(s1), saleHistoryDoc(s2), saleHistoryDoc(s4), saleHistoryDoc(s5))
	insert(t, profileCollection, profileDoc(userID, nil))

	a := mustCollect(t)

	if got := sorted(soldIDs(a.Sold.Fixable)...); !slices.Equal(got, sorted(s1, s2)) {
		t.Errorf("aplicables = %v, se esperaba %v", got, sorted(s1, s2))
	}
	wantManual := map[string]string{
		s3: "el historial no registra la venta: puede no haberse cobrado",
		s4: "sin agencyId para el historial",
		s5: "también tiene tookDate",
	}
	if got := details(a.Sold.Manual); !reflect.DeepEqual(got, wantManual) {
		t.Errorf("manuales = %v, se esperaba %v", got, wantManual)
	}
	if got := soldIDs(a.Sold.Paid); !slices.Equal(got, []string{p1}) {
		t.Errorf("cobrados dos veces = %v, se esperaba [%s]", got, p1)
	}
	if bonds, owners, total := soldSummary(a.Sold.Fixable); bonds != 2 || owners != 2 || total != 2000 {
		t.Errorf("resumen = %d bonos, %d dueños, %v; se esperaba 2, 2, 2000", bonds, owners, total)
	}
	// la auditoría del gateway cuenta cualquier estado con las dos fechas, sin borrados y de dueños con perfil
	if a.Sold.AuditBonds != 2 || a.Sold.AuditOwners != 1 {
		t.Errorf("auditoría del gateway = %d bonos de %d dueños, se esperaban 2 de 1", a.Sold.AuditBonds, a.Sold.AuditOwners)
	}
}

// La primera corrida madura los '1' vivos, sin fechas de venta ni canje, que vencen antes del fin del día UTC de
// la referencia. ms-feed descarta los que no tienen profile, bondType o bondObject, pero no los que apuntan a
// un producto borrado. Un bono que Go no decodifica corta la corrida en su lote.
func TestCollectMaturity(t *testing.T) {
	setup(t)
	insert(t, bondTypeCollection, typeDoc(legacyTypeID, nil), typeDoc(typeID, nil))
	insert(t, bondObjectCollection,
		objectDoc(objectID, legacyTypeID, nil),
		objectDoc(legacyObjectID, typeID, bson.M{"isRemove": true}))
	insert(t, profileCollection, profileDoc(userID, nil), profileDoc(user2ID, nil))

	goneUser := meteorID("GoneUser")
	m1, m2, m3 := meteorID("MatOne"), shared_domain.NewID(), meteorID("MatThree")
	m4, m5, m6 := shared_domain.NewID(), meteorID("MatFive"), shared_domain.NewID()
	m7, m8, m9 := meteorID("MatSeven"), shared_domain.NewID(), meteorID("MatNine")
	insert(t, bondBoundCollection,
		boundDoc(m1, nil),
		boundDoc(m2, bson.M{"bondObjectId": missingID}),
		boundDoc(m3, bson.M{"userId": goneUser}),
		boundDoc(m4, bson.M{"bondObjectId": legacyObjectID, "bondTypeId": typeID, "userId": user2ID}),
		boundDoc(m5, bson.M{"maturityDate": time.Date(2024, 6, 20, 0, 0, 0, 0, time.UTC)}),
		boundDoc(m6, bson.M{"isRemove": true}),
		boundDoc(m7, bson.M{"soldDate": fixtureAt.AddDate(0, 2, 0)}),
		boundDoc(m8, bson.M{"maturityDate": time.Date(2024, 6, 15, 23, 0, 0, 0, time.UTC)}),
		boundDoc(m9, bson.M{"amount": "abc"}),
	)

	m := mustCollect(t).Maturity

	if m.Bonds != 6 || m.Owners != 3 {
		t.Errorf("primera corrida = %d bonos de %d dueños, se esperaba 6 de 3", m.Bonds, m.Owners)
	}
	wantDropped := map[string]string{m2: "sin bondObject", m3: "sin profile"}
	if got := details(m.Dropped); !reflect.DeepEqual(got, wantDropped) || m.DroppedOwners != 2 {
		t.Errorf("descartados por ms-feed = %v de %d dueños, se esperaba %v de 2", got, m.DroppedOwners, wantDropped)
	}
	if !slices.Equal(m.Undecodable, []string{m9}) {
		t.Errorf("no decodifican = %v, se esperaba [%s]", m.Undecodable, m9)
	}
}

// Los strings canónicos de los numéricos se planean por documento y los demás se informan; isPrivateBond que
// no es booleano se planea con el booleano de Go, salvo NaN o un producto sin agencia, que van a revisión
// manual; el campo ausente no se toca, y se avisa del privado sin lista de autorizados que Meteor lista.
func TestCollectPrivateAndNumeric(t *testing.T) {
	setup(t)
	o1, o2, o3 := meteorID("ObjOne"), shared_domain.NewID(), meteorID("ObjThree")
	o4, o5, o6 := shared_domain.NewID(), meteorID("ObjFive"), shared_domain.NewID()
	without := func(doc bson.M, field string) bson.M {
		delete(doc, field)
		return doc
	}
	insert(t, bondObjectCollection,
		objectDoc(o1, legacyTypeID, bson.M{"maturityTimeInDays": "365", "isPrivateBond": legacyTypeID, "authorizedUserIds": bson.A{userID}}),
		objectDoc(o2, typeID, bson.M{"isPrivateBond": "", "amount": "1.000"}),
		objectDoc(o3, typeID, bson.M{"isPrivateBond": math.NaN()}),
		without(objectDoc(o4, typeID, nil), "isPrivateBond"),
		without(objectDoc(o5, typeID, bson.M{"isPrivateBond": true}), "authorizedUserIds"),
		objectDoc(o6, typeID, bson.M{"isPrivateBond": "x", "agencyId": bson.A{}}),
		without(objectDoc(meteorID("ObjSeven"), typeID, bson.M{"isPrivateBond": true, "isRemove": true}), "authorizedUserIds"),
		without(objectDoc(shared_domain.NewID(), typeID, bson.M{"isPrivateBond": true, "active": false}), "authorizedUserIds"),
	)
	n1, n2 := meteorID("NumOne"), shared_domain.NewID()
	terms := earlyCashTermDoc()
	terms[1].Value = "90"
	insert(t, bondBoundCollection,
		boundDoc(n1, bson.M{"maturityTimeInDays": "365", "earlyCashTerm": terms}),
		boundDoc(n2, nil),
	)

	a := mustCollect(t)

	if a.Numeric.count(bondObjectCollection) != 1 || a.Numeric.count(bondBoundCollection) != 1 {
		t.Fatalf("planes = %+v", a.Numeric.Plans)
	}
	for _, p := range a.Numeric.Plans {
		var want []numericChange
		switch stringOf(p.ID) {
		case o1:
			want = []numericChange{{Field: "maturityTimeInDays", Old: "365", New: int32(365)}}
		case n1:
			want = []numericChange{
				{Field: "maturityTimeInDays", Old: "365", New: int32(365)},
				{Field: "earlyCashTerm.inflectionPoint", Old: "90", New: int32(90)},
			}
		}
		if !reflect.DeepEqual(p.Changes, want) {
			t.Errorf("plan de %s = %+v, se esperaba %+v", stringOf(p.ID), p.Changes, want)
		}
	}
	wantNonCanonical := map[string]string{o2: "amount: formato no canónico, 5 caracteres"}
	if got := details(a.Numeric.NonCanonical); !reflect.DeepEqual(got, wantNonCanonical) {
		t.Errorf("no canónicos = %v, se esperaba %v", got, wantNonCanonical)
	}

	targets := map[string]bool{}
	for _, p := range a.Private.Auto {
		targets[stringOf(p.ID)] = p.Target
	}
	if !reflect.DeepEqual(targets, map[string]bool{o1: true, o2: false}) {
		t.Errorf("automáticos = %v", targets)
	}
	wantManual := map[string]string{
		o3: "double NaN; valor que Go y Meteor no leen igual o inesperado",
		o6: "string de 1 carácter; sin agencia para el historial",
	}
	if got := details(a.Private.Manual); !reflect.DeepEqual(got, wantManual) {
		t.Errorf("manuales = %v, se esperaba %v", got, wantManual)
	}
	if a.Private.Missing != 1 || !slices.Equal(issueIDs(a.Private.NoAuthorized), []string{o5}) {
		t.Errorf("sin el campo = %d, privados sin autorizados = %v", a.Private.Missing, issueIDs(a.Private.NoAuthorized))
	}
}

// Las claves repetidas incluyen los bonos borrados, en orden de alta. Los settings vivos de un mismo usuario se
// agrupan del último modificado al primero, y se informan los que no tienen userId o lo tienen como ObjectId.
func TestCollectDuplicates(t *testing.T) {
	setup(t)
	k1, k2, k3 := meteorID("KeyOne"), shared_domain.NewID(), meteorID("KeyThree")
	insert(t, bondBoundCollection,
		boundDoc(k1, bson.M{"key": "BP12-1", "statusId": "6"}),
		boundDoc(k2, bson.M{"key": "BP12-1", "isRemove": true, "createdAt": fixtureAt.Add(time.Hour)}),
		boundDoc(k3, bson.M{"key": "BP12-2"}),
	)
	g1, g2, g3 := meteorID("SetOne"), shared_domain.NewID(), meteorID("SetThree")
	g4, g5, g6 := shared_domain.NewID(), meteorID("SetFive"), shared_domain.NewID()
	insert(t, bondUserSettingCollection,
		settingDoc(g1, userID, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), nil),
		settingDoc(g2, userID, time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC), nil),
		settingDoc(g3, userID, time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC), bson.M{"isRemove": true}),
		settingDoc(g4, nil, fixtureAt, nil),
		settingDoc(g5, bson.NewObjectID(), fixtureAt, nil),
		settingDoc(g6, user2ID, fixtureAt, nil),
	)

	a := mustCollect(t)

	if len(a.Keys) != 1 || a.Keys[0].Key != "BP12-1" {
		t.Fatalf("claves = %+v", a.Keys)
	}
	wantBounds := []docIssue{
		{Collection: bondBoundCollection, ID: k1, Detail: `statusId "6", isRemove false`},
		{Collection: bondBoundCollection, ID: k2, Detail: `statusId "1", isRemove true`},
	}
	if !reflect.DeepEqual(a.Keys[0].Bounds, wantBounds) {
		t.Errorf("bonos de la clave = %+v, se esperaba %+v", a.Keys[0].Bounds, wantBounds)
	}
	if !reflect.DeepEqual(a.Settings.Groups, [][]string{{g2, g1}}) {
		t.Errorf("grupos = %v, se esperaba [[%s %s]]", a.Settings.Groups, g2, g1)
	}
	if !slices.Equal(a.Settings.NullUser, []string{g4}) || !slices.Equal(a.Settings.NonString, []string{g5}) {
		t.Errorf("sin userId = %v, userId no string = %v", a.Settings.NullUser, a.Settings.NonString)
	}
}

// Cada documento que Go no puede leer u operar aparece una vez, con su tipo de bloqueante; los perfiles y las
// cuentas, sin _id y con la causa. Los perfiles de los dueños se leen como los cruzan los listados y los
// inversionistas, también borrados. La auditoría no escribe nada.
func TestCollectBlockers(t *testing.T) {
	setup(t)
	bt1 := meteorID("TypeOne")
	brokenType := typeDoc(bt1, bson.M{"createdAt": "2023-06-01T12:00:00Z"})
	delete(brokenType, "isRemove")
	insert(t, bondTypeCollection, brokenType)

	x1, x2, x3 := meteorID("BadOne"), shared_domain.NewID(), meteorID("BadThree")
	x4, x5, x6 := shared_domain.NewID(), meteorID("BadFive"), shared_domain.NewID()
	x7 := meteorID("BadSeven")
	noInitialDate := boundDoc(x4, nil)
	delete(noInitialDate, "initialDate")
	insert(t, bondBoundCollection,
		boundDoc(x1, bson.M{"statusId": "9"}),
		boundDoc(x2, bson.M{"soldDate": fixtureAt.AddDate(0, 1, 0)}),
		boundDoc(x3, bson.M{"agencyId": ""}),
		noInitialDate,
		boundDoc(x5, bson.M{"amount": "abc"}),
		boundDoc(x6, bson.M{"statusId": "2", "maturityTimeInDays": int32(90)}),
		boundDoc(x7, bson.M{"statusId": "5", "soldDate": fixtureAt.AddDate(0, 1, 0), "maturityTimeInDays": int32(90)}),
	)
	h1 := shared_domain.NewID()
	history := saleHistoryDoc(x1)
	history["_id"], history["params"] = h1, bson.A{int32(1), int32(2)}
	insert(t, bondHistoryCollection, history)
	g1, g2 := meteorID("SetOne"), shared_domain.NewID()
	insert(t, bondUserSettingCollection,
		settingDoc(g1, bson.NewObjectID(), fixtureAt, nil),
		settingDoc(g2, user2ID, fixtureAt, bson.M{"agencyId": bson.A{agencyA}}),
	)
	p1, p2 := meteorID("ProfOne"), shared_domain.NewID()
	noIsRemove := profileDoc(p2, nil)
	delete(noIsRemove, "isRemove")
	insert(t, profileCollection, profileDoc(p1, bson.M{"id": int32(123)}), noIsRemove)
	// dueños de bonos vivos: uno con el perfil borrado y el nombre como número, que leen los listados; un
	// inversionista con avatar booleano; y otro con avatar booleano que solo tiene un bono vendido
	removedOwner, investor, seller := meteorID("OwnerGone"), shared_domain.NewID(), meteorID("OwnerSeller")
	insert(t, profileCollection,
		profileDoc(removedOwner, bson.M{"isRemove": true, "first_name": int32(7)}),
		profileDoc(investor, bson.M{"avatar": false}),
		profileDoc(seller, bson.M{"avatar": false}),
	)
	insert(t, bondBoundCollection,
		boundDoc(meteorID("OwnBondOne"), bson.M{"userId": removedOwner}),
		boundDoc(shared_domain.NewID(), bson.M{"userId": investor}),
		boundDoc(meteorID("OwnBondThree"), bson.M{"userId": seller, "statusId": "5", "soldDate": fixtureAt.AddDate(0, 1, 0)}),
	)
	insert(t, balanceAccountCollection,
		accountDoc(meteorID("AccOne"), bson.M{"available": "abc"}),
		accountDoc(shared_domain.NewID(), bson.M{"global": nil}),
		accountDoc(meteorID("AccThree"), bson.M{"available": true}),
		accountDoc(shared_domain.NewID(), bson.M{"global": 1200.5}),
	)

	before := snapshot(t)
	a := mustCollect(t)
	if after := snapshot(t); !reflect.DeepEqual(before, after) {
		t.Error("la auditoría modificó la base")
	}

	got := map[string][]string{}
	for _, b := range a.Blockers {
		ref := b.Collection + " " + b.ID
		if personalCollections[b.Collection] {
			if b.ID != "" {
				t.Errorf("bloqueante de %s con _id: %+v", b.Collection, b)
			}
			ref = b.Collection + " con " + b.Detail
		}
		got[b.Kind] = append(got[b.Kind], ref)
	}
	for kind := range got {
		got[kind] = sorted(got[kind]...)
	}
	want := map[string][]string{
		blockUndecodable: sorted(
			"bondBound "+x5, "bondHistory "+h1, "bondUserSetting "+g2,
			"profile con id: int", "profile con first_name: int", "profile con avatar: bool",
			"balanceAccount con available: string",
		),
		blockIsRemove:      sorted("bondType "+bt1, "profile con isRemove: ausente"),
		blockDateType:      sorted("bondType "+bt1, "bondBound "+x4),
		blockStatus:        {"bondBound " + x1},
		blockSoldInitiated: {"bondBound " + x2},
		blockNoAgency:      {"bondBound " + x3},
		blockInvalidTerms:  {"bondBound " + x6},
		blockBalance:       sorted("balanceAccount con available: bool", "balanceAccount con global: null"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("bloqueantes =\n%v\nse esperaba\n%v", got, want)
	}
}

// seedApplyFixtures guarda un producto con un numérico string y un isPrivateBond string, un bono con dos
// numéricos string y sin updatedAt, un producto con NaN, un bono con un numérico no canónico, un '2' vendido
// con la venta en el historial y otro sin ella.
func seedApplyFixtures(t *testing.T) (o1, o2, n1, n2, s1, s2 string) {
	t.Helper()
	o1, o2 = meteorID("ObjOne"), shared_domain.NewID()
	n1, n2 = meteorID("NumOne"), shared_domain.NewID()
	s1, s2 = shared_domain.NewID(), meteorID("SoldTwo")
	insert(t, bondObjectCollection,
		objectDoc(o1, legacyTypeID, bson.M{"maturityTimeInDays": "365", "isPrivateBond": legacyTypeID, "authorizedUserIds": bson.A{userID}}),
		objectDoc(o2, typeID, bson.M{"isPrivateBond": math.NaN()}),
	)
	terms := earlyCashTermDoc()
	terms[1].Value = "90"
	noUpdatedAt := boundDoc(n1, bson.M{"maturityTimeInDays": "365", "earlyCashTerm": terms})
	delete(noUpdatedAt, "updatedAt")
	insert(t, bondBoundCollection,
		noUpdatedAt,
		boundDoc(n2, bson.M{"amount": "1.000"}),
		boundDoc(s1, soldExtra(nil)),
		boundDoc(s2, soldExtra(nil)),
	)
	insert(t, bondHistoryCollection, saleHistoryDoc(s1))
	return
}

// -apply guarda los numéricos canónicos como int32, isPrivateBond como booleano con un BOND_UPDATED de sistema
// y los '2' vendidos como '5' con un cambio de estado de sistema, con updatedAt y un respaldo de cada cambio. No
// toca lo que va a revisión manual. Una segunda corrida no encuentra nada ni crea otro respaldo.
func TestApply(t *testing.T) {
	setup(t)
	o1, o2, n1, n2, s1, s2 := seedApplyFixtures(t)
	path := filepath.Join(t.TempDir(), "respaldo.jsonl")

	results := mustApply(t, mustCollect(t), runAt, path, allFixes...)

	for fix, want := range map[string]int{fixNumeric: 2, fixPrivate: 1, fixSold: 1} {
		r := results[fix]
		if r.Planned != want || r.Applied != want || len(r.Changed) != 0 || len(r.Failed) != 0 {
			t.Errorf("%s = %+v, se esperaban %d planeados y aplicados", fix, r, want)
		}
	}

	obj := rawDocument(t, bondObjectCollection, o1)
	if v := obj.Lookup("maturityTimeInDays"); v.Type != bson.TypeInt32 || v.Int32() != 365 {
		t.Errorf("maturityTimeInDays de %s = %v", o1, v)
	}
	if v := obj.Lookup("isPrivateBond"); v.Type != bson.TypeBoolean || !v.Boolean() {
		t.Errorf("isPrivateBond de %s = %v", o1, v)
	}
	if !obj.Lookup("updatedAt").Time().Equal(runAt) {
		t.Errorf("updatedAt de %s = %v", o1, obj.Lookup("updatedAt"))
	}
	bound := rawDocument(t, bondBoundCollection, n1)
	if v := bound.Lookup("earlyCashTerm", "inflectionPoint"); v.Type != bson.TypeInt32 || v.Int32() != 90 {
		t.Errorf("inflectionPoint de %s = %v", n1, v)
	}
	if !bound.Lookup("updatedAt").Time().Equal(runAt) {
		t.Errorf("updatedAt de %s = %v", n1, bound.Lookup("updatedAt"))
	}
	if v := rawDocument(t, bondBoundCollection, n2).Lookup("amount"); v.Type != bson.TypeString {
		t.Errorf("el no canónico de %s cambió: %v", n2, v)
	}
	if v := rawDocument(t, bondObjectCollection, o2).Lookup("isPrivateBond"); v.Type != bson.TypeDouble {
		t.Errorf("el NaN de %s cambió: %v", o2, v)
	}
	if len(historyOf(t, n1)) != 0 {
		t.Errorf("los numéricos no registran historial")
	}

	sold := rawDocument(t, bondBoundCollection, s1)
	if stringOf(sold.Lookup("statusId")) != "5" || sold.Lookup("soldDate").Type != bson.TypeDateTime {
		t.Errorf("%s = statusId %v, soldDate %v", s1, sold.Lookup("statusId"), sold.Lookup("soldDate"))
	}
	if stringOf(rawDocument(t, bondBoundCollection, s2).Lookup("statusId")) != "2" {
		t.Errorf("%s, sin la venta en el historial, no tenía que cambiar", s2)
	}

	entries := historyOf(t, s1)
	if len(entries) != 1 {
		t.Fatalf("historial de sistema de %s = %d registros, se esperaba 1", s1, len(entries))
	}
	h := entries[0]
	params, _ := h.Lookup("params").Array().Values()
	if !shared_domain.IsValidID(stringOf(h.Lookup("_id"))) || stringOf(h.Lookup("typeId")) != "2" ||
		stringOf(h.Lookup("text")) != shared_domain.TextBondBoundStatusChanged || stringOf(h.Lookup("agencyId")) != agencyA ||
		len(params) != 2 || stringOf(params[0]) != "2" || stringOf(params[1]) != "5" || h.Lookup("isRemove").Boolean() {
		t.Errorf("historial de %s = %v", s1, h)
	}

	entries = historyOf(t, o1)
	if len(entries) != 1 {
		t.Fatalf("historial de sistema de %s = %d registros, se esperaba 1", o1, len(entries))
	}
	h = entries[0]
	edited, _ := h.Lookup("editedProperties").Array().Values()
	if stringOf(h.Lookup("typeId")) != "1" || stringOf(h.Lookup("text")) != shared_domain.TextBondUpdated ||
		stringOf(h.Lookup("changes", "isPrivateBond", "old")) != legacyTypeID || !h.Lookup("changes", "isPrivateBond", "new").Boolean() ||
		len(edited) != 1 || stringOf(edited[0]) != "isPrivateBond" || stringOf(h.Lookup("agencyId")) != agencyA {
		t.Errorf("historial de %s = %v", o1, h)
	}

	_, changes, _, err := readBackup(path)
	if err != nil {
		t.Fatal(err)
	}
	fixes := []string{}
	for _, ch := range changes {
		fixes = append(fixes, ch.Fix+" "+idString(ch.ID))
	}
	if want := []string{fixNumeric + " " + o1, fixNumeric + " " + n1, fixPrivate + " " + o1, fixSold + " " + s1}; !slices.Equal(fixes, want) {
		t.Errorf("respaldo = %v, se esperaba %v", fixes, want)
	}
	if !slices.Equal(changes[1].Absent, []string{"updatedAt"}) || changes[0].Before.Lookup("maturityTimeInDays").Type != bson.TypeString {
		t.Errorf("respaldo de los numéricos = %+v / %+v", changes[0], changes[1])
	}
	if changes[3].HistoryID != stringOf(historyOf(t, s1)[0].Lookup("_id")) {
		t.Errorf("el respaldo no tiene el _id del historial de %s", s1)
	}

	again := mustCollect(t)
	if len(again.Numeric.Plans)+len(again.Private.Auto)+len(again.Sold.Fixable) != 0 {
		t.Errorf("después de aplicar quedan planes: %d numéricos, %d privados, %d vendidos",
			len(again.Numeric.Plans), len(again.Private.Auto), len(again.Sold.Fixable))
	}
	second := filepath.Join(t.TempDir(), "segunda.jsonl")
	for fix, r := range mustApply(t, again, runAt.Add(time.Hour), second, allFixes...) {
		if r.Planned != 0 {
			t.Errorf("la segunda corrida planeó %d en %s", r.Planned, fix)
		}
	}
	if _, err := os.Stat(second); !os.IsNotExist(err) {
		t.Errorf("la segunda corrida creó un respaldo: %v", err)
	}
	if len(historyOf(t, s1)) != 1 || len(historyOf(t, o1)) != 1 {
		t.Error("la segunda corrida registró historial")
	}
}

// Si un documento cambió entre la auditoría y la escritura, la guarda no lo encuentra: queda como cambiado, no se
// escribe el historial y se conserva el valor nuevo.
func TestApplyGuards(t *testing.T) {
	setup(t)
	o1, _, n1, _, s1, _ := seedApplyFixtures(t)
	a := mustCollect(t)

	ctx := context.Background()
	for _, u := range []struct {
		collection, id string
		set            bson.M
	}{
		{bondObjectCollection, o1, bson.M{"isPrivateBond": true, "maturityTimeInDays": int32(365)}},
		{bondBoundCollection, n1, bson.M{"maturityTimeInDays": int32(365)}},
		{bondBoundCollection, s1, bson.M{"statusId": "6", "tookDate": fixtureAt.AddDate(1, 0, 1)}},
	} {
		if _, err := testDB.UpdateOne(ctx, u.collection, bson.M{"_id": u.id}, bson.M{"$set": u.set}, nil); err != nil {
			t.Fatal(err)
		}
	}

	results := mustApply(t, a, runAt, filepath.Join(t.TempDir(), "respaldo.jsonl"), allFixes...)

	want := map[string][]string{fixNumeric: sorted(o1, n1), fixPrivate: {o1}, fixSold: {s1}}
	for fix, ids := range want {
		r := results[fix]
		if r.Applied != 0 || !slices.Equal(sorted(r.Changed...), ids) || len(r.Failed) != 0 {
			t.Errorf("%s = %+v, se esperaba que cambiaran %v", fix, r, ids)
		}
	}
	if len(historyOf(t, o1)) != 0 || len(historyOf(t, s1)) != 0 {
		t.Error("se registró historial de un documento que cambió")
	}
	if stringOf(rawDocument(t, bondBoundCollection, s1).Lookup("statusId")) != "6" {
		t.Errorf("%s perdió el canje concurrente", s1)
	}
}

// -only aplica solo las normalizaciones elegidas.
func TestApplyOnly(t *testing.T) {
	setup(t)
	o1, _, _, _, s1, _ := seedApplyFixtures(t)

	results := mustApply(t, mustCollect(t), runAt, filepath.Join(t.TempDir(), "respaldo.jsonl"), fixSold)

	if !results[fixNumeric].Skipped || !results[fixPrivate].Skipped || results[fixSold].Applied != 1 {
		t.Errorf("resultados = %+v", results)
	}
	if v := rawDocument(t, bondObjectCollection, o1).Lookup("maturityTimeInDays"); v.Type != bson.TypeString {
		t.Errorf("maturityTimeInDays de %s cambió sin numericos: %v", o1, v)
	}
	if stringOf(rawDocument(t, bondBoundCollection, s1).Lookup("statusId")) != "5" {
		t.Errorf("%s no pasó a '5'", s1)
	}
}

// -revert deja cada documento como estaba, byte a byte, también el que tuvo dos cambios en la misma corrida y el
// que no tenía updatedAt, y borra el historial de sistema. Omite el que cambió después de aplicar. Sin apply solo
// cuenta.
func TestRevert(t *testing.T) {
	setup(t)
	o1, _, n1, _, s1, _ := seedApplyFixtures(t)
	s3 := meteorID("SoldThree")
	insert(t, bondBoundCollection, boundDoc(s3, soldExtra(nil)))
	insert(t, bondHistoryCollection, saleHistoryDoc(s3))
	original := map[string]bson.Raw{
		o1: rawDocument(t, bondObjectCollection, o1),
		n1: rawDocument(t, bondBoundCollection, n1),
		s1: rawDocument(t, bondBoundCollection, s1),
	}

	path := filepath.Join(t.TempDir(), "respaldo.jsonl")
	mustApply(t, mustCollect(t), runAt, path, allFixes...)
	ctx := context.Background()
	later := bson.M{"$set": bson.M{"updatedAt": runAt.Add(time.Minute)}}
	if _, err := testDB.UpdateOne(ctx, bondBoundCollection, bson.M{"_id": s3}, later, nil); err != nil {
		t.Fatal(err)
	}

	_, changes, _, err := readBackup(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 5 {
		t.Fatalf("cambios en el respaldo = %d, se esperaban 5", len(changes))
	}

	applied := snapshot(t)
	dry, err := revertChanges(ctx, testDB, changes, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if dry.Reverted != 4 || !slices.Equal(dry.Skipped, []string{s3}) {
		t.Errorf("en seco = %+v, se esperaban 4 y %s omitido", dry, s3)
	}
	if !reflect.DeepEqual(snapshot(t), applied) {
		t.Error("la reversión en seco modificó la base")
	}

	res, err := revertChanges(ctx, testDB, changes, true, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Reverted != 4 || !slices.Equal(res.Skipped, []string{s3}) || len(res.Failed) != 0 {
		t.Errorf("reversión = %+v, se esperaban 4 y %s omitido", res, s3)
	}
	for id, raw := range original {
		collection := bondBoundCollection
		if id == o1 {
			collection = bondObjectCollection
		}
		if got := rawDocument(t, collection, id); !bytes.Equal(got, raw) {
			t.Errorf("%s no quedó como estaba:\n%v\nse esperaba\n%v", id, got, raw)
		}
	}
	if len(historyOf(t, o1)) != 0 || len(historyOf(t, s1)) != 0 {
		t.Error("quedó historial de sistema de lo revertido")
	}
	if len(historyOf(t, s3)) != 1 || stringOf(rawDocument(t, bondBoundCollection, s3).Lookup("statusId")) != "5" {
		t.Errorf("%s, que cambió después, no tenía que revertirse", s3)
	}
}
