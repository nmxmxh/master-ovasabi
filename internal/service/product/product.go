package product

import (
	"context"
	"errors"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	productpb "github.com/nmxmxh/master-ovasabi/api/protos/product/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

type EventEmitter interface {
	EmitEventWithLogging(ctx context.Context, emitter interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool)
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
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "missing product data", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		err := graceful.WrapErr(ctx, codes.Unauthenticated, "missing authentication", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	req.Product.OwnerId = authUserID
	meta, err := ExtractAndEnrichProductMetadata(req.Product.Metadata, authUserID, true)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.InvalidArgument, "invalid metadata", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	req.Product.Metadata = meta
	if req.Product.CampaignId == 0 {
		req.Product.CampaignId = 0
	}
	created, err := s.repo.CreateProduct(ctx, req.Product)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to create product", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "product created", created, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.Cache, CacheKey: created.Id, CacheValue: created, CacheTTL: 10 * time.Minute, Metadata: created.Metadata, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "product.created", EventID: created.Id, PatternType: "product", PatternID: created.Id, PatternMeta: created.Metadata})
	return &productpb.CreateProductResponse{Product: created}, nil
}

func (s *Service) UpdateProduct(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		err := graceful.WrapErr(ctx, codes.Unauthenticated, "missing authentication", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "product")
	product, err := s.repo.GetProduct(ctx, req.Product.Id)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.NotFound, "product not found", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	if !isAdmin && product.OwnerId != authUserID {
		err := graceful.WrapErr(ctx, codes.PermissionDenied, "cannot update product you do not own", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	if req == nil || req.Product == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "Product is required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	meta, err := ExtractAndEnrichProductMetadata(req.Product.Metadata, authUserID, false)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.InvalidArgument, "invalid metadata", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	req.Product.Metadata = meta
	if req.Product.CampaignId == 0 {
		req.Product.CampaignId = 0
	}
	updated, err := s.repo.UpdateProduct(ctx, req.Product)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to update product", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "product updated", updated, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.Cache, CacheKey: updated.Id, CacheValue: updated, CacheTTL: 10 * time.Minute, Metadata: updated.Metadata, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "product.updated", EventID: updated.Id, PatternType: "product", PatternID: updated.Id, PatternMeta: updated.Metadata})
	return &productpb.UpdateProductResponse{Product: updated}, nil
}

func (s *Service) DeleteProduct(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		err := graceful.WrapErr(ctx, codes.Unauthenticated, "missing authentication", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "product")
	product, err := s.repo.GetProduct(ctx, req.ProductId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.NotFound, "product not found", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	if !isAdmin && product.OwnerId != authUserID {
		err := graceful.WrapErr(ctx, codes.PermissionDenied, "cannot delete product you do not own", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	if req == nil || req.ProductId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "Product ID is required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	err = s.repo.DeleteProduct(ctx, req.ProductId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to delete product", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "product deleted", req.ProductId, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.Cache, CacheKey: req.ProductId, CacheValue: req.ProductId, CacheTTL: 10 * time.Minute, Metadata: nil, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "product.deleted", EventID: req.ProductId, PatternType: "product", PatternID: req.ProductId, PatternMeta: nil})
	return &productpb.DeleteProductResponse{Success: true}, nil
}

func (s *Service) GetProduct(ctx context.Context, req *productpb.GetProductRequest) (*productpb.GetProductResponse, error) {
	if req == nil || req.ProductId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "Product ID is required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	product, err := s.repo.GetProduct(ctx, req.ProductId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to get product", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	if product == nil {
		err := graceful.WrapErr(ctx, codes.NotFound, "Product not found", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	return &productpb.GetProductResponse{Product: product}, nil
}

func (s *Service) ListProducts(ctx context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "Request is required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
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
		err = graceful.WrapErr(ctx, codes.Internal, "failed to list products", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	totalCount := utils.ToInt32(total)
	totalPages := int32(0)
	if filter.PageSize > 0 {
		totalPages = utils.ToInt32((total + filter.PageSize - 1) / filter.PageSize)
	}
	for _, p := range products {
		meta, err := ExtractAndEnrichProductMetadata(p.Metadata, p.OwnerId, false)
		if err == nil && meta != nil {
			p.Metadata = meta
		}
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
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "Request is required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
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
		err = graceful.WrapErr(ctx, codes.Internal, "failed to search products", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	totalCount := utils.ToInt32(total)
	totalPages := int32(0)
	if filter.PageSize > 0 {
		totalPages = utils.ToInt32((total + filter.PageSize - 1) / filter.PageSize)
	}
	for _, p := range products {
		meta, err := ExtractAndEnrichProductMetadata(p.Metadata, p.OwnerId, false)
		if err == nil && meta != nil {
			p.Metadata = meta
		}
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
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "Variant ID is required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	variant, err := s.repo.UpdateInventory(ctx, req.VariantId, req.Delta)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to update inventory", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	return &productpb.UpdateInventoryResponse{Variant: variant}, nil
}

func (s *Service) ListProductVariants(ctx context.Context, req *productpb.ListProductVariantsRequest) (*productpb.ListProductVariantsResponse, error) {
	if req == nil || req.ProductId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "Product ID is required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	variants, err := s.repo.ListProductVariants(ctx, req.ProductId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to list product variants", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	return &productpb.ListProductVariantsResponse{Variants: variants}, nil
}
