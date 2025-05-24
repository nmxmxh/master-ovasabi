package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	productpb "github.com/nmxmxh/master-ovasabi/api/protos/product/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// Define a custom type for context keys to avoid linter errors
// and ensure type safety for metadata filters in context

// contextKey is a custom type for context keys
// Used for metadata filters in context
// This is required for robust metadata-driven filtering
// and to comply with Go best practices

type contextKey string

const metadataFiltersKey contextKey = "metadata_filters"

// ProductOpsHandler handles product-related actions via the "action" field.
//
// @Summary Product Operations
// @Description Handles product-related actions using the "action" field in the request body. Each action (e.g., create_product, update_product, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags product
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/product_ops [post]

func ProductOpsHandler(log *zap.Logger, container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var productSvc productpb.ProductServiceServer
		if err := container.Resolve(&productSvc); err != nil {
			log.Error("Failed to resolve ProductService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode product request JSON", zap.Error(err))
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in product request", zap.Any("value", req["action"]))
			http.Error(w, "missing or invalid action", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		switch action {
		case "create_product":
			product := &productpb.Product{}
			if v, ok := req["name"].(string); ok {
				product.Name = v
			}
			if v, ok := req["description"].(string); ok {
				product.Description = v
			}
			if v, ok := req["type"].(float64); ok {
				product.Type = productpb.ProductType(int32(v))
			}
			if v, ok := req["status"].(float64); ok {
				product.Status = productpb.ProductStatus(int32(v))
			}
			if arr, ok := req["tags"].([]interface{}); ok {
				for _, t := range arr {
					if s, ok := t.(string); ok {
						product.Tags = append(product.Tags, s)
					}
				}
			}
			ownerID, ok := req["owner_id"].(string)
			if !ok {
				log.Error("Missing or invalid owner_id in create_product", zap.Any("value", req["owner_id"]))
				http.Error(w, "missing or invalid owner_id", http.StatusBadRequest)
				return
			}
			product.OwnerId = ownerID
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
					http.Error(w, "invalid metadata", http.StatusBadRequest)
					return
				}
				product.Metadata = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			protoReq := &productpb.CreateProductRequest{Product: product}
			resp, err := productSvc.CreateProduct(ctx, protoReq)
			if err != nil {
				log.Error("Failed to create product", zap.Error(err))
				http.Error(w, "failed to create product", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"product": resp.Product}); err != nil {
				log.Error("Failed to write JSON response (create_product)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_product":
			productID, ok := req["product_id"].(string)
			if !ok {
				log.Error("Missing or invalid product_id in get_product", zap.Any("value", req["product_id"]))
				http.Error(w, "missing or invalid product_id", http.StatusBadRequest)
				return
			}
			protoReq := &productpb.GetProductRequest{ProductId: productID}
			resp, err := productSvc.GetProduct(ctx, protoReq)
			if err != nil {
				log.Error("Failed to get product", zap.Error(err))
				http.Error(w, "failed to get product", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"product": resp.Product}); err != nil {
				log.Error("Failed to write JSON response (get_product)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "update_product":
			productID, ok := req["product_id"].(string)
			if !ok {
				log.Error("Missing or invalid product_id in update_product", zap.Any("value", req["product_id"]))
				http.Error(w, "missing or invalid product_id", http.StatusBadRequest)
				return
			}
			product := &productpb.Product{Id: productID}
			if v, ok := req["name"].(string); ok {
				product.Name = v
			}
			if v, ok := req["description"].(string); ok {
				product.Description = v
			}
			if v, ok := req["type"].(float64); ok {
				product.Type = productpb.ProductType(int32(v))
			}
			if v, ok := req["status"].(float64); ok {
				product.Status = productpb.ProductStatus(int32(v))
			}
			if arr, ok := req["tags"].([]interface{}); ok {
				for _, t := range arr {
					if s, ok := t.(string); ok {
						product.Tags = append(product.Tags, s)
					}
				}
			}
			ownerID, ok := req["owner_id"].(string)
			if !ok {
				log.Error("Missing or invalid owner_id in update_product", zap.Any("value", req["owner_id"]))
				http.Error(w, "missing or invalid owner_id", http.StatusBadRequest)
				return
			}
			product.OwnerId = ownerID
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
					http.Error(w, "invalid metadata", http.StatusBadRequest)
					return
				}
				product.Metadata = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			protoReq := &productpb.UpdateProductRequest{Product: product}
			resp, err := productSvc.UpdateProduct(ctx, protoReq)
			if err != nil {
				log.Error("Failed to update product", zap.Error(err))
				http.Error(w, "failed to update product", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"product": resp.Product}); err != nil {
				log.Error("Failed to write JSON response (update_product)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "delete_product":
			productID, ok := req["product_id"].(string)
			if !ok {
				log.Error("Missing or invalid product_id in delete_product", zap.Any("value", req["product_id"]))
				http.Error(w, "missing or invalid product_id", http.StatusBadRequest)
				return
			}
			protoReq := &productpb.DeleteProductRequest{ProductId: productID}
			resp, err := productSvc.DeleteProduct(ctx, protoReq)
			if err != nil {
				log.Error("Failed to delete product", zap.Error(err))
				http.Error(w, "failed to delete product", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"success": resp.Success}); err != nil {
				log.Error("Failed to write JSON response (delete_product)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_products":
			page := int32(0)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			ownerID, ok := req["owner_id"].(string)
			if !ok {
				log.Error("Missing or invalid owner_id in list_products", zap.Any("value", req["owner_id"]))
				http.Error(w, "missing or invalid owner_id", http.StatusBadRequest)
				return
			}
			tags := []string{}
			if arr, ok := req["tags"].([]interface{}); ok {
				for _, t := range arr {
					if s, ok := t.(string); ok {
						tags = append(tags, s)
					}
				}
			}
			var metadataFilters map[string]interface{}
			if m, ok := req["metadata_filters"].(map[string]interface{}); ok {
				metadataFilters = m
			}
			if metadataFilters != nil {
				ctx = context.WithValue(ctx, metadataFiltersKey, metadataFilters)
			}
			protoReq := &productpb.ListProductsRequest{
				Page:     page,
				PageSize: pageSize,
				OwnerId:  ownerID,
				Tags:     tags,
			}
			resp, err := productSvc.ListProducts(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list products", zap.Error(err))
				http.Error(w, "failed to list products", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"products": resp.Products, "total_count": resp.TotalCount, "page": resp.Page, "total_pages": resp.TotalPages}); err != nil {
				log.Error("Failed to write JSON response (list_products)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "search_products":
			query, ok := req["query"].(string)
			if !ok {
				log.Error("Missing or invalid query in search_products", zap.Any("value", req["query"]))
				http.Error(w, "missing or invalid query", http.StatusBadRequest)
				return
			}
			page := int32(0)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			tags := []string{}
			if arr, ok := req["tags"].([]interface{}); ok {
				for _, t := range arr {
					if s, ok := t.(string); ok {
						tags = append(tags, s)
					}
				}
			}
			var metadataFilters map[string]interface{}
			if m, ok := req["metadata_filters"].(map[string]interface{}); ok {
				metadataFilters = m
			}
			if metadataFilters != nil {
				ctx = context.WithValue(ctx, metadataFiltersKey, metadataFilters)
			}
			protoReq := &productpb.SearchProductsRequest{
				Query:    query,
				Page:     page,
				PageSize: pageSize,
				Tags:     tags,
			}
			resp, err := productSvc.SearchProducts(ctx, protoReq)
			if err != nil {
				log.Error("Failed to search products", zap.Error(err))
				http.Error(w, "failed to search products", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"products": resp.Products, "total_count": resp.TotalCount, "page": resp.Page, "total_pages": resp.TotalPages}); err != nil {
				log.Error("Failed to write JSON response (search_products)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		default:
			log.Error("Unknown action in product_ops", zap.Any("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
}
