package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppEnv                 string
	AppName                string
	DBHost                 string
	DBPort                 string
	DBUser                 string
	DBPassword             string
	DBName                 string
	DBSSLMode              string
	RedisHost              string
	RedisPort              string
	RedisPassword          string
	RedisDB                int
	RedisPoolSize          int
	RedisMinIdleConns      int
	RedisMaxRetries        int
	AppPort                string
	MetricsPort            string
	JWTSecret              string
	LogLevel               string
	NexusGRPCAddr          string
	SchedulerGRPCAddr      string
	LibreTranslateEndpoint string
	LibreTranslateTimeout  string // duration as string, e.g. "10s"
}

func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:                 os.Getenv("APP_ENV"),
		AppName:                os.Getenv("APP_NAME"),
		DBHost:                 os.Getenv("DB_HOST"),
		DBPort:                 os.Getenv("DB_PORT"),
		DBUser:                 os.Getenv("DB_USER"),
		DBPassword:             os.Getenv("DB_PASSWORD"),
		DBName:                 os.Getenv("DB_NAME"),
		DBSSLMode:              os.Getenv("DB_SSL_MODE"),
		RedisHost:              os.Getenv("REDIS_HOST"),
		RedisPort:              os.Getenv("REDIS_PORT"),
		RedisPassword:          os.Getenv("REDIS_PASSWORD"),
		AppPort:                os.Getenv("APP_PORT"),
		MetricsPort:            os.Getenv("METRICS_PORT"),
		JWTSecret:              os.Getenv("JWT_SECRET"),
		LogLevel:               os.Getenv("LOG_LEVEL"),
		NexusGRPCAddr:          os.Getenv("NEXUS_GRPC_ADDR"),
		SchedulerGRPCAddr:      os.Getenv("SCHEDULER_GRPC_ADDR"),
		LibreTranslateEndpoint: os.Getenv("LIBRETRANSLATE_ENDPOINT"),
		LibreTranslateTimeout:  os.Getenv("LIBRETRANSLATE_TIMEOUT"),
	}
	if cfg.DBSSLMode == "" {
		cfg.DBSSLMode = "disable"
	}
	if cfg.NexusGRPCAddr == "" {
		cfg.NexusGRPCAddr = "nexus:50052"
	}
	if cfg.SchedulerGRPCAddr == "" {
		cfg.SchedulerGRPCAddr = "localhost:50053"
	}
	if cfg.LibreTranslateEndpoint == "" {
		cfg.LibreTranslateEndpoint = "http://localhost:5002"
	}
	if cfg.LibreTranslateTimeout == "" {
		cfg.LibreTranslateTimeout = "10s"
	}
	var err error
	if v := os.Getenv("REDIS_DB"); v != "" {
		cfg.RedisDB, err = strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
		}
	}
	if v := os.Getenv("REDIS_POOL_SIZE"); v != "" {
		cfg.RedisPoolSize, err = strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_POOL_SIZE: %w", err)
		}
	}
	if v := os.Getenv("REDIS_MIN_IDLE_CONNS"); v != "" {
		cfg.RedisMinIdleConns, err = strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_MIN_IDLE_CONNS: %w", err)
		}
	}
	if v := os.Getenv("REDIS_MAX_RETRIES"); v != "" {
		cfg.RedisMaxRetries, err = strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_MAX_RETRIES: %w", err)
		}
	}
	if cfg.AppEnv == "" || cfg.AppName == "" || cfg.DBHost == "" || cfg.DBPort == "" || cfg.DBUser == "" || cfg.DBPassword == "" || cfg.DBName == "" || cfg.RedisHost == "" || cfg.RedisPassword == "" {
		return nil, fmt.Errorf("missing required environment variables")
	}
	return cfg, nil
}
