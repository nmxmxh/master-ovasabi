package bootstrap

import (
	"context"
	"database/sql"

	"github.com/nmxmxh/master-ovasabi/config/registry"
	"github.com/nmxmxh/master-ovasabi/scripts"
	"go.uber.org/zap"
)

// BootstrapRegistries loads DB-backed service/event registry into memory on startup.
func BootstrapRegistries(ctx context.Context, db *sql.DB, log *zap.Logger) error {
	dbSvc := &registry.DBServiceRegistry{DB: db}
	dbEvt := &registry.DBEventRegistry{DB: db}

	// Seed service registry from config/service_registration.json if needed
	seeded := false
	if seedFn := getSeedServiceRegistryFunc(); seedFn != nil {
		var err error
		seeded, err = seedFn()
		if err != nil {
			log.Warn("Failed to seed service registry from JSON", zap.Error(err))
		} else if seeded {
			log.Info("Service registry seeded from config/service_registration.json")
		} else {
			log.Info("Service registry already seeded.")
		}
	}

	// Load and register services/events from DB as before
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

// getSeedServiceRegistryFunc returns the SeedServiceRegistry function from scripts if available.
func getSeedServiceRegistryFunc() func() (bool, error) {
	return scripts.SeedServiceRegistry
}
