package config

import (
	"fmt"
	"os"
	"strconv"
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
	JWTSecret         string
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
		JWTSecret:     os.Getenv("JWT_SECRET"),
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
