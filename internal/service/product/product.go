package product

import (
	"context"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	productpb "github.com/nmxmxh/master-ovasabi/api/protos/product/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

type EventEmitter interface {
	EmitEventWithLogging(context.Context, interface{}, *zap.Logger, string, string, *commonpb.Metadata) (string, bool)
	EmitRawEventWithLogging(context.Context, *zap.Logger, string, string, []byte) (string, bool)
}

type Service struct {
	productpb.UnimplementedProductServiceServer
	repo         RepositoryItf
	Cache        *redis.Cache
	log          *zap.Logger
	eventEmitter EventEmitter
	eventEnabled bool
}

func NewProductService(repo RepositoryItf, log *zap.Logger, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) *Service {
	return &Service{
		repo:         repo,
		log:          log,
		Cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
}

var _ productpb.ProductServiceServer = (*Service)(nil)

func (s *Service) CreateProduct(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	if req == nil || req.Product == nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "missing product data", codes.InvalidArgument))
	}
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "missing authentication", codes.Unauthenticated))
	}
	req.Product.OwnerId = authUserID
	metadata.MigrateMetadata(req.Product.Metadata)
	if err := metadata.ValidateMetadata(req.Product.Metadata); err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "invalid metadata", codes.InvalidArgument))
	}
	if req.Product.CampaignId == 0 {
		req.Product.CampaignId = 0
	}
	created, err := s.repo.CreateProduct(ctx, req.Product)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to create product", codes.Internal))
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "product created", created, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.Cache, CacheKey: created.Id, CacheValue: created, CacheTTL: 10 * time.Minute, Metadata: created.Metadata, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "product.created", EventID: created.Id, PatternType: "product", PatternID: created.Id, PatternMeta: created.Metadata})
	return &productpb.CreateProductResponse{Product: created}, nil
}

func (s *Service) UpdateProduct(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "missing authentication", codes.Unauthenticated))
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "product")
	product, err := s.repo.GetProduct(ctx, req.Product.Id)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "product not found", codes.NotFound))
	}
	if !isAdmin && product.OwnerId != authUserID {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "cannot update product you do not own", codes.PermissionDenied))
	}
	if req == nil || req.Product == nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "Product is required", codes.InvalidArgument))
	}
	metadata.MigrateMetadata(req.Product.Metadata)
	if err := metadata.ValidateMetadata(req.Product.Metadata); err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "invalid metadata", codes.InvalidArgument))
	}
	if req.Product.CampaignId == 0 {
		req.Product.CampaignId = 0
	}
	updated, err := s.repo.UpdateProduct(ctx, req.Product)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update product", codes.Internal))
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "product updated", updated, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.Cache, CacheKey: updated.Id, CacheValue: updated, CacheTTL: 10 * time.Minute, Metadata: updated.Metadata, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "product.updated", EventID: updated.Id, PatternType: "product", PatternID: updated.Id, PatternMeta: updated.Metadata})
	return &productpb.UpdateProductResponse{Product: updated}, nil
}

func (s *Service) DeleteProduct(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "missing authentication", codes.Unauthenticated))
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "product")
	product, err := s.repo.GetProduct(ctx, req.ProductId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "product not found", codes.NotFound))
	}
	if !isAdmin && product.OwnerId != authUserID {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "cannot delete product you do not own", codes.PermissionDenied))
	}
	if req == nil || req.ProductId == "" {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "Product ID is required", codes.InvalidArgument))
	}
	err = s.repo.DeleteProduct(ctx, req.ProductId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to delete product", codes.Internal))
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "product deleted", req.ProductId, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.Cache, CacheKey: req.ProductId, CacheValue: req.ProductId, CacheTTL: 10 * time.Minute, Metadata: nil, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "product.deleted", EventID: req.ProductId, PatternType: "product", PatternID: req.ProductId, PatternMeta: nil})
	return &productpb.DeleteProductResponse{Success: true}, nil
}

func (s *Service) GetProduct(ctx context.Context, req *productpb.GetProductRequest) (*productpb.GetProductResponse, error) {
	if req == nil || req.ProductId == "" {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "Product ID is required", codes.InvalidArgument))
	}
	product, err := s.repo.GetProduct(ctx, req.ProductId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to get product", codes.Internal))
	}
	if product == nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "Product not found", codes.NotFound))
	}
	return &productpb.GetProductResponse{Product: product}, nil
}

func (s *Service) ListProducts(ctx context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
	if req == nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "Request is required", codes.InvalidArgument))
	}
	var metadataFilters map[string]interface{}
	if mf, ok := ctx.Value("metadata_filters").(map[string]interface{}); ok {
		metadataFilters = mf
	}
	filter := ListProductsFilter{
		OwnerID:         req.OwnerId,
		Type:            req.Type,
		Status:          req.Status,
		Tags:            req.Tags,
		Page:            int(req.Page),
		PageSize:        int(req.PageSize),
		CampaignID:      req.CampaignId,
		MetadataFilters: metadataFilters,
	}
	products, total, err := s.repo.ListProducts(ctx, filter)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to list products", codes.Internal))
	}
	totalCount := utils.ToInt32(total)
	totalPages := int32(0)
	if filter.PageSize > 0 {
		totalPages = utils.ToInt32((total + filter.PageSize - 1) / filter.PageSize)
	}
	for _, p := range products {
		metadata.MigrateMetadata(p.Metadata)
	}
	return &productpb.ListProductsResponse{
		Products:   products,
		TotalCount: totalCount,
		Page:       req.Page,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) SearchProducts(ctx context.Context, req *productpb.SearchProductsRequest) (*productpb.SearchProductsResponse, error) {
	if req == nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "Request is required", codes.InvalidArgument))
	}
	filter := SearchProductsFilter{
		Query:      req.Query,
		Tags:       req.Tags,
		Type:       req.Type,
		Status:     req.Status,
		Page:       int(req.Page),
		PageSize:   int(req.PageSize),
		CampaignID: req.CampaignId,
	}
	products, total, err := s.repo.SearchProducts(ctx, filter)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to search products", codes.Internal))
	}
	totalCount := utils.ToInt32(total)
	totalPages := int32(0)
	if filter.PageSize > 0 {
		totalPages = utils.ToInt32((total + filter.PageSize - 1) / filter.PageSize)
	}
	for _, p := range products {
		metadata.MigrateMetadata(p.Metadata)
	}
	return &productpb.SearchProductsResponse{
		Products:   products,
		TotalCount: totalCount,
		Page:       req.Page,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) UpdateInventory(ctx context.Context, req *productpb.UpdateInventoryRequest) (*productpb.UpdateInventoryResponse, error) {
	if req == nil || req.VariantId == "" {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "Variant ID is required", codes.InvalidArgument))
	}
	variant, err := s.repo.UpdateInventory(ctx, req.VariantId, req.Delta)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update inventory", codes.Internal))
	}
	return &productpb.UpdateInventoryResponse{Variant: variant}, nil
}

func (s *Service) ListProductVariants(ctx context.Context, req *productpb.ListProductVariantsRequest) (*productpb.ListProductVariantsResponse, error) {
	if req == nil || req.ProductId == "" {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "Product ID is required", codes.InvalidArgument))
	}
	variants, err := s.repo.ListProductVariants(ctx, req.ProductId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to list product variants", codes.Internal))
	}
	return &productpb.ListProductVariantsResponse{Variants: variants}, nil
}
