package bondHistory_application

import (
	"time"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Apariencia de todos los registros del historial.
const (
	historyIcon            = "mdi-briefcase"
	historyColor           = "#000000"
	historyBackgroundColor = "#e5e5e5"
)

// bondBoundStatusNames son los nombres de los estados de un bono comprado, para mostrar sus cambios.
var bondBoundStatusNames = map[string]string{
	"1": "Iniciado",
	"2": "Madurado",
	"3": "Bloqueado",
	"4": "Desligado",
	"5": "Vendido",
	"6": "Canjeado",
}

// present arma los registros para mostrar con los perfiles de sus autores, en el mismo orden.
func present(histories []*bondHistory_domain.BondHistory, profiles []shared_domain.PublicProfile, now time.Time) []bondHistory_domain.HistoryEntry {
	profilesByID := make(map[string]shared_domain.PublicProfile, len(profiles))
	for _, p := range profiles {
		profilesByID[p.ID] = p
	}

	entries := make([]bondHistory_domain.HistoryEntry, len(histories))
	for i, h := range histories {
		entries[i] = bondHistory_domain.HistoryEntry{
			ID:               h.ID,
			RelatedID:        h.RelatedID,
			TypeID:           h.TypeID,
			Type:             bondHistory_domain.HistoryTypeRef{ID: h.TypeID, Name: h.TypeID.Name()},
			AgencyID:         h.AgencyID,
			Text:             historyText(h.Text, h.Params),
			User:             historyUser(profilesByID, h.UserID),
			Icon:             historyIcon,
			Color:            historyColor,
			BackgroundColor:  historyBackgroundColor,
			Changes:          h.Changes,
			EditedProperties: h.EditedProperties,
			CreatedAt:        h.CreatedAt,
			CreatedAtFromNow: shared_domain.FromNow(h.CreatedAt, now),
		}
	}
	return entries
}

// historyUser devuelve el nombre del perfil del autor o, si no tiene perfil, el sistema.
func historyUser(profilesByID map[string]shared_domain.PublicProfile, userID string) bondHistory_domain.HistoryUser {
	p, ok := profilesByID[userID]
	if !ok {
		return bondHistory_domain.HistoryUser{System: true}
	}
	return bondHistory_domain.HistoryUser{FirstName: p.FirstName, LastName: p.LastName}
}

// historyText formatea la plantilla de key con params; una clave fuera del catálogo da "". En un
// cambio de estado, los ids se muestran con su nombre.
func historyText(key string, params []string) string {
	tpl := shared_domain.Text(key)
	// Text devuelve la clave misma cuando no está en el catálogo
	if tpl == key {
		return ""
	}

	args := make([]any, len(params))
	for i, p := range params {
		if key == shared_domain.TextBondBoundStatusChanged {
			p = statusName(p)
		}
		args[i] = p
	}
	return shared_domain.Format(tpl, args...)
}

// statusName devuelve el nombre del estado de un bono comprado, o el id si no lo conoce.
func statusName(id string) string {
	if name, ok := bondBoundStatusNames[id]; ok {
		return name
	}
	return id
}
