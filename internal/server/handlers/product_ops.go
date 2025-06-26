package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	productpb "github.com/nmxmxh/master-ovasabi/api/protos/product/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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

func ProductOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var productSvc productpb.ProductServiceServer
		if err := container.Resolve(&productSvc); err != nil {
			log.Error("Failed to resolve ProductService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err)
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode product request JSON", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in product request", zap.Any("value", req["action"]))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil)
			return
		}
		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		roles := authCtx.Roles
		isGuest := userID == "" || (len(roles) == 1 && roles[0] == "guest")

		actionHandlers := map[string]func(){
			"create_product": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", nil)
					return
				}
				// The request body for create_product should contain a 'product' object.
				// We check the owner_id within that object.
				if productData, ok := req["product"].(map[string]interface{}); ok {
					if ownerID, ok := productData["owner_id"].(string); !ok || (!httputil.IsAdmin(roles) && ownerID != userID) {
						httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: only owner or admin can create", nil)
						return
					}
					// Enrich metadata with audit info
					if m, ok := productData["metadata"].(map[string]interface{}); ok {
						if ss, ok := m["service_specific"].(map[string]interface{}); ok {
							if prod, ok := ss["product"].(map[string]interface{}); ok {
								prod["audit"] = map[string]interface{}{"created_by": userID, "created_at": time.Now().UTC().Format(time.RFC3339)}
								ss["product"] = prod
								m["service_specific"] = ss
								productData["metadata"] = m
							}
						}
					}
					req["product"] = productData
				} else {
					httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing product data in request", nil)
					return
				}
				handleProductAction(w, ctx, log, req, &productpb.CreateProductRequest{}, productSvc.CreateProduct)
			},
			"get_product": func() {
				handleProductAction(w, ctx, log, req, &productpb.GetProductRequest{}, productSvc.GetProduct)
			},
			"update_product": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", nil)
					return
				}
				if productData, ok := req["product"].(map[string]interface{}); ok {
					if ownerID, ok := productData["owner_id"].(string); !ok || (!httputil.IsAdmin(roles) && ownerID != userID) {
						httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: only owner or admin can update", nil)
						return
					}
					// Enrich metadata with audit info
					if m, ok := productData["metadata"].(map[string]interface{}); ok {
						if ss, ok := m["service_specific"].(map[string]interface{}); ok {
							if prod, ok := ss["product"].(map[string]interface{}); ok {
								prod["audit"] = map[string]interface{}{"last_modified_by": userID, "last_modified_at": time.Now().UTC().Format(time.RFC3339)}
								ss["product"] = prod
								m["service_specific"] = ss
								productData["metadata"] = m
							}
						}
					}
					req["product"] = productData
				} else {
					httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing product data in request", nil)
					return
				}
				handleProductAction(w, ctx, log, req, &productpb.UpdateProductRequest{}, productSvc.UpdateProduct)
			},
			"delete_product": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", nil)
					return
				}
				// For delete, owner_id might be at the top level for permission check before deletion
				if ownerID, ok := req["owner_id"].(string); !ok || (!httputil.IsAdmin(roles) && ownerID != userID) {
					httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: only owner or admin can delete", nil)
					return
				}
				handleProductAction(w, ctx, log, req, &productpb.DeleteProductRequest{}, productSvc.DeleteProduct)
			},
			"list_products": func() {
				if m, ok := req["metadata_filters"].(map[string]interface{}); ok {
					ctx = context.WithValue(ctx, metadataFiltersKey, m)
				}
				handleProductAction(w, ctx, log, req, &productpb.ListProductsRequest{}, productSvc.ListProducts)
			},
			"search_products": func() {
				if m, ok := req["metadata_filters"].(map[string]interface{}); ok {
					ctx = context.WithValue(ctx, metadataFiltersKey, m)
				}
				handleProductAction(w, ctx, log, req, &productpb.SearchProductsRequest{}, productSvc.SearchProducts)
			},
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			log.Error("Unknown action in product_ops", zap.String("action", action))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil)
		}
	}
}

// handleProductAction is a generic helper to reduce boilerplate in ProductOpsHandler.
func handleProductAction[T proto.Message, U proto.Message](
	w http.ResponseWriter,
	ctx context.Context,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoProduct(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		log.Error("product service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil)
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProtoProduct converts a map[string]interface{} to a proto.Message using JSON as an intermediate.
func mapToProtoProduct(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonBytes, v)
}
