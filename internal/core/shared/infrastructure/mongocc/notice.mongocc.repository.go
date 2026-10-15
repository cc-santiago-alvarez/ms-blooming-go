package shared_mongocc_repository

import (
	"time"

	"github.com/codecraftkit/mongocc"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

// Colección de avisos del módulo legado. No se crean índices.
const noticeCollection = "notice"

// bestEffortTimeout es lo máximo que espera un insert best-effort.
const bestEffortTimeout = 5 * time.Second

// NoticeMongoccRepository implementa NoticePort sobre notice.
type NoticeMongoccRepository struct {
	DB *mongocc.MongoQueries
}

var _ shared_ports.NoticePort = (*NoticeMongoccRepository)(nil)
