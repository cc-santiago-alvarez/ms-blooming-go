package main

import (
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Un string canónico pasa a int32 si es entero y entra, o a double; en un campo entero solo valen los enteros que
// entran en int32. Lo que no es canónico devuelve por qué, sin el valor.
func TestParseCanonical(t *testing.T) {
	cases := []struct {
		in      string
		kind    numericKind
		want    any
		problem string
	}{
		{"365", kindInt, int32(365), ""},
		{"0", kindInt, int32(0), ""},
		{"1000", kindFloat, int32(1000), ""},
		{"15.5", kindFloat, 15.5, ""},
		{"0.0001", kindFloat, 0.0001, ""},
		{"2147483648", kindFloat, 2147483648.0, ""},
		{"15.5", kindInt, nil, "decimal en un campo entero"},
		{"2147483648", kindInt, nil, "fuera del rango de int32"},
		{"", kindInt, nil, "vacío"},
		{" 365", kindInt, nil, "con espacios"},
		{"365 ", kindFloat, nil, "con espacios"},
		{"365.0", kindInt, nil, "formato no canónico"},
		{"0365", kindFloat, nil, "formato no canónico"},
		{"1e3", kindFloat, nil, "formato no canónico"},
		{"-5", kindFloat, nil, "formato no canónico"},
		{"+5", kindFloat, nil, "formato no canónico"},
		{".5", kindFloat, nil, "formato no canónico"},
		{"abc", kindFloat, nil, "no numérico"},
		{"10%", kindFloat, nil, "no numérico"},
		{"1,5", kindFloat, nil, "no numérico"},
	}
	for _, tc := range cases {
		got, problem := parseCanonical(tc.in, tc.kind)
		if got != tc.want || problem != tc.problem {
			t.Errorf("parseCanonical(%q, %v) = (%#v, %q), se esperaba (%#v, %q)", tc.in, tc.kind, got, problem, tc.want, tc.problem)
		}
	}
}

// Go lee un string canónico con el mismo valor que el número que lo reemplaza, también truncado a entero.
func TestCanonicalReadsTheSameInGo(t *testing.T) {
	for _, s := range []string{"0", "7", "365", "15.5", "0.0001", "2147483648"} {
		v, problem := parseCanonical(s, kindFloat)
		if problem != "" {
			t.Fatalf("parseCanonical(%q) no es canónico: %s", s, problem)
		}
		typ, data, err := bson.MarshalValue(s)
		if err != nil {
			t.Fatal(err)
		}
		var n shared_domain.FlexNumber
		if err := n.UnmarshalBSONValue(byte(typ), data); err != nil {
			t.Fatalf("FlexNumber(%q): %v", s, err)
		}
		var want float64
		switch x := v.(type) {
		case int32:
			want = float64(x)
		case float64:
			want = x
		}
		if float64(n) != want {
			t.Errorf("FlexNumber(%q) = %v, el número que lo reemplaza es %v", s, float64(n), want)
		}
	}
}

// visit cuenta los tipos de cada numérico, planea los strings canónicos de un documento en un solo update e
// informa los no canónicos con el campo, el problema y el largo.
func TestNumericVisit(t *testing.T) {
	doc, err := bson.Marshal(bson.D{
		{Key: "_id", Value: "Bd9Xc4Vn7Mz2Lk5Jh"},
		{Key: "amount", Value: "1000"},
		{Key: "maturityTimeInDays", Value: "365"},
		{Key: "maturityRewardsPercentage", Value: 15.5},
		{Key: "earlyCashTerm", Value: bson.D{
			{Key: "initialPenaltyPercentage", Value: int32(100)},
			{Key: "inflectionPoint", Value: " 90"},
			{Key: "inflectionPenaltyPercentage", Value: "75.5"},
			{Key: "initialDayForWithdraw", Value: 0.0},
		}},
		{Key: "updatedAt", Value: fixedTime},
	})
	if err != nil {
		t.Fatal(err)
	}

	var n numericAudit
	n.visit(bondObjectCollection, bson.Raw(doc))

	if len(n.Plans) != 1 {
		t.Fatalf("planes = %d, se esperaba 1", len(n.Plans))
	}
	plan := n.Plans[0]
	want := []numericChange{
		{Field: "amount", Old: "1000", New: int32(1000)},
		{Field: "maturityTimeInDays", Old: "365", New: int32(365)},
		{Field: "earlyCashTerm.inflectionPenaltyPercentage", Old: "75.5", New: 75.5},
	}
	if len(plan.Changes) != len(want) {
		t.Fatalf("cambios = %#v, se esperaba %#v", plan.Changes, want)
	}
	for i := range want {
		if plan.Changes[i] != want[i] {
			t.Errorf("cambio %d = %#v, se esperaba %#v", i, plan.Changes[i], want[i])
		}
	}
	if stringOf(plan.ID) != "Bd9Xc4Vn7Mz2Lk5Jh" || plan.UpdatedAt.Type != bson.TypeDateTime {
		t.Errorf("el plan no guarda _id y updatedAt: %v %v", plan.ID, plan.UpdatedAt.Type)
	}

	if len(n.NonCanonical) != 1 || n.NonCanonical[0].Detail != "earlyCashTerm.inflectionPoint: con espacios, 3 caracteres" {
		t.Errorf("no canónicos = %#v", n.NonCanonical)
	}
	if got := n.Census["bondObject.maturityTimeInDays"]["string"]; got != 1 {
		t.Errorf("censo de maturityTimeInDays string = %d, se esperaba 1", got)
	}
	if got := n.Census["bondObject.earlyCashTerm.initialDayForWithdraw"]["double"]; got != 1 {
		t.Errorf("censo de initialDayForWithdraw double = %d, se esperaba 1", got)
	}
}
