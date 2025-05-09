package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppEnv            string
	AppName           string
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	RedisHost         string
	RedisPort         string
	RedisPassword     string
	RedisDB           int
	RedisPoolSize     int
	RedisMinIdleConns int
	RedisMaxRetries   int
	AppPort           string
	MetricsPort       string
}

func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:        os.Getenv("APP_ENV"),
		AppName:       os.Getenv("APP_NAME"),
		DBHost:        os.Getenv("DB_HOST"),
		DBPort:        os.Getenv("DB_PORT"),
		DBUser:        os.Getenv("DB_USER"),
		DBPassword:    os.Getenv("DB_PASSWORD"),
		DBName:        os.Getenv("DB_NAME"),
		RedisHost:     os.Getenv("REDIS_HOST"),
		RedisPort:     os.Getenv("REDIS_PORT"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		AppPort:       os.Getenv("APP_PORT"),
		MetricsPort:   os.Getenv("METRICS_PORT"),
	}
	// TODO: Parse ints and add validation
	if cfg.AppEnv == "" || cfg.AppName == "" || cfg.DBHost == "" || cfg.DBPort == "" || cfg.DBUser == "" || cfg.DBPassword == "" || cfg.DBName == "" || cfg.RedisHost == "" || cfg.RedisPassword == "" {
		return nil, fmt.Errorf("missing required environment variables")
	}
	return cfg, nil
}
