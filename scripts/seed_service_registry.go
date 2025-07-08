// scripts/seed_service_registry.go
// Usage: go run scripts/seed_service_registry.go
// Seeds the service_registry table from config/service_registration.json if the table is empty.
package scripts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

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
		return false, fmt.Errorf("DATABASE_URL env var required")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return false, fmt.Errorf("Failed to connect to DB: %w", err)
	}
	defer db.Close()

	ctx := context.Background()
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM service_registry").Scan(&count)
	if err != nil {
		return false, fmt.Errorf("Failed to count service_registry: %w", err)
	}
	if count > 0 {
		return false, nil
	}

	data, err := ioutil.ReadFile("config/service_registration.json")
	if err != nil {
		return false, fmt.Errorf("Failed to read config/service_registration.json: %w", err)
	}
	var services []ServiceRegistration
	if err := json.Unmarshal(data, &services); err != nil {
		return false, fmt.Errorf("Failed to unmarshal service_registration.json: %w", err)
	}

	for _, svc := range services {
		methods, err := json.Marshal(svc.Endpoints)
		if err != nil {
			return false, fmt.Errorf("Skipping %s: failed to marshal endpoints: %w", svc.Name, err)
		}
		_, err = db.ExecContext(ctx, `
			INSERT INTO service_registry (service_name, methods, registered_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (service_name) DO UPDATE SET methods = $2, registered_at = $3
		`, svc.Name, methods, time.Now().UTC())
		if err != nil {
			return false, fmt.Errorf("Failed to insert %s: %w", svc.Name, err)
		}
	}
	return true, nil
}
