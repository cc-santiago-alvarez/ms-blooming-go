package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type cfg struct {
	Env *ApiEnv
}

func InitConfig() *cfg {
	c := &cfg{}
	c.LoadEnvs()
	return c
}

func (c *cfg) LoadEnvs() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	envFile := ".env." + env
	if _, err := os.Stat(envFile); err == nil {
		if err := godotenv.Overload(envFile); err != nil {
			log.Fatalf("Error loading %s file: %v", envFile, err)
		}
	} else if os.IsNotExist(err) {
		log.Fatalf("Environment file %s not found", envFile)
	} else {
		log.Fatalf("Error checking environment file %s: %v", envFile, err)
	}

	c.Env = &ApiEnv{
		ApiName:   os.Getenv("API_NAME"),
		ApiPort:   os.Getenv("API_PORT"),
		ApiSecret: os.Getenv("API_SECRET"),
		ApiDebug:  os.Getenv("API_DEBUG") == "true",

		FuturappsMongoUrl:    os.Getenv("FA_MONGO_URL"),
		FuturappsMongoDbName: os.Getenv("FA_MONGO_DBNAME"),
		FuturappsMongoDebug:  os.Getenv("FA_MONGO_DEBUG") == "true",

		FuturappsPostgresUrl: os.Getenv("FA_POSTGRES_URL"),
	}
}
