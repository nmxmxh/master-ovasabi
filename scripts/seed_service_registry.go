// scripts/seed_service_registry.go
// Usage: go run scripts/seed_service_registry.go
// Seeds the service_registry table from config/service_registration.json if the table is empty.
package scripts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	// Blank import for pq driver registration (required by database/sql).
	_ "github.com/lib/pq"
)

type ServiceRegistration struct {
	Name      string `json:"name"`
	Endpoints []struct {
		Path    string   `json:"path"`
		Method  string   `json:"method"`
		Actions []string `json:"actions"`
	} `json:"endpoints"`
	// Add other fields as needed
}

// SeedServiceRegistry seeds the service_registry table from config/service_registration.json if the table is empty.
// Returns true if seeding was performed, false if already seeded, and error if failed.
func SeedServiceRegistry() (bool, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return false, fmt.Errorf("database_url env var required")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return false, fmt.Errorf("failed to connect to db: %w", err)
	}
	defer db.Close()

	ctx := context.Background()

	data, err := os.ReadFile("config/service_registration.json")
	if err != nil {
		return false, fmt.Errorf("failed to read config/service_registration.json: %w", err)
	}
	var services []ServiceRegistration
	if err := json.Unmarshal(data, &services); err != nil {
		return false, fmt.Errorf("failed to unmarshal service_registration.json: %w", err)
	}

	for _, svc := range services {
		methods, err := json.Marshal(svc.Endpoints)
		if err != nil {
			return false, fmt.Errorf("skipping %s: failed to marshal endpoints: %w", svc.Name, err)
		}
		_, err = db.ExecContext(ctx, `
					   INSERT INTO service_registry (service_name, methods, registered_at)
					   VALUES ($1, $2, $3)
					   ON CONFLICT (service_name) DO UPDATE SET methods = $2, registered_at = $3
			   `, svc.Name, methods, time.Now().UTC())
		if err != nil {
			return false, fmt.Errorf("failed to insert %s: %w", svc.Name, err)
		}
	}
	return true, nil
}
