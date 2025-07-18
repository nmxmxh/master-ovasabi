package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var log *zap.Logger

func SetLogger(l *zap.Logger) {
	log = l
}

func validatePassword(pw string) error {
	if len(pw) < 10 {
		return ErrPasswordTooShort
	}
	hasUpper, hasLower, hasDigit, hasSpecial := false, false, false, false
	for _, c := range pw {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}
	if !hasUpper {
		return ErrPasswordNoUpper
	}
	if !hasLower {
		return ErrPasswordNoLower
	}
	if !hasDigit {
		return ErrPasswordNoDigit
	}
	if !hasSpecial {
		return ErrPasswordNoSpecial
	}
	return nil
}

// User represents a user in the service_user table.
type User struct {
	ID           string             `db:"id"`
	MasterID     int64              `db:"master_id"`
	MasterUUID   string             `db:"master_uuid"`
	Username     string             `db:"username"`
	Email        string             `db:"email"`
	PasswordHash string             `db:"password_hash"`
	ReferralCode string             `db:"referral_code"`
	ReferredBy   string             `db:"referred_by"`
	DeviceHash   string             `db:"device_hash"`
	Locations    []string           `db:"location"`
	Profile      Profile            `db:"profile"`
	Roles        []string           `db:"roles"`
	Status       int32              `db:"status"`
	Metadata     *commonpb.Metadata `db:"metadata"`
	Tags         []string           `db:"tags"`
	ExternalIDs  map[string]string  `db:"external_ids"`
	CreatedAt    time.Time          `db:"created_at"`
	UpdatedAt    time.Time          `db:"updated_at"`
	// Score tracks the user's system currency, with subfields for balance and pending.
	Score Score `db:"score" json:"score"`
	// reserved for extensibility
}

// Score holds balance and pending fields for system currency.
type Score struct {
	Balance float64 `db:"score_balance" json:"balance"`
	Pending float64 `db:"score_pending" json:"pending"`
}

type Profile struct {
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

// Repository handles user data persistence.
type Repository struct {
	*repository.BaseRepository
	masterRepo repository.MasterRepository
}

// NewRepository creates a new user repository.
func NewRepository(db *sql.DB, log *zap.Logger, masterRepo repository.MasterRepository) *Repository {
	return &Repository{
		BaseRepository: repository.NewBaseRepository(db, log),
		masterRepo:     masterRepo,
	}
}

// validateUsername checks if a username is valid and available.
func (r *Repository) validateUsername(ctx context.Context, username string) error {
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
		`SELECT EXISTS(SELECT 1 FROM service_user_reserved_username WHERE username = $1)`,
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
func (r *Repository) Create(ctx context.Context, user *User) (*User, error) {
	// Validate username
	if err := r.validateUsername(ctx, user.Username); err != nil {
		return nil, err
	}

	// Validate password
	if err := validatePassword(user.PasswordHash); err != nil {
		return nil, err
	}

	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback() // Rollback on panic
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				r.GetLogger().Error("failed to rollback transaction on panic", zap.Error(rbErr))
			}
			panic(p) // Re-throw panic
		}
	}()

	// 1. Create the master record within the transaction
	user.MasterID, user.MasterUUID, err = r.masterRepo.Create(ctx, tx, repository.EntityTypeUser, user.Username)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			r.GetLogger().Error("failed to rollback transaction", zap.Error(rbErr))
		}
		return nil, fmt.Errorf("failed to create master entity for user: %w", err)
	}

	// 2. Insert into service_user using the same transaction
	err = tx.QueryRowContext(ctx, // Use tx here
		`INSERT INTO service_user (
			master_id, master_uuid, username, email, password_hash, referral_code, referred_by, device_hash, location, profile, roles, status, metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW()
		) RETURNING id, created_at, updated_at`,
		user.MasterID, user.MasterUUID, user.Username, user.Email, user.PasswordHash, user.ReferralCode, user.ReferredBy, user.DeviceHash, pq.Array(user.Locations), user.Profile, pq.Array(user.Roles), user.Status, user.Metadata, // Arguments
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt) // Scan results
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			r.GetLogger().Error("failed to rollback transaction", zap.Error(rbErr))
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
func (r *Repository) GetByUsername(ctx context.Context, username string) (*User, error) {
	user := &User{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT 
			id, master_id, master_uuid, username, email, password_hash, referral_code, referred_by, device_hash, location, profile, roles, status, metadata,
			created_at, updated_at
		FROM service_user 
		WHERE username = $1`,
		strings.ToLower(username),
	).Scan(
		&user.ID, &user.MasterID, &user.MasterUUID, &user.Username,
		&user.Email, &user.PasswordHash, &user.ReferralCode, &user.ReferredBy, &user.DeviceHash, pq.Array(&user.Locations), &user.Profile, pq.Array(&user.Roles), &user.Status, &user.Metadata,
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
func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT 
			id, master_id, master_uuid, username, email, password_hash, referral_code, referred_by, device_hash, location, profile, roles, status, metadata,
			created_at, updated_at
		FROM service_user 
		WHERE email = $1`,
		email,
	).Scan(
		&user.ID, &user.MasterID, &user.MasterUUID, &user.Username,
		&user.Email, &user.PasswordHash, &user.ReferralCode, &user.ReferredBy, &user.DeviceHash, pq.Array(&user.Locations), &user.Profile, pq.Array(&user.Roles), &user.Status, &user.Metadata,
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
func (r *Repository) GetByID(ctx context.Context, id string) (*User, error) {
	user := &User{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT 
			id, master_id, master_uuid, username, email, password_hash, referral_code, referred_by, device_hash, location, profile, roles, status, metadata,
			created_at, updated_at
		FROM service_user 
		WHERE id = $1`,
		id,
	).Scan(
		&user.ID, &user.MasterID, &user.MasterUUID, &user.Username,
		&user.Email, &user.PasswordHash, &user.ReferralCode, &user.ReferredBy, &user.DeviceHash, pq.Array(&user.Locations), &user.Profile, pq.Array(&user.Roles), &user.Status, &user.Metadata,
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
func (r *Repository) Update(ctx context.Context, user *User) error {
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

	// Validate password
	if err := validatePassword(user.PasswordHash); err != nil {
		return err
	}

	result, err := r.GetDB().ExecContext(ctx,
		`UPDATE service_user 
		SET username = $1, email = $2, password_hash = $3, referral_code = $4, referred_by = $5, device_hash = $6, location = $7, profile = $8, roles = $9, status = $10, metadata = $11, updated_at = NOW()
		WHERE id = $12`,
		user.Username, user.Email, user.PasswordHash, user.ReferralCode, user.ReferredBy, user.DeviceHash, pq.Array(user.Locations), user.Profile, pq.Array(user.Roles), user.Status, user.Metadata, user.ID,
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
func (r *Repository) Delete(ctx context.Context, id string) error {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return r.masterRepo.Delete(ctx, user.MasterID)
}

// List retrieves a paginated list of users.
func (r *Repository) List(ctx context.Context, limit, offset int) ([]*User, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT 
			id, master_id, master_uuid, username, email, password_hash, referral_code, referred_by, device_hash, location, profile, roles, status, metadata,
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
			&user.ID, &user.MasterID, &user.MasterUUID, &user.Username,
			&user.Email, &user.PasswordHash, &user.ReferralCode, &user.ReferredBy, &user.DeviceHash, pq.Array(&user.Locations), &user.Profile, pq.Array(&user.Roles), &user.Status, &user.Metadata,
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
func (r *Repository) ListFlexible(ctx context.Context, req *userv1.ListUsersRequest) ([]*User, int, error) {
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
	baseQuery := "SELECT id, master_id, master_uuid, username, email, password_hash, referral_code, referred_by, device_hash, location, profile, roles, status, metadata, tags, external_ids, created_at, updated_at FROM service_user"
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
			&user.ID, &user.MasterID, &user.MasterUUID, &user.Username, &user.Email, &user.PasswordHash, &user.ReferralCode, &user.ReferredBy, &user.DeviceHash, pq.Array(&user.Locations), pq.Array(&user.Roles), &user.Status, &metaRaw, &tagsRaw, &extIDsRaw, &user.CreatedAt, &user.UpdatedAt,
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

// ListUserEvents fetches user events by user ID with pagination.
func (r *Repository) ListUserEvents(ctx context.Context, userID string, page, pageSize int) ([]*userv1.UserEvent, int, error) {
	if page < 0 {
		page = 0
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := page * pageSize
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT id, user_id, master_id, event_type, description, occurred_at, metadata, payload
		 FROM service_user_event WHERE user_id = $1
		 ORDER BY occurred_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	events := []*userv1.UserEvent{}
	for rows.Next() {
		var (
			event      userv1.UserEvent
			occurredAt sql.NullTime
			metaRaw    []byte
			payloadRaw []byte
		)
		if err := rows.Scan(&event.Id, &event.UserId, &event.MasterId, &event.EventType, &event.Description, &occurredAt, &metaRaw, &payloadRaw); err != nil {
			return nil, 0, err
		}
		if occurredAt.Valid {
			event.OccurredAt = timestamppb.New(occurredAt.Time)
		}
		if len(metaRaw) > 0 {
			if event.Metadata == nil {
				event.Metadata = &commonpb.Metadata{}
			}
			if err := protojson.Unmarshal(metaRaw, event.Metadata); err != nil {
				return nil, 0, err
			}
		}
		if len(payloadRaw) > 0 {
			if event.Payload == nil {
				event.Payload = make(map[string]string)
			}
			if err := json.Unmarshal(payloadRaw, &event.Payload); err != nil {
				return nil, 0, err
			}
		}
		events = append(events, &event)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.GetDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM service_user_event WHERE user_id = $1`, userID).Scan(&total); err != nil {
		total = len(events)
	}
	return events, total, nil
}

// ListAuditLogs fetches audit logs by user ID with pagination.
func (r *Repository) ListAuditLogs(ctx context.Context, userID string, page, pageSize int) ([]*userv1.AuditLog, int, error) {
	if page < 0 {
		page = 0
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := page * pageSize
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT id, user_id, master_id, action, resource, occurred_at, metadata, payload
		 FROM service_user_audit_log WHERE user_id = $1
		 ORDER BY occurred_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	logs := []*userv1.AuditLog{}
	for rows.Next() {
		var (
			log        userv1.AuditLog
			occurredAt sql.NullTime
			metaRaw    []byte
			payloadRaw []byte
		)
		if err := rows.Scan(&log.Id, &log.UserId, &log.MasterId, &log.Action, &log.Resource, &occurredAt, &metaRaw, &payloadRaw); err != nil {
			return nil, 0, err
		}
		if occurredAt.Valid {
			log.OccurredAt = timestamppb.New(occurredAt.Time)
		}
		if len(metaRaw) > 0 {
			if log.Metadata == nil {
				log.Metadata = &commonpb.Metadata{}
			}
			if err := protojson.Unmarshal(metaRaw, log.Metadata); err != nil {
				return nil, 0, err
			}
		}
		if len(payloadRaw) > 0 {
			if log.Payload == nil {
				log.Payload = make(map[string]string)
			}
			if err := json.Unmarshal(payloadRaw, &log.Payload); err != nil {
				return nil, 0, err
			}
		}
		logs = append(logs, &log)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.GetDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM service_user_audit_log WHERE user_id = $1`, userID).Scan(&total); err != nil {
		total = len(logs)
	}
	return logs, total, nil
}

// --- Session Management ---.
func (r *Repository) CreateSession(ctx context.Context, session *userv1.Session) (*userv1.Session, error) {
	tx, err := r.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			if log != nil {
				log.Error("tx.Rollback failed", zap.Error(err))
			}
		}
	}()

	// Validate user exists
	var exists bool
	err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM service_user WHERE id = $1)", session.UserId).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("user check: %w", err)
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	// Insert session
	query := `
		INSERT INTO service_user_session (user_id, device_info, refresh_token, access_token, ip_address, metadata, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), $7)
		RETURNING id, created_at, expires_at
	`
	meta, err := metadatautil.MarshalCanonical(session.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	err = tx.QueryRowContext(ctx, query,
		session.UserId, session.DeviceInfo, session.RefreshToken, session.AccessToken, session.IpAddress, meta, session.ExpiresAt.AsTime(),
	).Scan(&session.Id, &session.CreatedAt, &session.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return session, nil
}

func (r *Repository) GetSession(ctx context.Context, sessionID string) (*userv1.Session, error) {
	query := `SELECT id, user_id, device_info, refresh_token, access_token, ip_address, metadata, created_at, expires_at
		FROM service_user_session WHERE id = $1`
	var metaRaw []byte
	s := &userv1.Session{}
	err := r.GetDB().QueryRowContext(ctx, query, sessionID).Scan(
		&s.Id, &s.UserId, &s.DeviceInfo, &s.RefreshToken, &s.AccessToken, &s.IpAddress, &metaRaw, &s.CreatedAt, &s.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if len(metaRaw) > 0 {
		if err := protojson.Unmarshal(metaRaw, s.Metadata); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (r *Repository) RevokeSession(ctx context.Context, sessionID string) error {
	_, err := r.GetDB().ExecContext(ctx, "DELETE FROM service_user_session WHERE id = $1", sessionID)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (r *Repository) ListSessions(ctx context.Context, userID string) ([]*userv1.Session, error) {
	rows, err := r.GetDB().QueryContext(ctx, `SELECT id, user_id, device_info, refresh_token, access_token, ip_address, metadata, created_at, expires_at
		FROM service_user_session WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()
	var sessions []*userv1.Session
	for rows.Next() {
		s := &userv1.Session{}
		var metaRaw []byte
		if err := rows.Scan(&s.Id, &s.UserId, &s.DeviceInfo, &s.RefreshToken, &s.AccessToken, &s.IpAddress, &metaRaw, &s.CreatedAt, &s.ExpiresAt); err != nil {
			return nil, err
		}
		if len(metaRaw) > 0 {
			if err := protojson.Unmarshal(metaRaw, s.Metadata); err != nil {
				return nil, err
			}
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// --- RBAC & Permissions ---.
func (r *Repository) AssignRole(ctx context.Context, userID, role string) error {
	// Use array_append if not present
	_, err := r.GetDB().ExecContext(ctx, `
		UPDATE service_user SET roles = array_append(roles, $1)
		WHERE id = $2 AND NOT (roles @> ARRAY[$1])
	`, role, userID)
	return err
}

func (r *Repository) RemoveRole(ctx context.Context, userID, role string) error {
	_, err := r.GetDB().ExecContext(ctx, `
		UPDATE service_user SET roles = array_remove(roles, $1)
		WHERE id = $2
	`, role, userID)
	return err
}

func (r *Repository) ListRoles(ctx context.Context, userID string) ([]string, error) {
	var roles []string
	err := r.GetDB().QueryRowContext(ctx, "SELECT roles FROM service_user WHERE id = $1", userID).Scan(pq.Array(&roles))
	return roles, err
}

func (r *Repository) ListPermissions(ctx context.Context, userID string) ([]string, error) {
	// Assume a table service_user_role_permission(role TEXT, permission TEXT)
	roles, err := r.ListRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, nil
	}
	rows, err := r.GetDB().QueryContext(ctx, `
		SELECT DISTINCT permission FROM service_user_role_permission WHERE role = ANY($1)
	`, pq.Array(roles))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var perms []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return perms, nil
}

// --- Social Graph: Friends & Follows ---.
func (r *Repository) AddFriend(ctx context.Context, userID, friendID string, metadata *commonpb.Metadata) (*userv1.Friendship, error) {
	// Insert or update friendship (pending/accepted)
	tx, err := r.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			if log != nil {
				log.Error("tx.Rollback failed", zap.Error(err))
			}
		}
	}()
	// Check for reciprocal request
	var reciprocalID string
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM service_user_friendship WHERE user_id = $1 AND friend_id = $2 AND status = 'pending'
	`, friendID, userID).Scan(&reciprocalID)
	status := "pending"
	if err == nil && reciprocalID != "" {
		// Accept both
		_, err = tx.ExecContext(ctx, `UPDATE service_user_friendship SET status = 'accepted' WHERE id = $1`, reciprocalID)
		if err != nil {
			return nil, fmt.Errorf("update friendship: %w", err)
		}
		status = "accepted"
	}
	meta, err := metadatautil.MarshalCanonical(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	var id string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO service_user_friendship (user_id, friend_id, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (user_id, friend_id) DO UPDATE SET status = EXCLUDED.status, updated_at = NOW()
		RETURNING id
	`, userID, friendID, status, meta).Scan(&id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &userv1.Friendship{Id: id, UserId: userID, FriendId: friendID, Status: userv1.FriendshipStatus(userv1.FriendshipStatus_value[strings.ToUpper(status)])}, nil
}

func (r *Repository) RemoveFriend(ctx context.Context, userID, friendID string) error {
	_, err := r.GetDB().ExecContext(ctx, `
		DELETE FROM service_user_friendship WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)
	`, userID, friendID)
	return err
}

func (r *Repository) ListFriends(ctx context.Context, userID string, page, pageSize int) ([]*userv1.User, int, error) {
	offset := page * pageSize
	rows, err := r.GetDB().QueryContext(ctx, `
		SELECT u.id, u.username, u.email, u.profile, u.metadata
		FROM service_user_friendship f
		JOIN service_user u ON u.id = f.friend_id
		WHERE f.user_id = $1 AND f.status = 'accepted'
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var users []*userv1.User
	for rows.Next() {
		u := &userv1.User{}
		var profileRaw, metaRaw []byte
		if err := rows.Scan(&u.Id, &u.Username, &u.Email, &profileRaw, &metaRaw); err != nil {
			return nil, 0, err
		}
		if err := protojson.Unmarshal(profileRaw, u.Profile); err != nil {
			return nil, 0, err
		}
		if err := protojson.Unmarshal(metaRaw, u.Metadata); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.GetDB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM service_user_friendship WHERE user_id = $1 AND status = 'accepted'
	`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *Repository) FollowUser(ctx context.Context, followerID, followeeID string, metadata *commonpb.Metadata) (*userv1.Follow, error) {
	meta, err := metadatautil.MarshalCanonical(metadata)
	if err != nil {
		return nil, err
	}
	var id string
	err = r.GetDB().QueryRowContext(ctx, `
		INSERT INTO service_user_follow (follower_id, followee_id, status, metadata, created_at, updated_at)
		VALUES ($1, $2, 'active', $3, NOW(), NOW())
		ON CONFLICT (follower_id, followee_id) DO UPDATE SET status = 'active', updated_at = NOW()
		RETURNING id
	`, followerID, followeeID, meta).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &userv1.Follow{Id: id, FollowerId: followerID, FolloweeId: followeeID, Status: userv1.FollowStatus_FOLLOW_STATUS_ACTIVE}, nil
}

func (r *Repository) UnfollowUser(ctx context.Context, followerID, followeeID string) error {
	_, err := r.GetDB().ExecContext(ctx, `
		UPDATE service_user_follow SET status = 'blocked', updated_at = NOW()
		WHERE follower_id = $1 AND followee_id = $2
	`, followerID, followeeID)
	return err
}

// ListFollowers returns a paginated list of users who follow the given user.
func (r *Repository) ListFollowers(ctx context.Context, userID string, page, pageSize int) ([]*userv1.User, int, error) {
	offset := page * pageSize
	rows, err := r.GetDB().QueryContext(ctx, `
		SELECT u.id, u.username, u.email, u.profile, u.metadata
		FROM service_user_follow f
		JOIN service_user u ON u.id = f.follower_id
		WHERE f.followee_id = $1 AND f.status = 'active'
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var users []*userv1.User
	for rows.Next() {
		u := &userv1.User{}
		var profileRaw, metaRaw []byte
		if err := rows.Scan(&u.Id, &u.Username, &u.Email, &profileRaw, &metaRaw); err != nil {
			return nil, 0, err
		}
		if err := protojson.Unmarshal(profileRaw, u.Profile); err != nil {
			return nil, 0, err
		}
		if err := protojson.Unmarshal(metaRaw, u.Metadata); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.GetDB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM service_user_follow WHERE followee_id = $1 AND status = 'active'
	`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// ListFollowing returns a paginated list of users whom the given user is following.
func (r *Repository) ListFollowing(ctx context.Context, userID string, page, pageSize int) ([]*userv1.User, int, error) {
	offset := page * pageSize
	rows, err := r.GetDB().QueryContext(ctx, `
		SELECT u.id, u.username, u.email, u.profile, u.metadata
		FROM service_user_follow f
		JOIN service_user u ON u.id = f.followee_id
		WHERE f.follower_id = $1 AND f.status = 'active'
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var users []*userv1.User
	for rows.Next() {
		u := &userv1.User{}
		var profileRaw, metaRaw []byte
		if err := rows.Scan(&u.Id, &u.Username, &u.Email, &profileRaw, &metaRaw); err != nil {
			return nil, 0, err
		}
		if err := protojson.Unmarshal(profileRaw, u.Profile); err != nil {
			return nil, 0, err
		}
		if err := protojson.Unmarshal(metaRaw, u.Metadata); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.GetDB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM service_user_follow WHERE follower_id = $1 AND status = 'active'
	`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// SuggestConnections suggests users with the most mutual friends (excluding the user).
func (r *Repository) SuggestConnections(ctx context.Context, userID string, metadata *commonpb.Metadata) ([]*userv1.User, error) {
	// Example: Use metadata for context-aware suggestions
	var (
		locale    string
		interests []string
	)
	if metadata != nil && metadata.ServiceSpecific != nil {
		if userField, ok := metadata.ServiceSpecific.Fields["user"]; ok && userField.GetStructValue() != nil {
			userMeta := userField.GetStructValue().Fields
			if l, ok := userMeta["locale"]; ok {
				locale = l.GetStringValue()
			}
			if ints, ok := userMeta["interests"]; ok && ints.GetListValue() != nil {
				for _, v := range ints.GetListValue().Values {
					interests = append(interests, v.GetStringValue())
				}
			}
		}
	}

	// Build base query
	query := `
		SELECT u.id, u.username, u.email, u.profile, u.metadata
		FROM service_user u
		WHERE u.id != $1`
	args := []interface{}{userID}

	// If interests are provided, filter users with overlapping interests
	if len(interests) > 0 {
		query += " AND ("
		for i, interest := range interests {
			if i > 0 {
				query += " OR "
			}
			query += fmt.Sprintf("u.metadata->'service_specific'->'user'->'interests' ? $%d", len(args)+1)
			args = append(args, interest)
		}
		query += ")"
	}
	// Optionally, filter by locale
	if locale != "" {
		query += fmt.Sprintf(" AND u.metadata->'service_specific'->'user'->>'locale' = $%d", len(args)+1)
		args = append(args, locale)
	}
	// Order by mutual friends as before
	query += `
		ORDER BY (
			SELECT COUNT(*) FROM service_user_friendship f
			WHERE f.user_id = $1 AND f.friend_id = u.id AND f.status = 'accepted'
		) DESC
		LIMIT 10`

	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*userv1.User
	for rows.Next() {
		u := &userv1.User{}
		var profileRaw, metaRaw []byte
		if err := rows.Scan(&u.Id, &u.Username, &u.Email, &profileRaw, &metaRaw); err != nil {
			return nil, err
		}
		if err := protojson.Unmarshal(profileRaw, u.Profile); err != nil {
			return nil, err
		}
		if err := protojson.Unmarshal(metaRaw, u.Metadata); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

// ListConnections lists connections of a given type (friend, follow, follower) for a user.
func (r *Repository) ListConnections(ctx context.Context, userID, connType string, metadata *commonpb.Metadata) ([]*userv1.User, error) {
	// Extract filtering criteria from metadata
	var (
		locale    string
		interests []string
	)
	if metadata != nil && metadata.ServiceSpecific != nil {
		if userField, ok := metadata.ServiceSpecific.Fields["user"]; ok && userField.GetStructValue() != nil {
			userMeta := userField.GetStructValue().Fields
			if l, ok := userMeta["locale"]; ok {
				locale = l.GetStringValue()
			}
			if ints, ok := userMeta["interests"]; ok && ints.GetListValue() != nil {
				for _, v := range ints.GetListValue().Values {
					interests = append(interests, v.GetStringValue())
				}
			}
		}
	}

	var users []*userv1.User
	var err error
	switch connType {
	case "friend":
		users, _, err = r.ListFriends(ctx, userID, 0, 100)
	case "follow":
		users, _, err = r.ListFollowing(ctx, userID, 0, 100)
	case "follower":
		users, _, err = r.ListFollowers(ctx, userID, 0, 100)
	default:
		return nil, errors.New("unsupported connection type")
	}
	if err != nil {
		return nil, err
	}

	// Filter users by locale and/or interests if provided in metadata
	filtered := users[:0]
	for _, u := range users {
		match := true
		if locale != "" && u.Metadata != nil && u.Metadata.ServiceSpecific != nil {
			userField, ok := u.Metadata.ServiceSpecific.Fields["user"]
			if ok && userField.GetStructValue() != nil {
				userMeta := userField.GetStructValue().Fields
				if l, ok := userMeta["locale"]; ok && l.GetStringValue() != locale {
					match = false
				}
			}
		}
		if len(interests) > 0 && u.Metadata != nil && u.Metadata.ServiceSpecific != nil {
			userField, ok := u.Metadata.ServiceSpecific.Fields["user"]
			if ok && userField.GetStructValue() != nil {
				userMeta := userField.GetStructValue().Fields
				if ints, ok := userMeta["interests"]; ok && ints.GetListValue() != nil {
					userInterests := map[string]struct{}{}
					for _, v := range ints.GetListValue().Values {
						userInterests[v.GetStringValue()] = struct{}{}
					}
					found := false
					for _, interest := range interests {
						if _, ok := userInterests[interest]; ok {
							found = true
							break
						}
					}
					if !found {
						match = false
					}
				}
			}
		}
		if match {
			filtered = append(filtered, u)
		}
	}
	return filtered, nil
}

// --- User Groups ---.
func (r *Repository) CreateUserGroup(ctx context.Context, group *userv1.UserGroup) (*userv1.UserGroup, error) {
	meta, err := metadatautil.MarshalCanonical(group.Metadata)
	if err != nil {
		return nil, err
	}
	roles, err := json.Marshal(group.Roles)
	if err != nil {
		return nil, err
	}
	memberIDs := pq.Array(group.MemberIds)
	var id string
	err = r.GetDB().QueryRowContext(ctx, `
		INSERT INTO service_user_group (name, description, member_ids, roles, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, group.Name, group.Description, memberIDs, roles, meta).Scan(&id)
	if err != nil {
		return nil, err
	}
	group.Id = id
	return group, nil
}

func (r *Repository) UpdateUserGroup(ctx context.Context, groupID string, group *userv1.UserGroup, fieldsToUpdate []string) (*userv1.UserGroup, error) {
	if len(fieldsToUpdate) == 0 {
		return nil, errors.New("no fields to update")
	}
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1
	for _, field := range fieldsToUpdate {
		switch field {
		case "name":
			setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
			args = append(args, group.Name)
			argIdx++
		case "description":
			setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
			args = append(args, group.Description)
			argIdx++
		case "member_ids":
			setClauses = append(setClauses, fmt.Sprintf("member_ids = $%d", argIdx))
			args = append(args, pq.Array(group.MemberIds))
			argIdx++
		case "roles":
			roles, err := json.Marshal(group.Roles)
			if err != nil {
				return nil, err
			}
			setClauses = append(setClauses, fmt.Sprintf("roles = $%d", argIdx))
			args = append(args, roles)
			argIdx++
		case "metadata":
			meta, err := metadatautil.MarshalCanonical(group.Metadata)
			if err != nil {
				return nil, err
			}
			setClauses = append(setClauses, fmt.Sprintf("metadata = $%d", argIdx))
			args = append(args, meta)
			argIdx++
		}
	}
	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, groupID)
	query := fmt.Sprintf("UPDATE service_user_group SET %s WHERE id = $%d RETURNING id, name, description, member_ids, roles, metadata, created_at, updated_at", strings.Join(setClauses, ", "), argIdx)
	var (
		id, name, description string
		memberIDs             []string
		rolesRaw              []byte
		metaRaw               []byte
		createdAt, updatedAt  time.Time
	)
	err := r.GetDB().QueryRowContext(ctx, query, args...).Scan(&id, &name, &description, pq.Array(&memberIDs), &rolesRaw, &metaRaw, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	group.Id = id
	group.Name = name
	group.Description = description
	group.MemberIds = memberIDs
	if err := json.Unmarshal(rolesRaw, &group.Roles); err != nil {
		return nil, err
	}
	if err := protojson.Unmarshal(metaRaw, group.Metadata); err != nil {
		return nil, err
	}
	group.CreatedAt = timestamppb.New(createdAt)
	group.UpdatedAt = timestamppb.New(updatedAt)
	return group, nil
}

func (r *Repository) DeleteUserGroup(ctx context.Context, groupID string) error {
	result, err := r.GetDB().ExecContext(ctx, "DELETE FROM service_user_group WHERE id = $1", groupID)
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

func (r *Repository) ListUserGroups(ctx context.Context, userID string, page, pageSize int) ([]*userv1.UserGroup, int, error) {
	offset := page * pageSize
	rows, err := r.GetDB().QueryContext(ctx, `
		SELECT id, name, description, member_ids, roles, metadata, created_at, updated_at
		FROM service_user_group
		WHERE $1 = ANY(member_ids)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	groups := []*userv1.UserGroup{}
	for rows.Next() {
		g := &userv1.UserGroup{}
		var memberIDs []string
		var rolesRaw []byte
		var metaRaw []byte
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&g.Id, &g.Name, &g.Description, pq.Array(&memberIDs), &rolesRaw, &metaRaw, &createdAt, &g.UpdatedAt); err != nil {
			return nil, 0, err
		}
		g.MemberIds = memberIDs
		if err := json.Unmarshal(rolesRaw, &g.Roles); err != nil {
			return nil, 0, err
		}
		if err := protojson.Unmarshal(metaRaw, g.Metadata); err != nil {
			return nil, 0, err
		}
		g.CreatedAt = timestamppb.New(createdAt)
		g.UpdatedAt = timestamppb.New(updatedAt)
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.GetDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM service_user_group WHERE $1 = ANY(member_ids)`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	return groups, total, nil
}

func (r *Repository) ListUserGroupMembers(ctx context.Context, groupID string, page, pageSize int) ([]*userv1.User, int, error) {
	offset := page * pageSize
	rows, err := r.GetDB().QueryContext(ctx, `
		SELECT u.id, u.username, u.email, u.profile, u.metadata
		FROM service_user_group g
		JOIN service_user u ON u.id = ANY(g.member_ids)
		WHERE g.id = $1
		ORDER BY u.created_at DESC
		LIMIT $2 OFFSET $3`, groupID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	users := []*userv1.User{}
	for rows.Next() {
		u := &userv1.User{}
		var profileRaw, metaRaw []byte
		if err := rows.Scan(&u.Id, &u.Username, &u.Email, &profileRaw, &metaRaw); err != nil {
			return nil, 0, err
		}
		if err := protojson.Unmarshal(profileRaw, u.Profile); err != nil {
			return nil, 0, err
		}
		if err := protojson.Unmarshal(metaRaw, u.Metadata); err != nil {
			return nil, 0, err
		}
		u.CreatedAt = nil // Set if needed
		u.UpdatedAt = nil // Set if needed
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.GetDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM service_user_group WHERE id = $1`, groupID).Scan(&total); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// MuteGroupIndividuals mutes all members of a group for a user (optionally, with a duration).
func (r *Repository) MuteGroupIndividuals(ctx context.Context, userID, groupID string, durationMinutes int, metadata *commonpb.Metadata) ([]string, error) {
	// Fetch all member IDs of the group (excluding userID), handling any group size via pagination
	page := 0
	pageSize := 1000
	var members []*userv1.User
	for {
		batch, total, err := r.ListUserGroupMembers(ctx, groupID, page, pageSize)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		members = append(members, batch...)
		if len(members) >= total {
			break
		}
		page++
	}

	// Parse metadata for selective muting
	targetUserIDs := map[string]struct{}{}
	excludeUserIDs := map[string]struct{}{}
	excludeFriends := false
	if metadata != nil && metadata.ServiceSpecific != nil {
		if userField, ok := metadata.ServiceSpecific.Fields["user"]; ok && userField.GetStructValue() != nil {
			userMeta := userField.GetStructValue().Fields
			if ids, ok := userMeta["target_user_ids"]; ok && ids.GetListValue() != nil {
				for _, v := range ids.GetListValue().Values {
					targetUserIDs[v.GetStringValue()] = struct{}{}
				}
			}
			if ids, ok := userMeta["exclude_user_ids"]; ok && ids.GetListValue() != nil {
				for _, v := range ids.GetListValue().Values {
					excludeUserIDs[v.GetStringValue()] = struct{}{}
				}
			}
			if ex, ok := userMeta["exclude_friends"]; ok {
				excludeFriends = ex.GetBoolValue()
			}
		}
	}

	// If excludeFriends is true, fetch friend IDs
	friendIDs := map[string]struct{}{}
	if excludeFriends {
		friends, _, err := r.ListFriends(ctx, userID, 0, 1000)
		if err != nil {
			return nil, err
		}
		for _, f := range friends {
			friendIDs[f.Id] = struct{}{}
		}
	}

	mutedUserIDs := make([]string, 0, len(members))
	expiration := time.Time{}
	if durationMinutes > 0 {
		expiration = time.Now().Add(time.Duration(durationMinutes) * time.Minute)
	}
	for _, member := range members {
		if member.Id == userID {
			continue
		}
		if len(targetUserIDs) > 0 {
			if _, ok := targetUserIDs[member.Id]; !ok {
				continue // skip users not in the target list
			}
		}
		if _, ok := excludeUserIDs[member.Id]; ok {
			continue // skip explicitly excluded users
		}
		if excludeFriends {
			if _, ok := friendIDs[member.Id]; ok {
				continue // skip friends
			}
		}
		mutedUserIDs = append(mutedUserIDs, member.Id)
	}
	if len(mutedUserIDs) == 0 {
		return nil, nil
	}
	// 2. Fetch user
	user, err := r.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Metadata == nil {
		user.Metadata = &commonpb.Metadata{}
	}
	// 3. Update metadata.service_specific.user.muted_users
	if user.Metadata.ServiceSpecific == nil {
		user.Metadata.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	// Build muted_users list as []map[string]interface{}
	mutedUsers := []map[string]interface{}{}
	for _, id := range mutedUserIDs {
		entry := map[string]interface{}{
			"user_id":  id,
			"group_id": groupID,
		}
		if !expiration.IsZero() {
			entry["expires_at"] = expiration.Format(time.RFC3339)
		}
		mutedUsers = append(mutedUsers, entry)
	}
	// Convert mutedUsers to structpb.Value
	mutedUsersVal, err := structpb.NewValue(mutedUsers)
	if err != nil {
		return nil, err
	}
	// Get or create the "user" namespace in service_specific
	userField, ok := user.Metadata.ServiceSpecific.Fields["user"]
	var userObj *structpb.Struct
	if ok && userField.GetStructValue() != nil {
		userObj = userField.GetStructValue()
	} else {
		userObj = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	userObj.Fields["muted_users"] = mutedUsersVal
	user.Metadata.ServiceSpecific.Fields["user"] = structpb.NewStructValue(userObj)
	// 4. Save updated metadata
	if err := r.Update(ctx, user); err != nil {
		return nil, err
	}
	return mutedUserIDs, nil
}

// BlockGroupIndividuals blocks all members of a group for a user (optionally, with a duration).
func (r *Repository) BlockGroupIndividuals(ctx context.Context, userID, groupID string, durationMinutes int, metadata *commonpb.Metadata) ([]string, error) {
	// Fetch all member IDs of the group (excluding userID), handling any group size via pagination
	page := 0
	pageSize := 1000
	var members []*userv1.User
	for {
		batch, total, err := r.ListUserGroupMembers(ctx, groupID, page, pageSize)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		members = append(members, batch...)
		if len(members) >= total {
			break
		}
		page++
	}

	// Parse metadata for selective blocking
	targetUserIDs := map[string]struct{}{}
	excludeUserIDs := map[string]struct{}{}
	excludeFriends := false
	if metadata != nil && metadata.ServiceSpecific != nil {
		if userField, ok := metadata.ServiceSpecific.Fields["user"]; ok && userField.GetStructValue() != nil {
			userMeta := userField.GetStructValue().Fields
			if ids, ok := userMeta["target_user_ids"]; ok && ids.GetListValue() != nil {
				for _, v := range ids.GetListValue().Values {
					targetUserIDs[v.GetStringValue()] = struct{}{}
				}
			}
			if ids, ok := userMeta["exclude_user_ids"]; ok && ids.GetListValue() != nil {
				for _, v := range ids.GetListValue().Values {
					excludeUserIDs[v.GetStringValue()] = struct{}{}
				}
			}
			if ex, ok := userMeta["exclude_friends"]; ok {
				excludeFriends = ex.GetBoolValue()
			}
		}
	}

	// If excludeFriends is true, fetch friend IDs
	friendIDs := map[string]struct{}{}
	if excludeFriends {
		friends, _, err := r.ListFriends(ctx, userID, 0, 1000)
		if err != nil {
			return nil, err
		}
		for _, f := range friends {
			friendIDs[f.Id] = struct{}{}
		}
	}

	blockedUserIDs := make([]string, 0, len(members))
	expiration := time.Time{}
	if durationMinutes > 0 {
		expiration = time.Now().Add(time.Duration(durationMinutes) * time.Minute)
	}
	for _, member := range members {
		if member.Id == userID {
			continue
		}
		if len(targetUserIDs) > 0 {
			if _, ok := targetUserIDs[member.Id]; !ok {
				continue // skip users not in the target list
			}
		}
		if _, ok := excludeUserIDs[member.Id]; ok {
			continue // skip explicitly excluded users
		}
		if excludeFriends {
			if _, ok := friendIDs[member.Id]; ok {
				continue // skip friends
			}
		}
		blockedUserIDs = append(blockedUserIDs, member.Id)
	}
	if len(blockedUserIDs) == 0 {
		return nil, nil
	}
	user, err := r.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Metadata == nil {
		user.Metadata = &commonpb.Metadata{}
	}
	if user.Metadata.ServiceSpecific == nil {
		user.Metadata.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	blockedUsers := []map[string]interface{}{}
	for _, id := range blockedUserIDs {
		entry := map[string]interface{}{
			"user_id":  id,
			"group_id": groupID,
		}
		if !expiration.IsZero() {
			entry["expires_at"] = expiration.Format(time.RFC3339)
		}
		blockedUsers = append(blockedUsers, entry)
	}
	blockedUsersVal, err := structpb.NewValue(blockedUsers)
	if err != nil {
		return nil, err
	}
	userField, ok := user.Metadata.ServiceSpecific.Fields["user"]
	var userObj *structpb.Struct
	if ok && userField.GetStructValue() != nil {
		userObj = userField.GetStructValue()
	} else {
		userObj = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	userObj.Fields["blocked_users"] = blockedUsersVal
	user.Metadata.ServiceSpecific.Fields["user"] = structpb.NewStructValue(userObj)
	if err := r.Update(ctx, user); err != nil {
		return nil, err
	}
	return blockedUserIDs, nil
}

// --- SSO, MFA, SCIM ---.
func (r *Repository) InitiateSSO(_ context.Context, provider, redirectURI string) (string, error) {
	// Generate SSO URL (use provider config, state, nonce)
	// Optionally, insert SSO initiation event in service_event
	return "https://sso.example.com/auth?provider=" + provider + "&redirect_uri=" + url.QueryEscape(redirectURI), nil
}

func (r *Repository) InitiateMFA(_ context.Context, _, _ string) (success bool, challengeID string, err error) {
	// Generate challenge, store in DB, return challengeID
	return true, "challenge-id", nil
}

func (r *Repository) SyncSCIM(_ context.Context, _ string) (bool, error) {
	// Parse SCIM, upsert users/groups/roles, log events
	return true, nil
}

// RegisterInterest creates or updates a pending user for interest registration.
func (r *Repository) RegisterInterest(ctx context.Context, email string) (*userv1.User, error) {
	user, err := r.GetByEmail(ctx, email)
	if err == nil && user != nil {
		if user.Status != int32(userv1.UserStatus_USER_STATUS_PENDING) {
			user.Status = int32(userv1.UserStatus_USER_STATUS_PENDING)
			if err := r.Update(ctx, user); err != nil {
				return nil, err
			}
		}
		return repoUserToProtoUser(user), nil
	}
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}
	newUser := &User{
		Email:    email,
		Status:   int32(userv1.UserStatus_USER_STATUS_PENDING),
		Metadata: &commonpb.Metadata{},
	}
	created, err := r.Create(ctx, newUser)
	if err != nil {
		return nil, err
	}
	return repoUserToProtoUser(created), nil
}

// CreateReferral generates a referral code for a user and campaign.
func (r *Repository) CreateReferral(ctx context.Context, userID, campaignSlug string) (string, error) {
	user, err := r.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	// Generate a unique referral code
	code := fmt.Sprintf("REF-%s-%s-%d", userID, campaignSlug, time.Now().Unix()%100000)
	user.ReferralCode = code
	if err := r.Update(ctx, user); err != nil {
		return "", err
	}
	return code, nil
}

func repoUserToProtoUser(user *User) *userv1.User {
	return &userv1.User{
		Id:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Profile:  repoProfileToProto(&user.Profile),
		Metadata: user.Metadata,
	}
}

func repoProfileToProto(p *Profile) *userv1.UserProfile {
	if p == nil {
		return nil
	}
	return &userv1.UserProfile{
		FirstName:    p.FirstName,
		LastName:     p.LastName,
		PhoneNumber:  p.PhoneNumber,
		AvatarUrl:    p.AvatarURL,
		Bio:          p.Bio,
		Timezone:     p.Timezone,
		Language:     p.Language,
		CustomFields: p.CustomFields,
	}
}

// UnblockUser removes a block between the current user and the target user.
func (r *Repository) UnblockUser(ctx context.Context, targetUserID string) error {
	user, err := r.GetByID(ctx, targetUserID)
	if err != nil {
		return err
	}
	if user.Metadata != nil {
		serviceMeta := getOrInitServiceUserMeta(user)
		if badActor, ok := serviceMeta["bad_actor"].(float64); ok && badActor > 0 {
			serviceMeta["bad_actor"] = badActor - 1
			err := setServiceUserMeta(user, serviceMeta)
			if err != nil {
				return err
			}
		}
	}
	return r.Update(ctx, user)
}

// UnmuteUser removes a mute between the current user and the target user.
func (r *Repository) UnmuteUser(ctx context.Context, targetUserID string) error {
	user, err := r.GetByID(ctx, targetUserID)
	if err != nil {
		return err
	}
	if user.Metadata != nil {
		serviceMeta := getOrInitServiceUserMeta(user)
		if badActor, ok := serviceMeta["bad_actor"].(float64); ok && badActor > 0 {
			serviceMeta["bad_actor"] = badActor - 1
			err := setServiceUserMeta(user, serviceMeta)
			if err != nil {
				return err
			}
		}
	}
	return r.Update(ctx, user)
}

// UnmuteGroup unmutes all members of a group for a user.
func (r *Repository) UnmuteGroup(ctx context.Context, userID, groupID string) error {
	user, err := r.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.Metadata == nil || user.Metadata.ServiceSpecific == nil {
		return nil
	}
	userField, ok := user.Metadata.ServiceSpecific.Fields["user"]
	if !ok || userField.GetStructValue() == nil {
		return nil
	}
	userObj := userField.GetStructValue()
	mutedVal, ok := userObj.Fields["muted_users"]
	if !ok || mutedVal.GetListValue() == nil {
		return nil
	}
	mutedUsers := mutedVal.GetListValue().Values
	filtered := []*structpb.Value{}
	for _, v := range mutedUsers {
		entry := v.GetStructValue()
		if entry == nil {
			continue
		}
		if entry.Fields["group_id"].GetStringValue() != groupID {
			filtered = append(filtered, v)
		}
	}
	userObj.Fields["muted_users"] = structpb.NewListValue(&structpb.ListValue{Values: filtered})
	user.Metadata.ServiceSpecific.Fields["user"] = structpb.NewStructValue(userObj)
	return r.Update(ctx, user)
}

// UnmuteGroupIndividuals unmutes specific users in a group for a user.
func (r *Repository) UnmuteGroupIndividuals(ctx context.Context, userID, groupID string, targetUserIDs []string) ([]string, error) {
	user, err := r.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Metadata == nil || user.Metadata.ServiceSpecific == nil {
		return nil, nil
	}
	userField, ok := user.Metadata.ServiceSpecific.Fields["user"]
	if !ok || userField.GetStructValue() == nil {
		return nil, nil
	}
	userObj := userField.GetStructValue()
	mutedVal, ok := userObj.Fields["muted_users"]
	if !ok || mutedVal.GetListValue() == nil {
		return nil, nil
	}
	mutedUsers := mutedVal.GetListValue().Values
	filtered := []*structpb.Value{}
	unmuted := []string{}
	for _, v := range mutedUsers {
		entry := v.GetStructValue()
		if entry == nil {
			continue
		}
		uid := entry.Fields["user_id"].GetStringValue()
		gid := entry.Fields["group_id"].GetStringValue()
		remove := false
		if gid == groupID {
			for _, tid := range targetUserIDs {
				if uid == tid {
					unmuted = append(unmuted, uid)
					remove = true
					break
				}
			}
		}
		if !remove {
			filtered = append(filtered, v)
		}
	}
	userObj.Fields["muted_users"] = structpb.NewListValue(&structpb.ListValue{Values: filtered})
	user.Metadata.ServiceSpecific.Fields["user"] = structpb.NewStructValue(userObj)
	if err := r.Update(ctx, user); err != nil {
		return nil, err
	}
	return unmuted, nil
}

// UnblockGroupIndividuals unblocks specific users in a group for a user.
func (r *Repository) UnblockGroupIndividuals(ctx context.Context, userID, groupID string, targetUserIDs []string) ([]string, error) {
	user, err := r.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Metadata == nil || user.Metadata.ServiceSpecific == nil {
		return nil, nil
	}
	userField, ok := user.Metadata.ServiceSpecific.Fields["user"]
	if !ok || userField.GetStructValue() == nil {
		return nil, nil
	}
	userObj := userField.GetStructValue()
	blockedVal, ok := userObj.Fields["blocked_users"]
	if !ok || blockedVal.GetListValue() == nil {
		return nil, nil
	}
	blockedUsers := blockedVal.GetListValue().Values
	filtered := []*structpb.Value{}
	unblocked := []string{}
	for _, v := range blockedUsers {
		entry := v.GetStructValue()
		if entry == nil {
			continue
		}
		uid := entry.Fields["user_id"].GetStringValue()
		gid := entry.Fields["group_id"].GetStringValue()
		remove := false
		if gid == groupID {
			for _, tid := range targetUserIDs {
				if uid == tid {
					unblocked = append(unblocked, uid)
					remove = true
					break
				}
			}
		}
		if !remove {
			filtered = append(filtered, v)
		}
	}
	userObj.Fields["blocked_users"] = structpb.NewListValue(&structpb.ListValue{Values: filtered})
	user.Metadata.ServiceSpecific.Fields["user"] = structpb.NewStructValue(userObj)
	if err := r.Update(ctx, user); err != nil {
		return nil, err
	}
	return unblocked, nil
}

// getOrInitServiceUserMeta extracts or initializes the user section of service_specific metadata for repository.User.
func getOrInitServiceUserMeta(user *User) map[string]interface{} {
	serviceMeta := map[string]interface{}{}
	if user.Metadata != nil && user.Metadata.ServiceSpecific != nil {
		m := user.Metadata.ServiceSpecific.AsMap()
		if u, ok := m["user"].(map[string]interface{}); ok {
			serviceMeta = u
		}
	}
	return serviceMeta
}

// setServiceUserMeta sets the user section of service_specific metadata for repository.User.
func setServiceUserMeta(user *User, serviceMeta map[string]interface{}) error {
	m := map[string]interface{}{}
	if user.Metadata != nil && user.Metadata.ServiceSpecific != nil {
		m = user.Metadata.ServiceSpecific.AsMap()
	}
	m["user"] = serviceMeta
	metaStruct, err := structpb.NewStruct(m)
	if err != nil {
		return err
	}
	if user.Metadata == nil {
		user.Metadata = &commonpb.Metadata{}
	}
	user.Metadata.ServiceSpecific = metaStruct
	return nil
}

// BlockGroupContent blocks a specific content item in a group for a user.
func (r *Repository) BlockGroupContent(_ context.Context, _, _, _ string, _ *commonpb.Metadata) error {
	// TODO: Implement DB logic to mark content as blocked for this user/group
	// Example: Insert or update a moderation table with block status
	return nil // Return error if operation fails
}

// ReportGroupContent reports a specific content item in a group.
func (r *Repository) ReportGroupContent(_ context.Context, _, _, _, _, _ string, _ *commonpb.Metadata) (string, error) {
	// TODO: Implement DB logic to insert a report record for this content
	// Example: Insert into a group_content_report table and return the report ID
	reportID := "report-123" // Replace with actual generated ID
	return reportID, nil     // Return error if operation fails
}

// MuteGroupContent mutes a specific content item in a group for a user.
func (r *Repository) MuteGroupContent(_ context.Context, _, _, _ string, _ int32, _ *commonpb.Metadata) error {
	// TODO: Implement DB logic to mark content as muted for this user/group
	// Example: Insert or update a moderation table with mute status and expiration
	return nil // Return error if operation fails
}

func (r *Repository) GetUserGroupByID(ctx context.Context, groupID string) (*userv1.UserGroup, error) {
	var (
		id, name, description string
		memberIDs             []string
		rolesRaw              []byte
		metaRaw               []byte
		createdAt, updatedAt  time.Time
	)
	err := r.GetDB().QueryRowContext(ctx, `SELECT id, name, description, member_ids, roles, metadata, created_at, updated_at FROM service_user_group WHERE id = $1`, groupID).Scan(&id, &name, &description, pq.Array(&memberIDs), &rolesRaw, &metaRaw, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	group := &userv1.UserGroup{
		Id:          id,
		Name:        name,
		Description: description,
		MemberIds:   memberIDs,
		Metadata:    &commonpb.Metadata{},
		CreatedAt:   timestamppb.New(createdAt),
		UpdatedAt:   timestamppb.New(updatedAt),
	}
	if err := json.Unmarshal(rolesRaw, &group.Roles); err != nil {
		return nil, err
	}
	if err := protojson.Unmarshal(metaRaw, group.Metadata); err != nil {
		return nil, err
	}
	return group, nil
}

// UpdateTx updates a user within a provided transaction (for atomic multi-user updates).
func (r *Repository) UpdateTx(ctx context.Context, tx *sql.Tx, user *User) error {
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

	// Validate password
	if err := validatePassword(user.PasswordHash); err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx,
		`UPDATE service_user 
		SET username = $1, email = $2, password_hash = $3, referral_code = $4, referred_by = $5, device_hash = $6, location = $7, profile = $8, roles = $9, status = $10, metadata = $11, updated_at = NOW()
		WHERE id = $12`,
		user.Username, user.Email, user.PasswordHash, user.ReferralCode, user.ReferredBy, user.DeviceHash, pq.Array(user.Locations), user.Profile, pq.Array(user.Roles), user.Status, user.Metadata, user.ID,
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
