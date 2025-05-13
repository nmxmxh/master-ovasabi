package productservice

import (
	"context"
	"math"
	"time"

	productpb "github.com/nmxmxh/master-ovasabi/api/protos/product/v1"
	productrepo "github.com/nmxmxh/master-ovasabi/internal/repository/product"
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	metadatautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	productpb.UnimplementedProductServiceServer
	repo  productrepo.Repository
	Cache *redis.Cache
	log   *zap.Logger
}

func NewProductService(repo productrepo.Repository, log *zap.Logger, cache *redis.Cache) *Service {
	return &Service{
		repo:  repo,
		log:   log,
		Cache: cache,
	}
}

var _ productpb.ProductServiceServer = (*Service)(nil)

func (s *Service) CreateProduct(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	if req == nil || req.Product == nil {
		return nil, status.Error(codes.InvalidArgument, "Product is required")
	}
	if err := metadatautil.ValidateMetadata(req.Product.Metadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	created, err := s.repo.CreateProduct(ctx, req.Product)
	if err != nil {
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
	return &productpb.CreateProductResponse{Product: created}, nil
}

func (s *Service) UpdateProduct(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	if req == nil || req.Product == nil {
		return nil, status.Error(codes.InvalidArgument, "Product is required")
	}
	if err := metadatautil.ValidateMetadata(req.Product.Metadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	updated, err := s.repo.UpdateProduct(ctx, req.Product)
	if err != nil {
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
	return &productpb.UpdateProductResponse{Product: updated}, nil
}

func (s *Service) DeleteProduct(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	if req == nil || req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "Product ID is required")
	}
	err := s.repo.DeleteProduct(ctx, req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete product: %v", err)
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
	filter := productrepo.ListProductsFilter{
		OwnerID:  req.OwnerId,
		Type:     req.Type,
		Status:   req.Status,
		Tags:     req.Tags,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}
	products, total, err := s.repo.ListProducts(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list products: %v", err)
	}
	var totalCount int32
	if total > math.MaxInt32 {
		totalCount = math.MaxInt32
		// TODO: log a warning about overflow
	} else {
		totalCount = int32(math.Min(float64(total), float64(math.MaxInt32)))
	}
	return &productpb.ListProductsResponse{
		Products:   products,
		TotalCount: totalCount,
		Page:       req.Page,
		TotalPages: 0, // TODO: implement
	}, nil
}

func (s *Service) SearchProducts(ctx context.Context, req *productpb.SearchProductsRequest) (*productpb.SearchProductsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "Request is required")
	}
	filter := productrepo.SearchProductsFilter{
		Query:    req.Query,
		Tags:     req.Tags,
		Type:     req.Type,
		Status:   req.Status,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}
	products, total, err := s.repo.SearchProducts(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search products: %v", err)
	}
	var totalCountSearch int32
	if total > math.MaxInt32 {
		totalCountSearch = math.MaxInt32
		// TODO: log a warning about overflow
	} else {
		totalCountSearch = int32(math.Min(float64(total), float64(math.MaxInt32)))
	}
	return &productpb.SearchProductsResponse{
		Products:   products,
		TotalCount: totalCountSearch,
		Page:       req.Page,
		TotalPages: 0, // TODO: implement
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
