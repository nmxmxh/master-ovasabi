package contentmoderation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	contentmoderationpb "github.com/nmxmxh/master-ovasabi/api/protos/contentmoderation/v1"
	repo "github.com/nmxmxh/master-ovasabi/internal/repository"
)

type ModerationResult struct {
	ContentID  string      `db:"content_id"`
	MasterID   string      `db:"master_id"`
	MasterUUID string      `db:"master_uuid"`
	UserID     string      `db:"user_id"`
	Status     string      `db:"status"`
	Reason     string      `db:"reason"`
	Metadata   interface{} `db:"metadata"`
	CampaignID int64       `db:"campaign_id"`
	CreatedAt  time.Time   `db:"created_at"`
	UpdatedAt  time.Time   `db:"updated_at"`
}

type PostgresRepository struct {
	db         *sql.DB
	masterRepo repo.MasterRepository
}

func NewPostgresRepository(db *sql.DB, masterRepo repo.MasterRepository) *PostgresRepository {
	return &PostgresRepository{db: db, masterRepo: masterRepo}
}

func (r *PostgresRepository) SubmitContentForModeration(ctx context.Context, contentID, masterID, masterUUID, userID, contentType, content string, metadata []byte, campaignID int64) (*contentmoderationpb.ModerationResult, error) {
	if contentID == "" || userID == "" {
		return nil, errors.New("content_id and user_id are required")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO service_contentmoderation_result (content_id, master_id, master_uuid, user_id, content_type, content, status, metadata, campaign_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now()) ON CONFLICT (content_id) DO UPDATE SET master_id = $2, master_uuid = $3, user_id = $4, content_type = $5, content = $6, status = $7, metadata = $8, campaign_id = $9, updated_at = now()`,
		contentID, masterID, masterUUID, userID, contentType, content, "PENDING", metadata, campaignID)
	if err != nil {
		return nil, err
	}
	return r.GetModerationResult(ctx, contentID)
}

func (r *PostgresRepository) GetModerationResult(ctx context.Context, contentID string) (*contentmoderationpb.ModerationResult, error) {
	row := r.db.QueryRowContext(ctx, `SELECT content_id, master_id, master_uuid, user_id, status, reason, metadata, campaign_id, created_at, updated_at FROM service_contentmoderation_result WHERE content_id = $1`, contentID)
	var res contentmoderationpb.ModerationResult
	var metaJSON []byte
	var createdAt, updatedAt time.Time
	var masterID, masterUUID string
	var campaignID int64
	if err := row.Scan(&res.ContentId, &masterID, &masterUUID, &res.UserId, &res.Status, &res.Reason, &metaJSON, &campaignID, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	if len(metaJSON) > 0 {
		if err := json.Unmarshal(metaJSON, &res.Metadata); err != nil {
			res.Metadata = nil
		}
	}
	res.CreatedAt = createdAt.Unix()
	res.UpdatedAt = updatedAt.Unix()
	return &res, nil
}

func (r *PostgresRepository) ListFlaggedContent(ctx context.Context, page, pageSize int, status string, campaignID int64) ([]*contentmoderationpb.ModerationResult, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM service_contentmoderation_result WHERE status = $1 AND campaign_id = $2`, status, campaignID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT content_id, master_id, master_uuid, user_id, status, reason, metadata, campaign_id, created_at, updated_at FROM service_contentmoderation_result WHERE status = $1 AND campaign_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`, status, campaignID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var results []*contentmoderationpb.ModerationResult
	for rows.Next() {
		var res contentmoderationpb.ModerationResult
		var metaJSON []byte
		var createdAt, updatedAt time.Time
		var masterID, masterUUID string
		var campaignID int64
		if err := rows.Scan(&res.ContentId, &masterID, &masterUUID, &res.UserId, &res.Status, &res.Reason, &metaJSON, &campaignID, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}
		if len(metaJSON) > 0 {
			if err := json.Unmarshal(metaJSON, &res.Metadata); err != nil {
				res.Metadata = nil
			}
		}
		res.CreatedAt = createdAt.Unix()
		res.UpdatedAt = updatedAt.Unix()
		results = append(results, &res)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (r *PostgresRepository) ApproveContent(ctx context.Context, contentID, masterID, masterUUID string, metadata []byte, campaignID int64) (*contentmoderationpb.ModerationResult, error) {
	_, err := r.db.ExecContext(ctx, `UPDATE service_contentmoderation_result SET status = $1, master_id = $2, master_uuid = $3, metadata = $4, campaign_id = $5, updated_at = now() WHERE content_id = $6`, "APPROVED", masterID, masterUUID, metadata, campaignID, contentID)
	if err != nil {
		return nil, err
	}
	return r.GetModerationResult(ctx, contentID)
}

func (r *PostgresRepository) RejectContent(ctx context.Context, contentID, masterID, masterUUID, reason string, metadata []byte, campaignID int64) (*contentmoderationpb.ModerationResult, error) {
	_, err := r.db.ExecContext(ctx, `UPDATE service_contentmoderation_result SET status = $1, reason = $2, master_id = $3, master_uuid = $4, metadata = $5, campaign_id = $6, updated_at = now() WHERE content_id = $7`, "REJECTED", reason, masterID, masterUUID, metadata, campaignID, contentID)
	if err != nil {
		return nil, err
	}
	return r.GetModerationResult(ctx, contentID)
}
