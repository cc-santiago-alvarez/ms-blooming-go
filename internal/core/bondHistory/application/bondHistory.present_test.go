package bondHistory_application_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	bondHistory_application "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/application"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// fakeRepository devuelve en FindByRelated los registros dados.
type fakeRepository struct {
	histories []*bondHistory_domain.BondHistory
}

func (r *fakeRepository) InsertOne(context.Context, *bondHistory_domain.BondHistory) error {
	return nil
}

func (r *fakeRepository) FindByRelated(context.Context, bondHistory_domain.HistoryType, string, *string) ([]*bondHistory_domain.BondHistory, error) {
	return r.histories, nil
}

func (r *fakeRepository) EnsureIndexes(context.Context) error {
	return nil
}

// fakeProfiles devuelve los perfiles pedidos que existan y cuenta las lecturas.
type fakeProfiles struct {
	profiles map[string]shared_domain.PublicProfile
	calls    int
}

func (p *fakeProfiles) GetPublic(_ context.Context, ids ...string) ([]shared_domain.PublicProfile, error) {
	p.calls++
	found := []shared_domain.PublicProfile{}
	for _, id := range ids {
		if profile, ok := p.profiles[id]; ok {
			found = append(found, profile)
		}
	}
	return found, nil
}

// juan es el único perfil que existe en los tests.
var juan = shared_domain.PublicProfile{ID: "user-1", FirstName: "Juan", LastName: "Pérez"}

// listHistory lista con una sesión de la agencia A el historial que devuelve el repositorio.
func listHistory(t *testing.T, profiles *fakeProfiles, histories ...*bondHistory_domain.BondHistory) []bondHistory_domain.HistoryEntry {
	t.Helper()
	app := &bondHistory_application.BondHistoryApplication{
		Repository: &fakeRepository{histories: histories},
		Profiles:   profiles,
	}
	ctx := session_domain.SetUserSession(context.Background(), &session_domain.UserSession{UserID: "user-1", AgencyID: "A"})
	entries, err := app.ListByRelated(ctx, bondHistory_domain.HistoryBondBound, "bound-1")
	if err != nil {
		t.Fatalf("ListByRelated: %v", err)
	}
	if len(entries) != len(histories) {
		t.Fatalf("ListByRelated devolvió %d registros, se esperaban %d", len(entries), len(histories))
	}
	return entries
}

// withProfiles es un lector con el perfil de Juan.
func withProfiles() *fakeProfiles {
	return &fakeProfiles{profiles: map[string]shared_domain.PublicProfile{juan.ID: juan}}
}

// history es un registro del bono bound-1 escrito por userID.
func history(id, userID, text string, params ...string) *bondHistory_domain.BondHistory {
	return &bondHistory_domain.BondHistory{
		ID:        id,
		RelatedID: "bound-1",
		UserID:    userID,
		TypeID:    bondHistory_domain.HistoryBondBound,
		AgencyID:  "A",
		Text:      text,
		Params:    params,
		CreatedAt: time.Now().Add(-time.Hour),
	}
}

// toJSON serializa v como lo recibe el front.
func toJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return string(raw)
}

// entryFields decodifica el JSON de un registro en un mapa.
func entryFields(t *testing.T, e bondHistory_domain.HistoryEntry) map[string]any {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal([]byte(toJSON(t, e)), &fields); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	return fields
}

// El autor se muestra con el nombre de su perfil; sin usuario o sin perfil, como System. Los perfiles se
// leen en una sola consulta. CA-08.5
func TestPresentUser(t *testing.T) {
	profiles := withProfiles()
	entries := listHistory(t, profiles,
		history("h1", juan.ID, shared_domain.TextBondAdded),
		history("h2", "", shared_domain.TextBondAdded),
		history("h3", "user-sin-perfil", shared_domain.TextBondAdded),
		history("h4", juan.ID, shared_domain.TextBondUpdated),
	)

	want := []string{
		`{"first_name":"Juan","last_name":"Pérez"}`,
		`{"name":"System"}`,
		`{"name":"System"}`,
		`{"first_name":"Juan","last_name":"Pérez"}`,
	}
	for i, e := range entries {
		if got := toJSON(t, e.User); got != want[i] {
			t.Errorf("user de %s = %s, se esperaba %s", e.ID, got, want[i])
		}
	}
	if profiles.calls != 1 {
		t.Errorf("se leyeron los perfiles %d veces, se esperaba 1", profiles.calls)
	}
}

// Cada registro trae el tipo, la apariencia fija y los campos del documento, sin params ni userId, en el
// orden del repositorio. CA-08.5
func TestPresentFormat(t *testing.T) {
	createdAt := time.Date(2026, 10, 12, 15, 0, 0, 0, time.UTC)
	edited := history("h1", juan.ID, shared_domain.TextBondUpdated, "x")
	edited.TypeID = bondHistory_domain.HistoryBond
	edited.RelatedID = "object-1"
	edited.Changes = map[string]bondHistory_domain.ChangeEntry{"amount": {Old: 1000.0, New: 1500.0}}
	edited.EditedProperties = []string{"amount"}
	edited.CreatedAt = createdAt
	plain := history("h2", "", shared_domain.TextBondBoundSold, "y")

	entries := listHistory(t, withProfiles(), edited, plain)

	t.Run("campos del registro editado", func(t *testing.T) {
		fields := entryFields(t, entries[0])
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		wantKeys := []string{"_id", "agencyId", "backgroundColor", "changes", "color", "createdAt",
			"createdAtFromNow", "editedProperties", "icon", "relatedId", "text", "type", "typeId", "user"}
		if !slices.Equal(keys, wantKeys) {
			t.Fatalf("claves = %v, se esperaban %v", keys, wantKeys)
		}

		want := map[string]string{
			"_id":              `"h1"`,
			"relatedId":        `"object-1"`,
			"typeId":           `"1"`,
			"type":             `{"_id":"1","name":"Bono"}`,
			"agencyId":         `"A"`,
			"icon":             `"mdi-briefcase"`,
			"color":            `"#000000"`,
			"backgroundColor":  `"#e5e5e5"`,
			"changes":          `{"amount":{"new":1500,"old":1000}}`,
			"editedProperties": `["amount"]`,
			"createdAt":        `"2026-10-12T15:00:00Z"`,
		}
		for k, w := range want {
			if got := toJSON(t, fields[k]); got != w {
				t.Errorf("%s = %s, se esperaba %s", k, got, w)
			}
		}
	})

	t.Run("sin changes ni editedProperties", func(t *testing.T) {
		fields := entryFields(t, entries[1])
		for _, k := range []string{"changes", "editedProperties", "params", "userId"} {
			if _, ok := fields[k]; ok {
				t.Errorf("la salida trae %s: %v", k, fields[k])
			}
		}
		if got := toJSON(t, fields["type"]); got != `{"_id":"2","name":"Bono Enlazado"}` {
			t.Errorf("type = %s, se esperaba Bono Enlazado", got)
		}
	})

	t.Run("orden del repositorio", func(t *testing.T) {
		if entries[0].ID != "h1" || entries[1].ID != "h2" {
			t.Fatalf("orden = %s, %s; se esperaba h1, h2", entries[0].ID, entries[1].ID)
		}
	})

	t.Run("sin registros", func(t *testing.T) {
		if got := toJSON(t, listHistory(t, withProfiles())); got != "[]" {
			t.Fatalf("historial vacío = %s, se esperaba []", got)
		}
	})
}

// El texto es la plantilla del catálogo formateada con params; en un cambio de estado los ids se
// muestran con su nombre, y una clave fuera del catálogo da "". CA-08.6
func TestPresentText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		params []string
		want   string
	}{
		{name: "cambio de estado", text: shared_domain.TextBondBoundStatusChanged, params: []string{"1", "5"}, want: "Estado cambiado de Iniciado a Vendido"},
		{name: "madurado a canjeado", text: shared_domain.TextBondBoundStatusChanged, params: []string{"2", "6"}, want: "Estado cambiado de Madurado a Canjeado"},
		{name: "bloqueado a desligado", text: shared_domain.TextBondBoundStatusChanged, params: []string{"3", "4"}, want: "Estado cambiado de Bloqueado a Desligado"},
		{name: "estado desconocido", text: shared_domain.TextBondBoundStatusChanged, params: []string{"1", "9"}, want: "Estado cambiado de Iniciado a 9"},
		{name: "bono añadido", text: shared_domain.TextBondAdded, want: "Bono Añadido"},
		{name: "bono vendido a un inversionista", text: shared_domain.TextBondSold, params: []string{"1"}, want: "Bono adquirido por un inversionista (1)"},
		{name: "clave desconocida", text: "NO_EXISTE", params: []string{"1"}, want: ""},
		{name: "sin texto", text: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := listHistory(t, withProfiles(), history("h1", juan.ID, tt.text, tt.params...))
			if got := entries[0].Text; got != tt.want {
				t.Fatalf("text de %s %v = %q, se esperaba %q", tt.text, tt.params, got, tt.want)
			}
		})
	}
}

// createdAtFromNow describe en inglés, como moment, cuánto pasó desde createdAt. CA-08.7
func TestPresentCreatedAtFromNow(t *testing.T) {
	tests := []struct {
		name string
		ago  time.Duration
		want string
	}{
		{name: "30 segundos", ago: 30 * time.Second, want: "a few seconds ago"},
		{name: "2 días", ago: 48 * time.Hour, want: "2 days ago"},
		{name: "400 días", ago: 400 * 24 * time.Hour, want: "a year ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := history("h1", juan.ID, shared_domain.TextBondAdded)
			h.CreatedAt = time.Now().Add(-tt.ago)
			entries := listHistory(t, withProfiles(), h)
			if got := entries[0].CreatedAtFromNow; got != tt.want {
				t.Fatalf("createdAtFromNow hace %v = %q, se esperaba %q", tt.ago, got, tt.want)
			}
		})
	}
}
