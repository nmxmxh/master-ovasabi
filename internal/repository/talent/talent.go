package talentrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	talentpb "github.com/nmxmxh/master-ovasabi/api/protos/talent/v1"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateTalentProfile(ctx context.Context, p *talentpb.TalentProfile) (*talentpb.TalentProfile, error) {
	tags := strings.Join(p.Tags, ",")
	skills := strings.Join(p.Skills, ",")
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO service_talent_profile (master_id, user_id, display_name, bio, skills, tags, location, avatar_url, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW(),NOW())
		RETURNING id
	`, p.MasterId, p.UserId, p.DisplayName, p.Bio, skills, tags, p.Location, p.AvatarUrl).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetTalentProfile(ctx, id)
}

func (r *PostgresRepository) UpdateTalentProfile(ctx context.Context, p *talentpb.TalentProfile) (*talentpb.TalentProfile, error) {
	tags := strings.Join(p.Tags, ",")
	skills := strings.Join(p.Skills, ",")
	_, err := r.db.ExecContext(ctx, `
		UPDATE service_talent_profile SET display_name=$1, bio=$2, skills=$3, tags=$4, location=$5, avatar_url=$6, updated_at=NOW() WHERE id=$7
	`, p.DisplayName, p.Bio, skills, tags, p.Location, p.AvatarUrl, p.Id)
	if err != nil {
		return nil, err
	}
	return r.GetTalentProfile(ctx, p.Id)
}

func (r *PostgresRepository) DeleteTalentProfile(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM service_talent_profile WHERE id = $1`, id)
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

func (r *PostgresRepository) GetTalentProfile(ctx context.Context, id string) (*talentpb.TalentProfile, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, master_id, user_id, display_name, bio, skills, tags, location, avatar_url, created_at, updated_at FROM service_talent_profile WHERE id = $1`, id)
	return scanTalentProfile(row)
}

func (r *PostgresRepository) ListTalentProfiles(ctx context.Context, page, pageSize int, skills, tags []string, masterID string) ([]*talentpb.TalentProfile, int, error) {
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
	if masterID != "" {
		where = append(where, fmt.Sprintf("master_id = $%d", argIdx))
		args = append(args, masterID)
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
	baseQuery := "SELECT id, master_id, user_id, display_name, bio, skills, tags, location, avatar_url, created_at, updated_at FROM service_talent_profile"
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

func (r *PostgresRepository) SearchTalentProfiles(ctx context.Context, query string, page, pageSize int, skills, tags []string, masterID string) ([]*talentpb.TalentProfile, int, error) {
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
	if masterID != "" {
		where = append(where, fmt.Sprintf("master_id = $%d", argIdx))
		args = append(args, masterID)
		argIdx++
	}
	args = append(args, pageSize, (page * pageSize))
	baseQuery := "SELECT id, master_id, user_id, display_name, bio, skills, tags, location, avatar_url, created_at, updated_at FROM service_talent_profile"
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
	var id, masterID, userID, displayName, bio, skills, tags, location, avatarURL string
	var createdAt, updatedAt time.Time
	if err := row.Scan(&id, &masterID, &userID, &displayName, &bio, &skills, &tags, &location, &avatarURL, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	skillList := []string{}
	tagList := []string{}
	if skills != "" {
		skillList = strings.Split(skills, ",")
	}
	if tags != "" {
		tagList = strings.Split(tags, ",")
	}
	return &talentpb.TalentProfile{
		Id:          id,
		MasterId:    masterID,
		UserId:      userID,
		DisplayName: displayName,
		Bio:         bio,
		Skills:      skillList,
		Tags:        tagList,
		Location:    location,
		AvatarUrl:   avatarURL,
		CreatedAt:   createdAt.Unix(),
		UpdatedAt:   updatedAt.Unix(),
	}, nil
}

func (r *PostgresRepository) BookTalent(_ context.Context, _, _ string, _, _ int64, _ string) (*talentpb.Booking, error) {
	// TODO: implement BookTalent logic
	return nil, errors.New("not implemented")
}

func (r *PostgresRepository) ListBookings(_ context.Context, _ string, _, _ int) ([]*talentpb.Booking, int, error) {
	// TODO: implement ListBookings logic
	return nil, 0, errors.New("not implemented")
}
