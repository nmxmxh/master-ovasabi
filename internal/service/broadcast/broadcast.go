package broadcast

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/api/protos/broadcast/v0"
	"github.com/nmxmxh/master-ovasabi/internal/shared/dbiface"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ServiceImpl implements the BroadcastService interface.
type ServiceImpl struct {
	broadcast.UnimplementedBroadcastServiceServer
	log     *zap.Logger
	db      dbiface.DB
	clients sync.Map // map[string]chan *broadcast.ActionSummary
}

// NewService creates a new instance of BroadcastService.
func NewService(log *zap.Logger, db dbiface.DB) *ServiceImpl {
	return &ServiceImpl{
		log: log,
		db:  db,
	}
}

// BroadcastAction implements the BroadcastAction RPC method.
func (s *ServiceImpl) BroadcastAction(ctx context.Context, req *broadcast.BroadcastActionRequest) (*broadcast.BroadcastActionResponse, error) {
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
		 VALUES ($1, $2, 'broadcast') 
		 RETURNING id`,
		req.UserId, "broadcast_action").Scan(&masterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create master record: %v", err)
	}

	// 2. Create service_broadcast record
	payload, err := json.Marshal(map[string]interface{}{
		"user_id":        req.UserId,
		"action_type":    req.ActionType,
		"application_id": req.ApplicationId,
		"metadata":       req.Metadata,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal payload: %v", err)
	}

	var broadcastID int32
	err = tx.QueryRowContext(ctx,
		`INSERT INTO service_broadcast 
		 (master_id, action_type, payload, created_at) 
		 VALUES ($1, $2, $3, NOW()) 
		 RETURNING id`,
		masterID, req.ActionType, payload).Scan(&broadcastID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create broadcast record: %v", err)
	}

	// 3. Log event
	_, err = tx.ExecContext(ctx,
		`INSERT INTO service_event 
		 (master_id, event_type, payload) 
		 VALUES ($1, 'broadcast_created', $2)`,
		masterID, payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to log event: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	// Create action summary for subscribers
	summary := &broadcast.ActionSummary{
		UserId:        req.UserId,
		ActionType:    req.ActionType,
		ApplicationId: req.ApplicationId,
		Metadata:      req.Metadata,
		Timestamp:     time.Now().Unix(),
	}

	// Notify subscribers
	s.clients.Range(func(key, value interface{}) bool {
		if ch, ok := value.(chan *broadcast.ActionSummary); ok {
			select {
			case ch <- summary:
			default:
				// Channel is full or closed, remove subscriber
				s.clients.Delete(key)
			}
		}
		return true
	})

	return &broadcast.BroadcastActionResponse{
		Success: true,
		Message: "Action broadcasted successfully",
	}, nil
}

// SubscribeToActions implements the SubscribeToActions streaming RPC method.
func (s *ServiceImpl) SubscribeToActions(req *broadcast.SubscribeRequest, stream broadcast.BroadcastService_SubscribeToActionsServer) error {
	ch := make(chan *broadcast.ActionSummary, 100)
	// Use application_id as the client key (or could use campaign_id if needed)
	clientID := req.ApplicationId

	// Store the subscriber
	s.clients.Store(clientID, ch)
	defer func() {
		s.clients.Delete(clientID)
		close(ch)
	}()

	// Query recent actions from database (filter by campaign if needed)
	rows, err := s.db.QueryContext(stream.Context(), `
		SELECT payload, created_at
		FROM service_broadcast
		WHERE created_at >= $1
		ORDER BY created_at DESC
		LIMIT 100`,
		time.Now().Add(-24*time.Hour))
	if err != nil {
		return status.Errorf(codes.Internal, "failed to query recent actions: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.log.Warn("failed to close rows", zap.Error(err))
		}
	}()

	// Send recent actions to the subscriber
	for rows.Next() {
		var payload []byte
		var createdAt time.Time

		if err := rows.Scan(&payload, &createdAt); err != nil {
			return status.Errorf(codes.Internal, "failed to scan action: %v", err)
		}

		var data map[string]interface{}
		if err := json.Unmarshal(payload, &data); err != nil {
			s.log.Error("failed to unmarshal action payload",
				zap.Error(err))
			continue
		}

		summary := &broadcast.ActionSummary{
			UserId:        toString(data["user_id"]),
			ActionType:    toString(data["action_type"]),
			ApplicationId: toString(data["application_id"]),
			Timestamp:     createdAt.Unix(),
		}

		if err := stream.Send(summary); err != nil {
			return err
		}
	}

	if err := rows.Err(); err != nil {
		return status.Errorf(codes.Internal, "error iterating rows: %v", err)
	}

	// Listen for new actions
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case summary := <-ch:
			if err := stream.Send(summary); err != nil {
				return err
			}
		}
	}
}

// Helper to safely convert interface{} to string.
func toString(val interface{}) string {
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

// GetBroadcast retrieves a specific broadcast by ID.
func (s *ServiceImpl) GetBroadcast(ctx context.Context, req *broadcast.GetBroadcastRequest) (*broadcast.GetBroadcastResponse, error) {
	var b broadcast.Broadcast
	var payload []byte
	var createdAt time.Time

	err := s.db.QueryRowContext(ctx, `
		SELECT id, campaign_id, channel, subject, message, payload, created_at, scheduled_at
		FROM service_broadcast
		WHERE id = $1`,
		req.BroadcastId).
		Scan(&b.Id, &b.CampaignId, &b.Channel, &b.Subject, &b.Message, &payload, &createdAt, &b.ScheduledAt)

	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "broadcast not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	// Parse payload (if needed)
	// Optionally, you can parse payload into b.Payload if your proto supports it
	b.CreatedAt = timestamppb.New(createdAt)

	return &broadcast.GetBroadcastResponse{
		Broadcast: &b,
	}, nil
}

// ListBroadcasts retrieves a list of broadcasts with pagination.
func (s *ServiceImpl) ListBroadcasts(ctx context.Context, req *broadcast.ListBroadcastsRequest) (*broadcast.ListBroadcastsResponse, error) {
	query := `
		SELECT id, campaign_id, channel, subject, message, payload, created_at, scheduled_at,
		       COUNT(*) OVER() as total_count
		FROM service_broadcast
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Apply campaign filter
	if req.CampaignId != 0 {
		query += ` AND campaign_id = $` + strconv.Itoa(argPos)
		args = append(args, req.CampaignId)
		argPos++
	}

	// Add pagination
	pageSize := int32(10)
	if req.PageSize > 0 {
		pageSize = req.PageSize
	}
	offset := req.Page * pageSize

	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argPos) + ` OFFSET $` + strconv.Itoa(argPos+1)
	args = append(args, pageSize, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.log.Warn("failed to close rows", zap.Error(err))
		}
	}()

	var broadcasts []*broadcast.Broadcast
	var totalCount int32

	for rows.Next() {
		var b broadcast.Broadcast
		var payload []byte
		var createdAt time.Time
		var scheduledAt time.Time

		err := rows.Scan(
			&b.Id,
			&b.CampaignId,
			&b.Channel,
			&b.Subject,
			&b.Message,
			&payload,
			&createdAt,
			&scheduledAt,
			&totalCount,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan row: %v", err)
		}

		b.CreatedAt = timestamppb.New(createdAt)
		b.ScheduledAt = timestamppb.New(scheduledAt)
		// Optionally parse payload into b.Payload if needed

		broadcasts = append(broadcasts, &b)
	}

	if err = rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "error iterating rows: %v", err)
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	return &broadcast.ListBroadcastsResponse{
		Broadcasts: broadcasts,
		TotalCount: totalCount,
		Page:       req.Page,
		TotalPages: totalPages,
	}, nil
}
