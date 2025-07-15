package product

import (
	"context"

	productpb "github.com/nmxmxh/master-ovasabi/api/protos/product/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

type Service struct {
	productpb.UnimplementedProductServiceServer
	repo         RepositoryItf
	Cache        *redis.Cache
	log          *zap.Logger
	eventEmitter events.EventEmitter
	eventEnabled bool
	handler      *graceful.Handler
}

func NewProductService(repo RepositoryItf, log *zap.Logger, cache *redis.Cache, eventEmitter events.EventEmitter, eventEnabled bool) *Service {
	return &Service{
		repo:         repo,
		log:          log,
		Cache:        cache,
		eventEnabled: eventEnabled,
		eventEmitter: eventEmitter, // Set via provider or DI
		handler:      graceful.NewHandler(log, eventEmitter, cache, "product", "", eventEnabled),
	}
}

var _ productpb.ProductServiceServer = (*Service)(nil)

func (s *Service) CreateProduct(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	if req == nil || req.Product == nil {
		s.handler.Error(ctx, "create_product", codes.InvalidArgument, "missing product data", nil, nil, "")
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "missing product data", codes.InvalidArgument))
	}
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		s.handler.Error(ctx, "create_product", codes.Unauthenticated, "missing authentication", nil, nil, "")
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "missing authentication", codes.Unauthenticated))
	}
	req.Product.OwnerId = authUserID
	metadata.MigrateMetadata(req.Product.Metadata)
	if err := metadata.ValidateMetadata(req.Product.Metadata); err != nil {
		s.handler.Error(ctx, "create_product", codes.InvalidArgument, "invalid metadata", err, nil, "")
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "invalid metadata", codes.InvalidArgument))
	}
	if req.Product.CampaignId == 0 {
		req.Product.CampaignId = 0
	}
	created, err := s.repo.CreateProduct(ctx, req.Product)
	if err != nil {
		s.handler.Error(ctx, "create_product", codes.Internal, "failed to create product", err, nil, "")
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to create product", codes.Internal))
	}
	resp := &productpb.CreateProductResponse{Product: created}
	s.handler.Success(ctx, "create_product", codes.OK, "product created", resp, created.Metadata, created.Id, nil)
	return resp, nil
}

func (s *Service) UpdateProduct(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		s.handler.Error(ctx, "update_product", codes.Unauthenticated, "missing authentication", nil, nil, "")
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "missing authentication", codes.Unauthenticated))
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "product")
	product, err := s.repo.GetProduct(ctx, req.Product.Id)
	if err != nil {
		s.handler.Error(ctx, "update_product", codes.NotFound, "product not found", err, nil, req.Product.Id)
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "product not found", codes.NotFound))
	}
	if !isAdmin && product.OwnerId != authUserID {
		s.handler.Error(ctx, "update_product", codes.PermissionDenied, "cannot update product you do not own", nil, nil, req.Product.Id)
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "cannot update product you do not own", codes.PermissionDenied))
	}
	if req == nil || req.Product == nil {
		s.handler.Error(ctx, "update_product", codes.InvalidArgument, "Product is required", nil, nil, "")
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "Product is required", codes.InvalidArgument))
	}
	metadata.MigrateMetadata(req.Product.Metadata)
	if err := metadata.ValidateMetadata(req.Product.Metadata); err != nil {
		s.handler.Error(ctx, "update_product", codes.InvalidArgument, "invalid metadata", err, nil, req.Product.Id)
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "invalid metadata", codes.InvalidArgument))
	}
	if req.Product.CampaignId == 0 {
		req.Product.CampaignId = 0
	}
	updated, err := s.repo.UpdateProduct(ctx, req.Product)
	if err != nil {
		s.handler.Error(ctx, "update_product", codes.Internal, "failed to update product", err, nil, req.Product.Id)
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update product", codes.Internal))
	}
	resp := &productpb.UpdateProductResponse{Product: updated}
	s.handler.Success(ctx, "update_product", codes.OK, "product updated", resp, updated.Metadata, updated.Id, nil)
	return resp, nil
}

func (s *Service) DeleteProduct(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		s.handler.Error(ctx, "delete_product", codes.Unauthenticated, "missing authentication", nil, nil, "")
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "missing authentication", codes.Unauthenticated))
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "product")
	product, err := s.repo.GetProduct(ctx, req.ProductId)
	if err != nil {
		s.handler.Error(ctx, "delete_product", codes.NotFound, "product not found", err, nil, req.ProductId)
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "product not found", codes.NotFound))
	}
	if !isAdmin && product.OwnerId != authUserID {
		s.handler.Error(ctx, "delete_product", codes.PermissionDenied, "cannot delete product you do not own", nil, nil, req.ProductId)
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "cannot delete product you do not own", codes.PermissionDenied))
	}
	if req == nil || req.ProductId == "" {
		s.handler.Error(ctx, "delete_product", codes.InvalidArgument, "Product ID is required", nil, nil, "")
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "Product ID is required", codes.InvalidArgument))
	}
	err = s.repo.DeleteProduct(ctx, req.ProductId)
	if err != nil {
		s.handler.Error(ctx, "delete_product", codes.Internal, "failed to delete product", err, nil, req.ProductId)
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to delete product", codes.Internal))
	}
	resp := &productpb.DeleteProductResponse{Success: true}
	s.handler.Success(ctx, "delete_product", codes.OK, "product deleted", resp, nil, req.ProductId, nil)
	return resp, nil
}

func (s *Service) GetProduct(ctx context.Context, req *productpb.GetProductRequest) (*productpb.GetProductResponse, error) {
	if req == nil || req.ProductId == "" {
		s.handler.Error(ctx, "get_product", codes.InvalidArgument, "Product ID is required", nil, nil, "")
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "Product ID is required", codes.InvalidArgument))
	}
	product, err := s.repo.GetProduct(ctx, req.ProductId)
	if err != nil {
		s.handler.Error(ctx, "get_product", codes.Internal, "failed to get product", err, nil, req.ProductId)
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to get product", codes.Internal))
	}
	if product == nil {
		s.handler.Error(ctx, "get_product", codes.NotFound, "Product not found", nil, nil, req.ProductId)
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "Product not found", codes.NotFound))
	}
	resp := &productpb.GetProductResponse{Product: product}
	s.handler.Success(ctx, "get_product", codes.OK, "product fetched", resp, product.Metadata, product.Id, nil)
	return resp, nil
}

func (s *Service) ListProducts(ctx context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
	if req == nil {
		s.handler.Error(ctx, "list_products", codes.InvalidArgument, "Request is required", nil, nil, "")
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
		s.handler.Error(ctx, "list_products", codes.Internal, "failed to list products", err, nil, "")
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
	resp := &productpb.ListProductsResponse{
		Products:   products,
		TotalCount: totalCount,
		Page:       req.Page,
		TotalPages: totalPages,
	}
	s.handler.Success(ctx, "list_products", codes.OK, "products listed", resp, nil, "", nil)
	return resp, nil
}

func (s *Service) SearchProducts(ctx context.Context, req *productpb.SearchProductsRequest) (*productpb.SearchProductsResponse, error) {
	if req == nil {
		s.handler.Error(ctx, "search_products", codes.InvalidArgument, "Request is required", nil, nil, "")
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
		s.handler.Error(ctx, "search_products", codes.Internal, "failed to search products", err, nil, "")
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
	resp := &productpb.SearchProductsResponse{
		Products:   products,
		TotalCount: totalCount,
		Page:       req.Page,
		TotalPages: totalPages,
	}
	s.handler.Success(ctx, "search_products", codes.OK, "products searched", resp, nil, "", nil)
	return resp, nil
}

func (s *Service) UpdateInventory(ctx context.Context, req *productpb.UpdateInventoryRequest) (*productpb.UpdateInventoryResponse, error) {
	if req == nil || req.VariantId == "" {
		s.handler.Error(ctx, "update_inventory", codes.InvalidArgument, "Variant ID is required", nil, nil, "")
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "Variant ID is required", codes.InvalidArgument))
	}
	variant, err := s.repo.UpdateInventory(ctx, req.VariantId, req.Delta)
	if err != nil {
		s.handler.Error(ctx, "update_inventory", codes.Internal, "failed to update inventory", err, nil, req.VariantId)
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update inventory", codes.Internal))
	}
	resp := &productpb.UpdateInventoryResponse{Variant: variant}
	s.handler.Success(ctx, "update_inventory", codes.OK, "inventory updated", resp, nil, req.VariantId, nil)
	return resp, nil
}

func (s *Service) ListProductVariants(ctx context.Context, req *productpb.ListProductVariantsRequest) (*productpb.ListProductVariantsResponse, error) {
	if req == nil || req.ProductId == "" {
		s.handler.Error(ctx, "list_product_variants", codes.InvalidArgument, "Product ID is required", nil, nil, "")
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "Product ID is required", codes.InvalidArgument))
	}
	variants, err := s.repo.ListProductVariants(ctx, req.ProductId)
	if err != nil {
		s.handler.Error(ctx, "list_product_variants", codes.Internal, "failed to list product variants", err, nil, req.ProductId)
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to list product variants", codes.Internal))
	}
	resp := &productpb.ListProductVariantsResponse{Variants: variants}
	s.handler.Success(ctx, "list_product_variants", codes.OK, "product variants listed", resp, nil, req.ProductId, nil)
	return resp, nil
}
