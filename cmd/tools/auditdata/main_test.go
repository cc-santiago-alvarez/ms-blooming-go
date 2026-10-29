package main

import (
	"maps"
	"strings"
	"testing"
	"time"
)

// -apply exige -confirm; -only solo acepta las tres normalizaciones y por defecto son todas; -revert no se
// combina con -only ni con -backup.
func TestParseConfig(t *testing.T) {
	now := time.Date(2026, 10, 29, 15, 4, 5, 0, time.UTC)

	cfg, err := parseConfig(nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.apply || !cfg.at.Equal(now) || !maps.Equal(cfg.only, map[string]bool{fixNumeric: true, fixPrivate: true, fixSold: true}) {
		t.Errorf("valores por defecto = %+v", cfg)
	}

	cfg, err = parseConfig([]string{"-apply", "-confirm", "testing/futurapps", "-only", "vendidos", "-at", "2023-07-06", "-pausa", "10ms"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.apply || cfg.confirm != "testing/futurapps" || !maps.Equal(cfg.only, map[string]bool{fixSold: true}) ||
		cfg.pause != 10*time.Millisecond || !cfg.at.Equal(time.Date(2023, 7, 6, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("opciones = %+v", cfg)
	}

	for _, args := range [][]string{
		{"-apply"},
		{"-only", "vendidos,claves"},
		{"-revert", "respaldo.jsonl", "-only", "vendidos"},
		{"-revert", "respaldo.jsonl", "-backup", "otro.jsonl"},
		{"-pausa", "-1s"},
		{"-at", "ayer"},
		{"futurapps"},
	} {
		if _, err := parseConfig(args, now); err == nil {
			t.Errorf("parseConfig(%s) no dio error", strings.Join(args, " "))
		}
	}
}

// -confirm tiene que ser el entorno y la base de la conexión: la base se llama igual en testing y en live.
func TestCheckConfirm(t *testing.T) {
	if err := checkConfirm("testing/futurapps", "testing", "futurapps"); err != nil {
		t.Errorf("el entorno y la base correctos no pasan: %v", err)
	}
	for _, confirm := range []string{"futurapps", "production/futurapps", "testing/parity", ""} {
		if checkConfirm(confirm, "testing", "futurapps") == nil {
			t.Errorf("-confirm %q pasó contra testing/futurapps", confirm)
		}
	}
}
