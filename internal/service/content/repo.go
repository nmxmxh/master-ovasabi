package content

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	contentpb "github.com/nmxmxh/master-ovasabi/api/protos/content/v1"
	repo "github.com/nmxmxh/master-ovasabi/internal/repository"
	metadatautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

type Repository struct {
	db         *sql.DB
	masterRepo repo.MasterRepository
}

func NewRepository(db *sql.DB, masterRepo repo.MasterRepository) *Repository {
	return &Repository{db: db, masterRepo: masterRepo}
}

// Content CRUD.
func (r *Repository) CreateContent(ctx context.Context, c *contentpb.Content) (*contentpb.Content, error) {
	meta, err := metadatautil.MarshalCanonical(c.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	tags := strings.Join(c.Tags, ",")
	media := strings.Join(c.MediaUrls, ",")
	var id string
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO service_content_main (master_id, author_id, type, title, body, media_urls, metadata, tags, parent_id, visibility, campaign_id, created_at, updated_at, search_vector)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW(),NOW(), to_tsvector('english', $4 || ' ' || $5 || ' ' || $8 || ' ' || $9))
		RETURNING id
	`, c.MasterId, c.AuthorId, c.Type, c.Title, c.Body, media, meta, tags, c.ParentId, c.Visibility, c.CampaignId).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetContent(ctx, id)
}

func (r *Repository) GetContent(ctx context.Context, id string) (*contentpb.Content, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, master_id, author_id, type, title, body, media_urls, metadata, tags, parent_id, visibility, campaign_id, created_at, updated_at FROM service_content_main WHERE id = $1`, id)
	return scanContent(row)
}

func (r *Repository) UpdateContent(ctx context.Context, c *contentpb.Content) (*contentpb.Content, error) {
	meta, err := metadatautil.MarshalCanonical(c.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	tags := strings.Join(c.Tags, ",")
	media := strings.Join(c.MediaUrls, ",")
	_, err = r.db.ExecContext(ctx, `
		UPDATE service_content_main SET title=$1, body=$2, media_urls=$3, metadata=$4, tags=$5, parent_id=$6, visibility=$7, campaign_id=$8, updated_at=NOW(), search_vector=to_tsvector('english', $1 || ' ' || $2 || ' ' || $5 || ' ' || $6) WHERE id=$9
	`, c.Title, c.Body, media, meta, tags, c.ParentId, c.Visibility, c.CampaignId, c.Id)
	if err != nil {
		return nil, err
	}
	return r.GetContent(ctx, c.Id)
}

func (r *Repository) DeleteContent(ctx context.Context, id string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM service_content_main WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *Repository) ListContent(ctx context.Context, authorID, ctype string, campaignID int64, page, pageSize int) ([]*contentpb.Content, int, error) {
	args := []interface{}{}
	where := []string{}
	if authorID != "" {
		where = append(where, "author_id = ?")
		args = append(args, authorID)
	}
	if ctype != "" {
		where = append(where, "type = ?")
		args = append(args, ctype)
	}
	if campaignID != 0 {
		where = append(where, "campaign_id = ?")
		args = append(args, campaignID)
	}
	if page < 0 {
		page = 0
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := page * pageSize
	args = append(args, pageSize, offset)
	baseQuery := "SELECT id, master_id, author_id, type, title, body, media_urls, metadata, tags, parent_id, visibility, campaign_id, created_at, updated_at, comment_count, reaction_counts FROM service_content_main"
	if len(where) > 0 {
		baseQuery += " WHERE " + strings.Join(where, " AND ")
	}
	baseQuery += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	// Replace '?' with proper $N for Postgres
	for i := 1; i <= len(args); i++ {
		baseQuery = strings.Replace(baseQuery, "?", "$"+fmt.Sprintf("%d", i), 1)
	}
	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	results := make([]*contentpb.Content, 0, pageSize)
	for rows.Next() {
		c, err := scanContentWithCounters(rows)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	total := len(results)
	return results, total, nil
}

// Full-text/context search with master_id support.
func (r *Repository) SearchContent(ctx context.Context, query, contextQuery, masterID string, page, pageSize int) ([]*contentpb.Content, int, error) {
	args := []interface{}{query, contextQuery}
	where := []string{"(search_vector @@ plainto_tsquery('english', $1) OR search_vector @@ plainto_tsquery('english', $2))"}
	argIdx := 3
	if masterID != "" {
		where = append(where, fmt.Sprintf("master_id = $%d", argIdx))
		args = append(args, masterID)
		argIdx++
	}
	args = append(args, pageSize, page*pageSize)
	baseQuery := "SELECT id, master_id, author_id, type, title, body, media_urls, metadata, tags, parent_id, visibility, created_at, updated_at FROM service_content_main"
	if len(where) > 0 {
		baseQuery += " WHERE " + strings.Join(where, " AND ")
	}
	baseQuery += fmt.Sprintf(" ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	results := make([]*contentpb.Content, 0, pageSize)
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	countQuery := "SELECT COUNT(*) FROM service_content_main"
	if len(where) > 0 {
		countQuery += " WHERE " + strings.Join(where, " AND ")
	}
	countArgs := args[:len(args)-2]
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		total = len(results)
	}
	return results, total, nil
}

// Reactions.
func (r *Repository) AddReaction(ctx context.Context, contentID, userID, reaction string) (int, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO content_reactions (content_id, user_id, reaction, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (content_id, user_id, reaction) DO NOTHING
	`, contentID, userID, reaction)
	if err != nil {
		return 0, err
	}
	var countsRaw []byte
	err = r.db.QueryRowContext(ctx, `SELECT reaction_counts FROM service_content_main WHERE id = $1`, contentID).Scan(&countsRaw)
	if err != nil {
		return 0, err
	}
	var counts map[string]int
	if err := json.Unmarshal(countsRaw, &counts); err != nil {
		return 0, err
	}
	return counts[reaction], nil
}

func (r *Repository) ListReactions(ctx context.Context, contentID string) (map[string]int, error) {
	var countsRaw []byte
	err := r.db.QueryRowContext(ctx, `SELECT reaction_counts FROM service_content_main WHERE id = $1`, contentID).Scan(&countsRaw)
	if err != nil {
		return nil, err
	}
	var counts map[string]int
	if err := json.Unmarshal(countsRaw, &counts); err != nil {
		return nil, err
	}
	return counts, nil
}

// Helper to scan a content row.
func scanContent(row interface {
	Scan(dest ...interface{}) error
},
) (*contentpb.Content, error) {
	var id, masterID, authorID, ctype, title, body, media, meta, tags, parentID, visibility string
	var campaignID int64
	var createdAt, updatedAt time.Time
	if err := row.Scan(&id, &masterID, &authorID, &ctype, &title, &body, &media, &meta, &tags, &parentID, &visibility, &campaignID, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	masterIDInt, err := strconv.ParseInt(masterID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid masterID: %w", err)
	}
	metadata := &commonpb.Metadata{}
	if err := protojson.Unmarshal([]byte(meta), metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	var tagList, mediaList []string
	if tags != "" {
		tagList = strings.Split(tags, ",")
	}
	if media != "" {
		mediaList = strings.Split(media, ",")
	}
	return &contentpb.Content{
		Id:         id,
		MasterId:   masterIDInt,
		AuthorId:   authorID,
		Type:       ctype,
		Title:      title,
		Body:       body,
		MediaUrls:  mediaList,
		Metadata:   metadata,
		Tags:       tagList,
		ParentId:   parentID,
		Visibility: visibility,
		CampaignId: campaignID,
		CreatedAt:  createdAt.Unix(),
		UpdatedAt:  updatedAt.Unix(),
	}, nil
}

// Helper to scan a content row with counters.
func scanContentWithCounters(row interface {
	Scan(dest ...interface{}) error
},
) (*contentpb.Content, error) {
	var id, masterID, authorID, ctype, title, body, media, meta, tags, parentID, visibility string
	var campaignID int64
	var createdAt, updatedAt time.Time
	var commentCount int
	var reactionCountsRaw []byte
	if err := row.Scan(&id, &masterID, &authorID, &ctype, &title, &body, &media, &meta, &tags, &parentID, &visibility, &campaignID, &createdAt, &updatedAt, &commentCount, &reactionCountsRaw); err != nil {
		return nil, err
	}
	masterIDInt, err := strconv.ParseInt(masterID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid masterID: %w", err)
	}
	metadata := &commonpb.Metadata{}
	if err := protojson.Unmarshal([]byte(meta), metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	var tagList, mediaList []string
	if tags != "" {
		tagList = strings.Split(tags, ",")
	}
	if media != "" {
		mediaList = strings.Split(media, ",")
	}
	var reactionCounts map[string]int
	if err := json.Unmarshal(reactionCountsRaw, &reactionCounts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reaction_counts: %w", err)
	}
	// Convert to map[string]int32 for proto
	reactionCounts32 := make(map[string]int32, len(reactionCounts))
	for k, v := range reactionCounts {
		if v > int(^int32(0)) || v < int(^int32(0))*-1 {
			return nil, fmt.Errorf("reaction count overflow for type %s", k)
		}
		reactionCounts32[k] = int32(v)
	}
	if commentCount > int(^int32(0)) || commentCount < int(^int32(0))*-1 {
		return nil, fmt.Errorf("comment count overflow")
	}
	return &contentpb.Content{
		Id:             id,
		MasterId:       masterIDInt,
		AuthorId:       authorID,
		Type:           ctype,
		Title:          title,
		Body:           body,
		MediaUrls:      mediaList,
		Metadata:       metadata,
		Tags:           tagList,
		ParentId:       parentID,
		Visibility:     visibility,
		CampaignId:     campaignID,
		CreatedAt:      createdAt.Unix(),
		UpdatedAt:      updatedAt.Unix(),
		CommentCount:   int32(commentCount),
		ReactionCounts: reactionCounts32,
	}, nil
}

// AddComment adds a comment to content.
func (r *Repository) AddComment(ctx context.Context, contentID, authorID, body string, metadata *commonpb.Metadata) (*contentpb.Comment, error) {
	meta, err := metadatautil.MarshalCanonical(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	var id string
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO service_content_comment (content_id, author_id, body, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING id
	`, contentID, authorID, body, meta).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetComment(ctx, id)
}

// ListComments lists comments for a content item.
func (r *Repository) ListComments(ctx context.Context, contentID string, page, pageSize int) ([]*contentpb.Comment, int, error) {
	if page < 0 {
		page = 0
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := page * pageSize
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, content_id, author_id, body, metadata, created_at, updated_at FROM service_content_comment WHERE content_id = $1 ORDER BY created_at ASC LIMIT $2 OFFSET $3
	`, contentID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	results := make([]*contentpb.Comment, 0, pageSize)
	for rows.Next() {
		var id, contentID, authorID, body string
		var metaRaw []byte
		var createdAt, updatedAt time.Time
		meta := &commonpb.Metadata{}
		if err := rows.Scan(&id, &contentID, &authorID, &body, &metaRaw, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}
		if len(metaRaw) > 0 {
			if err := protojson.Unmarshal(metaRaw, meta); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}
		results = append(results, &contentpb.Comment{
			Id:        id,
			ContentId: contentID,
			AuthorId:  authorID,
			Body:      body,
			Metadata:  meta,
			CreatedAt: createdAt.Unix(),
			UpdatedAt: updatedAt.Unix(),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM service_content_comment WHERE content_id = $1`, contentID).Scan(&total); err != nil {
		total = len(results)
	}
	return results, total, nil
}

// GetComment fetches a single comment by ID.
func (r *Repository) GetComment(ctx context.Context, commentID string) (*contentpb.Comment, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, content_id, author_id, body, metadata, created_at, updated_at FROM service_content_comment WHERE id = $1`, commentID)
	var id, contentID, authorID, body string
	var metaRaw []byte
	var createdAt, updatedAt time.Time
	meta := &commonpb.Metadata{}
	if err := row.Scan(&id, &contentID, &authorID, &body, &metaRaw, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	if len(metaRaw) > 0 {
		if err := protojson.Unmarshal(metaRaw, meta); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	return &contentpb.Comment{
		Id:        id,
		ContentId: contentID,
		AuthorId:  authorID,
		Body:      body,
		Metadata:  meta,
		CreatedAt: createdAt.Unix(),
		UpdatedAt: updatedAt.Unix(),
	}, nil
}

// DeleteComment deletes a comment by ID.
func (r *Repository) DeleteComment(ctx context.Context, commentID string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM service_content_comment WHERE id = $1`, commentID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// LogContentEvent logs a content event for analytics/audit.
func (r *Repository) LogContentEvent(ctx context.Context, event *contentpb.ContentEvent) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO content_events (content_id, master_id, event_type, user_id, occurred_at, payload)
		VALUES ($1, $2, $3, $4, to_timestamp($5), $6)
	`, event.ContentId, event.MasterId, event.EventType, event.UserId, event.OccurredAt, payload)
	return err
}

// Moderation stubs.
func (r *Repository) ModerateContent(ctx context.Context, contentID, moderatorID, status, reason string) error {
	// This implementation assumes a `moderation_status` column exists on `service_content_main`.
	// It also enriches the content's metadata with moderation details.
	_, err := r.db.ExecContext(ctx, `
		UPDATE service_content_main
		SET
			moderation_status = $1,
			metadata = jsonb_set(
				jsonb_set(COALESCE(metadata, '{}'::jsonb), '{moderation,status}', to_jsonb($1::text)),
				'{moderation,reason}', to_jsonb($2::text)
			),
			updated_at = NOW()
		WHERE id = $3
	`, status, reason, contentID)
	if err != nil {
		return fmt.Errorf("failed to update content moderation status: %w", err)
	}
	return nil
}

// Flexible ListContent with filters.
func (r *Repository) ListContentFlexible(ctx context.Context, req *contentpb.ListContentRequest) ([]*contentpb.Content, int, error) {
	args := []interface{}{}
	where := []string{}
	argIdx := 1
	if req.AuthorId != "" {
		where = append(where, fmt.Sprintf("author_id = $%d", argIdx))
		args = append(args, req.AuthorId)
		argIdx++
	}
	if req.Type != "" {
		where = append(where, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, req.Type)
		argIdx++
	}
	if req.ParentId != "" {
		where = append(where, fmt.Sprintf("parent_id = $%d", argIdx))
		args = append(args, req.ParentId)
		argIdx++
	}
	if req.Visibility != "" {
		where = append(where, fmt.Sprintf("visibility = $%d", argIdx))
		args = append(args, req.Visibility)
		argIdx++
	}
	if len(req.Tags) > 0 {
		where = append(where, fmt.Sprintf("tags && $%d", argIdx))
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
	if req.SearchQuery != "" {
		where = append(where, fmt.Sprintf("search_vector @@ plainto_tsquery('english', $%d)", argIdx))
		args = append(args, req.SearchQuery)
		argIdx++
	}
	if req.Page < 0 {
		req.Page = 0
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	offset := int(req.Page) * int(req.PageSize)
	args = append(args, req.PageSize, offset)
	baseQuery := "SELECT id, master_id, author_id, type, title, body, media_urls, metadata, tags, parent_id, visibility, created_at, updated_at, comment_count, reaction_counts FROM service_content_main"
	if len(where) > 0 {
		baseQuery += " WHERE " + strings.Join(where, " AND ")
	}
	baseQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	results := make([]*contentpb.Content, 0, req.PageSize)
	for rows.Next() {
		c, err := scanContentWithCounters(rows)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	countQuery := "SELECT COUNT(*) FROM service_content_main"
	if len(where) > 0 {
		countQuery += " WHERE " + strings.Join(where, " AND ")
	}
	countArgs := args[:len(args)-2]
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		total = len(results)
	}
	return results, total, nil
}

// SearchContent with flexible filters.
func (r *Repository) SearchContentFlexible(ctx context.Context, req *contentpb.SearchContentRequest) ([]*contentpb.Content, int, error) {
	args := []interface{}{}
	where := []string{}
	argIdx := 1
	if req.Query != "" {
		where = append(where, fmt.Sprintf("search_vector @@ plainto_tsquery('english', $%d)", argIdx))
		args = append(args, req.Query)
		argIdx++
	}
	if len(req.Tags) > 0 {
		where = append(where, fmt.Sprintf("tags && $%d", argIdx))
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
	if req.Page < 0 {
		req.Page = 0
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	offset := int(req.Page) * int(req.PageSize)
	args = append(args, req.PageSize, offset)
	baseQuery := "SELECT id, master_id, author_id, type, title, body, media_urls, metadata, tags, parent_id, visibility, created_at, updated_at, comment_count, reaction_counts FROM service_content_main"
	if len(where) > 0 {
		baseQuery += " WHERE " + strings.Join(where, " AND ")
	}
	baseQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	results := make([]*contentpb.Content, 0, req.PageSize)
	for rows.Next() {
		c, err := scanContentWithCounters(rows)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	countQuery := "SELECT COUNT(*) FROM service_content_main"
	if len(where) > 0 {
		countQuery += " WHERE " + strings.Join(where, " AND ")
	}
	countArgs := args[:len(args)-2]
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		total = len(results)
	}
	return results, total, nil
}
