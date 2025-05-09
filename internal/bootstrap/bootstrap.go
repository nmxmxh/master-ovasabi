package bootstrap

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/config"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/logger"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

type Dependencies struct {
	Logger   *zap.Logger
	DB       *sql.DB
	Provider *service.Provider
}

func Initialize(cfg *config.Config) (*Dependencies, error) {
	loggerInstance, err := logger.New(logger.Config{
		Environment: cfg.AppEnv,
		LogLevel:    "info",
		ServiceName: cfg.AppName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	log := loggerInstance.GetZapLogger()

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	redisConfig := redis.Config{
		Host:     cfg.RedisHost,
		Port:     cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       0, // TODO: parse from cfg
	}

	provider, err := service.NewProvider(log, db, redisConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize service provider: %w", err)
	}

	return &Dependencies{
		Logger:   log,
		DB:       db,
		Provider: provider,
	}, nil
}
