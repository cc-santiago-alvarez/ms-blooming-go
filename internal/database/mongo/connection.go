package mongo

import (
	"log"

	"github.com/codecraftkit/mongocc"
	api_config "github.com/futurapps/ms-petty-cash-api/cmd/api/config"
	_ "github.com/lib/pq"
)

var allowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}

var allowedHeaders = []string{
	"Origin",
	"X-Requested-With",
	"Content-Type",
	"Accept",
	"X-Secret",
	"X-Token",
}

// func initServerConfig() (*nexus.Server, *serverDeps) {
// 	deps := &serverDeps{}

// 	// 0. Cargar configuracion
// 	cfg := api_config.InitConfig()

// 	// Debug: print masked API secret and whether session secret is set
// 	maskedSecret := cfg.Env.ApiSecret
// 	if len(maskedSecret) > 8 {
// 		maskedSecret = maskedSecret[:4] + "..." + maskedSecret[len(maskedSecret)-4:]
// 	}

// 	// 1. Conexiones a bases de datos
// 	initDatabaseConnections(deps, cfg.Env)

// 	// 2. AppRole repo (necesario antes del middleware)
// 	deps.appRoleRepo = &appRole_mongocc_repository.AppRoleMongoccRepository{DB: deps.mongoDB}
// 	deps.initialAdminIDs = cfg.Env.InitialAdminIDs
// 	deps.notificationServiceURL = cfg.Env.NotificationServiceURL
// 	deps.notificationSecret = cfg.Env.NotificationSecret
// 	deps.balanceCheckCron = cfg.Env.BalanceCheckCron
// 	deps.linkAuditCron = cfg.Env.LinkAuditCron
// 	deps.telegramBotUsername = cfg.Env.TelegramBotUsername
// 	roleMiddleware := appRole_middleware.NewRoleMiddleware(deps.appRoleRepo)

// 	// 3. Adaptadores compartidos
// 	deps.utilsAdapter = utils.NewUtils()
// 	jwtccAdapter, err := jwtcc_adapters.NewJWTccAdapter(
// 		cfg.Env.JwtSecret,
// 		cfg.Env.JwtAlgorithm,
// 	)
// 	if err != nil {
// 		log.Fatalf("Failed to initialize JWT adapter: %v", err)
// 	}
// 	deps.jwtccAdapter = jwtccAdapter

// 	i18nAdapter, err := i18n.New("es")
// 	if err != nil {
// 		log.Fatalf("Failed to initialize i18n: %v", err)
// 	}

// 	// 3. Middleware
// 	middlewareConfig := &middlewares_adapters.MiddlewareAdapterConfig{
// 		Debug:                cfg.Env.ApiDebug,
// 		EnableRequestLog:     cfg.Env.EnableRequestLog,
// 		DefaultLang:          "es",
// 		JwtSecret:            cfg.Env.JwtSecret,
// 		JwtAlgorithm:         cfg.Env.JwtAlgorithm,
// 		SessionServiceURL:    cfg.Env.SessionServiceURL,
// 		SessionServiceSecret: cfg.Env.SessionServiceSecret,
// 	}

// 	middlewaresAdapter := middlewares_adapters.NewMiddlewareAdapter(&middlewares_adapters.NewMiddlewareAdapterParams{
// 		Config: middlewareConfig,
// 		Utils:  deps.utilsAdapter,
// 		JwtCC:  deps.jwtccAdapter,
// 		I18n:   i18nAdapter,
// 	})

// 	// 4. Crear servidor Nexus
// 	server := &nexus.Server{
// 		ServerName: cfg.Env.ApiName,
// 		Secret:     cfg.Env.ApiSecret,
// 		Port:       cfg.Env.ApiPort,
// 		Debug:      cfg.Env.ApiDebug,
// 		CorsOptions: cors.Options{
// 			AllowOriginFunc:  middlewaresAdapter.IsOriginAllowed,
// 			AllowedMethods:   allowedMethods,
// 			AllowedHeaders:   allowedHeaders,
// 			Debug:            cfg.Env.ApiDebug,
// 			AllowCredentials: true,
// 		},
// 		Settings: &nexus.Settings{
// 			PathPrefix: "/api/v1",
// 		},
// 	}

// 	// 5. Registrar middleware (orden importa)
// 	server.Use(middlewaresAdapter.RequestLogMiddleware)
// 	server.Use(middlewaresAdapter.ValidateApiSecret)
// 	server.Use(middlewaresAdapter.VerifySession)
// 	server.Use(roleMiddleware.LoadUserRole)
// 	server.Use(middlewaresAdapter.SetLanguage)

// 	return server, deps
// }

func initDatabaseConnections(deps *serverDeps, env *api_config.ApiEnv) {
	// MongoDB
	mongoOpts := mongocc.ClientOptions{Debug: env.FuturappsMongoDebug}
	mongoDB, err := mongocc.Connect(env.FuturappsMongoUrl, env.FuturappsMongoDbName, &mongoOpts)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	deps.mongoDB = mongoDB

	// PostgreSQL
	//if err = database.RunMigrations(env.FuturappsPostgresUrl); err != nil {
	//	log.Fatalf("Failed to run migrations: %v", err)
	//}
	//
	//db, err := sql.Open("postgres", env.FuturappsPostgresUrl)
	//if err != nil {
	//	log.Fatalf("Failed to open PostgreSQL connection: %v", err)
	//}
	//
	//if err = db.Ping(); err != nil {
	//	log.Fatalf("Failed to ping PostgreSQL: %v", err)
	//}
	//
	//deps.postgresQueries = database.New(db)
}
