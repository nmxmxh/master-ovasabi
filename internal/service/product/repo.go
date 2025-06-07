package product

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	productpb "github.com/nmxmxh/master-ovasabi/api/protos/product/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
)

type RepositoryItf interface {
	CreateProduct(ctx context.Context, product *productpb.Product) (*productpb.Product, error)
	UpdateProduct(ctx context.Context, product *productpb.Product) (*productpb.Product, error)
	DeleteProduct(ctx context.Context, productID string) error
	GetProduct(ctx context.Context, productID string) (*productpb.Product, error)
	ListProducts(ctx context.Context, filter ListProductsFilter) ([]*productpb.Product, int, error)
	SearchProducts(ctx context.Context, filter SearchProductsFilter) ([]*productpb.Product, int, error)
	UpdateInventory(ctx context.Context, variantID string, delta int32) (*productpb.ProductVariant, error)
	ListProductVariants(ctx context.Context, productID string) ([]*productpb.ProductVariant, error)
}

type ListProductsFilter struct {
	OwnerID         string
	Type            productpb.ProductType
	Status          productpb.ProductStatus
	Tags            []string
	Page            int
	PageSize        int
	MasterID        string
	MasterUUID      string
	CampaignID      int64
	MetadataFilters map[string]interface{}
}

type SearchProductsFilter struct {
	Query      string
	Tags       []string
	Type       productpb.ProductType
	Status     productpb.ProductStatus
	Page       int
	PageSize   int
	MasterID   string
	MasterUUID string
	CampaignID int64
}

// Model uses Go naming conventions for DB operations.
type Model struct {
	ID          string
	MasterID    int64
	MasterUUID  string
	Name        string
	Description string
	Type        int32
	Status      int32
	Tags        string
	Metadata    *commonpb.Metadata
	CampaignID  int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Product struct {
	ID          string                  `db:"id"`
	MasterID    int64                   `db:"master_id"`
	MasterUUID  string                  `db:"master_uuid"`
	Name        string                  `db:"name"`
	Description string                  `db:"description"`
	Type        productpb.ProductType   `db:"type"`
	Status      productpb.ProductStatus `db:"status"`
	Tags        []string                `db:"tags"`
	Metadata    *commonpb.Metadata      `db:"metadata"`
	CampaignID  int64                   `db:"campaign_id"`
	CreatedAt   time.Time               `db:"created_at"`
	UpdatedAt   time.Time               `db:"updated_at"`
}

type Repository struct {
	db         *sql.DB
	masterRepo repository.MasterRepository
}

func NewRepository(db *sql.DB, masterRepo repository.MasterRepository) *Repository {
	return &Repository{db: db, masterRepo: masterRepo}
}

func (r *Repository) GetDB() *sql.DB {
	return r.db
}

func (r *Repository) CreateProduct(ctx context.Context, p *productpb.Product) (*productpb.Product, error) {
	tags := strings.Join(p.Tags, ",")
	var id string
	model := Model{
		MasterID:    p.MasterId,
		MasterUUID:  p.MasterUuid,
		Name:        p.Name,
		Description: p.Description,
		Type:        int32(p.Type),
		Status:      int32(p.Status),
		Tags:        tags,
		Metadata:    p.Metadata,
		CampaignID:  p.CampaignId,
	}
	err := r.GetDB().QueryRowContext(ctx, `
		INSERT INTO service_product_main (master_id, master_uuid, name, description, type, status, tags, metadata, campaign_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW())
		RETURNING id
	`, model.MasterID, model.MasterUUID, model.Name, model.Description, model.Type, model.Status, model.Tags, model.Metadata, model.CampaignID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetProduct(ctx, id)
}

func (r *Repository) GetProduct(ctx context.Context, id string) (*productpb.Product, error) {
	row := r.GetDB().QueryRowContext(ctx, `SELECT id, master_id, master_uuid, name, description, type, status, tags, metadata, campaign_id, created_at, updated_at FROM service_product_main WHERE id = $1`, id)
	return scanProduct(row)
}

func (r *Repository) UpdateProduct(ctx context.Context, p *productpb.Product) (*productpb.Product, error) {
	tags := strings.Join(p.Tags, ",")
	_, err := r.GetDB().ExecContext(ctx, `
		UPDATE service_product_main SET name=$1, description=$2, type=$3, status=$4, tags=$5, metadata=$6, campaign_id=$7, updated_at=NOW() WHERE id=$8
	`, p.Name, p.Description, p.Type, p.Status, tags, p.Metadata, p.CampaignId, p.Id)
	if err != nil {
		return nil, err
	}
	return r.GetProduct(ctx, p.Id)
}

func (r *Repository) DeleteProduct(ctx context.Context, id string) error {
	res, err := r.GetDB().ExecContext(ctx, `DELETE FROM service_product_main WHERE id = $1`, id)
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

func (r *Repository) ListProducts(ctx context.Context, filter ListProductsFilter) ([]*productpb.Product, int, error) {
	args := []interface{}{}
	where := []string{}
	if filter.OwnerID != "" {
		where = append(where, "owner_id = ?")
		args = append(args, filter.OwnerID)
	}
	if filter.Type != 0 {
		where = append(where, "type = ?")
		args = append(args, filter.Type)
	}
	if filter.Status != 0 {
		where = append(where, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.MasterID != "" {
		where = append(where, "master_id = ?")
		args = append(args, filter.MasterID)
	}
	if filter.MasterUUID != "" {
		where = append(where, "master_uuid = ?")
		args = append(args, filter.MasterUUID)
	}
	if filter.CampaignID != 0 {
		where = append(where, "campaign_id = ?")
		args = append(args, filter.CampaignID)
	}
	if filter.MetadataFilters != nil {
		for path, value := range filter.MetadataFilters {
			pgPath := "{" + strings.ReplaceAll(path, ".", ",") + "}"
			where = append(where, fmt.Sprintf("metadata->'service_specific' #> '%s' = ?", pgPath))
			args = append(args, value)
		}
	}
	if filter.Page < 0 {
		filter.Page = 0
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	offset := filter.Page * filter.PageSize
	args = append(args, filter.PageSize, offset)
	baseQuery := "SELECT id, master_id, master_uuid, name, description, type, status, tags, metadata, campaign_id, created_at, updated_at FROM service_product_main"
	if len(where) > 0 {
		baseQuery += " WHERE " + strings.Join(where, " AND ")
	}
	baseQuery += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	for i := 1; i <= len(args); i++ {
		baseQuery = strings.Replace(baseQuery, "?", "$"+fmt.Sprintf("%d", i), 1)
	}
	rows, err := r.GetDB().QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	results := make([]*productpb.Product, 0, filter.PageSize)
	for rows.Next() {
		p, err := scanProduct(rows)
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

func (r *Repository) SearchProducts(ctx context.Context, filter SearchProductsFilter) ([]*productpb.Product, int, error) {
	args := []interface{}{filter.Query}
	where := []string{"search_vector @@ plainto_tsquery('english', $1)"}
	argIdx := 2
	if filter.MasterID != "" {
		where = append(where, fmt.Sprintf("master_id = $%d", argIdx))
		args = append(args, filter.MasterID)
		argIdx++
	}
	if filter.MasterUUID != "" {
		where = append(where, fmt.Sprintf("master_uuid = $%d", argIdx))
		args = append(args, filter.MasterUUID)
		argIdx++
	}
	if filter.CampaignID != 0 {
		where = append(where, fmt.Sprintf("campaign_id = $%d", argIdx))
		args = append(args, filter.CampaignID)
		argIdx++
	}
	args = append(args, filter.PageSize, (filter.Page * filter.PageSize))
	baseQuery := "SELECT id, master_id, master_uuid, name, description, type, status, tags, metadata, campaign_id, created_at, updated_at FROM service_product_main"
	if len(where) > 0 {
		baseQuery += " WHERE " + strings.Join(where, " AND ")
	}
	baseQuery += fmt.Sprintf(" ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	rows, err := r.GetDB().QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	results := make([]*productpb.Product, 0, filter.PageSize)
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	countQuery := "SELECT COUNT(*) FROM service_product_main"
	if len(where) > 0 {
		countQuery += " WHERE " + strings.Join(where, " AND ")
	}
	countArgs := args[:len(args)-2]
	if err := r.GetDB().QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		total = len(results)
	}
	return results, total, nil
}

// Helper to scan a product row.
func scanProduct(row interface {
	Scan(dest ...interface{}) error
},
) (*productpb.Product, error) {
	var model Model
	if err := row.Scan(&model.ID, &model.MasterID, &model.MasterUUID, &model.Name, &model.Description, &model.Type, &model.Status, &model.Tags, &model.Metadata, &model.CampaignID, &model.CreatedAt, &model.UpdatedAt); err != nil {
		return nil, err
	}
	var tagList []string
	if model.Tags != "" {
		tagList = strings.Split(model.Tags, ",")
	}
	return &productpb.Product{
		Id:          model.ID,
		MasterId:    model.MasterID,
		MasterUuid:  model.MasterUUID,
		Name:        model.Name,
		Description: model.Description,
		Type:        productpb.ProductType(model.Type),
		Status:      productpb.ProductStatus(model.Status),
		Tags:        tagList,
		Metadata:    model.Metadata,
		CampaignId:  model.CampaignID,
		CreatedAt:   model.CreatedAt.Unix(),
		UpdatedAt:   model.UpdatedAt.Unix(),
	}, nil
}

func (r *Repository) UpdateInventory(_ context.Context, _ string, _ int32) (*productpb.ProductVariant, error) {
	// TODO: implement UpdateInventory logic
	return nil, errors.New("not implemented")
}

func (r *Repository) ListProductVariants(_ context.Context, _ string) ([]*productpb.ProductVariant, error) {
	// TODO: implement ListProductVariants logic
	return nil, errors.New("not implemented")
}
