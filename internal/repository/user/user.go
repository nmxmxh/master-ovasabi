package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"go.uber.org/zap"
)

var log *zap.Logger

func SetLogger(l *zap.Logger) {
	log = l
}

// User represents a user in the service_user table
type User struct {
	ID        int64           `db:"id"`
	MasterID  int64           `db:"master_id"`
	Email     string          `db:"email"`
	Password  string          `db:"password_hash"`
	Metadata  json.RawMessage `db:"metadata"`
	CreatedAt time.Time       `db:"created_at"`
	UpdatedAt time.Time       `db:"updated_at"`
}

// UserRepository handles operations on the service_user table
type UserRepository struct {
	*repository.BaseRepository
	masterRepo repository.MasterRepository
}

// NewUserRepository creates a new user repository instance
func NewUserRepository(db *sql.DB, masterRepo repository.MasterRepository) *UserRepository {
	return &UserRepository{
		BaseRepository: repository.NewBaseRepository(db),
		masterRepo:     masterRepo,
	}
}

// Create inserts a new user record
func (r *UserRepository) Create(ctx context.Context, user *User) (*User, error) {
	// First create the master record
	masterID, err := r.masterRepo.Create(ctx, repository.EntityTypeUser)
	if err != nil {
		return nil, err
	}

	user.MasterID = masterID
	err = r.GetDB().QueryRowContext(ctx,
		`INSERT INTO service_user (
			master_id, email, password_hash, metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, NOW(), NOW()
		) RETURNING id, created_at, updated_at`,
		user.MasterID, user.Email, user.Password, user.Metadata,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		// If user creation fails, clean up the master record
		_ = r.masterRepo.Delete(ctx, masterID)
		return nil, err
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT 
			id, master_id, email, password_hash, metadata,
			created_at, updated_at
		FROM service_user 
		WHERE email = $1`,
		email,
	).Scan(
		&user.ID, &user.MasterID, &user.Email,
		&user.Password, &user.Metadata,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*User, error) {
	user := &User{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT 
			id, master_id, email, password_hash, metadata,
			created_at, updated_at
		FROM service_user 
		WHERE id = $1`,
		id,
	).Scan(
		&user.ID, &user.MasterID, &user.Email,
		&user.Password, &user.Metadata,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// Update updates a user record
func (r *UserRepository) Update(ctx context.Context, user *User) error {
	result, err := r.GetDB().ExecContext(ctx,
		`UPDATE service_user 
		SET email = $1, password_hash = $2, metadata = $3, updated_at = NOW()
		WHERE id = $4`,
		user.Email, user.Password, user.Metadata, user.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Delete removes a user and its master record
func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// The master record deletion will cascade to the user due to foreign key
	return r.masterRepo.Delete(ctx, user.MasterID)
}

// List retrieves a paginated list of users
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*User, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT 
			id, master_id, email, password_hash, metadata,
			created_at, updated_at
		FROM service_user 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			if log != nil {
				log.Error("error closing rows", zap.Error(cerr))
			}
		}
	}()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID, &user.MasterID, &user.Email,
			&user.Password, &user.Metadata,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}
