package main

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// rawDoc arma un documento BSON para los tests.
func rawDoc(t *testing.T, doc bson.D) bson.Raw {
	t.Helper()
	data, err := bson.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// La reversión exige que cada campo siga con el valor y el tipo que dejó el cambio, y vuelve a los valores de
// antes; un campo que no estaba se quita.
func TestRevertFilterAndUpdate(t *testing.T) {
	idType, idData, _ := bson.MarshalValue("Bd9Xc4Vn7Mz2Lk5Jh")
	ch := backupLine{
		Collection: bondObjectCollection,
		ID:         bson.RawValue{Type: idType, Value: idData},
		Before:     rawDoc(t, bson.D{{Key: "maturityTimeInDays", Value: "365"}}),
		Absent:     []string{"updatedAt"},
		After:      rawDoc(t, bson.D{{Key: "maturityTimeInDays", Value: int32(365)}, {Key: "updatedAt", Value: fixedTime}}),
	}

	filter, err := revertFilter(ch)
	if err != nil {
		t.Fatal(err)
	}
	days, ok := filter["maturityTimeInDays"].(bson.M)
	if !ok || days["$type"] != "int" || days["$eq"].(bson.RawValue).Int32() != 365 {
		t.Errorf("guarda de maturityTimeInDays = %#v", filter["maturityTimeInDays"])
	}
	updated, ok := filter["updatedAt"].(bson.M)
	if !ok || updated["$type"] != "date" {
		t.Errorf("guarda de updatedAt = %#v", filter["updatedAt"])
	}

	update := revertUpdate(ch)
	set, _ := update["$set"].(bson.M)
	if v, ok := set["maturityTimeInDays"].(bson.RawValue); !ok || v.Type != bson.TypeString || stringOf(v) != "365" {
		t.Errorf("$set = %#v", update["$set"])
	}
	if unset, _ := update["$unset"].(bson.M); len(unset) != 1 || unset["updatedAt"] != "" {
		t.Errorf("$unset = %#v", update["$unset"])
	}
}

// Un null se guarda con $type, porque {campo: null} también encuentra los documentos sin el campo.
func TestExactMatchNull(t *testing.T) {
	rv := rawDoc(t, bson.D{{Key: "v", Value: nil}}).Lookup("v")
	got := exactMatch(rv)
	if len(got) != 1 || got["$type"] != "null" {
		t.Errorf("exactMatch(null) = %#v", got)
	}
}
