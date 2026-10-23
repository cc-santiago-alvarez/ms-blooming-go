package bondBound_application_test

import (
	"context"
	"time"

	bondBound_application "github.com/futurapps/ms-blooming-go/internal/core/bondBound/application"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

const (
	agencyA     = "agencyAAAAAAAAAAA"
	ownerID     = "user1AAAAAAAAAAAA"
	otherUserID = "user2AAAAAAAAAAAA"
	adminID     = "adminAAAAAAAAAAAA"
	boundID     = "5a6b7c8d-9e0f-4a1b-8c2d-3e4f5a6b7c8d"
	missingID   = "0e1f2a3b-4c5d-4e6f-8a7b-9c0d1e2f3a4b"

	// legacyBoundID es un _id Meteor de 17 caracteres.
	legacyBoundID = "Kp3Wq8Zr5Tn2Yx7Lm"
)

// boundCreatedAt es la fecha de compra de los bonos guardados.
var boundCreatedAt = time.Date(2026, 1, 1, 15, 0, 0, 0, time.UTC)

// fakeRepository guarda los bonos en memoria y registra las lecturas.
type fakeRepository struct {
	bondBound_ports.BondBoundMongoccRepositoryPorts

	bounds map[string]*bondBound_domain.BondBound
	err    error

	reads []string
}

// FindOneByID devuelve err si está puesto, nil si el bono no existe o está borrado, y si no una copia.
func (r *fakeRepository) FindOneByID(_ context.Context, id string) (*bondBound_domain.BondBound, error) {
	r.reads = append(r.reads, id)
	if r.err != nil {
		return nil, r.err
	}
	b, ok := r.bounds[id]
	if !ok || b.IsRemove {
		return nil, nil
	}
	found := *b
	return &found, nil
}

// fakes son las dependencias de la aplicación.
type fakes struct {
	repo *fakeRepository
}

// newFakes arma las dependencias con los bonos dados.
func newFakes(bounds ...*bondBound_domain.BondBound) *fakes {
	repo := &fakeRepository{bounds: map[string]*bondBound_domain.BondBound{}}
	for _, b := range bounds {
		stored := *b
		repo.bounds[b.ID] = &stored
	}
	return &fakes{repo: repo}
}

// app arma la aplicación con los fakes.
func (f *fakes) app() *bondBound_application.BondBoundApplication {
	return &bondBound_application.BondBoundApplication{Repository: f.repo}
}

// withSession es un context con la sesión de userID en la agencia A y la zona utcDiff.
func withSession(userID string, utcDiff int) context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID:   userID,
		AgencyID: agencyA,
		UtcDiff:  session_domain.OptionalUtcDiff{Hours: utcDiff, Valid: true},
	})
}

// withAdmin es un context con la sesión de un admin de Blooming en la agencia A, en UTC-5.
func withAdmin() context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID:        adminID,
		AgencyID:      agencyA,
		AdminBlooming: true,
		UtcDiff:       session_domain.OptionalUtcDiff{Hours: -5, Valid: true},
	})
}

// storedBound es un bono iniciado de ownerID, comprado el 1 de enero a las 15:00 UTC.
func storedBound(id string) *bondBound_domain.BondBound {
	return &bondBound_domain.BondBound{
		ID:                        id,
		Key:                       "BP12-1",
		UserID:                    ownerID,
		AgencyID:                  agencyA,
		StatusID:                  bondBound_domain.StatusInitiated,
		Amount:                    1000,
		MaturityTimeInDays:        365,
		MaturityRewardsPercentage: 15,
		EarlyCashTerm: shared_domain.EarlyCashTerm{
			InitialPenaltyPercentage:    100,
			InflectionPoint:             30,
			InflectionPenaltyPercentage: 75,
			InitialDayForWithdraw:       10,
		},
		InitialDate:  boundCreatedAt,
		MaturityDate: boundCreatedAt.AddDate(0, 0, 365),
		CreatedAt:    boundCreatedAt,
		UpdatedAt:    boundCreatedAt,
	}
}
