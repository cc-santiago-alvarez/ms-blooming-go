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
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	oauth_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/adapters/oauth"
	shared_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/mongocc"
	middlewares_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/middlewares"
)

func initServerConfig() (*nexus.Server, *serverDeps) {
	// 0. Cargar configuración
	cfg := api_config.InitConfig()
	deps := &serverDeps{env: cfg.Env}
	session_domain.SetDefaultUtcDiff(cfg.Env.DefaultUtcDiff)

	// 1. Conexiones a bases de datos
	initDatabaseConnections(deps)

	// 2. Adaptadores compartidos
	deps.sessionVerifier = oauth_adapters.NewSessionClient(cfg.Env.OAuthURL, cfg.Env.OAuthSecret)
	deps.profileReader = &shared_mongocc_repository.ProfileMongoccRepository{DB: deps.mongoDB}
	deps.activityPort = &shared_mongocc_repository.ActivityMongoccRepository{DB: deps.mongoDB}

	// 3. CORS
	corsOptions, err := middlewares_adapters.NewCorsOptions(cfg.Env.CorsOrigins)
	if err != nil {
		log.Fatalf("[server] CORS_ORIGINS inválido: %v", err)
	}
	if middlewares_adapters.AllowsAnyOrigin(cfg.Env.CorsOrigins) {
		log.Printf("[server] CORS abierto a cualquier origen")
	}

	// 4. Crear servidor Nexus
	server := &nexus.Server{
		ServerName:  cfg.Env.ApiName,
		Secret:      cfg.Env.ApiSecret,
		Port:        cfg.Env.ApiPort,
		Debug:       cfg.Env.ApiDebug,
		CorsOptions: corsOptions,
		Settings: &nexus.Settings{
			PathPrefix: "/api/v1",
		},
	}

	// 5. Middlewares (orden importa: el primero envuelve a los demás).
	// RequestLog va primero para registrar también los 500 de recoverPanic y los 401 de autenticación.
	server.Use(middlewares_adapters.RequestLog)
	server.Use(hideRoutesUnlessDebug)
	server.Use(recoverPanic)
	server.Use(middlewares_adapters.ValidateApiSecret)
	server.Use(middlewares_adapters.VerifySession(deps.sessionVerifier))

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

// hideRoutesUnlessDebug responde 404 en /_routes y /_routes/raw cuando API_DEBUG es falso.
func hideRoutesUnlessDebug(next http.Handler, server *nexus.Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !server.Debug && (r.URL.Path == "/_routes" || strings.HasPrefix(r.URL.Path, "/_routes/")) {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// recoverPanic convierte el panic de un handler en 500 ERR_INTERNAL y loguea el stack.
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
