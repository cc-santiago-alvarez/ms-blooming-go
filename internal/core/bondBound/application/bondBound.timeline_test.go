package bondBound_application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// timelineItems son las compras que responde el repositorio.
func timelineItems() []bondBound_ports.TimelineItem {
	return []bondBound_ports.TimelineItem{
		{ID: boundID, Text: "Bono Plus 12M (BP12-1)", Start: "2026-12-31 15:00", End: "2026-12-31 15:00"},
	}
}

// Sin userId, o con el propio, son las compras del usuario de la sesión; las de otro usuario exigen admin,
// y un no admin recibe ErrForbidden sin consultar. Siempre se consulta en la zona de la sesión. CA-06.12
func TestTimelineAccess(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		userID   string
		wantUser string
		wantZone int
		wantErr  error
	}{
		{name: "sin userId", ctx: withSession(ownerID, -5), userID: "", wantUser: ownerID, wantZone: -5},
		{name: "userId con espacios", ctx: withSession(ownerID, 14), userID: "  ", wantUser: ownerID, wantZone: 14},
		{name: "el propio userId", ctx: withSession(ownerID, -5), userID: ownerID, wantUser: ownerID, wantZone: -5},
		{name: "admin con userId ajeno", ctx: withAdmin(), userID: otherUserID, wantUser: otherUserID, wantZone: -5},
		{name: "admin sin userId", ctx: withAdmin(), userID: "", wantUser: adminID, wantZone: -5},
		{name: "no admin con userId ajeno", ctx: withSession(ownerID, -5), userID: otherUserID, wantErr: shared_domain.ErrForbidden},
		{name: "userId inválido", ctx: withAdmin(), userID: "abc", wantErr: shared_domain.ErrInvalidID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()
			f.repo.timeline = timelineItems()

			items, err := f.app().Timeline(tt.ctx, tt.userID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || items != nil {
					t.Fatalf("Timeline = %+v, %v, se esperaba %v", items, err, tt.wantErr)
				}
				assertNoQueries(t, f)
				return
			}
			if err != nil {
				t.Fatalf("Timeline: %v", err)
			}
			want := []repoQuery{{method: "Timeline", userID: tt.wantUser, utcDiff: tt.wantZone}}
			if !reflect.DeepEqual(f.repo.queries, want) {
				t.Errorf("consultas %+v, se esperaba %+v", f.repo.queries, want)
			}
			if !reflect.DeepEqual(items, timelineItems()) {
				t.Errorf("Timeline devolvió %+v, se esperaba lo del repositorio", items)
			}
		})
	}
}

// Sin sesión es un error y no consulta. CA-06.12
func TestTimelineWithoutSession(t *testing.T) {
	f := newFakes()

	items, err := f.app().Timeline(context.Background(), "")

	if err == nil || items != nil {
		t.Fatalf("Timeline = %+v, %v, se esperaba un error", items, err)
	}
	assertNoQueries(t, f)
}

// Un error del repositorio se devuelve sin cambiarlo. CA-06.12
func TestTimelineRepositoryError(t *testing.T) {
	f := newFakes()
	f.repo.err = errors.New("mongo caído")

	_, err := f.app().Timeline(withSession(ownerID, -5), "")

	if !errors.Is(err, f.repo.err) {
		t.Fatalf("Timeline = %v, se esperaba el error del repositorio", err)
	}
}
