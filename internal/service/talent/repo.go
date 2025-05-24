package talent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	talentpb "github.com/nmxmxh/master-ovasabi/api/protos/talent/v1"
	repo "github.com/nmxmxh/master-ovasabi/internal/repository"
)

type Repository struct {
	db         *sql.DB
	masterRepo repo.MasterRepository
}

func NewRepository(db *sql.DB, masterRepo repo.MasterRepository) *Repository {
	return &Repository{db: db, masterRepo: masterRepo}
}

type Profile struct {
	ID          string             `db:"id"`
	MasterID    int64              `db:"master_id"`
	MasterUUID  string             `db:"master_uuid"`
	UserID      string             `db:"user_id"`
	DisplayName string             `db:"display_name"`
	Bio         string             `db:"bio"`
	Skills      []string           `db:"skills"`
	Tags        []string           `db:"tags"`
	Location    string             `db:"location"`
	AvatarURL   string             `db:"avatar_url"`
	CreatedAt   time.Time          `db:"created_at"`
	UpdatedAt   time.Time          `db:"updated_at"`
	Metadata    *commonpb.Metadata `db:"metadata"`
}

func (r *Repository) CreateTalentProfile(ctx context.Context, p *talentpb.TalentProfile, campaignID int64) (*talentpb.TalentProfile, error) {
	tags := strings.Join(p.Tags, ",")
	skills := strings.Join(p.Skills, ",")
	metaJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return nil, err
	}
	var id string
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO service_talent_profile (master_id, master_uuid, user_id, display_name, bio, skills, tags, location, avatar_url, metadata, campaign_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW(),NOW())
		RETURNING id
	`, p.MasterId, p.MasterUuid, p.UserId, p.DisplayName, p.Bio, skills, tags, p.Location, p.AvatarUrl, metaJSON, campaignID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetTalentProfile(ctx, id, campaignID)
}

func (r *Repository) UpdateTalentProfile(ctx context.Context, p *talentpb.TalentProfile, campaignID int64) (*talentpb.TalentProfile, error) {
	tags := strings.Join(p.Tags, ",")
	skills := strings.Join(p.Skills, ",")
	metaJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return nil, err
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE service_talent_profile SET display_name=$1, bio=$2, skills=$3, tags=$4, location=$5, avatar_url=$6, metadata=$7, campaign_id=$8, updated_at=NOW() WHERE id=$9
	`, p.DisplayName, p.Bio, skills, tags, p.Location, p.AvatarUrl, metaJSON, campaignID, p.Id)
	if err != nil {
		return nil, err
	}
	return r.GetTalentProfile(ctx, p.Id, campaignID)
}

func (r *Repository) DeleteTalentProfile(ctx context.Context, id string, campaignID int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM service_talent_profile WHERE id = $1 AND campaign_id = $2`, id, campaignID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) GetTalentProfile(ctx context.Context, id string, campaignID int64) (*talentpb.TalentProfile, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, master_id, master_uuid, user_id, display_name, bio, skills, tags, location, avatar_url, metadata, campaign_id, created_at, updated_at FROM service_talent_profile WHERE id = $1 AND campaign_id = $2`, id, campaignID)
	return scanTalentProfile(row)
}

func (r *Repository) ListTalentProfiles(ctx context.Context, page, pageSize int, skills, tags []string, location string, campaignID int64) ([]*talentpb.TalentProfile, int, error) {
	args := []interface{}{}
	where := []string{}
	argIdx := 1
	if len(skills) > 0 {
		where = append(where, fmt.Sprintf("skills && $%d", argIdx))
		args = append(args, strings.Join(skills, ","))
		argIdx++
	}
	if len(tags) > 0 {
		where = append(where, fmt.Sprintf("tags && $%d", argIdx))
		args = append(args, strings.Join(tags, ","))
		argIdx++
	}
	if location != "" {
		where = append(where, fmt.Sprintf("location = $%d", argIdx))
		args = append(args, location)
		argIdx++
	}
	if campaignID != 0 {
		where = append(where, fmt.Sprintf("campaign_id = $%d", argIdx))
		args = append(args, campaignID)
		argIdx++
	}
	if page < 0 {
		page = 0
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := page * pageSize
	args = append(args, pageSize, offset)
	baseQuery := "SELECT id, master_id, master_uuid, user_id, display_name, bio, skills, tags, location, avatar_url, metadata, campaign_id, created_at, updated_at FROM service_talent_profile"
	if len(where) > 0 {
		baseQuery += " WHERE " + strings.Join(where, " AND ")
	}
	baseQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	results := make([]*talentpb.TalentProfile, 0, pageSize)
	for rows.Next() {
		p, err := scanTalentProfile(rows)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	total := len(results)
	return results, total, nil
}

func (r *Repository) SearchTalentProfiles(ctx context.Context, query string, page, pageSize int, skills, tags []string, location string, campaignID int64) ([]*talentpb.TalentProfile, int, error) {
	args := []interface{}{query}
	where := []string{"search_vector @@ plainto_tsquery('english', $1)"}
	argIdx := 2
	if len(skills) > 0 {
		where = append(where, fmt.Sprintf("skills && $%d", argIdx))
		args = append(args, strings.Join(skills, ","))
		argIdx++
	}
	if len(tags) > 0 {
		where = append(where, fmt.Sprintf("tags && $%d", argIdx))
		args = append(args, strings.Join(tags, ","))
		argIdx++
	}
	if location != "" {
		where = append(where, fmt.Sprintf("location = $%d", argIdx))
		args = append(args, location)
		argIdx++
	}
	if campaignID != 0 {
		where = append(where, fmt.Sprintf("campaign_id = $%d", argIdx))
		args = append(args, campaignID)
		argIdx++
	}
	args = append(args, pageSize, (page * pageSize))
	baseQuery := "SELECT id, master_id, master_uuid, user_id, display_name, bio, skills, tags, location, avatar_url, metadata, campaign_id, created_at, updated_at FROM service_talent_profile"
	if len(where) > 0 {
		baseQuery += " WHERE " + strings.Join(where, " AND ")
	}
	baseQuery += fmt.Sprintf(" ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	results := make([]*talentpb.TalentProfile, 0, pageSize)
	for rows.Next() {
		p, err := scanTalentProfile(rows)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	countQuery := "SELECT COUNT(*) FROM service_talent_profile"
	if len(where) > 0 {
		countQuery += " WHERE " + strings.Join(where, " AND ")
	}
	countArgs := args[:len(args)-2]
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		total = len(results)
	}
	return results, total, nil
}

// Helper to scan a talent profile row.
func scanTalentProfile(row interface {
	Scan(dest ...interface{}) error
},
) (*talentpb.TalentProfile, error) {
	var id string
	var masterID int64
	var masterUUID, userID, displayName, bio, skills, tags, location, avatarURL string
	var metaJSON []byte
	var createdAt, updatedAt time.Time
	if err := row.Scan(&id, &masterID, &masterUUID, &userID, &displayName, &bio, &skills, &tags, &location, &avatarURL, &metaJSON, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	var meta *commonpb.Metadata
	if len(metaJSON) > 0 {
		var m commonpb.Metadata
		if err := json.Unmarshal(metaJSON, &m); err == nil {
			meta = &m
		}
	}
	return &talentpb.TalentProfile{
		Id:          id,
		MasterId:    masterID,
		MasterUuid:  masterUUID,
		UserId:      userID,
		DisplayName: displayName,
		Bio:         bio,
		Skills:      strings.Split(skills, ","),
		Tags:        strings.Split(tags, ","),
		Location:    location,
		AvatarUrl:   avatarURL,
		CreatedAt:   createdAt.Unix(),
		UpdatedAt:   updatedAt.Unix(),
		Metadata:    meta,
	}, nil
}

func (r *Repository) BookTalent(ctx context.Context, talentID, userID string, start, end int64, notes string, campaignID int64) (*talentpb.Booking, error) {
	// Insert a new booking into the service_talent_booking table
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO service_talent_booking (talent_id, user_id, start_time, end_time, notes, campaign_id, created_at, updated_at)
		VALUES ($1, $2, to_timestamp($3), to_timestamp($4), $5, $6, NOW(), NOW())
		RETURNING id
	`, talentID, userID, start, end, notes, campaignID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetBooking(ctx, id, campaignID)
}

func (r *Repository) ListBookings(ctx context.Context, talentID string, page, pageSize int, campaignID int64) ([]*talentpb.Booking, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, talent_id, user_id, start_time, end_time, notes, campaign_id, created_at, updated_at
		FROM service_talent_booking
		WHERE talent_id = $1 AND campaign_id = $2
		ORDER BY start_time DESC
		LIMIT $3 OFFSET $4
	`, talentID, campaignID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var bookings []*talentpb.Booking
	for rows.Next() {
		var b talentpb.Booking
		var startTime, endTime, createdAt, updatedAt time.Time
		if err := rows.Scan(&b.Id, &b.TalentId, &b.UserId, &startTime, &endTime, &b.Notes, &b.CampaignId, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}
		b.StartTime = startTime.Unix()
		b.EndTime = endTime.Unix()
		b.CreatedAt = createdAt.Unix()
		bookings = append(bookings, &b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	// Count total
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM service_talent_booking WHERE talent_id = $1 AND campaign_id = $2`, talentID, campaignID).Scan(&total); err != nil {
		total = len(bookings)
	}
	return bookings, total, nil
}

func (r *Repository) GetBooking(ctx context.Context, id string, campaignID int64) (*talentpb.Booking, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, talent_id, user_id, start_time, end_time, notes, campaign_id, created_at, updated_at FROM service_talent_booking WHERE id = $1 AND campaign_id = $2`, id, campaignID)
	var b talentpb.Booking
	var startTime, endTime, createdAt, updatedAt time.Time
	if err := row.Scan(&b.Id, &b.TalentId, &b.UserId, &startTime, &endTime, &b.Notes, &b.CampaignId, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	b.StartTime = startTime.Unix()
	b.EndTime = endTime.Unix()
	b.CreatedAt = createdAt.Unix()
	return &b, nil
}
