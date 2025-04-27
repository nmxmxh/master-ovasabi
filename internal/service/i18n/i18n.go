package i18n

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/nmxmxh/master-ovasabi/api/protos/i18n/v0"
	"github.com/nmxmxh/master-ovasabi/internal/shared/dbiface"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ServiceImpl implements the I18nService interface.
type ServiceImpl struct {
	i18n.UnimplementedI18NServiceServer
	log              *zap.Logger
	db               dbiface.DB
	supportedLocales []string
	defaultLocale    string
}

// NewService creates a new instance of I18nService.
func NewService(log *zap.Logger, db dbiface.DB) *ServiceImpl {
	return &ServiceImpl{
		log:              log,
		db:               db,
		supportedLocales: []string{"en", "es", "fr"},
		defaultLocale:    "en",
	}
}

func (s *ServiceImpl) CreateTranslation(ctx context.Context, req *i18n.CreateTranslationRequest) (*i18n.CreateTranslationResponse, error) {
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
		 VALUES ($1, $2, 'i18n') 
		 RETURNING id`,
		uuid.New().String(), "i18n_entry").Scan(&masterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create master record: %v", err)
	}

	// 2. Create service_translation record
	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal metadata: %v", err)
	}

	var translation i18n.Translation
	err = tx.QueryRowContext(ctx,
		`INSERT INTO service_translation 
		(master_id, campaign_id, key, language, value, metadata, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, NOW()) 
		RETURNING id, created_at`,
		masterID, req.CampaignId, req.Key, req.Language, req.Value, metadata).
		Scan(&translation.Id, &translation.CreatedAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create translation: %v", err)
	}

	// Fill in the rest of the translation fields
	translation.MasterId = masterID
	translation.CampaignId = req.CampaignId
	translation.Key = req.Key
	translation.Language = req.Language
	translation.Value = req.Value
	translation.Metadata = req.Metadata

	// 3. Log event
	_, err = tx.ExecContext(ctx,
		`INSERT INTO service_event 
		(master_id, event_type, payload) 
		VALUES ($1, 'translation_created', $2)`,
		masterID, metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to log event: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	return &i18n.CreateTranslationResponse{
		Translation: &translation,
		Success:     true,
	}, nil
}

func (s *ServiceImpl) GetTranslation(ctx context.Context, req *i18n.GetTranslationRequest) (*i18n.GetTranslationResponse, error) {
	var translation i18n.Translation
	var metadataBytes []byte
	var createdAt time.Time

	err := s.db.QueryRowContext(ctx, `
		SELECT t.id, t.master_id, t.campaign_id, t.key, t.language, t.value, t.metadata, t.created_at
		FROM service_translation t
		WHERE t.id = $1`,
		req.TranslationId).
		Scan(&translation.Id, &translation.MasterId, &translation.CampaignId,
			&translation.Key, &translation.Language, &translation.Value,
			&metadataBytes, &createdAt)

	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "translation not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	// Parse metadata
	if err := json.Unmarshal(metadataBytes, &translation.Metadata); err != nil {
		s.log.Warn("failed to unmarshal translation metadata",
			zap.Int32("translation_id", translation.Id),
			zap.Error(err))
	}

	translation.CreatedAt = timestamppb.New(createdAt)

	return &i18n.GetTranslationResponse{
		Translation: &translation,
	}, nil
}

func (s *ServiceImpl) ListTranslations(ctx context.Context, req *i18n.ListTranslationsRequest) (*i18n.ListTranslationsResponse, error) {
	query := `
		SELECT t.id, t.master_id, t.campaign_id, t.key, t.language, t.value, t.metadata, t.created_at,
		       COUNT(*) OVER() as total_count
		FROM service_translation t
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Apply filters
	if req.CampaignId != 0 {
		query += ` AND t.campaign_id = $` + strconv.Itoa(argPos)
		args = append(args, req.CampaignId)
		argPos++
	}

	if req.Language != "" {
		query += ` AND t.language = $` + strconv.Itoa(argPos)
		args = append(args, req.Language)
		argPos++
	}

	// Add pagination
	pageSize := int32(10)
	if req.PageSize > 0 {
		pageSize = req.PageSize
	}
	offset := req.Page * pageSize

	query += ` ORDER BY t.created_at DESC LIMIT $` + strconv.Itoa(argPos) + ` OFFSET $` + strconv.Itoa(argPos+1)
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

	var translations []*i18n.Translation
	var totalCount int32

	for rows.Next() {
		var translation i18n.Translation
		var metadataBytes []byte
		var createdAt time.Time

		err := rows.Scan(
			&translation.Id,
			&translation.MasterId,
			&translation.CampaignId,
			&translation.Key,
			&translation.Language,
			&translation.Value,
			&metadataBytes,
			&createdAt,
			&totalCount,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan row: %v", err)
		}

		if err := json.Unmarshal(metadataBytes, &translation.Metadata); err != nil {
			s.log.Warn("failed to unmarshal translation metadata",
				zap.Int32("translation_id", translation.Id),
				zap.Error(err))
		}

		translation.CreatedAt = timestamppb.New(createdAt)
		translations = append(translations, &translation)
	}

	if err = rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "error iterating rows: %v", err)
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	return &i18n.ListTranslationsResponse{
		Translations: translations,
		TotalCount:   totalCount,
		Page:         req.Page,
		TotalPages:   totalPages,
	}, nil
}
