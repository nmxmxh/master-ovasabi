package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v0"
	"github.com/nmxmxh/master-ovasabi/internal/shared/dbiface"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service implements the UserService gRPC interface.
type Service struct {
	userpb.UnimplementedUserServiceServer
	log *zap.Logger
	db  dbiface.DB
}

// NewUserService creates a new instance of UserService.
func NewUserService(log *zap.Logger, db dbiface.DB) userpb.UserServiceServer {
	return &Service{
		log: log,
		db:  db,
	}
}

// CreateUser creates a new user following the Master-Client-Service-Event pattern.
func (s *Service) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	s.log.Info("Creating user", zap.String("email", req.Email))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.log.Warn("failed to rollback tx", zap.Error(err))
		}
	}()

	// 1. Create master record
	var masterID int32
	err = tx.QueryRowContext(ctx,
		`INSERT INTO master (uuid, name, type) 
		 VALUES ($1, $2, 'user') 
		 RETURNING id`,
		req.Email, req.Username).Scan(&masterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create master record: %v", err)
	}

	// 2. Create service_user record
	var userID int32
	err = tx.QueryRowContext(ctx,
		`INSERT INTO service_user 
		 (master_id, email, referral_code, device_hash, location, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) 
		 RETURNING id`,
		masterID, req.Email, req.Metadata["referral_code"],
		req.Metadata["device_hash"], req.Metadata["location"]).Scan(&userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create service_user record: %v", err)
	}

	// 3. Log event
	_, err = tx.ExecContext(ctx,
		`INSERT INTO service_event 
		 (master_id, event_type, payload) 
		 VALUES ($1, 'user_created', $2)`,
		masterID, req.Metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to log event: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	// Return the created user
	return &userpb.CreateUserResponse{
		User: &userpb.User{
			Id:           userID,
			Email:        req.Email,
			ReferralCode: req.Metadata["referral_code"],
			DeviceHash:   req.Metadata["device_hash"],
			Location:     req.Metadata["location"],
			CreatedAt:    timestamppb.Now(),
			UpdatedAt:    timestamppb.Now(),
		},
	}, nil
}

// GetUser retrieves user information.
func (s *Service) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {
	var user userpb.User
	err := s.db.QueryRowContext(ctx,
		`SELECT u.id, u.email, u.referral_code, u.device_hash, u.location, 
		        u.created_at, u.updated_at
		 FROM service_user u
		 JOIN master m ON m.id = u.master_id
		 WHERE u.id = $1`, req.UserId).
		Scan(&user.Id, &user.Email, &user.ReferralCode, &user.DeviceHash,
			&user.Location, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	return &userpb.GetUserResponse{User: &user}, nil
}

func (s *Service) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	userID, err := strconv.Atoi(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	// Build dynamic update query
	fields := []string{}
	args := []interface{}{}
	argPos := 1

	if len(req.FieldsToUpdate) > 0 {
		for _, field := range req.FieldsToUpdate {
			switch field {
			case "email":
				fields = append(fields, "email = $"+strconv.Itoa(argPos))
				args = append(args, req.User.Email)
				argPos++
			case "referral_code":
				fields = append(fields, "referral_code = $"+strconv.Itoa(argPos))
				args = append(args, req.User.ReferralCode)
				argPos++
			case "location":
				fields = append(fields, "location = $"+strconv.Itoa(argPos))
				args = append(args, req.User.Location)
				argPos++
			}
		}
	}

	if len(fields) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no fields to update")
	}

	fields = append(fields, "updated_at = NOW()")
	query := "UPDATE service_user SET " +
		strings.Join(fields, ", ") +
		" WHERE id = $" + strconv.Itoa(argPos)
	args = append(args, userID)

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, status.Error(codes.AlreadyExists, "duplicate value for unique field")
		}
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}

	// Fetch updated user
	getResp, err := s.GetUser(ctx, &userpb.GetUserRequest{UserId: req.UserId})
	if err != nil {
		return nil, err
	}

	return &userpb.UpdateUserResponse{
		User: getResp.User,
	}, nil
}

func (s *Service) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	userID, err := strconv.Atoi(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	res, err := s.db.ExecContext(ctx, "DELETE FROM service_user WHERE id = $1", userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete user: %v", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	if n == 0 {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &userpb.DeleteUserResponse{
		Success: true,
	}, nil
}

// ListUsers retrieves a list of users with pagination and filtering.
func (s *Service) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	// Input validation
	if req.Page < 0 {
		return nil, status.Error(codes.InvalidArgument, "page number cannot be negative")
	}
	if req.PageSize < 0 || req.PageSize > 100 {
		return nil, status.Error(codes.InvalidArgument, "page size must be between 0 and 100")
	}

	// Build the query with filters
	query := `
		SELECT u.id, u.email, u.referral_code, u.device_hash, u.location, 
		       u.created_at, u.updated_at,
		       COUNT(*) OVER() as total_count
		FROM service_user u
		JOIN master m ON m.id = u.master_id
		WHERE 1=1
	`
	args := []any{}
	argPos := 1

	// Apply filters
	for key, value := range req.Filters {
		switch key {
		case "email":
			query += fmt.Sprintf(" AND u.email = $%d", argPos)
			args = append(args, value)
			argPos++
		case "location":
			query += fmt.Sprintf(" AND u.location = $%d", argPos)
			args = append(args, value)
			argPos++
		}
	}

	// Add pagination
	pageSize := int32(10)
	if req.PageSize > 0 {
		pageSize = req.PageSize
	}
	offset := req.Page * pageSize
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, pageSize, offset)

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		s.log.Error("failed to execute list users query",
			zap.Error(err),
			zap.String("query", query),
			zap.Any("args", args))
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.log.Warn("failed to close rows", zap.Error(err))
		}
	}()

	var users []*userpb.User
	var totalCount int32

	for rows.Next() {
		var user userpb.User
		err := rows.Scan(
			&user.Id, &user.Email, &user.ReferralCode, &user.DeviceHash,
			&user.Location, &user.CreatedAt, &user.UpdatedAt, &totalCount)
		if err != nil {
			s.log.Error("failed to scan user row", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "failed to scan row: %v", err)
		}
		users = append(users, &user)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		s.log.Error("error iterating over user rows", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "error iterating rows: %v", err)
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	return &userpb.ListUsersResponse{
		Users:      users,
		TotalCount: totalCount,
		Page:       req.Page,
		TotalPages: totalPages,
	}, nil
}

// UpdatePassword implements the UpdatePassword RPC method.
func (s *Service) UpdatePassword(_ context.Context, _ *userpb.UpdatePasswordRequest) (*userpb.UpdatePasswordResponse, error) {
	// In a real implementation, you would:
	// 1. Verify the current password
	// 2. Hash the new password
	// 3. Update the password in the database
	// For this example, we'll just return success
	return &userpb.UpdatePasswordResponse{
		Success:   true,
		UpdatedAt: time.Now().Unix(),
	}, nil
}

// Fixed issues with User struct alignment and field handling.
func (s *Service) UpdateProfile(ctx context.Context, req *userpb.UpdateProfileRequest) (*userpb.UpdateProfileResponse, error) {
	userID, err := strconv.Atoi(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	fields := []string{}
	args := []any{}
	argPos := 1

	// For backward compatibility, also support updating email, referral_code, device_hash, location if present in FieldsToUpdate
	for _, field := range req.FieldsToUpdate {
		switch field {
		case "email":
			if req.Profile != nil && req.Profile.CustomFields["email"] != "" {
				fields = append(fields, "email = $"+strconv.Itoa(argPos))
				args = append(args, req.Profile.CustomFields["email"])
				argPos++
			}
		case "referral_code":
			if req.Profile != nil && req.Profile.CustomFields["referral_code"] != "" {
				fields = append(fields, "referral_code = $"+strconv.Itoa(argPos))
				args = append(args, req.Profile.CustomFields["referral_code"])
				argPos++
			}
		case "device_hash":
			if req.Profile != nil && req.Profile.CustomFields["device_hash"] != "" {
				fields = append(fields, "device_hash = $"+strconv.Itoa(argPos))
				args = append(args, req.Profile.CustomFields["device_hash"])
				argPos++
			}
		case "location":
			if req.Profile != nil && req.Profile.CustomFields["location"] != "" {
				fields = append(fields, "location = $"+strconv.Itoa(argPos))
				args = append(args, req.Profile.CustomFields["location"])
				argPos++
			}
		}
	}

	if len(fields) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no fields to update")
	}

	fields = append(fields, "updated_at = NOW()")
	query := "UPDATE service_user SET " +
		strings.Join(fields, ", ") +
		" WHERE id = $" + strconv.Itoa(argPos)
	args = append(args, userID)

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, status.Error(codes.AlreadyExists, "duplicate value for unique field")
		}
		return nil, status.Errorf(codes.Internal, "failed to update profile: %v", err)
	}

	// Fetch updated user
	getResp, err := s.GetUser(ctx, &userpb.GetUserRequest{UserId: req.UserId})
	if err != nil {
		return nil, err
	}

	return &userpb.UpdateProfileResponse{
		User: getResp.User,
	}, nil
}

// Helper to check for unique constraint violation.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pqErr *pq.Error
	return errors.As(err, &pqErr)
}
