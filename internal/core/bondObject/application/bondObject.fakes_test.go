package bondObject_application_test

import (
	"bytes"
	"context"
	"log"
	"slices"
	"strings"
	"testing"
	"time"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondHistory_ports "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/ports"
	bondObject_application "github.com/futurapps/ms-blooming-go/internal/core/bondObject/application"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	bondObject_ports "github.com/futurapps/ms-blooming-go/internal/core/bondObject/ports"
	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	bondType_ports "github.com/futurapps/ms-blooming-go/internal/core/bondType/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

const (
	agencyA        = "agencyAAAAAAAAAAA"
	agencyB        = "agencyBBBBBBBBBBB"
	agencyMain     = "agencyMAAAAAAAAAA"
	userID         = "user1AAAAAAAAAAAA"
	user2ID        = "user2AAAAAAAAAAAA"
	adminID        = "adminAAAAAAAAAAAA"
	objectID       = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"
	otherObjectID  = "1b2c3d4e-5f6a-4b7c-9d8e-0f1a2b3c4d5e"
	sharedObjectID = "5c4d3e2f-1a0b-4c9d-8e7f-6a5b4c3d2e1f"
	missingID      = "0e1f2a3b-4c5d-4e6f-8a7b-9c0d1e2f3a4b"
	typeID         = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"
	otherTypeID    = "9a8b7c6d-5e4f-4a3b-9c2d-1e0f2a3b4c5d"

	// legacyObjectID es un _id Meteor de 17 caracteres.
	legacyObjectID = "Bd9Xc4Vn7Mz2Lk5Jh"
)

// storedAt es la fecha de alta de los productos que ya están guardados.
var storedAt = time.Date(2026, 1, 10, 15, 0, 0, 0, time.UTC)

// keyUpdate son los argumentos de una llamada a UpdateKeys.
type keyUpdate struct {
	ids      []string
	key      string
	agencyID *string
}

// fakeRepository guarda los productos en memoria y registra las llamadas.
type fakeRepository struct {
	bondObject_ports.BondObjectMongoccRepositoryPorts

	objects map[string]*bondObject_domain.BondObject
	// keysResult es lo que devuelve UpdateKeys.
	keysResult bondObject_domain.UpdateKeysResult

	reads        []string
	finds        [][]string
	groupedFinds [][]string
	visible      []bondObject_domain.VisibleFilter
	inserted     []bondObject_domain.BondObject
	updated      []bondObject_domain.BondObject
	keyUpdates   []keyUpdate
	deleted      []string
}

// Find registra las agencias del filtro y no devuelve productos.
func (r *fakeRepository) Find(_ context.Context, agencyIDs []string) ([]*bondObject_domain.BondObject, error) {
	r.finds = append(r.finds, slices.Clone(agencyIDs))
	return []*bondObject_domain.BondObject{}, nil
}

// FindGroupedByType registra las agencias del filtro y no devuelve grupos.
func (r *fakeRepository) FindGroupedByType(_ context.Context, agencyIDs []string) ([]bondObject_ports.BondObjectGroup, error) {
	r.groupedFinds = append(r.groupedFinds, slices.Clone(agencyIDs))
	return []bondObject_ports.BondObjectGroup{}, nil
}

// FindVisible aplica el filtro en memoria como el adapter: no borrados, de la agencia, privados solo
// donde UserID está autorizado, del más reciente al más antiguo. Registra cada filtro.
func (r *fakeRepository) FindVisible(_ context.Context, f bondObject_domain.VisibleFilter) ([]*bondObject_domain.BondObject, error) {
	r.visible = append(r.visible, f)
	out := []*bondObject_domain.BondObject{}
	for _, o := range r.objects {
		switch {
		case o.IsRemove, !slices.Contains(o.AgencyID, f.AgencyID):
		case o.IsPrivateBond && !slices.Contains(o.AuthorizedUserIDs, f.UserID):
		case f.BondTypeID != "" && o.BondTypeID != f.BondTypeID:
		case len(f.IDs) > 0 && !slices.Contains(f.IDs, o.ID):
		case f.OnlyActive && !o.Active:
		default:
			found := *o
			out = append(out, &found)
		}
	}
	slices.SortFunc(out, func(a, b *bondObject_domain.BondObject) int {
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	if f.Limit > 0 && int64(len(out)) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

// FindOneByID devuelve una copia del producto, o nil si no existe o está borrado.
func (r *fakeRepository) FindOneByID(_ context.Context, id string) (*bondObject_domain.BondObject, error) {
	r.reads = append(r.reads, id)
	o, ok := r.objects[id]
	if !ok || o.IsRemove {
		return nil, nil
	}
	found := *o
	return &found, nil
}

func (r *fakeRepository) InsertOne(_ context.Context, o *bondObject_domain.BondObject) error {
	r.inserted = append(r.inserted, *o)
	stored := *o
	r.objects[o.ID] = &stored
	return nil
}

// UpdateOne guarda o si el producto existe y no está borrado.
func (r *fakeRepository) UpdateOne(_ context.Context, o *bondObject_domain.BondObject) (bool, error) {
	r.updated = append(r.updated, *o)
	current, ok := r.objects[o.ID]
	if !ok || current.IsRemove {
		return false, nil
	}
	stored := *o
	r.objects[o.ID] = &stored
	return true, nil
}

// UpdateKeys registra la llamada y devuelve keysResult.
func (r *fakeRepository) UpdateKeys(_ context.Context, ids []string, key string, agencyID *string) (bondObject_domain.UpdateKeysResult, error) {
	r.keyUpdates = append(r.keyUpdates, keyUpdate{ids: slices.Clone(ids), key: key, agencyID: agencyID})
	return r.keysResult, nil
}

// SoftDelete marca el producto como borrado; devuelve false si no existe o ya estaba borrado.
func (r *fakeRepository) SoftDelete(_ context.Context, id string) (bool, error) {
	r.deleted = append(r.deleted, id)
	o, ok := r.objects[id]
	if !ok || o.IsRemove {
		return false, nil
	}
	o.IsRemove = true
	return true, nil
}

// fakeBondTypes conoce los tipos de existing, con sus agencias, y registra las consultas.
type fakeBondTypes struct {
	bondType_ports.BondTypeApplicationPorts

	existing map[string][]string
	reads    []string
}

// GetByID devuelve el tipo si existe y está en el alcance de agencia de la sesión, como el de bondType; si
// no, ErrBondTypeNotFound.
func (b *fakeBondTypes) GetByID(ctx context.Context, id string) (*bondType_domain.BondType, error) {
	b.reads = append(b.reads, id)
	agencies, ok := b.existing[id]
	if !ok || !shared_application.InAgencyScope(ctx, agencies...) {
		return nil, bondType_domain.ErrBondTypeNotFound
	}
	return &bondType_domain.BondType{ID: id, AgencyID: agencies, Active: true}, nil
}

// fakeHistory registra los eventos y devuelve err.
type fakeHistory struct {
	bondHistory_ports.BondHistoryPort

	err   error
	added []bondHistory_domain.AddParams
}

func (h *fakeHistory) Add(_ context.Context, p bondHistory_domain.AddParams) error {
	h.added = append(h.added, p)
	return h.err
}

// fakeActivity registra las actividades.
type fakeActivity struct {
	shared_ports.ActivityPort

	added []shared_domain.Activity
}

func (a *fakeActivity) Add(_ context.Context, activity shared_domain.Activity) {
	a.added = append(a.added, activity)
}

// fakeProfiles responde los perfiles conocidos y registra los ids de cada llamada.
type fakeProfiles struct {
	shared_ports.ProfileReader

	profiles map[string]shared_domain.PublicProfile
	calls    [][]string
}

// GetPublic devuelve, en el orden pedido, los perfiles que conoce.
func (p *fakeProfiles) GetPublic(_ context.Context, ids ...string) ([]shared_domain.PublicProfile, error) {
	p.calls = append(p.calls, slices.Clone(ids))
	out := []shared_domain.PublicProfile{}
	for _, id := range ids {
		if profile, ok := p.profiles[id]; ok {
			out = append(out, profile)
		}
	}
	return out, nil
}

// fakes son las dependencias de la aplicación.
type fakes struct {
	repo     *fakeRepository
	types    *fakeBondTypes
	history  *fakeHistory
	activity *fakeActivity
	profiles *fakeProfiles
}

// newFakes arma las dependencias con los productos dados y el tipo typeID de las agencias A y B.
func newFakes(objects ...*bondObject_domain.BondObject) *fakes {
	repo := &fakeRepository{objects: map[string]*bondObject_domain.BondObject{}}
	for _, o := range objects {
		stored := *o
		repo.objects[o.ID] = &stored
	}
	return &fakes{
		repo:     repo,
		types:    &fakeBondTypes{existing: map[string][]string{typeID: {agencyA, agencyB}}},
		history:  &fakeHistory{},
		activity: &fakeActivity{},
		profiles: &fakeProfiles{profiles: map[string]shared_domain.PublicProfile{}},
	}
}

// app arma la aplicación con los fakes.
func (f *fakes) app() *bondObject_application.BondObjectApplication {
	return &bondObject_application.BondObjectApplication{
		Repository: f.repo,
		BondTypes:  f.types,
		History:    f.history,
		Activity:   f.activity,
		Profiles:   f.profiles,
	}
}

// withSession es un context con la sesión de userID en agencyID.
func withSession(userID, agencyID string) context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{UserID: userID, AgencyID: agencyID})
}

// withAdmin es un context con la sesión de un admin de Blooming en agencyID; isMain indica si es la agencia
// principal.
func withAdmin(agencyID string, isMain bool) context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID:        adminID,
		AgencyID:      agencyID,
		Agency:        session_domain.SessionAgency{ID: agencyID, IsMain: isMain},
		AdminBlooming: true,
	})
}

// storedObject es un producto válido de la agencia A guardado el 10 de enero.
func storedObject(id string) *bondObject_domain.BondObject {
	return &bondObject_domain.BondObject{
		ID:                        id,
		Name:                      "Bono Plus 12M",
		Description:               "Renta fija 12 meses",
		AgencyID:                  shared_domain.AgencyIDs{agencyA},
		Key:                       "BP12",
		BondTypeID:                typeID,
		Amount:                    1000,
		AuthorizedUserIDs:         []string{},
		MaturityTimeInDays:        365,
		MaturityRewardsPercentage: 15,
		Active:                    true,
		Available:                 true,
		EarlyCashTerm: shared_domain.EarlyCashTerm{
			InitialPenaltyPercentage:    100,
			InflectionPenaltyPercentage: 100,
		},
		CreatedAt: storedAt,
		UpdatedAt: storedAt,
	}
}

// agencyObjects son los productos de las pruebas de alcance: objectID de A, otherObjectID de B y
// sharedObjectID compartido por A y B.
func agencyObjects() []*bondObject_domain.BondObject {
	onlyB := storedObject(otherObjectID)
	onlyB.AgencyID = shared_domain.AgencyIDs{agencyB}
	shared := storedObject(sharedObjectID)
	shared.AgencyID = shared_domain.AgencyIDs{agencyA, agencyB}
	return []*bondObject_domain.BondObject{storedObject(objectID), onlyB, shared}
}

// paramsFrom es una edición con los mismos valores que el producto guardado.
func paramsFrom(o *bondObject_domain.BondObject) bondObject_domain.UpdateParams {
	return bondObject_domain.UpdateParams{
		Name:                      o.Name,
		Description:               o.Description,
		AgencyID:                  slices.Clone(o.AgencyID),
		Key:                       o.Key,
		BondTypeID:                o.BondTypeID,
		Amount:                    o.Amount,
		IsPrivateBond:             o.IsPrivateBond,
		AuthorizedUserIDs:         slices.Clone(o.AuthorizedUserIDs),
		MaturityTimeInDays:        o.MaturityTimeInDays,
		MaturityRewardsPercentage: o.MaturityRewardsPercentage,
		Active:                    o.Active,
		Available:                 o.Available,
		SaleInitDate:              o.SaleInitDate,
		SaleEndDate:               o.SaleEndDate,
		EarlyCashTerm:             o.EarlyCashTerm,
	}
}

// createParams es un alta válida para las agencias A y B, sin isPrivateBond ni
// inflectionPenaltyPercentage.
func createParams() bondObject_domain.CreateParams {
	return bondObject_domain.CreateParams{
		Name:                      "Bono Plus 12M",
		Description:               "Renta fija 12 meses",
		AgencyID:                  shared_domain.AgencyIDs{agencyA, agencyB},
		Key:                       "BP12",
		BondTypeID:                typeID,
		Amount:                    1000,
		MaturityTimeInDays:        365,
		MaturityRewardsPercentage: 15,
		Active:                    true,
		Available:                 true,
		EarlyCashTerm: shared_domain.EarlyCashTerm{
			InitialPenaltyPercentage: 100,
			InflectionPoint:          90,
			InitialDayForWithdraw:    30,
		},
	}
}

// captureLog redirige el log estándar a un buffer mientras dura el test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(previous) })
	return &buf
}
