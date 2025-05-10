package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/lib/pq"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"go.uber.org/zap"
)

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrInvalidUsername       = errors.New("invalid username")
	ErrUsernameReserved      = errors.New("username is reserved")
	ErrUsernameTaken         = errors.New("username is already taken")
	ErrUsernameBadWord       = errors.New("username contains inappropriate content")
	ErrUsernameInvalidFormat = errors.New("username contains invalid characters or format")
)

var log *zap.Logger

func SetLogger(l *zap.Logger) {
	log = l
}

// User represents a user in the service_user table.
type User struct {
	ID        int64           `db:"id"`
	MasterID  int64           `db:"master_id"`
	Username  string          `db:"username"`
	Email     string          `db:"email"`
	Password  string          `db:"password_hash"`
	Metadata  json.RawMessage `db:"metadata"`
	CreatedAt time.Time       `db:"created_at"`
	UpdatedAt time.Time       `db:"updated_at"`
}

// UserRepository handles operations on the service_user table.
type UserRepository struct {
	*repository.BaseRepository
	masterRepo repository.MasterRepository
}

// NewUserRepository creates a new user repository instance.
func NewUserRepository(db *sql.DB, masterRepo repository.MasterRepository) *UserRepository {
	return &UserRepository{
		BaseRepository: repository.NewBaseRepository(db),
		masterRepo:     masterRepo,
	}
}

// validateUsername checks if a username is valid and available.
func (r *UserRepository) validateUsername(ctx context.Context, username string) error {
	// Basic validation
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 64 {
		return ErrInvalidUsername
	}

	// Check for invalid characters
	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && !unicode.IsSymbol(r) && !unicode.IsMark(r) && r != '_' && r != '-' && r != '.' {
			return ErrUsernameInvalidFormat
		}
	}

	// Check for reserved usernames
	var exists bool
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM reserved_usernames WHERE username = $1)`,
		username,
	).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrUsernameReserved
	}

	// Check if username is taken (case-insensitive)
	err = r.GetDB().QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM service_user WHERE lower(username) = lower($1))`,
		username,
	).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrUsernameTaken
	}

	return nil
}

// Create inserts a new user record.
func (r *UserRepository) Create(ctx context.Context, user *User) (*User, error) {
	// Validate username
	if err := r.validateUsername(ctx, user.Username); err != nil {
		return nil, err
	}

	// First create the master record with username as name
	masterID, err := r.masterRepo.Create(ctx, repository.EntityTypeUser, user.Username)
	if err != nil {
		return nil, err
	}

	user.MasterID = masterID
	err = r.GetDB().QueryRowContext(ctx,
		`INSERT INTO service_user (
			master_id, username, email, password_hash, metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, NOW(), NOW()
		) RETURNING id, created_at, updated_at`,
		user.MasterID, user.Username, user.Email, user.Password, user.Metadata,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		// If user creation fails, clean up the master record
		if err := r.masterRepo.Delete(ctx, masterID); err != nil {
			if log != nil {
				log.Error("service not implemented", zap.Error(err))
			}
		}

		// Check for specific PostgreSQL errors
		pqErr := &pq.Error{}
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			case "23505": // unique_violation
				if strings.Contains(pqErr.Message, "username") {
					return nil, ErrUsernameTaken
				}
			case "23514": // check_violation
				if strings.Contains(pqErr.Message, "inappropriate content") {
					return nil, ErrUsernameBadWord
				}
				if strings.Contains(pqErr.Message, "between 3 and 64") {
					return nil, ErrInvalidUsername
				}
			}
		}
		return nil, err
	}

	return user, nil
}

// GetByUsername retrieves a user by username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	user := &User{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT 
			id, master_id, username, email, password_hash, metadata,
			created_at, updated_at
		FROM service_user 
		WHERE username = $1`,
		strings.ToLower(username),
	).Scan(
		&user.ID, &user.MasterID, &user.Username,
		&user.Email, &user.Password, &user.Metadata,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetByEmail retrieves a user by email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT 
			id, master_id, username, email, password_hash, metadata,
			created_at, updated_at
		FROM service_user 
		WHERE email = $1`,
		email,
	).Scan(
		&user.ID, &user.MasterID, &user.Username,
		&user.Email, &user.Password, &user.Metadata,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*User, error) {
	user := &User{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT 
			id, master_id, username, email, password_hash, metadata,
			created_at, updated_at
		FROM service_user 
		WHERE id = $1`,
		id,
	).Scan(
		&user.ID, &user.MasterID, &user.Username,
		&user.Email, &user.Password, &user.Metadata,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// Update updates a user record.
func (r *UserRepository) Update(ctx context.Context, user *User) error {
	// If username is being changed, validate it
	if user.Username != "" {
		currentUser, err := r.GetByID(ctx, user.ID)
		if err != nil {
			return err
		}
		if currentUser.Username != user.Username {
			if err := r.validateUsername(ctx, user.Username); err != nil {
				return err
			}
			// Update master record name
			master := &repository.Master{
				ID:   user.MasterID,
				Name: user.Username,
			}
			if err := r.masterRepo.Update(ctx, master); err != nil {
				return err
			}
		}
	}

	result, err := r.GetDB().ExecContext(ctx,
		`UPDATE service_user 
		SET username = $1, email = $2, password_hash = $3, metadata = $4, updated_at = NOW()
		WHERE id = $5`,
		user.Username, user.Email, user.Password, user.Metadata, user.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Delete removes a user and its master record.
func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// The master record deletion will cascade to the user due to foreign key
	return r.masterRepo.Delete(ctx, user.MasterID)
}

// List retrieves a paginated list of users.
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*User, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT 
			id, master_id, username, email, password_hash, metadata,
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
			&user.ID, &user.MasterID, &user.Username,
			&user.Email, &user.Password, &user.Metadata,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}
