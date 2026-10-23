package bondUserSetting_application_test

import (
	"context"
	"slices"
	"strings"
	"time"

	bondUserSetting_application "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/application"
	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

const (
	agencyA    = "agencyAAAAAAAAAAA"
	agencyB    = "agencyBBBBBBBBBBB"
	agencyMain = "agencyMAAAAAAAAAA"
	userID     = "user1AAAAAAAAAAAA"
	user2ID    = "user2AAAAAAAAAAAA"
	user3ID    = "user3AAAAAAAAAAAA"

	settingID   = "6a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d"
	setting3ID  = "8d9e0f1a-2b3c-4d4e-a5f6-7a8b9c0d1e2f"
	missingID   = "0e1f2a3b-4c5d-4e6f-8a7b-9c0d1e2f3a4b"
	bondBoundID = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"

	// legacySettingID es un _id Meteor de 17 caracteres.
	legacySettingID = "Bd9Xc4Vn7Mz2Lk5Jh"

	// accountOld y accountNew son cuentas de userID, la primera es la más antigua; accountOther es de
	// user2ID.
	accountOld   = "account1AAAAAAAAA"
	accountNew   = "account2AAAAAAAAA"
	accountOther = "account3AAAAAAAAA"
)

// storedAt es la fecha de alta de los settings que ya están guardados.
var storedAt = time.Date(2026, 1, 10, 15, 0, 0, 0, time.UTC)

// upsertCall es una llamada a UpsertField.
type upsertCall struct {
	setting bondUserSetting_domain.BondUserSetting
	field   bondUserSetting_ports.SettingField
}

// pageCall es una llamada a FindPage.
type pageCall struct {
	agencyID *string
	p        shared_domain.Pagination
}

// fakeRepository guarda los settings en memoria, emula los upserts del adapter y registra las llamadas.
type fakeRepository struct {
	bondUserSetting_ports.BondUserSettingMongoccRepositoryPorts

	settings map[string]*bondUserSetting_domain.BondUserSetting

	userReads []string
	defaults  []bondUserSetting_domain.BondUserSetting
	upserts   []upsertCall
	pages     []pageCall
	reads     []string
	deleted   []string
}

// live devuelve el setting vivo de userID, o nil.
func (r *fakeRepository) live(userID string) *bondUserSetting_domain.BondUserSetting {
	for _, s := range r.settings {
		if s.UserID == userID && !s.IsRemove {
			return s
		}
	}
	return nil
}

func (r *fakeRepository) FindByUser(_ context.Context, userID string) (*bondUserSetting_domain.BondUserSetting, error) {
	r.userReads = append(r.userReads, userID)
	return copyOf(r.live(userID)), nil
}

// UpsertDefault devuelve el setting vivo de s.UserID; si no tiene, guarda s.
func (r *fakeRepository) UpsertDefault(_ context.Context, s *bondUserSetting_domain.BondUserSetting) (*bondUserSetting_domain.BondUserSetting, error) {
	r.defaults = append(r.defaults, *s)
	if existing := r.live(s.UserID); existing != nil {
		return copyOf(existing), nil
	}
	r.settings[s.ID] = copyOf(s)
	return copyOf(s), nil
}

// UpsertField cambia solo field y updatedAt en el setting vivo de s.UserID; si no tiene, guarda s entero.
func (r *fakeRepository) UpsertField(_ context.Context, s *bondUserSetting_domain.BondUserSetting, field bondUserSetting_ports.SettingField) (*bondUserSetting_domain.BondUserSetting, error) {
	r.upserts = append(r.upserts, upsertCall{setting: *s, field: field})
	stored := r.live(s.UserID)
	if stored == nil {
		stored = copyOf(s)
		r.settings[s.ID] = stored
		return copyOf(stored), nil
	}
	switch field {
	case bondUserSetting_ports.FieldLanguage:
		stored.Language = s.Language
	case bondUserSetting_ports.FieldSelectedAccountID:
		stored.SelectedAccountID = s.SelectedAccountID
	case bondUserSetting_ports.FieldSelectedBondBoundID:
		stored.SelectedBondBoundID = s.SelectedBondBoundID
	}
	stored.UpdatedAt = shared_domain.Now()
	return copyOf(stored), nil
}

// FindPage devuelve la página p de los settings vivos de agencyID, o de todas si es nil, del más reciente
// al más antiguo. Registra cada llamada.
func (r *fakeRepository) FindPage(_ context.Context, agencyID *string, p shared_domain.Pagination) (shared_domain.Paginated[*bondUserSetting_domain.BondUserSetting], error) {
	r.pages = append(r.pages, pageCall{agencyID: agencyID, p: p})
	all := []*bondUserSetting_domain.BondUserSetting{}
	for _, s := range r.settings {
		if s.IsRemove || (agencyID != nil && s.AgencyID != *agencyID) {
			continue
		}
		all = append(all, copyOf(s))
	}
	slices.SortFunc(all, func(a, b *bondUserSetting_domain.BondUserSetting) int {
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	start := max(0, min((p.Page-1)*p.Limit, len(all)))
	end := max(start, min(start+p.Limit, len(all)))
	return shared_domain.NewPaginated(all[start:end], int64(len(all)), p), nil
}

// FindOneByID devuelve una copia del setting, o nil si no existe o está borrado.
func (r *fakeRepository) FindOneByID(_ context.Context, id string) (*bondUserSetting_domain.BondUserSetting, error) {
	r.reads = append(r.reads, id)
	s, ok := r.settings[id]
	if !ok || s.IsRemove {
		return nil, nil
	}
	return copyOf(s), nil
}

// SoftDelete marca el setting como borrado; devuelve false si no existe o ya estaba borrado.
func (r *fakeRepository) SoftDelete(_ context.Context, id string) (bool, error) {
	r.deleted = append(r.deleted, id)
	s, ok := r.settings[id]
	if !ok || s.IsRemove {
		return false, nil
	}
	s.IsRemove = true
	return true, nil
}

// accountCall es una llamada a GetAccount.
type accountCall struct {
	accountID string
	userID    string
}

// fakeBalance conoce las cuentas de accounts y registra las consultas.
type fakeBalance struct {
	shared_ports.BalancePort

	accounts map[string]shared_domain.BalanceAccount
	calls    []accountCall
}

// GetAccount devuelve la cuenta solo si existe y es de userID, como el adapter.
func (b *fakeBalance) GetAccount(_ context.Context, accountID, userID string) (*shared_domain.BalanceAccount, error) {
	b.calls = append(b.calls, accountCall{accountID: accountID, userID: userID})
	a, ok := b.accounts[accountID]
	if !ok || a.UserID != userID {
		return nil, nil
	}
	return &a, nil
}

// ownerCall es una consulta por usuario y agencia.
type ownerCall struct {
	userID   string
	agencyID string
}

// fakeFirstAccount conoce la cuenta más antigua de cada usuario y registra las consultas.
type fakeFirstAccount struct {
	bondUserSetting_ports.FirstAccountPort

	oldest map[string]string
	calls  []ownerCall
}

func (f *fakeFirstAccount) FirstAccountID(_ context.Context, userID, agencyID string) (*string, error) {
	f.calls = append(f.calls, ownerCall{userID: userID, agencyID: agencyID})
	id, ok := f.oldest[userID]
	if !ok {
		return nil, nil
	}
	return &id, nil
}

// fakeBondBounds responde un resumen vacío y registra las consultas.
type fakeBondBounds struct {
	bondUserSetting_ports.BoundSummaryPort

	calls []ownerCall
}

func (b *fakeBondBounds) BoundSummary(_ context.Context, userID, agencyID string) (bondUserSetting_domain.BoundSummary, error) {
	b.calls = append(b.calls, ownerCall{userID: userID, agencyID: agencyID})
	return bondUserSetting_domain.BoundSummary{}, nil
}

// fakeProfiles responde los perfiles que conoce.
type fakeProfiles struct {
	shared_ports.ProfileReader

	profiles map[string]shared_domain.PublicProfile
}

func (p *fakeProfiles) GetPublic(_ context.Context, ids ...string) ([]shared_domain.PublicProfile, error) {
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
	repo         *fakeRepository
	balance      *fakeBalance
	firstAccount *fakeFirstAccount
	bondBounds   *fakeBondBounds
	profiles     *fakeProfiles
}

// newFakes arma las dependencias con los settings dados. userID tiene las cuentas accountOld (la más
// antigua) y accountNew, y user2ID la cuenta accountOther.
func newFakes(settings ...*bondUserSetting_domain.BondUserSetting) *fakes {
	repo := &fakeRepository{settings: map[string]*bondUserSetting_domain.BondUserSetting{}}
	for _, s := range settings {
		repo.settings[s.ID] = copyOf(s)
	}
	return &fakes{
		repo: repo,
		balance: &fakeBalance{accounts: map[string]shared_domain.BalanceAccount{
			accountOld:   {ID: accountOld, UserID: userID, AgencyID: agencyA, StatusID: "1", Number: "3466-01-000000001"},
			accountNew:   {ID: accountNew, UserID: userID, AgencyID: agencyA, StatusID: "1", Number: "3466-01-000000002"},
			accountOther: {ID: accountOther, UserID: user2ID, AgencyID: agencyA, StatusID: "1", Number: "3466-01-000000003"},
		}},
		firstAccount: &fakeFirstAccount{oldest: map[string]string{userID: accountOld, user2ID: accountOther}},
		bondBounds:   &fakeBondBounds{},
		profiles:     &fakeProfiles{profiles: map[string]shared_domain.PublicProfile{}},
	}
}

// app arma la aplicación con los fakes.
func (f *fakes) app() *bondUserSetting_application.BondUserSettingApplication {
	return &bondUserSetting_application.BondUserSettingApplication{
		Repository:   f.repo,
		BondBounds:   f.bondBounds,
		FirstAccount: f.firstAccount,
		Balance:      f.balance,
		Profiles:     f.profiles,
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
		UserID:        "adminAAAAAAAAAAAA",
		AgencyID:      agencyID,
		Agency:        session_domain.SessionAgency{ID: agencyID, IsMain: isMain},
		AdminBlooming: true,
	})
}

// storedSetting es un setting vivo de userID en agencyID con idioma "es", sin cuenta ni bono, guardado el
// 10 de enero.
func storedSetting(id, userID, agencyID string) *bondUserSetting_domain.BondUserSetting {
	return &bondUserSetting_domain.BondUserSetting{
		ID:        id,
		UserID:    userID,
		AgencyID:  agencyID,
		Language:  bondUserSetting_domain.DefaultLanguage,
		CreatedAt: storedAt,
		UpdatedAt: storedAt,
	}
}

// copyOf devuelve una copia del setting, o nil.
func copyOf(s *bondUserSetting_domain.BondUserSetting) *bondUserSetting_domain.BondUserSetting {
	if s == nil {
		return nil
	}
	c := *s
	return &c
}

// ptr devuelve un puntero a s.
func ptr(s string) *string {
	return &s
}

// val devuelve el valor de p, o "<nil>", para los mensajes de error.
func val(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}
