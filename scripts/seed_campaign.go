// scripts/seed_campaign.go
// Usage: go run scripts/seed_campaign.go
// Seeds or updates the campaign table from config/campaign.json.
package scripts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type Campaign struct {
	Slug           string      `json:"slug"`
	Title          string      `json:"title"`
	Description    string      `json:"description"`
	RankingFormula string      `json:"ranking_formula"`
	OwnerID        string      `json:"owner_id"`
	StartDate      string      `json:"start_date"`
	EndDate        string      `json:"end_date"`
	Metadata       interface{} `json:"metadata"`
}

// SeedCampaign seeds or updates the campaign table from config/campaign.json.
// Returns true if seeding was performed, false if error.
func SeedCampaign() (bool, error) {
	fmt.Println("[seed_campaign] Starting campaign seeding process (default/first campaign)...")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("[seed_campaign][ERROR] DATABASE_URL env var required")
		return false, fmt.Errorf("database_url env var required")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("[seed_campaign][ERROR] Failed to connect to db: %v\n", err)
		return false, fmt.Errorf("failed to connect to db: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("[seed_campaign] Reading config/campaign.json...")
	data, err := os.ReadFile("config/campaign.json")
	if err != nil {
		fmt.Printf("[seed_campaign][ERROR] Failed to read config/campaign.json: %v\n", err)
		return false, fmt.Errorf("failed to read config/campaign.json: %w", err)
	}
	var campaigns []Campaign
	if err := json.Unmarshal(data, &campaigns); err != nil {
		fmt.Printf("[seed_campaign][ERROR] Failed to unmarshal campaign.json: %v\n", err)
		return false, fmt.Errorf("failed to unmarshal campaign.json: %w", err)
	}

	if len(campaigns) == 0 {
		fmt.Println("[seed_campaign][WARN] No campaigns found in config/campaign.json.")
		return false, nil
	}

	successCount := 0
	// --- Ensure admin user exists, create if missing ---
	var masterID int64
	var masterUUID string
	adminUsername := "admin"
	adminEmail := "admin@ovasabi.com"
	// Try to get admin user by username and master_id
	row := db.QueryRowContext(ctx, "SELECT master_id, master_uuid FROM service_user_master WHERE username = $1", adminUsername)
	err = row.Scan(&masterID, &masterUUID)
	if err == sql.ErrNoRows {
		// Check if master record exists for admin user
		masterRow := db.QueryRowContext(ctx, "SELECT id, uuid FROM master WHERE name = $1 AND type = $2", adminUsername, "user")
		errMaster := masterRow.Scan(&masterID, &masterUUID)
		if errMaster == sql.ErrNoRows {
			// Create master record for admin user
			err = db.QueryRowContext(ctx, `INSERT INTO master (uuid, name, type, description, version, created_at, updated_at) VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW()) RETURNING id, uuid`, adminUsername, "user", "Default admin user", "1.0.0").Scan(&masterID, &masterUUID)
			if err != nil {
				fmt.Printf("[seed_campaign][ERROR] Failed to create master record for admin user: %v\n", err)
				return false, err
			}
		} else if errMaster != nil {
			fmt.Printf("[seed_campaign][ERROR] Failed to query master record for admin user: %v\n", errMaster)
			return false, errMaster
		}
		// Hash default password
		defaultPassword := "admin123"
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			fmt.Printf("[seed_campaign][ERROR] Failed to hash default admin password: %v\n", err)
			return false, err
		}
		// Create admin user record with password_hash and status (use integer for status)
		_, err = db.ExecContext(ctx, `INSERT INTO service_user_master (master_id, master_uuid, username, email, password_hash, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`, masterID, masterUUID, adminUsername, adminEmail, passwordHash, 1)

		if err != nil {
			fmt.Printf("[seed_campaign][ERROR] Failed to create admin user: %v\n", err)
			return false, err
		}
		fmt.Printf("[seed_campaign] Created default admin user: %s (%s)\n", adminUsername, masterUUID)
	} else if err != nil {
		fmt.Printf("[seed_campaign][ERROR] Failed to query admin user: %v\n", err)
		return false, err
	} else {
		fmt.Printf("[seed_campaign] Found existing admin user: %s (%s)\n", adminUsername, masterUUID)
	}

	for i, c := range campaigns {
		fmt.Printf("[seed_campaign] Seeding campaign %d/%d: slug=%s, title=%s, start_date=%s, end_date=%s\n", i+1, len(campaigns), c.Slug, c.Title, c.StartDate, c.EndDate)
		metadataJSON, err := json.Marshal(c.Metadata)
		if err != nil {
			fmt.Printf("[seed_campaign][ERROR] Failed to marshal metadata for campaign %s: %v\n", c.Slug, err)
			continue
		}
		// Look up or create master record for campaign owner
		var ownerMasterID int64
		var ownerMasterUUID string
		var ownerUsername string
		// Try to parse OwnerID as UUID
		isUUID := false
		if len(c.OwnerID) == 36 && c.OwnerID[8] == '-' && c.OwnerID[13] == '-' && c.OwnerID[18] == '-' && c.OwnerID[23] == '-' {
			isUUID = true
		}
		if isUUID {
			ownerRow := db.QueryRowContext(ctx, "SELECT id, uuid, name FROM master WHERE uuid = $1 AND type = $2", c.OwnerID, "user")
			errOwner := ownerRow.Scan(&ownerMasterID, &ownerMasterUUID, &ownerUsername)
			if errOwner == sql.ErrNoRows {
				// Create master record for campaign owner
				errOwner = db.QueryRowContext(ctx, `INSERT INTO master (uuid, name, type, description, version, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id, uuid, name`, c.OwnerID, c.OwnerID, "user", "Campaign owner", "1.0.0").Scan(&ownerMasterID, &ownerMasterUUID, &ownerUsername)
				if errOwner != nil {
					fmt.Printf("[seed_campaign][ERROR] Failed to create master record for campaign owner %s: %v\n", c.OwnerID, errOwner)
					continue
				}
			} else if errOwner != nil {
				fmt.Printf("[seed_campaign][ERROR] Failed to query master record for campaign owner %s: %v\n", c.OwnerID, errOwner)
				continue
			}
		} else {
			// OwnerID is not a UUID, treat as name
			ownerRow := db.QueryRowContext(ctx, "SELECT id, uuid, name FROM master WHERE name = $1 AND type = $2", c.OwnerID, "user")
			errOwner := ownerRow.Scan(&ownerMasterID, &ownerMasterUUID, &ownerUsername)
			if errOwner == sql.ErrNoRows {
				// Create master record for campaign owner
				errOwner = db.QueryRowContext(ctx, `INSERT INTO master (uuid, name, type, description, version, created_at, updated_at) VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW()) RETURNING id, uuid, name`, c.OwnerID, "user", "Campaign owner", "1.0.0").Scan(&ownerMasterID, &ownerMasterUUID, &ownerUsername)
				if errOwner != nil {
					fmt.Printf("[seed_campaign][ERROR] Failed to create master record for campaign owner %s: %v\n", c.OwnerID, errOwner)
					continue
				}
			} else if errOwner != nil {
				fmt.Printf("[seed_campaign][ERROR] Failed to query master record for campaign owner %s: %v\n", c.OwnerID, errOwner)
				continue
			}
		}

		// Ensure user record exists for campaign owner in service_user_master
		var ownerUserID string
		userRow := db.QueryRowContext(ctx, "SELECT id FROM service_user_master WHERE master_uuid = $1", ownerMasterUUID)
		errUser := userRow.Scan(&ownerUserID)
		if errUser == sql.ErrNoRows {
			// Create user record for campaign owner
			defaultEmail := fmt.Sprintf("%s@ovasabi.com", ownerUsername)
			defaultPassword := "owner123"
			passwordHash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
			if err != nil {
				fmt.Printf("[seed_campaign][ERROR] Failed to hash default owner password for %s: %v\n", ownerUsername, err)
				continue
			}
			_, err = db.ExecContext(ctx, `INSERT INTO service_user_master (master_id, master_uuid, username, email, password_hash, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`, ownerMasterID, ownerMasterUUID, ownerUsername, defaultEmail, passwordHash, 1)
			if err != nil {
				fmt.Printf("[seed_campaign][ERROR] Failed to create user record for campaign owner %s: %v\n", ownerUsername, err)
				continue
			}
		} else if errUser != nil {
			fmt.Printf("[seed_campaign][ERROR] Failed to query user record for campaign owner %s: %v\n", ownerUsername, errUser)
			continue
		}

		// Look up or create master record for campaign itself (type 'campaign', name=slug)
		var campaignMasterID int64
		var campaignMasterUUID string
		campaignRow := db.QueryRowContext(ctx, "SELECT id, uuid FROM master WHERE name = $1 AND type = $2", c.Slug, "campaign")
		errCampaign := campaignRow.Scan(&campaignMasterID, &campaignMasterUUID)
		if errCampaign == sql.ErrNoRows {
			// Create master record for campaign
			errCampaign = db.QueryRowContext(ctx, `INSERT INTO master (uuid, name, type, description, version, created_at, updated_at) VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW()) RETURNING id, uuid`, c.Slug, "campaign", "Campaign entity", "1.0.0").Scan(&campaignMasterID, &campaignMasterUUID)
			if errCampaign != nil {
				fmt.Printf("[seed_campaign][ERROR] Failed to create master record for campaign %s: %v\n", c.Slug, errCampaign)
				continue
			}
		} else if errCampaign != nil {
			fmt.Printf("[seed_campaign][ERROR] Failed to query master record for campaign %s: %v\n", c.Slug, errCampaign)
			continue
		}

		// Insert or update campaign with correct owner_id and campaign master_id/master_uuid
		_, err = db.ExecContext(ctx, `
		INSERT INTO service_campaign_main (
			slug, name, title, description, ranking_formula, owner_id, master_id, master_uuid, start_date, end_date, metadata, updated_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
		ON CONFLICT (slug) DO UPDATE SET
			name = $2,
			title = $3,
			description = $4,
			ranking_formula = $5,
			owner_id = $6,
			master_id = $7,
			master_uuid = $8,
			start_date = $9,
			end_date = $10,
			metadata = $11,
			updated_at = $12
	`,
			c.Slug,
			c.Title,
			c.Title,
			c.Description,
			c.RankingFormula,
			ownerUserID, // use the actual user id from service_user_master for owner_id
			campaignMasterID,
			campaignMasterUUID,
			c.StartDate,
			c.EndDate,
			metadataJSON,
			time.Now().UTC(),
			time.Now().UTC(),
		)
		if err != nil {
			fmt.Printf("[seed_campaign][ERROR] Failed to insert/update campaign %s: %v\n", c.Slug, err)
			continue
		}
		fmt.Printf("[seed_campaign] Successfully seeded/updated campaign: %s\n", c.Slug)
		successCount++
	}
	fmt.Printf("[seed_campaign] Campaign seeding process complete. %d/%d campaigns seeded/updated.\n", successCount, len(campaigns))
	return successCount > 0, nil
}

// Optional: main function for standalone script usage
//
//nolint:unused // main is for standalone script usage
func main() {
	// Automate seeding if SEED_CAMPAIGN_ON_STARTUP env var is set to "true"
	if os.Getenv("SEED_CAMPAIGN_ON_STARTUP") == "true" {
		ok, err := SeedCampaign()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if ok {
			fmt.Println("Campaign table seeded/updated from config/campaign.json")
		} else {
			fmt.Println("No campaigns seeded/updated.")
		}
	}
}
