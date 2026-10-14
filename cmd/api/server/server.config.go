package server

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/codecraftkit/mongocc"
	"github.com/codecraftkit/nexus"
	api_config "github.com/futurapps/ms-blooming-go/cmd/api/config"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	"github.com/rs/cors"
)

func initServerConfig() (*nexus.Server, *serverDeps) {
	// 0. Cargar configuración
	cfg := api_config.InitConfig()
	deps := &serverDeps{env: cfg.Env}

	// 1. Conexiones a bases de datos
	initDatabaseConnections(deps)

	// 2. Crear servidor Nexus
	server := &nexus.Server{
		ServerName: cfg.Env.ApiName,
		Secret:     cfg.Env.ApiSecret,
		Port:       cfg.Env.ApiPort,
		Debug:      cfg.Env.ApiDebug,
		CorsOptions: cors.Options{
			// Con opciones vacías rs/cors acepta cualquier origen. Hasta que BLM-0008 configure
			// CORS_ORIGINS (RF-00.6) no se acepta ninguno.
			AllowOriginFunc: func(string) bool { return false },
		},
		Settings: &nexus.Settings{
			PathPrefix: "/api/v1",
		},
	}

	// 3. Middlewares (orden importa: el primero envuelve a los demás).
	// RequestLog (BLM-0003) va antes de recoverPanic para registrar también los 500;
	// ValidateApiSecret (BLM-0004) y VerifySession (BLM-0005) van después.
	server.Use(hideRoutesUnlessDebug)
	server.Use(recoverPanic)

	return server, deps
}

func initDatabaseConnections(deps *serverDeps) {
	// MongoDB. mongocc oculta las credenciales del URI en su log de conexión.
	mongoOpts := mongocc.ClientOptions{Debug: deps.env.FuturappsMongoDebug}
	mongoDB, err := mongocc.Connect(deps.env.FuturappsMongoUrl, deps.env.FuturappsMongoDbName, &mongoOpts)
	if err != nil {
		log.Fatalf("[server] no se pudo conectar a MongoDB: %v", err)
	}
	deps.mongoDB = mongoDB
}

// hideRoutesUnlessDebug oculta /_routes y /_routes/raw cuando API_DEBUG es falso (spec 00 §4).
// nexus los registra siempre como públicos; sin debug responden como una ruta inexistente.
func hideRoutesUnlessDebug(next http.Handler, server *nexus.Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !server.Debug && (r.URL.Path == "/_routes" || strings.HasPrefix(r.URL.Path, "/_routes/")) {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// recoverPanic convierte el panic de un handler en 500 ERR_INTERNAL; el detalle y el stack solo
// van al log (RF-00.7).
func recoverPanic(next http.Handler, _ *nexus.Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			// net/http usa ErrAbortHandler para cortar una respuesta a propósito.
			if rec == http.ErrAbortHandler {
				panic(rec)
			}
			handlers_shared.ResponseWithError(w, r, http.StatusInternalServerError, fmt.Errorf("panic: %v\n%s", rec, debug.Stack()))
		}()
		next.ServeHTTP(w, r)
	})
}
