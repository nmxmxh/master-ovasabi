package connect

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/nmxmxh/master-ovasabi/internal/config"
	"go.uber.org/zap"
)

// ConnectPostgres establishes a connection to Postgres with retries and config tuning.
func ConnectPostgres(ctx context.Context, log *zap.Logger, cfg *config.Config) (*sql.DB, error) {
	maxRetries := 5
	var db *sql.DB
	var err error
	for i := 1; i <= maxRetries; i++ {
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.DBHost,
			cfg.DBPort,
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBName,
			cfg.DBSSLMode,
		)
		log.Info("Attempting database connection", zap.Int("attempt", i))
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			log.Error("Failed to open database", zap.Error(err))
			time.Sleep(3 * time.Second)
			continue
		}
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = db.PingContext(ctx)
		cancel()
		if err == nil {
			db.SetMaxOpenConns(cfg.DBMaxOpenConns)
			db.SetMaxIdleConns(cfg.DBMaxIdleConns)
			db.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxLifetimeMinutes) * time.Minute)
			log.Info("Database connection established")
			return db, nil
		}
		log.Error("Database ping failed", zap.Error(err))
		_ = db.Close()
		time.Sleep(3 * time.Second)
	}
	return nil, fmt.Errorf("failed to connect to database after %d retries: %w", maxRetries, err)
}
