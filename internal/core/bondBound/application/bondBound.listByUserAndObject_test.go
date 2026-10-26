package bondBound_application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// legacyProductID es un _id Meteor de 17 caracteres de un producto.
const legacyProductID = "Bd9Xc4Vn7Mz2Lk5Jh"

// Consulta los bonos del usuario en el producto dentro del alcance de la sesión: su agencia, o todas si es
// la principal. El bondObjectId se recorta y se aceptan ids legacy. CA-06.7
func TestListByUserAndObject(t *testing.T) {
	tests := []struct {
		name       string
		ctx        context.Context
		userID     string
		objectID   string
		wantObject string
		wantScope  *string
	}{
		{name: "admin de la agencia A", ctx: withAdmin(), userID: ownerID, objectID: objectID, wantObject: objectID, wantScope: agency(agencyA)},
		{name: "admin de la agencia principal", ctx: withMainAdmin(), userID: ownerID, objectID: objectID, wantObject: objectID},
		{name: "bondObjectId con espacios", ctx: withAdmin(), userID: ownerID, objectID: " " + objectID + " ", wantObject: objectID, wantScope: agency(agencyA)},
		{name: "producto legacy", ctx: withAdmin(), userID: ownerID, objectID: legacyProductID, wantObject: legacyProductID, wantScope: agency(agencyA)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()
			f.repo.listed = []*bondBound_domain.BondBound{storedBound(boundID), storedBound(legacyBoundID)}

			bounds, err := f.app().ListByUserAndObject(tt.ctx, tt.userID, tt.objectID)
			if err != nil {
				t.Fatalf("ListByUserAndObject: %v", err)
			}

			want := []repoQuery{{method: "ListByUserAndObject", userID: tt.userID, bondObjectID: tt.wantObject, agencyID: tt.wantScope}}
			if !reflect.DeepEqual(f.repo.queries, want) {
				t.Errorf("consultas %+v, se esperaba %+v", f.repo.queries, want)
			}
			if !reflect.DeepEqual(bounds, f.repo.listed) {
				t.Errorf("ListByUserAndObject devolvió %+v, se esperaba lo del repositorio", bounds)
			}
		})
	}
}

// Sin bondObjectId es ErrBondObjectIDRequired y un id fuera de formato es ErrInvalidID; en los dos casos no
// consulta. CA-06.7
func TestListByUserAndObjectInvalid(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		objectID string
		wantErr  error
	}{
		{name: "sin bondObjectId", userID: ownerID, objectID: "", wantErr: bondBound_domain.ErrBondObjectIDRequired},
		{name: "bondObjectId de espacios", userID: ownerID, objectID: "  ", wantErr: bondBound_domain.ErrBondObjectIDRequired},
		{name: "bondObjectId inválido", userID: ownerID, objectID: "abc", wantErr: shared_domain.ErrInvalidID},
		{name: "userId inválido", userID: "abc", objectID: objectID, wantErr: shared_domain.ErrInvalidID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()

			bounds, err := f.app().ListByUserAndObject(withAdmin(), tt.userID, tt.objectID)

			if !errors.Is(err, tt.wantErr) || bounds != nil {
				t.Fatalf("ListByUserAndObject = %+v, %v, se esperaba %v", bounds, err, tt.wantErr)
			}
			assertNoQueries(t, f)
		})
	}
}

// Un error del repositorio se devuelve sin cambiarlo. CA-06.7
func TestListByUserAndObjectRepositoryError(t *testing.T) {
	f := newFakes()
	f.repo.err = errors.New("mongo caído")

	_, err := f.app().ListByUserAndObject(withAdmin(), ownerID, objectID)

	if !errors.Is(err, f.repo.err) {
		t.Fatalf("ListByUserAndObject = %v, se esperaba el error del repositorio", err)
	}
}
