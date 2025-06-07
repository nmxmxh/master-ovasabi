package localization

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	repo "github.com/nmxmxh/master-ovasabi/internal/repository"
	metadatautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
)

var (
	ErrTranslationNotFound = errors.New("translation not found")
	ErrPricingRuleNotFound = errors.New("pricing rule not found")
	ErrLocaleNotFound      = errors.New("locale not found")
)

// Repository handles translations, pricing rules, and locale metadata for the unified LocalizationService.
type Repository struct {
	db         *sql.DB
	masterRepo repo.MasterRepository
	log        *zap.Logger
}

func NewRepository(db *sql.DB, masterRepo repo.MasterRepository) *Repository {
	return &Repository{db: db, masterRepo: masterRepo}
}

// Translate returns a translation for a given key and locale.
func (r *Repository) Translate(ctx context.Context, key, locale string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx,
		`SELECT value FROM service_i18n WHERE key = $1 AND locale = $2 ORDER BY updated_at DESC LIMIT 1`,
		key, locale,
	).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrTranslationNotFound
		}
		return "", err
	}
	return value, nil
}

// BatchTranslate returns translations for multiple keys in a given locale.
// Returns:
//   - result: map of key to value for found translations
//   - missing: slice of keys that were not found
//   - firstErr: the first error encountered (if any), or nil if all succeeded
func (r *Repository) BatchTranslate(ctx context.Context, keys []string, locale string) (result map[string]string, missing []string, firstErr error) {
	if len(keys) == 0 {
		return map[string]string{}, nil, nil
	}
	query := `SELECT key, value FROM service_i18n WHERE locale = $1 AND key = ANY($2)`
	rows, err := r.db.QueryContext(ctx, query, locale, pq.Array(keys))
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	result = make(map[string]string)
	found := make(map[string]struct{})
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			if r.log != nil {
				r.log.Warn("Failed to scan translation in batch", zap.String("key", k), zap.Error(err))
			}
			continue
		}
		result[k] = v
		found[k] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		if firstErr == nil {
			firstErr = err
		}
		if r.log != nil {
			r.log.Warn("Error after iterating translation rows", zap.Error(err))
		}
	}
	for _, k := range keys {
		if _, ok := found[k]; !ok {
			missing = append(missing, k)
		}
	}
	return result, missing, firstErr
}

// CreateTranslation creates a new translation entry.
func (r *Repository) CreateTranslation(ctx context.Context, key, language, value, masterID, masterUUID string, metadata *commonpb.Metadata, campaignID int64) (string, error) {
	meta, err := metadatautil.MarshalCanonical(metadata)
	if err != nil {
		return "", err
	}
	var id int64
	err = r.db.QueryRowContext(ctx,
		`INSERT INTO service_i18n (master_id, master_uuid, key, locale, value, metadata, campaign_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW()) RETURNING id`,
		masterID, masterUUID, key, language, value, meta, campaignID,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(id), nil
}

// GetTranslation retrieves a translation by ID.
func (r *Repository) GetTranslation(ctx context.Context, translationID string) (*Translation, error) {
	var t Translation
	var metaRaw []byte
	row := r.db.QueryRowContext(ctx,
		`SELECT id, master_id, master_uuid, key, locale, value, metadata, created_at, campaign_id FROM service_i18n WHERE id = $1`, translationID)
	if err := row.Scan(&t.ID, &t.MasterID, &t.MasterUUID, &t.Key, &t.Language, &t.Value, &metaRaw, &t.CreatedAt, &t.CampaignID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTranslationNotFound
		}
		return nil, err
	}
	var meta commonpb.Metadata
	if err := metadatautil.UnmarshalCanonical(metaRaw, &meta); err != nil {
		meta = commonpb.Metadata{}
	}
	t.Metadata = &meta
	return &t, nil
}

// ListTranslations lists translations for a language with pagination.
func (r *Repository) ListTranslations(ctx context.Context, language string, page, pageSize int, campaignID int64) ([]*Translation, int, error) {
	if page < 0 {
		page = 0
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := page * pageSize
	query := `SELECT id, master_id, master_uuid, key, locale, value, metadata, created_at, campaign_id FROM service_i18n WHERE locale = $1 AND campaign_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.db.QueryContext(ctx, query, language, campaignID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var trs []*Translation
	for rows.Next() {
		var t Translation
		var metaRaw []byte
		if err := rows.Scan(&t.ID, &t.MasterID, &t.MasterUUID, &t.Key, &t.Language, &t.Value, &metaRaw, &t.CreatedAt, &t.CampaignID); err != nil {
			return nil, 0, err
		}
		var meta commonpb.Metadata
		if err := metadatautil.UnmarshalCanonical(metaRaw, &meta); err != nil {
			meta = commonpb.Metadata{}
		}
		t.Metadata = &meta
		trs = append(trs, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM service_i18n WHERE locale = $1 AND campaign_id = $2`, language, campaignID).Scan(&total)
	if err != nil {
		total = len(trs)
	}
	return trs, total, nil
}

// GetPricingRule retrieves a pricing rule for a location.
func (r *Repository) GetPricingRule(ctx context.Context, country, region, city string) (*PricingRule, error) {
	var rule PricingRule
	row := r.db.QueryRowContext(ctx,
		`SELECT id, country_code, region, city, currency_code, affluence_tier, demand_level, multiplier, base_price, effective_from, effective_to, notes, metadata, created_at, updated_at
		 FROM service_pricing_rule
		 WHERE country_code = $1 AND (region = $2 OR $2 IS NULL) AND (city = $3 OR $3 IS NULL)
		 ORDER BY effective_from DESC, updated_at DESC LIMIT 1`,
		country, region, city)
	var metaRaw []byte
	if err := row.Scan(&rule.ID, &rule.CountryCode, &rule.Region, &rule.City, &rule.CurrencyCode, &rule.AffluenceTier, &rule.DemandLevel, &rule.Multiplier, &rule.BasePrice, &rule.EffectiveFrom, &rule.EffectiveTo, &rule.Notes, &metaRaw, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPricingRuleNotFound
		}
		return nil, err
	}
	var meta commonpb.Metadata
	if err := metadatautil.UnmarshalCanonical(metaRaw, &meta); err != nil {
		meta = commonpb.Metadata{}
	}
	rule.Metadata = &meta
	return &rule, nil
}

// SetPricingRule creates or updates a pricing rule.
func (r *Repository) SetPricingRule(ctx context.Context, rule *PricingRule) error {
	meta, err := metadatautil.MarshalCanonical(rule.Metadata)
	if err != nil {
		return err
	}
	// Upsert by country, region, city, and effective_from
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO service_pricing_rule (country_code, region, city, currency_code, affluence_tier, demand_level, multiplier, base_price, effective_from, effective_to, notes, metadata, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW(),NOW())
		 ON CONFLICT (country_code, region, city, effective_from) DO UPDATE SET
		  currency_code=EXCLUDED.currency_code, affluence_tier=EXCLUDED.affluence_tier, demand_level=EXCLUDED.demand_level, multiplier=EXCLUDED.multiplier, base_price=EXCLUDED.base_price, effective_to=EXCLUDED.effective_to, notes=EXCLUDED.notes, metadata=EXCLUDED.metadata, updated_at=NOW()`,
		rule.CountryCode, rule.Region, rule.City, rule.CurrencyCode, rule.AffluenceTier, rule.DemandLevel, rule.Multiplier, rule.BasePrice, rule.EffectiveFrom, rule.EffectiveTo, rule.Notes, meta)
	return err
}

// ListPricingRules lists pricing rules for a country/region with pagination.
func (r *Repository) ListPricingRules(ctx context.Context, country, region string, page, pageSize int) ([]*PricingRule, int, error) {
	if page < 0 {
		page = 0
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := page * pageSize
	query := `SELECT id, country_code, region, city, currency_code, affluence_tier, demand_level, multiplier, base_price, effective_from, effective_to, notes, metadata, created_at, updated_at
		FROM service_pricing_rule WHERE country_code = $1 AND ($2 IS NULL OR region = $2)
		ORDER BY effective_from DESC, updated_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.db.QueryContext(ctx, query, country, region, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var rules []*PricingRule
	for rows.Next() {
		var rule PricingRule
		var metaRaw []byte
		if err := rows.Scan(&rule.ID, &rule.CountryCode, &rule.Region, &rule.City, &rule.CurrencyCode, &rule.AffluenceTier, &rule.DemandLevel, &rule.Multiplier, &rule.BasePrice, &rule.EffectiveFrom, &rule.EffectiveTo, &rule.Notes, &metaRaw, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, 0, err
		}
		var meta commonpb.Metadata
		if err := metadatautil.UnmarshalCanonical(metaRaw, &meta); err != nil {
			meta = commonpb.Metadata{}
		}
		rule.Metadata = &meta
		rules = append(rules, &rule)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM service_pricing_rule WHERE country_code = $1 AND ($2 IS NULL OR region = $2)`, country, region).Scan(&total)
	if err != nil {
		total = len(rules)
	}
	return rules, total, nil
}

// ListLocales returns all supported locales.
func (r *Repository) ListLocales(ctx context.Context) ([]*Locale, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT code, language, country, currency, regions, metadata, created_at, updated_at FROM service_locale`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var locales []*Locale
	for rows.Next() {
		var l Locale
		var metaRaw []byte
		if err := rows.Scan(&l.Code, &l.Language, &l.Country, &l.Currency, pq.Array(&l.Regions), &metaRaw, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		var meta commonpb.Metadata
		if err := metadatautil.UnmarshalCanonical(metaRaw, &meta); err != nil {
			meta = commonpb.Metadata{}
		}
		l.Metadata = &meta
		locales = append(locales, &l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return locales, nil
}

// GetLocaleMetadata returns metadata for a locale.
func (r *Repository) GetLocaleMetadata(ctx context.Context, locale string) (*Locale, error) {
	var l Locale
	var metaRaw []byte
	row := r.db.QueryRowContext(ctx, `SELECT code, language, country, currency, regions, metadata, created_at, updated_at FROM service_locale WHERE code = $1`, locale)
	if err := row.Scan(&l.Code, &l.Language, &l.Country, &l.Currency, pq.Array(&l.Regions), &metaRaw, &l.CreatedAt, &l.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrLocaleNotFound
		}
		return nil, err
	}
	var meta commonpb.Metadata
	if err := metadatautil.UnmarshalCanonical(metaRaw, &meta); err != nil {
		meta = commonpb.Metadata{}
	}
	l.Metadata = &meta
	return &l, nil
}

// --- Data Models ---.
type Translation struct {
	ID         string             `db:"id"`
	MasterID   int64              `db:"master_id"`
	MasterUUID string             `db:"master_uuid"`
	Key        string             `db:"key"`
	Language   string             `db:"locale"`
	Value      string             `db:"value"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	CreatedAt  time.Time          `db:"created_at"`
	CampaignID int64              `db:"campaign_id"`
}

type PricingRule struct {
	ID            string
	CountryCode   string
	Region        string
	City          string
	CurrencyCode  string
	AffluenceTier string
	DemandLevel   string
	Multiplier    float64
	BasePrice     float64
	EffectiveFrom time.Time
	EffectiveTo   time.Time
	Notes         string
	Metadata      *commonpb.Metadata
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Locale struct {
	Code      string
	Language  string
	Country   string
	Currency  string
	Regions   []string
	Metadata  *commonpb.Metadata
	CreatedAt time.Time
	UpdatedAt time.Time
}

// --- Translation CRUD ---.
func (r *Repository) UpdateTranslation(ctx context.Context, id, value string, metadata *commonpb.Metadata) error {
	meta, err := metadatautil.MarshalCanonical(metadata)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE service_i18n SET value = $1, metadata = $2, updated_at = NOW() WHERE id = $3`,
		value, meta, id)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) DeleteTranslation(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM service_i18n WHERE id = $1`, id)
	return err
}

// --- PricingRule CRUD ---.
func (r *Repository) CreatePricingRule(ctx context.Context, rule *PricingRule) (int64, error) {
	meta, err := metadatautil.MarshalCanonical(rule.Metadata)
	if err != nil {
		return 0, err
	}
	var id int64
	err = r.db.QueryRowContext(ctx,
		`INSERT INTO service_pricing_rule (country_code, region, city, currency_code, affluence_tier, demand_level, multiplier, base_price, effective_from, effective_to, notes, metadata, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW(),NOW()) RETURNING id`,
		rule.CountryCode, rule.Region, rule.City, rule.CurrencyCode, rule.AffluenceTier, rule.DemandLevel, rule.Multiplier, rule.BasePrice, rule.EffectiveFrom, rule.EffectiveTo, rule.Notes, meta).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) UpdatePricingRule(ctx context.Context, id int64, rule *PricingRule) error {
	meta, err := metadatautil.MarshalCanonical(rule.Metadata)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE service_pricing_rule SET country_code=$1, region=$2, city=$3, currency_code=$4, affluence_tier=$5, demand_level=$6, multiplier=$7, base_price=$8, effective_from=$9, effective_to=$10, notes=$11, metadata=$12, updated_at=NOW() WHERE id = $13`,
		rule.CountryCode, rule.Region, rule.City, rule.CurrencyCode, rule.AffluenceTier, rule.DemandLevel, rule.Multiplier, rule.BasePrice, rule.EffectiveFrom, rule.EffectiveTo, rule.Notes, meta, id)
	return err
}

func (r *Repository) DeletePricingRule(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM service_pricing_rule WHERE id = $1`, id)
	return err
}

// --- Locale CRUD ---.
func (r *Repository) CreateLocale(ctx context.Context, locale *Locale) error {
	meta, err := metadatautil.MarshalCanonical(locale.Metadata)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO service_locale (code, language, country, currency, regions, metadata, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,NOW(),NOW())`,
		locale.Code, locale.Language, locale.Country, locale.Currency, pq.Array(locale.Regions), meta)
	return err
}

func (r *Repository) UpdateLocale(ctx context.Context, code string, locale *Locale) error {
	meta, err := metadatautil.MarshalCanonical(locale.Metadata)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE service_locale SET language=$1, country=$2, currency=$3, regions=$4, metadata=$5, updated_at=NOW() WHERE code = $6`,
		locale.Language, locale.Country, locale.Currency, pq.Array(locale.Regions), meta, code)
	return err
}

func (r *Repository) DeleteLocale(ctx context.Context, code string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM service_locale WHERE code = $1`, code)
	return err
}

type Localization struct {
	ID         string             `db:"id"`
	MasterID   int64              `db:"master_id"`
	MasterUUID string             `db:"master_uuid"`
	Locale     string             `db:"locale"`
	ContentID  string             `db:"content_id"`
	Status     int16              `db:"status"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	CreatedAt  time.Time          `db:"created_at"`
	UpdatedAt  time.Time          `db:"updated_at"`
}
