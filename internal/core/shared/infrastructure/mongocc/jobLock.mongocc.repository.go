package shared_mongocc_repository

import (
	"github.com/codecraftkit/mongocc"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

// Colección de los locks de los jobs. Es nueva y solo la usa este ms; alcanza con el índice de _id.
const jobLockCollection = "bondJobLock"

// JobLockMongoccRepository implementa JobLockPort sobre bondJobLock. Owner identifica a la réplica, con su
// hostname; solo esa réplica libera los locks que tomó.
type JobLockMongoccRepository struct {
	DB    *mongocc.MongoQueries
	Owner string
}

var _ shared_ports.JobLockPort = (*JobLockMongoccRepository)(nil)
