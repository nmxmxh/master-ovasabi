package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
	repo "github.com/nmxmxh/master-ovasabi/internal/repository"
)

type Repository struct {
	db         *sql.DB
	masterRepo repo.MasterRepository
}

func NewRepository(db *sql.DB, masterRepo repo.MasterRepository) *Repository {
	return &Repository{db: db, masterRepo: masterRepo}
}

// User management.
func (r *Repository) CreateUser(ctx context.Context, user *adminpb.User) (*adminpb.User, error) {
	var metadataJSON []byte
	if user.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(user.Metadata)
		if err != nil {
			return nil, err
		}
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO service_admin_user (id, master_id, master_uuid, email, name, roles, is_active, created_at, updated_at, user_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, to_timestamp($8), to_timestamp($9), $10, $11, $12)
	`, user.Id, user.MasterId, user.MasterUuid, user.Email, user.Name, pq.Array(user.Roles), user.IsActive, user.CreatedAt, user.UpdatedAt, user.UserId, metadataJSON)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) UpdateUser(ctx context.Context, user *adminpb.User) (*adminpb.User, error) {
	var metadataJSON []byte
	if user.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(user.Metadata)
		if err != nil {
			return nil, err
		}
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE service_admin_user SET email=$1, name=$2, roles=$3, is_active=$4, updated_at=to_timestamp($5), user_id=$6, metadata=$7 WHERE id=$8
	`, user.Email, user.Name, pq.Array(user.Roles), user.IsActive, user.UpdatedAt, user.UserId, metadataJSON, user.Id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) DeleteUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM service_admin_user WHERE id=$1`, userID)
	return err
}

func (r *Repository) ListUsers(ctx context.Context, page, pageSize int) ([]*adminpb.User, int, error) {
	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, master_id, master_uuid, email, name, roles, is_active, EXTRACT(EPOCH FROM created_at), EXTRACT(EPOCH FROM updated_at), user_id, metadata FROM service_admin_user ORDER BY email LIMIT $1 OFFSET $2
	`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var users []*adminpb.User
	for rows.Next() {
		var user adminpb.User
		var roles []string
		var createdAt, updatedAt float64
		var metadataJSON []byte
		if err := rows.Scan(&user.Id, &user.MasterId, &user.MasterUuid, &user.Email, &user.Name, pq.Array(&roles), &user.IsActive, &createdAt, &updatedAt, &user.UserId, &metadataJSON); err != nil {
			return nil, 0, err
		}
		user.Roles = roles
		user.CreatedAt = int64(createdAt)
		user.UpdatedAt = int64(updatedAt)
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &user.Metadata); err != nil {
				return nil, 0, err
			}
		}
		users = append(users, &user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	row := r.db.QueryRowContext(ctx, `SELECT count(*) FROM service_admin_user`)
	err = row.Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to scan total: %w", err)
	}
	return users, total, nil
}

func (r *Repository) GetUser(ctx context.Context, id string) (*adminpb.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, master_id, master_uuid, email, name, roles, is_active, EXTRACT(EPOCH FROM created_at), EXTRACT(EPOCH FROM updated_at), user_id, metadata
		FROM service_admin_user WHERE id = $1
	`, id)
	var user adminpb.User
	var roles []string
	var createdAt, updatedAt float64
	var metadataJSON []byte
	if err := row.Scan(&user.Id, &user.MasterId, &user.MasterUuid, &user.Email, &user.Name, pq.Array(&roles), &user.IsActive, &createdAt, &updatedAt, &user.UserId, &metadataJSON); err != nil {
		return nil, err
	}
	user.Roles = roles
	user.CreatedAt = int64(createdAt)
	user.UpdatedAt = int64(updatedAt)
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &user.Metadata); err != nil {
			return nil, err
		}
	}
	return &user, nil
}

// Role management.
func (r *Repository) CreateRole(ctx context.Context, role *adminpb.Role) (*adminpb.Role, error) {
	var metadataJSON []byte
	if role.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(role.Metadata)
		if err != nil {
			return nil, err
		}
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO service_admin_role (id, master_id, name, permissions, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, now())
	`, role.Id, role.MasterId, role.Name, pq.Array(role.Permissions), metadataJSON)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (r *Repository) UpdateRole(ctx context.Context, role *adminpb.Role) (*adminpb.Role, error) {
	var metadataJSON []byte
	if role.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(role.Metadata)
		if err != nil {
			return nil, err
		}
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE service_admin_role SET name=$1, permissions=$2, metadata=$3 WHERE id=$4
	`, role.Name, pq.Array(role.Permissions), metadataJSON, role.Id)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (r *Repository) DeleteRole(ctx context.Context, roleID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM service_admin_role WHERE id=$1`, roleID)
	return err
}

func (r *Repository) ListRoles(ctx context.Context, page, pageSize int) ([]*adminpb.Role, int, error) {
	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, master_id, name, permissions, metadata FROM service_admin_role ORDER BY name LIMIT $1 OFFSET $2
	`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var roles []*adminpb.Role
	for rows.Next() {
		var role adminpb.Role
		var permissions []string
		var metadataJSON []byte
		if err := rows.Scan(&role.Id, &role.MasterId, &role.Name, pq.Array(&permissions), &metadataJSON); err != nil {
			return nil, 0, err
		}
		role.Permissions = permissions
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &role.Metadata); err != nil {
				return nil, 0, err
			}
		}
		roles = append(roles, &role)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM service_admin_role`).Scan(&total); err != nil {
		return nil, 0, err
	}
	return roles, total, nil
}

// Role assignment.
func (r *Repository) AssignRole(ctx context.Context, userID, roleID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO service_admin_user_role (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, userID, roleID)
	return err
}

func (r *Repository) RevokeRole(ctx context.Context, userID, roleID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM service_admin_user_role WHERE user_id=$1 AND role_id=$2`, userID, roleID)
	return err
}

// Settings management.
func (r *Repository) GetSettings(ctx context.Context) (*adminpb.Settings, error) {
	row := r.db.QueryRowContext(ctx, `SELECT values, metadata FROM service_admin_settings ORDER BY updated_at DESC LIMIT 1`)
	var valuesJSON []byte
	var metaJSON []byte
	settings := &adminpb.Settings{}
	if err := row.Scan(&valuesJSON, &metaJSON); err != nil {
		return nil, err
	}
	if len(valuesJSON) > 0 {
		if err := json.Unmarshal(valuesJSON, &settings.Values); err != nil {
			return nil, err
		}
	}
	if len(metaJSON) > 0 {
		if err := json.Unmarshal(metaJSON, &settings.Metadata); err != nil {
			return nil, err
		}
	}
	return settings, nil
}

func (r *Repository) UpdateSettings(ctx context.Context, s *adminpb.Settings) (*adminpb.Settings, error) {
	valuesJSON, err := json.Marshal(s.Values)
	if err != nil {
		return nil, err
	}
	metaJSON, err := json.Marshal(s.Metadata)
	if err != nil {
		return nil, err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE service_admin_settings SET values=$1, metadata=$2, updated_at=now()`, valuesJSON, metaJSON)
	if err != nil {
		return nil, err
	}
	return r.GetSettings(ctx)
}

// Audit log management.
func (r *Repository) GetAuditLogs(ctx context.Context, page, pageSize int, userID, action string) ([]*adminpb.AuditLog, int, error) {
	query := `SELECT id, master_id, user_id, action, resource, details, timestamp, metadata FROM service_admin_audit_log WHERE 1=1`
	args := []interface{}{}
	if userID != "" {
		query += " AND user_id = $1"
		args = append(args, userID)
	}
	if action != "" {
		query += " AND action = $2"
		args = append(args, action)
	}
	query += " ORDER BY timestamp DESC LIMIT $3 OFFSET $4"
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var logs []*adminpb.AuditLog
	for rows.Next() {
		var l adminpb.AuditLog
		var metaJSON []byte
		if err := rows.Scan(&l.Id, &l.MasterId, &l.UserId, &l.Action, &l.Resource, &l.Details, &l.Timestamp, &metaJSON); err != nil {
			return nil, 0, err
		}
		if len(metaJSON) > 0 {
			if err := json.Unmarshal(metaJSON, &l.Metadata); err != nil {
				return nil, 0, err
			}
		}
		logs = append(logs, &l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	// Count total
	total := len(logs)
	return logs, total, nil
}
