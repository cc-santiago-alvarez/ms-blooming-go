//go:build integration

package bondObject_mongocc_repository_test

import (
	"context"
	"regexp"
	"slices"
	"testing"
	"time"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Para un usuario de la agencia A no autorizado en el privado P, el listado del tipo excluye P, los
// inactivos, los borrados, los de la agencia B y los de otro tipo, ordena por createdAt descendente e
// incluye maturityDateFormatted; si el usuario está en authorizedUserIds de P, P aparece. CA-02.11
func TestListByType(t *testing.T) {
	older, newer, private := meteor1, uuid1, meteor2

	tests := []struct {
		name       string
		authorized bson.A
		want       []string
	}{
		{name: "sin autorizar en P", authorized: bson.A{user2ID}, want: []string{newer, older}},
		{name: "autorizado en P", authorized: bson.A{user2ID, userID}, want: []string{private, newer, older}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)
			insert(t, objectCollection,
				objectDoc(older, "Antiguo", typeID, agencyA, nil),
				objectDoc(newer, "Nuevo", typeID, bson.A{agencyA, agencyB},
					bson.M{"createdAt": fixtureAt.Add(time.Hour), "maturityTimeInDays": daysUntilNextFifth()}),
				objectDoc(private, "Privado", typeID, bson.A{agencyA},
					bson.M{"createdAt": fixtureAt.Add(2 * time.Hour), "isPrivateBond": true, "authorizedUserIds": tt.authorized}),
				objectDoc(uuid2, "Inactivo", typeID, bson.A{agencyA}, bson.M{"active": false}),
				objectDoc(meteor3, "De B", typeID, bson.A{agencyB}, nil),
				objectDoc(uuid3, "Borrado", typeID, bson.A{agencyA}, bson.M{"isRemove": true}),
				objectDoc(meteor4, "Otro tipo", legacyTypeID, bson.A{agencyA}, nil),
			)

			before := time.Now()
			got, err := newApp().ListByType(investorSession(userID, agencyA), typeID)
			after := time.Now()

			if err != nil {
				t.Fatalf("ListByType() = %v", err)
			}
			gotIDs := make([]string, len(got))
			for i, o := range got {
				gotIDs[i] = o.ID
			}
			if !slices.Equal(gotIDs, tt.want) {
				t.Fatalf("ListByType() = %v, se esperaba %v", gotIDs, tt.want)
			}
			for _, o := range got {
				assertMaturity(t, o, before, after)
				if o.ID == newer && !singleDigitDayLabel.MatchString(o.MaturityDateFormatted) {
					t.Errorf("maturityDateFormatted de %s = %q, se esperaba el formato D MMM YYYY con el día en una cifra",
						o.ID, o.MaturityDateFormatted)
				}
			}
		})
	}
}

// Dados P con isPrivateBond string y Q con isPrivateBond "" del mismo tipo, P se lee por id como privado; el
// listado del tipo devuelve Q sin P a un usuario no autorizado, y P aparece si el usuario está en
// authorizedUserIds de P. CA-02.17
func TestStringIsPrivateBond(t *testing.T) {
	repo := setup(t)
	private, public := meteor1, uuid1
	insert(t, objectCollection,
		objectDoc(private, "Privado", typeID, bson.A{agencyA}, bson.M{
			"createdAt":         fixtureAt.Add(time.Hour),
			"isPrivateBond":     "A8DjCnAETjXarQ2YA",
			"authorizedUserIds": bson.A{user2ID},
		}),
		objectDoc(public, "Público", typeID, bson.A{agencyA}, bson.M{"isPrivateBond": ""}),
	)

	t.Run("el detalle lo lee como privado", func(t *testing.T) {
		got, err := repo.FindOneByID(context.Background(), private)
		if err != nil || got == nil {
			t.Fatalf("FindOneByID(%s) = %v, %v", private, got, err)
		}
		if !got.IsPrivateBond {
			t.Errorf("isPrivateBond de %s = false, se esperaba true", private)
		}
	})

	tests := []struct {
		name   string
		userID string
		want   []string
	}{
		{name: "sin autorizar en P", userID: userID, want: []string{public}},
		{name: "autorizado en P", userID: user2ID, want: []string{private, public}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newApp().ListByType(investorSession(tt.userID, agencyA), typeID)
			if err != nil {
				t.Fatalf("ListByType() = %v", err)
			}
			gotIDs := make([]string, len(got))
			for i, o := range got {
				gotIDs[i] = o.ID
			}
			if !slices.Equal(gotIDs, tt.want) {
				t.Fatalf("ListByType() = %v, se esperaba %v", gotIDs, tt.want)
			}
			for _, o := range got {
				if want := o.ID == private; o.IsPrivateBond != want {
					t.Errorf("isPrivateBond de %s = %v, se esperaba %v", o.ID, o.IsPrivateBond, want)
				}
			}
		})
	}
}

// singleDigitDayLabel es el formato legacy "D MMM YYYY" de un día de una cifra, por ejemplo "5 Mar 2027".
var singleDigitDayLabel = regexp.MustCompile(`^[1-9] [A-Z][a-z]{2} \d{4}$`)

// daysUntilNextFifth son los días desde hoy, en la zona de la sesión, hasta el 5 del mes siguiente: una
// maduración que cae en un día de una cifra.
func daysUntilNextFifth() int {
	now := time.Now().In(time.FixedZone("", investorUtcDiff*3600))
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	fifth := time.Date(now.Year(), now.Month()+1, 5, 0, 0, 0, 0, time.UTC)
	return int(fifth.Sub(today).Hours() / 24)
}

// assertMaturity verifica la fecha de maduración de una compra hecha entre before y after.
func assertMaturity(t *testing.T, o bondObject_domain.BondObjectForInvestor, before, after time.Time) {
	t.Helper()
	for _, at := range []time.Time{before, after} {
		estimated, formatted := expectedMaturity(at, o.MaturityTimeInDays)
		if o.MaturityDateEstimated.Equal(estimated) && o.MaturityDateFormatted == formatted {
			return
		}
	}
	estimated, formatted := expectedMaturity(before, o.MaturityTimeInDays)
	t.Errorf("maduración de %s = %v %q, se esperaba %v %q",
		o.ID, o.MaturityDateEstimated, o.MaturityDateFormatted, estimated, formatted)
}

// expectedMaturity es el inicio del día de at en la zona de la sesión más days días, y esa fecha con el
// formato legacy "D MMM YYYY".
func expectedMaturity(at time.Time, days int) (time.Time, string) {
	zone := time.FixedZone("", investorUtcDiff*3600)
	local := at.In(zone)
	maturity := time.Date(local.Year(), local.Month(), local.Day()+days, 0, 0, 0, 0, zone)
	return maturity.UTC(), maturity.Format("2 Jan 2006")
}

// Latest devuelve como máximo 3 productos, por createdAt descendente, solo de la agencia de la sesión y
// sin privados no autorizados ni borrados. CA-02.13
func TestLatest(t *testing.T) {
	setup(t)
	at := func(hours int) bson.M {
		return bson.M{"createdAt": fixtureAt.Add(time.Duration(hours) * time.Hour)}
	}
	insert(t, objectCollection,
		objectDoc(meteor1, "Uno", typeID, bson.A{agencyA}, at(1)),
		objectDoc(uuid1, "Dos", legacyTypeID, agencyA, at(2)),
		objectDoc(meteor2, "Tres", typeID, bson.A{agencyA, agencyB}, at(3)),
		objectDoc(uuid2, "Cuatro", typeID, bson.A{agencyA}, at(4)),
		objectDoc(meteor3, "Cinco", legacyTypeID, bson.A{agencyA}, at(5)),
		objectDoc(uuid3, "De B", typeID, bson.A{agencyB}, at(6)),
		objectDoc(meteor4, "Privado", typeID, bson.A{agencyA},
			bson.M{"createdAt": fixtureAt.Add(7 * time.Hour), "isPrivateBond": true, "authorizedUserIds": bson.A{user2ID}}),
		objectDoc(uuid4, "Borrado", typeID, bson.A{agencyA},
			bson.M{"createdAt": fixtureAt.Add(8 * time.Hour), "isRemove": true}),
	)

	got, err := newApp().Latest(investorSession(userID, agencyA))

	if err != nil {
		t.Fatalf("Latest() = %v", err)
	}
	if want := []string{meteor3, uuid2, meteor2}; !slices.Equal(ids(got), want) {
		t.Fatalf("Latest() = %v, se esperaba %v", ids(got), want)
	}
}
