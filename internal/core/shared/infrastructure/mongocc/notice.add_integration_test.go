//go:build integration

package shared_mongocc_repository_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/mongocc"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// NoticePort.Add guarda el aviso con el typeId de su tipo, categoryId '1' y unread null, aunque el
// ctx del llamador esté cancelado. CA-00.20
func TestNoticeAdd(t *testing.T) {
	tests := []struct {
		name       string
		noticeType shared_domain.NoticeType
		wantTypeID string
		cancelled  bool
	}{
		{name: "aviso de bono", noticeType: shared_domain.NoticeTypeBond, wantTypeID: "3"},
		{name: "aviso de saldo", noticeType: shared_domain.NoticeTypeBalance, wantTypeID: "1"},
		{name: "ctx cancelado", noticeType: shared_domain.NoticeTypeBond, wantTypeID: "3", cancelled: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setup(t)
			ctx, cancel := context.WithCancel(context.Background())
			if tt.cancelled {
				cancel()
			} else {
				defer cancel()
			}
			repo := &shared_mongocc_repository.NoticeMongoccRepository{DB: db}

			repo.Add(ctx, shared_domain.Notice{
				Type:     tt.noticeType,
				UserID:   meteorUserID,
				AgencyID: uuidAgencyID,
				Title:    "Bono comprado",
				Message:  "Compraste el bono AB-3 por 1,500.00",
			})

			records := findAll[shared_domain.NoticeRecord](t, db, "notice")
			if len(records) != 1 {
				t.Fatalf("hay %d avisos, se esperaba 1", len(records))
			}
			r := records[0]
			assertUUIDv4(t, r.ID)
			assertFieldType(t, rawDoc(t, db, "notice", r.ID), "unread", bson.TypeNull)
			if time.Since(r.CreatedAt) > time.Minute {
				t.Fatalf("createdAt %v no es de ahora", r.CreatedAt)
			}
			want := shared_domain.NoticeRecord{
				ID:         r.ID,
				TypeID:     tt.wantTypeID,
				UserID:     meteorUserID,
				AgencyID:   uuidAgencyID,
				Message:    "Compraste el bono AB-3 por 1,500.00",
				Title:      "Bono comprado",
				CategoryID: "1",
				CreatedAt:  r.CreatedAt,
				UpdatedAt:  r.CreatedAt,
			}
			if !reflect.DeepEqual(r, want) {
				t.Fatalf("notice:\n got  %+v\n want %+v", r, want)
			}
		})
	}
}
