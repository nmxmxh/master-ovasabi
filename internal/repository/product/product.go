package productrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	productpb "github.com/nmxmxh/master-ovasabi/api/protos/product/v1"
	metadatautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

type Repository interface {
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
	OwnerID  string
	Type     productpb.ProductType
	Status   productpb.ProductStatus
	Tags     []string
	Page     int
	PageSize int
	MasterID string
}

type SearchProductsFilter struct {
	Query    string
	Tags     []string
	Type     productpb.ProductType
	Status   productpb.ProductStatus
	Page     int
	PageSize int
	MasterID string
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetDB() *sql.DB {
	return r.db
}

func (r *PostgresRepository) CreateProduct(ctx context.Context, p *productpb.Product) (*productpb.Product, error) {
	err := metadatautil.ValidateMetadata(p.Metadata)
	if err != nil {
		return nil, err
	}
	meta, err := protojson.Marshal(p.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	tags := strings.Join(p.Tags, ",")
	var id string
	err = r.GetDB().QueryRowContext(ctx, `
		INSERT INTO service_product_main (master_id, name, description, type, status, tags, metadata, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NOW(),NOW())
		RETURNING id
	`, p.MasterId, p.Name, p.Description, p.Type, p.Status, tags, meta).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetProduct(ctx, id)
}

func (r *PostgresRepository) GetProduct(ctx context.Context, id string) (*productpb.Product, error) {
	row := r.GetDB().QueryRowContext(ctx, `SELECT id, master_id, name, description, type, status, tags, metadata, created_at, updated_at FROM service_product_main WHERE id = $1`, id)
	return scanProduct(row)
}

func (r *PostgresRepository) UpdateProduct(ctx context.Context, p *productpb.Product) (*productpb.Product, error) {
	err := metadatautil.ValidateMetadata(p.Metadata)
	if err != nil {
		return nil, err
	}
	meta, err := protojson.Marshal(p.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	tags := strings.Join(p.Tags, ",")
	_, err = r.GetDB().ExecContext(ctx, `
		UPDATE service_product_main SET name=$1, description=$2, type=$3, status=$4, tags=$5, metadata=$6, updated_at=NOW() WHERE id=$7
	`, p.Name, p.Description, p.Type, p.Status, tags, meta, p.Id)
	if err != nil {
		return nil, err
	}
	return r.GetProduct(ctx, p.Id)
}

func (r *PostgresRepository) DeleteProduct(ctx context.Context, id string) error {
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

func (r *PostgresRepository) ListProducts(ctx context.Context, filter ListProductsFilter) ([]*productpb.Product, int, error) {
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
	if filter.Page < 0 {
		filter.Page = 0
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	offset := filter.Page * filter.PageSize
	args = append(args, filter.PageSize, offset)
	baseQuery := "SELECT id, master_id, name, description, type, status, tags, metadata, created_at, updated_at FROM service_product_main"
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

func (r *PostgresRepository) SearchProducts(ctx context.Context, filter SearchProductsFilter) ([]*productpb.Product, int, error) {
	args := []interface{}{filter.Query}
	where := []string{"search_vector @@ plainto_tsquery('english', $1)"}
	argIdx := 2
	if filter.MasterID != "" {
		where = append(where, fmt.Sprintf("master_id = $%d", argIdx))
		args = append(args, filter.MasterID)
		argIdx++
	}
	args = append(args, filter.PageSize, (filter.Page * filter.PageSize))
	baseQuery := "SELECT id, master_id, name, description, type, status, tags, metadata, created_at, updated_at FROM service_product_main"
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
	var id, masterID, name, description, tags, meta string
	var ptype, status int32
	var createdAt, updatedAt time.Time
	if err := row.Scan(&id, &masterID, &name, &description, &ptype, &status, &tags, &meta, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	metadata := &commonpb.Metadata{}
	if err := protojson.Unmarshal([]byte(meta), metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	var tagList []string
	if tags != "" {
		tagList = strings.Split(tags, ",")
	}
	return &productpb.Product{
		Id:          id,
		MasterId:    masterID,
		Name:        name,
		Description: description,
		Type:        productpb.ProductType(ptype),
		Status:      productpb.ProductStatus(status),
		Tags:        tagList,
		Metadata:    metadata,
		CreatedAt:   createdAt.Unix(),
		UpdatedAt:   updatedAt.Unix(),
	}, nil
}

func (r *PostgresRepository) UpdateInventory(_ context.Context, _ string, _ int32) (*productpb.ProductVariant, error) {
	// TODO: implement UpdateInventory logic
	return nil, errors.New("not implemented")
}

func (r *PostgresRepository) ListProductVariants(_ context.Context, _ string) ([]*productpb.ProductVariant, error) {
	// TODO: implement ListProductVariants logic
	return nil, errors.New("not implemented")
}
