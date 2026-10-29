package main

import (
	"math"
	"testing"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// objectWith arma un producto vivo y activo con isPrivateBond y la lista de autorizados; nil en un valor deja el
// campo afuera, y extra agrega campos o pisa los anteriores.
func objectWith(t *testing.T, isPrivateBond, authorized any, extra ...bson.E) bson.Raw {
	t.Helper()
	doc := bson.D{
		{Key: "_id", Value: "WizkTxQg8LPFpnp58"},
		{Key: "bondTypeId", Value: "q8zp69zAa7ygqCXxT"},
		{Key: "agencyId", Value: bson.A{"tDr6NwSsDe4Lnccp8"}},
	}
	base := map[string]any{"isRemove": false, "active": true}
	for _, e := range extra {
		base[e.Key] = e.Value
	}
	doc = append(doc, bson.E{Key: "isRemove", Value: base["isRemove"]}, bson.E{Key: "active", Value: base["active"]})
	if isPrivateBond != nil {
		doc = append(doc, bson.E{Key: "isPrivateBond", Value: isPrivateBond})
	}
	if authorized != nil {
		doc = append(doc, bson.E{Key: "authorizedUserIds", Value: authorized})
	}
	data, err := bson.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// decodeObject decodifica el producto como lo hace Go.
func decodeObject(t *testing.T, raw bson.Raw) (bondObject_domain.BondObject, error) {
	t.Helper()
	var o bondObject_domain.BondObject
	err := bson.Unmarshal(raw, &o)
	return o, err
}

// Se normalizan solos los strings, null, los enteros y los doubles finitos, con el booleano que ya les da el
// decoder de Go, que coincide con JavaScript. NaN, Decimal128, documentos y arreglos van a revisión manual.
func TestAutoPrivateMatchesTheDecoder(t *testing.T) {
	cases := []struct {
		value any
		auto  bool
		want  bool
	}{
		{"q8zp69zAa7ygqCXxT", true, true},
		{"", true, false},
		{"false", true, true},
		{"0", true, true},
		{bson.Null{}, true, false},
		{int32(0), true, false},
		{int32(1), true, true},
		{int64(0), true, false},
		{0.0, true, false},
		{2.5, true, true},
		{math.NaN(), false, true},
		{math.Inf(1), false, true},
		{bson.NewDecimal128(0, 0), false, false},
		{bson.D{{Key: "a", Value: 1}}, false, true},
		{bson.A{}, false, true},
	}
	for _, tc := range cases {
		raw := objectWith(t, tc.value, bson.A{})
		rv := raw.Lookup("isPrivateBond")
		if got := autoPrivate(rv); got != tc.auto {
			t.Errorf("autoPrivate(%v) = %v, se esperaba %v", tc.value, got, tc.auto)
		}
		o, err := decodeObject(t, raw)
		if err != nil {
			t.Fatalf("decode con isPrivateBond %v: %v", tc.value, err)
		}
		if o.IsPrivateBond != tc.want {
			t.Errorf("Go lee isPrivateBond %v como %v, se esperaba %v", tc.value, o.IsPrivateBond, tc.want)
		}
	}
}

// Un string se planea con el booleano de Go, sin mostrar el valor, y avisa si es el bondTypeId del producto;
// el booleano y la falta del campo no se tocan; NaN va a revisión manual. El privado sin lista de autorizados
// solo se avisa si Meteor lo lista: vivo y activo.
func TestPrivateVisit(t *testing.T) {
	var p privateAudit
	visit := func(value, authorized any) {
		raw := objectWith(t, value, authorized)
		o, err := decodeObject(t, raw)
		p.visit(raw, o, err)
	}
	visit("q8zp69zAa7ygqCXxT", bson.A{"Hn4Wq8Ls2Dx6Vb9Ty"})
	visit(true, bson.A{})
	visit(nil, nil)
	visit(math.NaN(), bson.A{})
	visit(true, nil)
	for _, extra := range []bson.E{{Key: "isRemove", Value: true}, {Key: "active", Value: false}} {
		raw := objectWith(t, true, nil, extra)
		o, err := decodeObject(t, raw)
		p.visit(raw, o, err)
	}

	if len(p.Auto) != 1 || !p.Auto[0].Target || p.Auto[0].AgencyID != "tDr6NwSsDe4Lnccp8" {
		t.Fatalf("automáticos = %#v", p.Auto)
	}
	if p.Auto[0].Detail != "string de 17 caracteres, igual a su bondTypeId" {
		t.Errorf("detalle = %q", p.Auto[0].Detail)
	}
	if p.Missing != 1 {
		t.Errorf("sin el campo = %d, se esperaba 1", p.Missing)
	}
	if len(p.Manual) != 1 || p.Manual[0].Detail != "double NaN; valor que Go y Meteor no leen igual o inesperado" {
		t.Errorf("manuales = %#v", p.Manual)
	}
	if len(p.NoAuthorized) != 1 || p.NoAuthorized[0].Detail != "authorizedUserIds ausente" {
		t.Errorf("privados sin autorizados = %#v", p.NoAuthorized)
	}
}

// El historial guarda el valor anterior tal cual: string, número o null.
func TestHistoryValue(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  any
	}{
		{"q8zp69zAa7ygqCXxT", "q8zp69zAa7ygqCXxT"},
		{int32(1), int32(1)},
		{int64(0), int64(0)},
		{2.5, 2.5},
		{bson.Null{}, nil},
	} {
		rv := objectWith(t, tc.value, nil).Lookup("isPrivateBond")
		if got := historyValue(rv); got != tc.want {
			t.Errorf("historyValue(%v) = %#v, se esperaba %#v", tc.value, got, tc.want)
		}
	}
}
