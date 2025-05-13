package adminrepo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"
	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// User management.
func (r *PostgresRepository) CreateUser(ctx context.Context, user *adminpb.User) (*adminpb.User, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO service_admin_user (id, master_id, email, name, roles, is_active, created_at, updated_at, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, to_timestamp($7), to_timestamp($8), $9)
	`, user.Id, user.MasterId, user.Email, user.Name, pq.Array(user.Roles), user.IsActive, user.CreatedAt, user.UpdatedAt, user.UserId)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *PostgresRepository) UpdateUser(_ context.Context, _ *adminpb.User) (*adminpb.User, error) {
	// TODO: implement UpdateUser logic
	return nil, errors.New("not implemented")
}

func (r *PostgresRepository) DeleteUser(_ context.Context, _ string) error {
	// TODO: implement DeleteUser logic
	return errors.New("not implemented")
}

func (r *PostgresRepository) ListUsers(_ context.Context, _, _ int) ([]*adminpb.User, int, error) {
	// TODO: implement ListUsers logic
	return nil, 0, errors.New("not implemented")
}

func (r *PostgresRepository) GetUser(ctx context.Context, id string) (*adminpb.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, master_id, email, name, roles, is_active, EXTRACT(EPOCH FROM created_at), EXTRACT(EPOCH FROM updated_at), user_id
		FROM service_admin_user WHERE id = $1
	`, id)
	var user adminpb.User
	var roles []string
	var createdAt, updatedAt float64
	if err := row.Scan(&user.Id, &user.MasterId, &user.Email, &user.Name, pq.Array(&roles), &user.IsActive, &createdAt, &updatedAt, &user.UserId); err != nil {
		return nil, err
	}
	user.Roles = roles
	user.CreatedAt = int64(createdAt)
	user.UpdatedAt = int64(updatedAt)
	return &user, nil
}

// Role management.
func (r *PostgresRepository) CreateRole(_ context.Context, _ *adminpb.Role) (*adminpb.Role, error) {
	// TODO: implement CreateRole logic
	return nil, errors.New("not implemented")
}

func (r *PostgresRepository) UpdateRole(_ context.Context, _ *adminpb.Role) (*adminpb.Role, error) {
	// TODO: implement UpdateRole logic
	return nil, errors.New("not implemented")
}

func (r *PostgresRepository) DeleteRole(_ context.Context, _ string) error {
	// TODO: implement DeleteRole logic
	return errors.New("not implemented")
}

func (r *PostgresRepository) ListRoles(_ context.Context, _, _ int) ([]*adminpb.Role, int, error) {
	// TODO: implement ListRoles logic
	return nil, 0, errors.New("not implemented")
}

// Role assignment.
func (r *PostgresRepository) AssignRole(_ context.Context, _, _ string) error {
	// TODO: implement AssignRole logic
	return errors.New("not implemented")
}

func (r *PostgresRepository) RevokeRole(_ context.Context, _, _ string) error {
	// TODO: implement RevokeRole logic
	return errors.New("not implemented")
}

// Audit logs.
func (r *PostgresRepository) GetAuditLogs(_ context.Context, _, _ int, _, _ string) ([]*adminpb.AuditLog, int, error) {
	// TODO: implement GetAuditLogs logic
	return nil, 0, errors.New("not implemented")
}

// Settings.
func (r *PostgresRepository) GetSettings(_ context.Context) (*adminpb.Settings, error) {
	// TODO: implement GetSettings logic
	return nil, errors.New("not implemented")
}

func (r *PostgresRepository) UpdateSettings(_ context.Context, _ *adminpb.Settings) (*adminpb.Settings, error) {
	// TODO: implement UpdateSettings logic
	return nil, errors.New("not implemented")
}
