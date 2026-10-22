package server

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codecraftkit/nexus"
	"github.com/rs/cors"
)

const (
	// shutdownTimeout es lo que se espera a los requests en curso antes de cortarlos.
	shutdownTimeout = 20 * time.Second
	// disconnectTimeout es lo que se espera a que Mongo cierre sus conexiones.
	disconnectTimeout = 5 * time.Second
	// indexesTimeout es lo que se espera a que un repositorio cree sus índices al arrancar.
	indexesTimeout = 30 * time.Second
)

// scheduler es lo que el apagado necesita del cron: deja de programar jobs y cierra el context
// devuelto cuando termina el que está en curso.
type scheduler interface {
	Stop() context.Context
}

// App es el servicio armado: el servidor nexus y lo que hay que cerrar al apagarlo.
type App struct {
	server *nexus.Server
	deps   *serverDeps
	cron   scheduler // nil si no hay jobs registrados
}

// InitServer arma el servidor, inicializa los dominios, registra los handlers y arranca el cron.
func InitServer() *App {
	server, deps := initServerConfig()

	// ====================== HEXAGONAL ARCHITECTURE =====================

	// Init domains. Un dominio va antes que los que usan su ApplicationPorts.
	// bondHistory → bondUserSetting → bondType → bondObject → bondBound
	initBondHistoryDomain(deps)
	initBondUserSettingDomain(deps)
	initBondTypeDomain(deps)
	initBondObjectDomain(deps)
	// bondType y bondObject se usan mutuamente: se conectan cuando los dos existen
	linkBondObjectToBondType(deps)

	// Init and start cron scheduler

	// Register handlers
	registerBondTypeHandlers(server, deps)
	registerBondObjectHandlers(server, deps)
	registerBondUserSettingHandlers(server, deps)

	return &App{server: server, deps: deps}
}

// indexer es un repositorio que declara sus índices.
type indexer interface {
	EnsureIndexes(ctx context.Context) error
}

// ensureIndexes crea los índices de un repositorio al arrancar; un fallo solo se loguea.
func ensureIndexes(name string, repository indexer) {
	ctx, cancel := context.WithTimeout(context.Background(), indexesTimeout)
	defer cancel()
	if err := repository.EnsureIndexes(ctx); err != nil {
		log.Printf("[%s] warning: no se pudieron crear los índices: %v", name, err)
	}
}

// Run atiende requests hasta recibir SIGINT o SIGTERM y entonces apaga el servicio ordenadamente.
// Si el puerto está ocupado o el apagado no termina a tiempo, sale con código 1.
func (app *App) Run() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// Tras la primera señal vuelve el comportamiento por defecto: una segunda corta el proceso.
	context.AfterFunc(ctx, stop)

	ln, err := net.Listen("tcp", ":"+app.server.Port)
	if err != nil {
		log.Fatalf("[server] no se pudo escuchar en el puerto %s: %v", app.server.Port, err)
	}
	if err := app.serve(ctx, ln); err != nil {
		log.Fatalf("[server] %v", err)
	}
}

// serve atiende en ln hasta que ctx se cancela o el servidor HTTP se detiene solo. Después cierra
// el servidor HTTP, el cron y Mongo, en ese orden.
func (app *App) serve(ctx context.Context, ln net.Listener) error {
	httpServer := app.httpServer()
	served := make(chan error, 1)
	go func() { served <- httpServer.Serve(ln) }()
	log.Printf("[server] %s escuchando en %s", app.server.ServerName, ln.Addr())

	var err error
	select {
	case err = <-served:
		err = fmt.Errorf("el servidor HTTP se detuvo: %w", err)
	case <-ctx.Done():
		log.Printf("[server] señal recibida, apagando")
		err = shutdownHTTP(httpServer)
	}

	app.stopCron()
	app.disconnectMongo()
	if err == nil {
		log.Printf("[server] apagado completo")
	}
	return err
}

// httpServer arma el mismo http.Server que nexus.Server.Run, que no lo expone y por eso no deja
// apagarlo.
func (app *App) httpServer() *http.Server {
	mux := app.server.PrepareEndpoints()
	handler := cors.New(app.server.CorsOptions).Handler(app.server.ApplyMiddlewares(mux))
	settings := app.server.Settings
	return &http.Server{
		Handler:      handler,
		ReadTimeout:  cmp.Or(settings.ReadTimeout, nexus.DefaultReadTimeout),
		WriteTimeout: cmp.Or(settings.WriteTimeout, nexus.DefaultWriteTimeout),
	}
}

// shutdownHTTP deja de aceptar conexiones y espera los requests en curso. Si no terminan a tiempo,
// cierra las conexiones a la fuerza.
func shutdownHTTP(httpServer *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		_ = httpServer.Close()
		return fmt.Errorf("apagado forzado del servidor HTTP: %w", err)
	}
	return nil
}

// stopCron detiene el scheduler y espera el job en curso.
func (app *App) stopCron() {
	if app.cron == nil {
		return
	}
	<-app.cron.Stop().Done()
}

// disconnectMongo cierra las conexiones a Mongo. Un error solo se loguea: el proceso ya termina.
func (app *App) disconnectMongo() {
	ctx, cancel := context.WithTimeout(context.Background(), disconnectTimeout)
	defer cancel()
	if err := app.deps.mongoDB.Database().Client().Disconnect(ctx); err != nil {
		log.Printf("[server] error al desconectar MongoDB: %v", err)
	}
}
