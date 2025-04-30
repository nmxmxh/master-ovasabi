package i18n

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/repository"
)

var (
	ErrTranslationNotFound = errors.New("translation not found")
	ErrTranslationExists   = errors.New("translation already exists")
)

// Translation represents a translation record for i18n
// (move from shared repository types if needed)
type Translation struct {
	ID          int64     `db:"id"`
	MasterID    int64     `db:"master_id"`
	Key         string    `db:"key"`
	Locale      string    `db:"locale"`
	Value       string    `db:"value"`
	Description string    `db:"description"`
	Tags        string    `db:"tags"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// Repository handles operations on the service_i18n table
type Repository struct {
	*repository.BaseRepository
	masterRepo repository.MasterRepository
}

// NewRepository creates a new i18n repository instance
func NewRepository(db *sql.DB, masterRepo repository.MasterRepository) *Repository {
	return &Repository{
		BaseRepository: repository.NewBaseRepository(db),
		masterRepo:     masterRepo,
	}
}

// Create inserts a new translation record
func (r *Repository) Create(ctx context.Context, translation *Translation) (*Translation, error) {
	// Generate a descriptive name for the master record
	masterName := r.GenerateMasterName(repository.EntityTypeI18n,
		translation.Key,
		translation.Locale)

	masterID, err := r.masterRepo.Create(ctx, repository.EntityTypeI18n, masterName)
	if err != nil {
		return nil, err
	}

	translation.MasterID = masterID
	err = r.GetDB().QueryRowContext(ctx,
		`INSERT INTO service_i18n (master_id, key, locale, value, description, tags, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		 RETURNING id, created_at, updated_at`,
		translation.MasterID, translation.Key, translation.Locale, translation.Value, translation.Description, translation.Tags,
	).Scan(&translation.ID, &translation.CreatedAt, &translation.UpdatedAt)

	if err != nil {
		_ = r.masterRepo.Delete(ctx, masterID)
		return nil, err
	}

	return translation, nil
}

// GetByID retrieves a translation by ID
func (r *Repository) GetByID(ctx context.Context, id int64) (*Translation, error) {
	translation := &Translation{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT id, master_id, key, locale, value, description, tags, created_at, updated_at
		 FROM service_i18n WHERE id = $1`,
		id,
	).Scan(&translation.ID, &translation.MasterID, &translation.Key, &translation.Locale, &translation.Value, &translation.Description, &translation.Tags, &translation.CreatedAt, &translation.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTranslationNotFound
		}
		return nil, err
	}
	return translation, nil
}

// GetByKeyAndLocale retrieves a translation by key and locale
func (r *Repository) GetByKeyAndLocale(ctx context.Context, key, locale string) (*Translation, error) {
	translation := &Translation{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT id, master_id, key, locale, value, description, tags, created_at, updated_at
		 FROM service_i18n WHERE key = $1 AND locale = $2`,
		key, locale,
	).Scan(&translation.ID, &translation.MasterID, &translation.Key, &translation.Locale, &translation.Value, &translation.Description, &translation.Tags, &translation.CreatedAt, &translation.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTranslationNotFound
		}
		return nil, err
	}
	return translation, nil
}

// Update updates a translation record
func (r *Repository) Update(ctx context.Context, translation *Translation) error {
	result, err := r.GetDB().ExecContext(ctx,
		`UPDATE service_i18n SET value = $1, description = $2, tags = $3, updated_at = NOW() WHERE id = $4`,
		translation.Value, translation.Description, translation.Tags, translation.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTranslationNotFound
	}
	return nil
}

// Delete removes a translation and its master record
func (r *Repository) Delete(ctx context.Context, id int64) error {
	translation, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return r.masterRepo.Delete(ctx, translation.MasterID)
}

// List retrieves a paginated list of translations
func (r *Repository) List(ctx context.Context, limit, offset int) ([]*Translation, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT id, master_id, key, locale, value, description, tags, created_at, updated_at
		 FROM service_i18n ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var translations []*Translation
	for rows.Next() {
		translation := &Translation{}
		err := rows.Scan(&translation.ID, &translation.MasterID, &translation.Key, &translation.Locale, &translation.Value, &translation.Description, &translation.Tags, &translation.CreatedAt, &translation.UpdatedAt)
		if err != nil {
			return nil, err
		}
		translations = append(translations, translation)
	}
	return translations, nil
}

// ListByLocale retrieves all translations for a specific locale
func (r *Repository) ListByLocale(ctx context.Context, locale string, limit, offset int) ([]*Translation, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT id, master_id, key, locale, value, description, tags, created_at, updated_at
		 FROM service_i18n WHERE locale = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		locale, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var translations []*Translation
	for rows.Next() {
		translation := &Translation{}
		err := rows.Scan(&translation.ID, &translation.MasterID, &translation.Key, &translation.Locale, &translation.Value, &translation.Description, &translation.Tags, &translation.CreatedAt, &translation.UpdatedAt)
		if err != nil {
			return nil, err
		}
		translations = append(translations, translation)
	}
	return translations, nil
}
