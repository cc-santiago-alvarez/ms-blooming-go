package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// El respaldo se crea con el primer cambio, con la cabecera primero, solo para su dueño, y conserva los tipos
// BSON de cada valor: el string "365", el int32 365, el booleano, la fecha y los campos que no estaban.
func TestBackupWriterRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auditdata", "futurapps.jsonl")
	runAt := time.Date(2026, 10, 29, 15, 4, 5, 123_000_000, time.UTC)
	w := newBackupWriter(path, backupHeader{Kind: lineRun, Run: runAt, Env: "testing", DB: "futurapps", Only: allFixes})

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("sin cambios no tiene que existir el archivo: %v", err)
	}

	idType, idData, _ := bson.MarshalValue("Bd9Xc4Vn7Mz2Lk5Jh")
	numeric := backupChange{
		Kind: lineChange, Fix: fixNumeric, Collection: bondObjectCollection,
		ID:     bson.RawValue{Type: idType, Value: idData},
		Before: bson.D{{Key: "maturityTimeInDays", Value: "365"}},
		After:  bson.D{{Key: "maturityTimeInDays", Value: int32(365)}},
	}
	numeric.withUpdatedAt(bson.RawValue{}, runAt)
	private := backupChange{
		Kind: lineChange, Fix: fixPrivate, Collection: bondObjectCollection,
		ID:        bson.RawValue{Type: idType, Value: idData},
		Before:    bson.D{{Key: "isPrivateBond", Value: "q8zp69zAa7ygqCXxT"}, {Key: "updatedAt", Value: fixedTime}},
		After:     bson.D{{Key: "isPrivateBond", Value: true}, {Key: "updatedAt", Value: runAt}},
		HistoryID: "0b6f8c2e-3d4a-4e5b-9c6d-7e8f9a0b1c2d",
	}
	for _, ch := range []backupChange{numeric, private} {
		if err := w.Write(ch); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("permisos del respaldo = %v, se esperaba 0600", info.Mode().Perm())
	}

	header, changes, torn, err := readBackup(path)
	if err != nil || torn {
		t.Fatalf("readBackup: %v, a medias %v", err, torn)
	}
	if header.DB != "futurapps" || !header.Run.Equal(runAt) {
		t.Errorf("cabecera = %+v", header)
	}
	if len(changes) != 2 {
		t.Fatalf("cambios = %d, se esperaban 2", len(changes))
	}

	got := changes[0]
	if got.Fix != fixNumeric || stringOf(got.ID) != "Bd9Xc4Vn7Mz2Lk5Jh" || !slices.Equal(got.Absent, []string{"updatedAt"}) {
		t.Errorf("cambio numérico = %+v", got)
	}
	if v := got.Before.Lookup("maturityTimeInDays"); v.Type != bson.TypeString || stringOf(v) != "365" {
		t.Errorf("antes = %v, se esperaba el string 365", v)
	}
	if v := got.After.Lookup("maturityTimeInDays"); v.Type != bson.TypeInt32 || v.Int32() != 365 {
		t.Errorf("después = %v, se esperaba el int32 365", v)
	}
	if v := got.After.Lookup("updatedAt"); v.Type != bson.TypeDateTime || !v.Time().Equal(runAt) {
		t.Errorf("updatedAt = %v, se esperaba %s", v, runAt)
	}

	got = changes[1]
	if got.HistoryID != private.HistoryID || got.Before.Lookup("updatedAt").Type != bson.TypeDateTime {
		t.Errorf("cambio de isPrivateBond = %+v", got)
	}
	if v := got.After.Lookup("isPrivateBond"); v.Type != bson.TypeBoolean || !v.Boolean() {
		t.Errorf("isPrivateBond después = %v", v)
	}
}

// El respaldo nunca pisa un archivo que ya existe.
func TestBackupWriterNeverOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "previo.jsonl")
	if err := os.WriteFile(path, []byte("corrida anterior\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := newBackupWriter(path, backupHeader{Kind: lineRun, DB: "futurapps"})
	if err := w.Write(backupChange{Kind: lineChange}); err == nil {
		t.Fatal("se escribió sobre un respaldo existente")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "corrida anterior\n" {
		t.Errorf("el archivo existente cambió: %q", data)
	}
}

// Un respaldo sin cabecera o con líneas de otro tipo no se lee.
func TestReadBackupRejectsMalformed(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"vacio.jsonl":       "",
		"sinCabecera.jsonl": `{"kind":"cambio","fix":"vendidos"}` + "\n",
		"basura.jsonl":      "no es json\n",
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := readBackup(path); err == nil {
			t.Errorf("%s: se leyó sin error", name)
		}
	}
}

// Una última línea sin salto de línea es una escritura a medias: se ignora, porque su cambio no llegó a la base,
// y el resto del respaldo se puede revertir.
func TestReadBackupIgnoresTornLastLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cortado.jsonl")
	content := `{"kind":"corrida","db":"futurapps","env":"testing"}` + "\n" +
		`{"kind":"cambio","fix":"vendidos","collection":"bondBound","_id":"Bd9Xc4Vn7Mz2Lk5Jh","before":{"statusId":"2"},"after":{"statusId":"5"}}` + "\n" +
		`{"kind":"cambio","fix":"vendidos","collection":"bondBound","_id":"Rv6Tb1Yn5Um9Io3Pa","before":{"sta`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	header, changes, torn, err := readBackup(path)
	if err != nil {
		t.Fatal(err)
	}
	if !torn || len(changes) != 1 || stringOf(changes[0].ID) != "Bd9Xc4Vn7Mz2Lk5Jh" || header.Env != "testing" {
		t.Errorf("a medias %v, cambios %+v, cabecera %+v", torn, changes, header)
	}
}
