//go:build integration

package bondBound_mongocc_repository_test

import (
	"context"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// La baja deja isRemove true y actualiza updatedAt; repetirla o pedirla para un id inexistente devuelve
// false. El bono borrado deja de salir en el listado de administración y en mis bonos. CA-06.13
func TestSoftDelete(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	insert(t, boundCollection, boundAt(legacyBoundID, 0, nil), boundAt(boundID, 1, nil))
	ctx := context.Background()

	ok, err := repo.SoftDelete(ctx, legacyBoundID)
	if err != nil || !ok {
		t.Fatalf("SoftDelete = %v, %v; se esperaba true", ok, err)
	}

	raw := rawDoc(t, boundCollection, legacyBoundID)
	if !raw.Lookup("isRemove").Boolean() {
		t.Errorf("isRemove %v, se esperaba true", raw.Lookup("isRemove"))
	}
	if updatedAt := raw.Lookup("updatedAt").Time(); !updatedAt.After(fixtureAt) {
		t.Errorf("updatedAt %v, se esperaba posterior a %v", updatedAt, fixtureAt)
	}
	if other := rawDoc(t, boundCollection, boundID); other.Lookup("isRemove").Boolean() {
		t.Errorf("la baja también borró %s", boundID)
	}

	for _, id := range []string{legacyBoundID, missingID} {
		if ok, err := repo.SoftDelete(ctx, id); err != nil || ok {
			t.Errorf("SoftDelete(%s) = %v, %v; se esperaba false", id, ok, err)
		}
	}

	admin, err := repo.ListAdmin(ctx, agencyScope(agencyA), bondBound_domain.ListFilters{}, shared_domain.Pagination{Page: 1, Limit: 10}, -5)
	if err != nil {
		t.Fatalf("ListAdmin: %v", err)
	}
	if got := adminRowIDs(admin.Items); !slices.Equal(got, []string{boundID}) {
		t.Errorf("listado de administración %v, se esperaba solo %s", got, boundID)
	}
	mine, err := repo.ListMine(ctx, userID, agencyA, nil, shared_domain.Pagination{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("ListMine: %v", err)
	}
	if got := withObjectIDs(mine.Items); !slices.Equal(got, []string{boundID}) {
		t.Errorf("mis bonos %v, se esperaba solo %s", got, boundID)
	}
}
