package api_config

// ApiEnv contiene la configuración del servicio leída del entorno.
type ApiEnv struct {
	// Entorno (development, testing o production): se declara en .env y selecciona .env.<ENV>
	Env string

	// Servidor
	ApiName   string
	ApiPort   string
	ApiSecret string
	ApiDebug  bool

	// CORS: orígenes permitidos; admite patrones *.dominio.com
	CorsOrigins []string

	// MongoDB (replica set, requerido por las transacciones)
	FuturappsMongoUrl    string
	FuturappsMongoDbName string
	FuturappsMongoDebug  bool

	// ms-oauth: base de la API; la sesión se valida en /session
	OAuthURL    string
	OAuthSecret string

	// ms-notification
	NotificationServiceURL string
	NotificationSecret     string

	// ms-exporter
	ExporterServiceURL string
	ExporterSecret     string

	// Blooming
	BloomingAdminEmails     []string
	DefaultUtcDiff          int
	BulkPurchaseMaxQuantity int
}
