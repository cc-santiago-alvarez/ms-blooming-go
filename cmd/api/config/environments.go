package config

type ApiEnv struct {
	ApiName   string
	ApiPort   string
	ApiSecret string
	ApiDebug  bool

	FuturappsMongoUrl    string
	FuturappsMongoDbName string
	FuturappsMongoDebug  bool

	FuturappsPostgresUrl string
}
