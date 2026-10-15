package shared_mongocc_repository

import (
	"context"
	"log"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Add guarda el aviso. Se escribe aunque ctx ya esté cancelado; un fallo solo se loguea.
func (repo *NoticeMongoccRepository) Add(ctx context.Context, n shared_domain.Notice) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), bestEffortTimeout)
	defer cancel()

	if _, err := repo.DB.InsertOne(ctx, noticeCollection, shared_domain.NewNoticeRecord(n)); err != nil {
		log.Printf("[NoticePort] no se pudo guardar el aviso para %s: %v", n.UserID, err)
	}
}
