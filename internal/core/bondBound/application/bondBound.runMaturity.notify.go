package bondBound_application

import (
	"context"
	"html"
	"log"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// notifyBondMatured es la notificación de ms-notification de los bonos madurados.
const notifyBondMatured = "notify-blooming-bondMatured"

// maturedGroup son los bonos de un dueño que maduraron en la corrida.
type maturedGroup struct {
	userID string
	bounds []*bondBound_domain.BondBound
}

// notifyMaturity avisa a cada dueño sus bonos madurados: guarda un aviso con la tabla de los bonos, en la
// agencia de su perfil o, si no la tiene, en la del primer bono, y manda el email. Los nombres de tipo y de
// producto y los perfiles se leen una vez para toda la corrida; lo que no se puede leer queda vacío. Ningún
// fallo llega al llamador.
func (app *BondBoundApplication) notifyMaturity(ctx context.Context, groups []maturedGroup) {
	if len(groups) == 0 {
		return
	}
	typeNames, objectNames := app.maturityNames(ctx, groups)
	profiles := app.maturityProfiles(ctx, groups)

	for _, g := range groups {
		rows := make([]bondBound_domain.MaturityRow, len(g.bounds))
		bonds := make([]map[string]any, len(g.bounds))
		for i, b := range g.bounds {
			rows[i] = bondBound_domain.MaturityRow{Key: b.Key, Type: typeNames[b.BondTypeID], Name: objectNames[b.BondObjectID]}
			bonds[i] = map[string]any{"key": rows[i].Key, "type": rows[i].Type, "name": rows[i].Name}
		}

		profile := profiles[g.userID]
		agencyID := profile.AgencyID
		if agencyID == "" {
			agencyID = g.bounds[0].AgencyID
		}

		app.Notices.Add(ctx, shared_domain.Notice{
			Type:     shared_domain.NoticeTypeBond,
			UserID:   g.userID,
			AgencyID: agencyID,
			Title:    shared_domain.Text(shared_domain.TextBondBoundMessageMaturity),
			Message: shared_domain.Format(shared_domain.Text(shared_domain.TextBondBoundMessageMaturitySucess),
				html.EscapeString(profile.FirstName), html.EscapeString(profile.LastName),
				bondBound_domain.BuildMaturityTable(rows)),
		})
		app.Notifications.Send(ctx, groupUserIDs, notifyBondMatured, map[string]any{
			"userIds": []string{g.userID},
			"bonds":   bonds,
		})
	}
}

// maturityNames devuelve el nombre de cada tipo, de los no borrados, y de cada producto, aunque esté borrado,
// de los bonos madurados.
func (app *BondBoundApplication) maturityNames(ctx context.Context, groups []maturedGroup) (typeNames, objectNames map[string]string) {
	typeNames, objectNames = map[string]string{}, map[string]string{}
	var typeIDs []string
	seenTypes := map[string]bool{}
	for _, g := range groups {
		for _, b := range g.bounds {
			if !seenTypes[b.BondTypeID] {
				seenTypes[b.BondTypeID] = true
				typeIDs = append(typeIDs, b.BondTypeID)
			}
			if _, ok := objectNames[b.BondObjectID]; !ok {
				objectNames[b.BondObjectID] = app.bondName(ctx, b.BondObjectID)
			}
		}
	}

	types, err := app.BondTypes.GetManyByIDs(ctx, typeIDs)
	if err != nil {
		log.Printf("[bondBound] no se pudieron leer los tipos para los avisos de maduración: %v", err)
	}
	for _, t := range types {
		typeNames[t.ID] = t.Name
	}
	return typeNames, objectNames
}

// maturityProfiles devuelve el perfil público de cada dueño que lo tiene; si no se pueden leer, ninguno.
func (app *BondBoundApplication) maturityProfiles(ctx context.Context, groups []maturedGroup) map[string]shared_domain.PublicProfile {
	userIDs := make([]string, len(groups))
	for i, g := range groups {
		userIDs[i] = g.userID
	}

	byID := map[string]shared_domain.PublicProfile{}
	profiles, err := app.Profiles.GetPublic(ctx, userIDs...)
	if err != nil {
		log.Printf("[bondBound] no se pudieron leer los perfiles para los avisos de maduración: %v", err)
	}
	for _, p := range profiles {
		byID[p.ID] = p
	}
	return byID
}
