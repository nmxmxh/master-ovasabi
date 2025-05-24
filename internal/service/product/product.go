package product

import (
	"context"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	productpb "github.com/nmxmxh/master-ovasabi/api/protos/product/v1"
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	events "github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type EventEmitter interface {
	EmitEvent(ctx context.Context, eventType, entityID string, metadata *commonpb.Metadata) error
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
		return nil, status.Error(codes.InvalidArgument, "Product is required")
	}
	userID := req.Product.OwnerId
	meta, err := ExtractAndEnrichProductMetadata(req.Product.Metadata, userID, true)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{
				"error":    err.Error(),
				"owner_id": userID,
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for product event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "product.create_failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit product.create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	req.Product.Metadata = meta
	if req.Product.CampaignId == 0 {
		req.Product.CampaignId = 0
	}
	created, err := s.repo.CreateProduct(ctx, req.Product)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{
				"error":    err.Error(),
				"owner_id": userID,
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for product event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "product.create_failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit product.create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create product: %v", err)
	}
	if s.Cache != nil && created.Metadata != nil {
		if err := pattern.CacheMetadata(ctx, s.log, s.Cache, "product", created.Id, created.Metadata, 10*time.Minute); err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	if err := pattern.RegisterSchedule(ctx, s.log, "product", created.Id, created.Metadata); err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "product", created.Id, created.Metadata); err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	if err := pattern.RegisterWithNexus(ctx, s.log, "product", created.Metadata); err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	created.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "product.created", created.Id, created.Metadata)
	return &productpb.CreateProductResponse{Product: created}, nil
}

func (s *Service) UpdateProduct(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	if req == nil || req.Product == nil {
		return nil, status.Error(codes.InvalidArgument, "Product is required")
	}
	userID := req.Product.OwnerId
	meta, err := ExtractAndEnrichProductMetadata(req.Product.Metadata, userID, false)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{
				"error":    err.Error(),
				"owner_id": userID,
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for product event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "product.update_failed", req.Product.Id, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit product.update_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	req.Product.Metadata = meta
	if req.Product.CampaignId == 0 {
		req.Product.CampaignId = 0
	}
	updated, err := s.repo.UpdateProduct(ctx, req.Product)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{
				"error":    err.Error(),
				"owner_id": userID,
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for product event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "product.update_failed", req.Product.Id, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit product.update_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to update product: %v", err)
	}
	if s.Cache != nil && updated.Metadata != nil {
		if err := pattern.CacheMetadata(ctx, s.log, s.Cache, "product", updated.Id, updated.Metadata, 10*time.Minute); err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	if err := pattern.RegisterSchedule(ctx, s.log, "product", updated.Id, updated.Metadata); err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "product", updated.Id, updated.Metadata); err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	if err := pattern.RegisterWithNexus(ctx, s.log, "product", updated.Metadata); err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	updated.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "product.updated", updated.Id, updated.Metadata)
	return &productpb.UpdateProductResponse{Product: updated}, nil
}

func (s *Service) DeleteProduct(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	if req == nil || req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "Product ID is required")
	}
	err := s.repo.DeleteProduct(ctx, req.ProductId)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{
				"error":      err.Error(),
				"product_id": req.ProductId,
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for product event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "product.delete_failed", req.ProductId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit product.delete_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to delete product: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEmit := s.eventEmitter.EmitEvent(ctx, "product.deleted", req.ProductId, nil)
		if errEmit != nil {
			s.log.Warn("Failed to emit product.deleted event", zap.Error(errEmit))
		}
	}
	return &productpb.DeleteProductResponse{Success: true}, nil
}

func (s *Service) GetProduct(ctx context.Context, req *productpb.GetProductRequest) (*productpb.GetProductResponse, error) {
	if req == nil || req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "Product ID is required")
	}
	product, err := s.repo.GetProduct(ctx, req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get product: %v", err)
	}
	if product == nil {
		return nil, status.Error(codes.NotFound, "Product not found")
	}
	return &productpb.GetProductResponse{Product: product}, nil
}

func (s *Service) ListProducts(ctx context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "Request is required")
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
		return nil, status.Errorf(codes.Internal, "failed to list products: %v", err)
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
		return nil, status.Error(codes.InvalidArgument, "Request is required")
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
		return nil, status.Errorf(codes.Internal, "failed to search products: %v", err)
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
		return nil, status.Error(codes.InvalidArgument, "Variant ID is required")
	}
	variant, err := s.repo.UpdateInventory(ctx, req.VariantId, req.Delta)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update inventory: %v", err)
	}
	return &productpb.UpdateInventoryResponse{Variant: variant}, nil
}

func (s *Service) ListProductVariants(ctx context.Context, req *productpb.ListProductVariantsRequest) (*productpb.ListProductVariantsResponse, error) {
	if req == nil || req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "Product ID is required")
	}
	variants, err := s.repo.ListProductVariants(ctx, req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list product variants: %v", err)
	}
	return &productpb.ListProductVariantsResponse{Variants: variants}, nil
}
