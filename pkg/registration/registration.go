package registration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// ServiceRegisterFunc defines the canonical signature for all service Register functions.
type ServiceRegisterFunc func(
	ctx context.Context,
	container *di.Container,
	eventEmitter interface{},
	db *sql.DB,
	masterRepo interface{},
	redisProvider *redis.Provider,
	log *zap.Logger,
	serviceEnabled bool,
	provider interface{},
) error

// ServiceRegistrationEntry defines a struct for service registration metadata and logic.
type ServiceRegistrationEntry struct {
	Name     string              `json:"name"`
	Enabled  bool                `json:"enabled"`
	Register ServiceRegisterFunc `json:"-"` // Not in JSON, set in Go
}

// JSONServiceRegistration mirrors the JSON structure (minimal for registration).
type JSONServiceRegistration struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Enabled *bool  `json:"enabled,omitempty"`
	// ... other fields omitted for brevity ...
}

// LoadServiceRegistrations loads service registrations from service_registration.json.
func LoadServiceRegistrations(path string) ([]JSONServiceRegistration, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bytes, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var regs []JSONServiceRegistration
	if err := json.Unmarshal(bytes, &regs); err != nil {
		return nil, err
	}
	return regs, nil
}

// RegisterAllFromJSON registers all services listed in the JSON, using the provided Go mapping.
func RegisterAllFromJSON(
	ctx context.Context,
	container *di.Container,
	eventEmitter interface{},
	db *sql.DB,
	masterRepo interface{},
	redisProvider *redis.Provider,
	log *zap.Logger,
	eventEnabled bool,
	provider interface{},
	jsonPath string,
	registerFuncs map[string]ServiceRegisterFunc,
) error {
	jsonRegs, err := LoadServiceRegistrations(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to load service_registration.json: %w", err)
	}
	for _, reg := range jsonRegs {
		enabled := true
		if reg.Enabled != nil {
			enabled = *reg.Enabled
		}
		if !enabled {
			continue
		}
		regFunc, ok := registerFuncs[reg.Name]
		if !ok {
			log.Warn("No Go registration function for service", zap.String("service", reg.Name))
			continue
		}
		if err := regFunc(ctx, container, eventEmitter, db, masterRepo, redisProvider, log, eventEnabled, provider); err != nil {
			log.Error("Failed to register service", zap.String("service", reg.Name), zap.Error(err))
			return err
		}
		log.Info("Registered service", zap.String("service", reg.Name))
	}
	return nil
}

// Ouroboros/Jormungand Note:
// Nexus as both a server and a service is an intentional ouroboros: it orchestrates itself and others.
// This is a powerful pattern for meta-orchestration, but requires careful bootstrapping and health checks.
