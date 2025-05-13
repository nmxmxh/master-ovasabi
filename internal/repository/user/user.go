package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/lib/pq"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	userv1 "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	metadatautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
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
	ID           string             `db:"id"`
	MasterID     string             `db:"master_id"`
	Username     string             `db:"username"`
	Email        string             `db:"email"`
	PasswordHash string             `db:"password_hash"`
	ReferralCode string             `db:"referral_code"`
	ReferredBy   string             `db:"referred_by"`
	DeviceHash   string             `db:"device_hash"`
	Location     string             `db:"location"`
	Profile      UserProfile        `db:"profile"`
	Roles        []string           `db:"roles"`
	Status       int32              `db:"status"`
	Metadata     *commonpb.Metadata `db:"metadata"`
	Tags         []string           `db:"tags"`
	ExternalIDs  map[string]string  `db:"external_ids"`
	CreatedAt    time.Time          `db:"created_at"`
	UpdatedAt    time.Time          `db:"updated_at"`
	// reserved for extensibility
}

type UserProfile struct {
	FirstName    string            `json:"first_name" db:"first_name"`
	LastName     string            `json:"last_name" db:"last_name"`
	PhoneNumber  string            `json:"phone_number" db:"phone_number"`
	AvatarURL    string            `json:"avatar_url" db:"avatar_url"`
	Bio          string            `json:"bio" db:"bio"`
	Timezone     string            `json:"timezone" db:"timezone"`
	Language     string            `json:"language" db:"language"`
	CustomFields map[string]string `json:"custom_fields" db:"custom_fields"`
	// reserved for extensibility
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

	// Validate metadata
	if err := metadatautil.ValidateMetadata(user.Metadata); err != nil {
		return nil, err
	}

	// First create the master record with username as name
	masterID, err := r.masterRepo.Create(ctx, repository.EntityTypeUser, user.Username)
	if err != nil {
		return nil, err
	}

	user.MasterID = strconv.FormatInt(masterID, 10)
	err = r.GetDB().QueryRowContext(ctx,
		`INSERT INTO service_user (
			master_id, username, email, password_hash, referral_code, referred_by, device_hash, location, profile, roles, status, metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW()
		) RETURNING id, created_at, updated_at`,
		masterID, user.Username, user.Email, user.PasswordHash, user.ReferralCode, user.ReferredBy, user.DeviceHash, user.Location, user.Profile, user.Roles, user.Status, user.Metadata,
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
			id, master_id, username, email, password_hash, referral_code, referred_by, device_hash, location, profile, roles, status, metadata,
			created_at, updated_at
		FROM service_user 
		WHERE username = $1`,
		strings.ToLower(username),
	).Scan(
		&user.ID, &user.MasterID, &user.Username,
		&user.Email, &user.PasswordHash, &user.ReferralCode, &user.ReferredBy, &user.DeviceHash, &user.Location, &user.Profile, &user.Roles, &user.Status, &user.Metadata,
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
			id, master_id, username, email, password_hash, referral_code, referred_by, device_hash, location, profile, roles, status, metadata,
			created_at, updated_at
		FROM service_user 
		WHERE email = $1`,
		email,
	).Scan(
		&user.ID, &user.MasterID, &user.Username,
		&user.Email, &user.PasswordHash, &user.ReferralCode, &user.ReferredBy, &user.DeviceHash, &user.Location, &user.Profile, &user.Roles, &user.Status, &user.Metadata,
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
func (r *UserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	user := &User{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT 
			id, master_id, username, email, password_hash, referral_code, referred_by, device_hash, location, profile, roles, status, metadata,
			created_at, updated_at
		FROM service_user 
		WHERE id = $1`,
		id,
	).Scan(
		&user.ID, &user.MasterID, &user.Username,
		&user.Email, &user.PasswordHash, &user.ReferralCode, &user.ReferredBy, &user.DeviceHash, &user.Location, &user.Profile, &user.Roles, &user.Status, &user.Metadata,
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
			masterIDInt, err := strconv.ParseInt(user.MasterID, 10, 64)
			if err != nil {
				return err
			}
			master := &repository.Master{
				ID:   masterIDInt,
				Name: user.Username,
			}
			if err := r.masterRepo.Update(ctx, master); err != nil {
				return err
			}
		}
	}

	// Validate metadata
	if err := metadatautil.ValidateMetadata(user.Metadata); err != nil {
		return err
	}

	result, err := r.GetDB().ExecContext(ctx,
		`UPDATE service_user 
		SET username = $1, email = $2, password_hash = $3, referral_code = $4, referred_by = $5, device_hash = $6, location = $7, profile = $8, roles = $9, status = $10, metadata = $11, updated_at = NOW()
		WHERE id = $12`,
		user.Username, user.Email, user.PasswordHash, user.ReferralCode, user.ReferredBy, user.DeviceHash, user.Location, user.Profile, user.Roles, user.Status, user.Metadata, user.ID,
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
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	masterIDInt, err := strconv.ParseInt(user.MasterID, 10, 64)
	if err != nil {
		return err
	}
	return r.masterRepo.Delete(ctx, masterIDInt)
}

// List retrieves a paginated list of users.
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*User, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT 
			id, master_id, username, email, password_hash, referral_code, referred_by, device_hash, location, profile, roles, status, metadata,
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
			&user.Email, &user.PasswordHash, &user.ReferralCode, &user.ReferredBy, &user.DeviceHash, &user.Location, &user.Profile, &user.Roles, &user.Status, &user.Metadata,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// ListFlexible retrieves a paginated, filtered list of users with flexible search.
func (r *UserRepository) ListFlexible(ctx context.Context, req *userv1.ListUsersRequest) ([]*User, int, error) {
	args := []interface{}{}
	where := []string{}
	argIdx := 1
	if req.SearchQuery != "" {
		where = append(where, "(username ILIKE $"+fmt.Sprint(argIdx)+" OR email ILIKE $"+fmt.Sprint(argIdx)+")")
		args = append(args, "%"+req.SearchQuery+"%")
		argIdx++
	}
	if len(req.Tags) > 0 {
		where = append(where, "tags && $"+fmt.Sprint(argIdx))
		args = append(args, pq.Array(req.Tags))
		argIdx++
	}
	if req.Metadata != nil && req.Metadata.ServiceSpecific != nil {
		for k, v := range req.Metadata.ServiceSpecific.Fields {
			where = append(where, fmt.Sprintf("metadata->'service_specific'->>'%s' = $%d", k, argIdx))
			args = append(args, v.GetStringValue())
			argIdx++
		}
	}
	// TODO: Handle filters using req.Metadata fields if needed
	if req.Page < 0 {
		req.Page = 0
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	offset := int(req.Page) * int(req.PageSize)
	args = append(args, req.PageSize, offset)
	baseQuery := "SELECT id, master_id, username, email, password_hash, referral_code, referred_by, device_hash, location, profile, roles, status, metadata, tags, external_ids, created_at, updated_at FROM service_user"
	if len(where) > 0 {
		baseQuery += " WHERE " + strings.Join(where, " AND ")
	}
	orderBy := "created_at DESC"
	if req.SortBy != "" {
		orderBy = req.SortBy
		if req.SortDesc {
			orderBy += " DESC"
		} else {
			orderBy += " ASC"
		}
	}
	baseQuery += " ORDER BY " + orderBy + " LIMIT $" + fmt.Sprint(argIdx) + " OFFSET $" + fmt.Sprint(argIdx+1)
	rows, err := r.GetDB().QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	users := []*User{}
	for rows.Next() {
		user := &User{}
		var metaRaw, tagsRaw, extIDsRaw []byte
		if err := rows.Scan(
			&user.ID, &user.MasterID, &user.Username, &user.Email, &user.PasswordHash, &user.ReferralCode, &user.ReferredBy, &user.DeviceHash, &user.Location, &user.Profile, &user.Roles, &user.Status, &metaRaw, &tagsRaw, &extIDsRaw, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		if len(metaRaw) > 0 {
			if err := protojson.Unmarshal(metaRaw, user.Metadata); err != nil {
				return nil, 0, err
			}
		}
		if len(tagsRaw) > 0 {
			if err := json.Unmarshal(tagsRaw, &user.Tags); err != nil {
				return nil, 0, err
			}
		}
		if len(extIDsRaw) > 0 {
			if err := json.Unmarshal(extIDsRaw, &user.ExternalIDs); err != nil {
				return nil, 0, err
			}
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	countQuery := "SELECT COUNT(*) FROM service_user"
	if len(where) > 0 {
		countQuery += " WHERE " + strings.Join(where, " AND ")
	}
	countArgs := args[:len(args)-2]
	if err := r.GetDB().QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		total = len(users)
	}
	return users, total, nil
}
