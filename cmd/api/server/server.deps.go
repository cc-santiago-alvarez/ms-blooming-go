package server

import (
	"github.com/codecraftkit/mongocc"
)

// serverDeps holds all repositories and applications wired during server initialization.
type serverDeps struct {
	// ====================== Database Connections ======================
	mongoDB *mongocc.MongoQueries
}
