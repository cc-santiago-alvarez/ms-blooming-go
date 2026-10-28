package bondType_application_test

import (
	"context"
	"slices"
	"time"

	bondType_application "github.com/futurapps/ms-blooming-go/internal/core/bondType/application"
	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	bondType_ports "github.com/futurapps/ms-blooming-go/internal/core/bondType/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
)

const (
	agencyA      = "agencyAAAAAAAAAAA"
	agencyB      = "agencyBBBBBBBBBBB"
	agencyMain   = "agencyMAAAAAAAAAA"
	typeID       = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"
	otherTypeID  = "9a8b7c6d-5e4f-4a3b-9c2d-1e0f2a3b4c5d"
	sharedTypeID = "5c4d3e2f-1a0b-4c9d-8e7f-6a5b4c3d2e1f"
	missingID    = "0e1f2a3b-4c5d-4e6f-8a7b-9c0d1e2f3a4b"
)

// storedAt es la fecha de alta de los tipos que ya están guardados.
var storedAt = time.Date(2026, 1, 10, 15, 0, 0, 0, time.UTC)

// findCall son los argumentos de una llamada a Find.
type findCall struct {
	agencyIDs  []string
	onlyActive bool
}

// fakeRepository guarda los tipos en memoria y registra las llamadas.
type fakeRepository struct {
	bondType_ports.BondTypeMongoccRepositoryPorts

	types map[string]*bondType_domain.BondType
	// found es lo que devuelve Find.
	found []*bondType_domain.BondType
	// staleUpdate hace que UpdateOne no encuentre el tipo, como si se hubiera borrado después de leerlo.
	staleUpdate bool

	finds    []findCall
	reads    []string
	inserted []bondType_domain.BondType
	updated  []bondType_domain.BondType
	deleted  []string
}

// newRepository es un repositorio con los tipos dados.
func newRepository(types ...*bondType_domain.BondType) *fakeRepository {
	r := &fakeRepository{types: map[string]*bondType_domain.BondType{}}
	for _, t := range types {
		stored := *t
		r.types[t.ID] = &stored
	}
	return r
}

func (r *fakeRepository) Find(_ context.Context, agencyIDs []string, onlyActive bool) ([]*bondType_domain.BondType, error) {
	r.finds = append(r.finds, findCall{agencyIDs: slices.Clone(agencyIDs), onlyActive: onlyActive})
	return r.found, nil
}

// FindOneByID devuelve una copia del tipo, o nil si no existe o está borrado.
func (r *fakeRepository) FindOneByID(_ context.Context, id string) (*bondType_domain.BondType, error) {
	r.reads = append(r.reads, id)
	t, ok := r.types[id]
	if !ok || t.IsRemove {
		return nil, nil
	}
	found := *t
	return &found, nil
}

func (r *fakeRepository) InsertOne(_ context.Context, t *bondType_domain.BondType) error {
	r.inserted = append(r.inserted, *t)
	stored := *t
	r.types[t.ID] = &stored
	return nil
}

// UpdateOne guarda t si el tipo existe y no está borrado.
func (r *fakeRepository) UpdateOne(_ context.Context, t *bondType_domain.BondType) (bool, error) {
	r.updated = append(r.updated, *t)
	current, ok := r.types[t.ID]
	if r.staleUpdate || !ok || current.IsRemove {
		return false, nil
	}
	stored := *t
	r.types[t.ID] = &stored
	return true, nil
}

func (r *fakeRepository) SoftDelete(_ context.Context, id string) error {
	r.deleted = append(r.deleted, id)
	if t, ok := r.types[id]; ok {
		t.IsRemove = true
	}
	return nil
}

// countCall son los argumentos de una llamada a CountActiveByType.
type countCall struct {
	ids    []string
	userID string
}

// fakeCounter responde los conteos dados y registra las llamadas.
type fakeCounter struct {
	bondType_ports.BondObjectCounterPort

	counts map[string]int
	inUse  bool
	err    error

	countCalls  []countCall
	existsCalls []string
}

func (c *fakeCounter) CountActiveByType(_ context.Context, ids []string, userID string) (map[string]int, error) {
	c.countCalls = append(c.countCalls, countCall{ids: slices.Clone(ids), userID: userID})
	if c.err != nil {
		return nil, c.err
	}
	return c.counts, nil
}

func (c *fakeCounter) ExistsNotRemovedByType(_ context.Context, id string) (bool, error) {
	c.existsCalls = append(c.existsCalls, id)
	return c.inUse, c.err
}

// newApp arma la aplicación con los fakes.
func newApp(repo *fakeRepository, counter *fakeCounter) *bondType_application.BondTypeApplication {
	return &bondType_application.BondTypeApplication{Repository: repo, BondObjects: counter}
}

// withSession es un context con la sesión de userID en agencyID.
func withSession(userID, agencyID string) context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{UserID: userID, AgencyID: agencyID})
}

// withAdmin es un context con la sesión de un admin de Blooming en agencyID; isMain indica si es la agencia
// principal.
func withAdmin(agencyID string, isMain bool) context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID:        "adminAAAAAAAAAAAA",
		AgencyID:      agencyID,
		Agency:        session_domain.SessionAgency{ID: agencyID, IsMain: isMain},
		AdminBlooming: true,
	})
}

// storedType es un tipo activo de la agencia A guardado el 10 de enero.
func storedType(id string) *bondType_domain.BondType {
	return &bondType_domain.BondType{
		ID:          id,
		AgencyID:    []string{agencyA},
		Name:        "Bono Plus",
		Description: "Renta fija 12 meses",
		Active:      true,
		CreatedAt:   storedAt,
		UpdatedAt:   storedAt,
	}
}

// agencyTypes son los tipos de las pruebas de alcance: typeID de A, otherTypeID de B y sharedTypeID
// compartido por A y B.
func agencyTypes() []*bondType_domain.BondType {
	onlyB := storedType(otherTypeID)
	onlyB.AgencyID = []string{agencyB}
	shared := storedType(sharedTypeID)
	shared.AgencyID = []string{agencyA, agencyB}
	return []*bondType_domain.BondType{storedType(typeID), onlyB, shared}
}
