package bootstrap

import (
	"context"
	"database/sql"

	"github.com/nmxmxh/master-ovasabi/config/registry"
	"go.uber.org/zap"
)

// BootstrapRegistries loads DB-backed service/event registry into memory on startup.
func BootstrapRegistries(ctx context.Context, db *sql.DB, log *zap.Logger) error {
	dbSvc := &registry.DBServiceRegistry{DB: db}
	dbEvt := &registry.DBEventRegistry{DB: db}

	services, err := dbSvc.LoadAll(ctx)
	if err != nil {
		log.Warn("Failed to load service registry from DB", zap.Error(err))
	} else {
		for _, svc := range services {
			registry.RegisterService(svc)
		}
		log.Info("Loaded service registry from DB", zap.Int("count", len(services)))
	}
	events, err := dbEvt.LoadAll(ctx)
	if err != nil {
		log.Warn("Failed to load event registry from DB", zap.Error(err))
	} else {
		for _, evt := range events {
			registry.RegisterEvent(evt)
		}
		log.Info("Loaded event registry from DB", zap.Int("count", len(events)))
	}
	return nil
}
