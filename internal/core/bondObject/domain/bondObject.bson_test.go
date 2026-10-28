package bondObject_domain_test

import (
	"reflect"
	"testing"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// legacyDoc es un producto con la forma que dejaba Meteor y todos los campos con valor; cada llamada
// devuelve un documento nuevo.
func legacyDoc() bson.M {
	return bson.M{
		"_id":                       "Bd9Xc4Vn7Mz2Lk5Jh",
		"name":                      "Bono Plus 12M",
		"description":               "Renta fija 12 meses",
		"agencyId":                  agencyA,
		"key":                       "BP12",
		"bondTypeId":                typeID,
		"amount":                    "1000",
		"isPrivateBond":             false,
		"authorizedUserIds":         bson.A{userID},
		"maturityTimeInDays":        365,
		"maturityRewardsPercentage": 15.0,
		"active":                    true,
		"available":                 true,
		"saleInitDate":              saleInit,
		"saleEndDate":               nil,
		"earlyCashTerm": bson.M{
			"initialPenaltyPercentage":    100.0,
			"inflectionPoint":             90,
			"inflectionPenaltyPercentage": 75.0,
			"blockWithdrawals":            false,
			"initialDayForWithdraw":       30,
		},
		"createdAt": storedAt,
		"updatedAt": storedAt,
		"isRemove":  false,
	}
}

// decode serializa doc y lo lee con el decoder del producto.
func decode(t *testing.T, doc bson.M) bondObject_domain.BondObject {
	t.Helper()
	data, err := bson.Marshal(doc)
	if err != nil {
		t.Fatalf("bson.Marshal() = %v", err)
	}
	var o bondObject_domain.BondObject
	if err := bson.Unmarshal(data, &o); err != nil {
		t.Fatalf("bson.Unmarshal() = %v", err)
	}
	return o
}

// El decoder lee isPrivateBond con el criterio de verdad de JS: false, null, ausente, "" y 0 dan false;
// true, un id de Meteor, "false", "0", 1, un documento y un arreglo dan true. Los demás campos se leen
// igual que con un booleano. CA-02.16
func TestUnmarshalBSONIsPrivateBond(t *testing.T) {
	tests := []struct {
		name  string
		value any
		// absent, si está, quita el campo del documento.
		absent bool
		want   bool
	}{
		{name: "false", value: false, want: false},
		{name: "null", value: nil, want: false},
		{name: "ausente", absent: true, want: false},
		{name: "string vacío", value: "", want: false},
		{name: "0 int32", value: int32(0), want: false},
		{name: "0 int64", value: int64(0), want: false},
		{name: "0 double", value: 0.0, want: false},

		{name: "true", value: true, want: true},
		{name: "id de Meteor", value: "A8DjCnAETjXarQ2YA", want: true},
		{name: "string false", value: "false", want: true},
		{name: "string 0", value: "0", want: true},
		{name: "1 int32", value: int32(1), want: true},
		{name: "1 double", value: 1.0, want: true},
		{name: "documento vacío", value: bson.D{}, want: true},
		{name: "arreglo vacío", value: bson.A{}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := legacyDoc()
			if tt.absent {
				delete(doc, "isPrivateBond")
			} else {
				doc["isPrivateBond"] = tt.value
			}

			got := decode(t, doc)

			if got.IsPrivateBond != tt.want {
				t.Fatalf("isPrivateBond = %v, se esperaba %v", got.IsPrivateBond, tt.want)
			}
			withBool := legacyDoc()
			withBool["isPrivateBond"] = tt.want
			if want := decode(t, withBool); !reflect.DeepEqual(got, want) {
				t.Errorf("producto = %+v, se esperaba %+v", got, want)
			}
		})
	}
}
