package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	commercepb "github.com/nmxmxh/master-ovasabi/api/protos/commerce/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	auth "github.com/nmxmxh/master-ovasabi/pkg/auth"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	shield "github.com/nmxmxh/master-ovasabi/pkg/shield"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// CommerceOpsHandler handles commerce-related actions via the "action" field.
//
// @Summary Commerce Operations
// @Description Handles commerce-related actions using the "action" field in the request body. Each action (e.g., create_order, process_payment, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags commerce
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/commerce_ops [post]

// CommerceOpsHandler: Composable, robust handler for commerce operations.
func CommerceOpsHandler(log *zap.Logger, container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var commerceSvc commercepb.CommerceServiceServer
		if err := container.Resolve(&commerceSvc); err != nil {
			log.Error("Failed to resolve CommerceService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Extract authentication context
		authCtx := auth.FromContext(r.Context())
		userID := authCtx.UserID
		isGuest := userID == "" || (len(authCtx.Roles) == 1 && authCtx.Roles[0] == "guest")
		if isGuest {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		meta := shield.BuildRequestMetadata(r, userID, isGuest)
		var securitySvc securitypb.SecurityServiceClient
		if err := container.Resolve(&securitySvc); err != nil {
			log.Error("Failed to resolve SecurityService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode commerce request JSON", zap.Error(err))
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in commerce request", zap.Any("value", req["action"]))
			http.Error(w, "missing or invalid action", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		// Strict permission check for all commerce actions
		err := shield.CheckPermission(ctx, securitySvc, action, "commerce", shield.WithMetadata(meta))
		switch {
		case err == nil:
			// allowed, proceed
		case errors.Is(err, shield.ErrUnauthenticated):
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		case errors.Is(err, shield.ErrPermissionDenied):
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		switch action {
		case "create_quote":
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in create_quote", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			productID, ok := req["product_id"].(string)
			if !ok {
				log.Error("Missing or invalid product_id in create_quote", zap.Any("value", req["product_id"]))
				http.Error(w, "missing or invalid product_id", http.StatusBadRequest)
				return
			}
			amount, ok := req["amount"].(float64)
			if !ok {
				log.Error("Missing or invalid amount in create_quote", zap.Any("value", req["amount"]))
				http.Error(w, "missing or invalid amount", http.StatusBadRequest)
				return
			}
			currency, ok := req["currency"].(string)
			if !ok {
				log.Error("Missing or invalid currency in create_quote", zap.Any("value", req["currency"]))
				http.Error(w, "missing or invalid currency", http.StatusBadRequest)
				return
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
					http.Error(w, "invalid metadata", http.StatusBadRequest)
					return
				}
				meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &commercepb.CreateQuoteRequest{
				UserId:     userID,
				ProductId:  productID,
				Amount:     amount,
				Currency:   currency,
				Metadata:   meta,
				CampaignId: campaignID,
			}
			resp, err := commerceSvc.CreateQuote(ctx, protoReq)
			if err != nil {
				log.Error("Failed to create quote", zap.Error(err))
				http.Error(w, "failed to create quote", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"quote": resp.Quote}); err != nil {
				log.Error("Failed to write JSON response (create_quote)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_quote":
			quoteID, ok := req["quote_id"].(string)
			if !ok {
				log.Error("Missing or invalid quote_id in get_quote", zap.Any("value", req["quote_id"]))
				http.Error(w, "missing or invalid quote_id", http.StatusBadRequest)
				return
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &commercepb.GetQuoteRequest{QuoteId: quoteID, CampaignId: campaignID}
			resp, err := commerceSvc.GetQuote(ctx, protoReq)
			if err != nil {
				log.Error("Failed to get quote", zap.Error(err))
				http.Error(w, "failed to get quote", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"quote": resp.Quote}); err != nil {
				log.Error("Failed to write JSON response (get_quote)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "create_order":
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in create_order", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			currency, ok := req["currency"].(string)
			if !ok {
				log.Error("Missing or invalid currency in create_order", zap.Any("value", req["currency"]))
				http.Error(w, "missing or invalid currency", http.StatusBadRequest)
				return
			}
			var items []*commercepb.OrderItem
			if arr, ok := req["items"].([]interface{}); ok {
				for _, it := range arr {
					m, ok := it.(map[string]interface{})
					if !ok {
						continue
					}
					item := &commercepb.OrderItem{}
					if v, ok := m["product_id"].(string); ok {
						item.ProductId = v
					}
					if v, ok := m["quantity"].(float64); ok {
						item.Quantity = int32(v)
					}
					if v, ok := m["price"].(float64); ok {
						item.Price = v
					}
					items = append(items, item)
				}
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
					http.Error(w, "invalid metadata", http.StatusBadRequest)
					return
				}
				meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &commercepb.CreateOrderRequest{
				UserId:     userID,
				Currency:   currency,
				Items:      items,
				Metadata:   meta,
				CampaignId: campaignID,
			}
			resp, err := commerceSvc.CreateOrder(ctx, protoReq)
			if err != nil {
				log.Error("Failed to create order", zap.Error(err))
				http.Error(w, "failed to create order", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"order": resp.Order}); err != nil {
				log.Error("Failed to write JSON response (create_order)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_order":
			orderID, ok := req["order_id"].(string)
			if !ok {
				log.Error("Missing or invalid order_id in get_order", zap.Any("value", req["order_id"]))
				http.Error(w, "missing or invalid order_id", http.StatusBadRequest)
				return
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &commercepb.GetOrderRequest{OrderId: orderID, CampaignId: campaignID}
			resp, err := commerceSvc.GetOrder(ctx, protoReq)
			if err != nil {
				log.Error("Failed to get order", zap.Error(err))
				http.Error(w, "failed to get order", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"order": resp.Order}); err != nil {
				log.Error("Failed to write JSON response (get_order)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "initiate_payment":
			orderID, ok := req["order_id"].(string)
			if !ok {
				log.Error("Missing or invalid order_id in initiate_payment", zap.Any("value", req["order_id"]))
				http.Error(w, "missing or invalid order_id", http.StatusBadRequest)
				return
			}
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in initiate_payment", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			amount, ok := req["amount"].(float64)
			if !ok {
				log.Error("Missing or invalid amount in initiate_payment", zap.Any("value", req["amount"]))
				http.Error(w, "missing or invalid amount", http.StatusBadRequest)
				return
			}
			currency, ok := req["currency"].(string)
			if !ok {
				log.Error("Missing or invalid currency in initiate_payment", zap.Any("value", req["currency"]))
				http.Error(w, "missing or invalid currency", http.StatusBadRequest)
				return
			}
			method, ok := req["method"].(string)
			if !ok {
				log.Error("Missing or invalid method in initiate_payment", zap.Any("value", req["method"]))
				http.Error(w, "missing or invalid method", http.StatusBadRequest)
				return
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
					http.Error(w, "invalid metadata", http.StatusBadRequest)
					return
				}
				meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &commercepb.InitiatePaymentRequest{
				OrderId:    orderID,
				UserId:     userID,
				Amount:     amount,
				Currency:   currency,
				Method:     method,
				Metadata:   meta,
				CampaignId: campaignID,
			}
			resp, err := commerceSvc.InitiatePayment(ctx, protoReq)
			if err != nil {
				log.Error("Failed to initiate payment", zap.Error(err))
				http.Error(w, "failed to initiate payment", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"payment": resp.Payment}); err != nil {
				log.Error("Failed to write JSON response (initiate_payment)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "confirm_payment":
			paymentID, ok := req["payment_id"].(string)
			if !ok {
				log.Error("Missing or invalid payment_id in confirm_payment", zap.Any("value", req["payment_id"]))
				http.Error(w, "missing or invalid payment_id", http.StatusBadRequest)
				return
			}
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in confirm_payment", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
					http.Error(w, "invalid metadata", http.StatusBadRequest)
					return
				}
				meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &commercepb.ConfirmPaymentRequest{
				PaymentId:  paymentID,
				UserId:     userID,
				Metadata:   meta,
				CampaignId: campaignID,
			}
			resp, err := commerceSvc.ConfirmPayment(ctx, protoReq)
			if err != nil {
				log.Error("Failed to confirm payment", zap.Error(err))
				http.Error(w, "failed to confirm payment", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"payment": resp.Payment}); err != nil {
				log.Error("Failed to write JSON response (confirm_payment)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "refund_payment":
			paymentID, ok := req["payment_id"].(string)
			if !ok {
				log.Error("Missing or invalid payment_id in refund_payment", zap.Any("value", req["payment_id"]))
				http.Error(w, "missing or invalid payment_id", http.StatusBadRequest)
				return
			}
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in refund_payment", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
					http.Error(w, "invalid metadata", http.StatusBadRequest)
					return
				}
				meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &commercepb.RefundPaymentRequest{
				PaymentId:  paymentID,
				UserId:     userID,
				Metadata:   meta,
				CampaignId: campaignID,
			}
			resp, err := commerceSvc.RefundPayment(ctx, protoReq)
			if err != nil {
				log.Error("Failed to refund payment", zap.Error(err))
				http.Error(w, "failed to refund payment", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"payment": resp.Payment}); err != nil {
				log.Error("Failed to write JSON response (refund_payment)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_transaction":
			transactionID, ok := req["transaction_id"].(string)
			if !ok {
				log.Error("Missing or invalid transaction_id in get_transaction", zap.Any("value", req["transaction_id"]))
				http.Error(w, "missing or invalid transaction_id", http.StatusBadRequest)
				return
			}
			protoReq := &commercepb.GetTransactionRequest{TransactionId: transactionID}
			resp, err := commerceSvc.GetTransaction(ctx, protoReq)
			if err != nil {
				log.Error("Failed to get transaction", zap.Error(err))
				http.Error(w, "failed to get transaction", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"transaction": resp.Transaction}); err != nil {
				log.Error("Failed to write JSON response (get_transaction)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_balance":
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in get_balance", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			currency, ok := req["currency"].(string)
			if !ok {
				log.Error("Missing or invalid currency in get_balance", zap.Any("value", req["currency"]))
				http.Error(w, "missing or invalid currency", http.StatusBadRequest)
				return
			}
			protoReq := &commercepb.GetBalanceRequest{
				UserId:   userID,
				Currency: currency,
			}
			resp, err := commerceSvc.GetBalance(ctx, protoReq)
			if err != nil {
				log.Error("Failed to get balance", zap.Error(err))
				http.Error(w, "failed to get balance", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"balance": resp.Balance}); err != nil {
				log.Error("Failed to write JSON response (get_balance)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_quotes":
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in list_quotes", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			page := int32(1)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &commercepb.ListQuotesRequest{
				UserId:     userID,
				Page:       page,
				PageSize:   pageSize,
				CampaignId: campaignID,
			}
			resp, err := commerceSvc.ListQuotes(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list quotes", zap.Error(err))
				http.Error(w, "failed to list quotes", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list_quotes)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_orders":
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in list_orders", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			page := int32(1)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &commercepb.ListOrdersRequest{
				UserId:     userID,
				Page:       page,
				PageSize:   pageSize,
				CampaignId: campaignID,
			}
			resp, err := commerceSvc.ListOrders(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list orders", zap.Error(err))
				http.Error(w, "failed to list orders", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list_orders)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "update_order_status":
			orderID, ok := req["order_id"].(string)
			if !ok {
				log.Error("Missing or invalid order_id in update_order_status", zap.Any("value", req["order_id"]))
				http.Error(w, "missing or invalid order_id", http.StatusBadRequest)
				return
			}
			statusVal, ok := req["status"].(string)
			if !ok {
				log.Error("Missing or invalid status in update_order_status", zap.Any("value", req["status"]))
				http.Error(w, "missing or invalid status", http.StatusBadRequest)
				return
			}
			var statusEnum commercepb.OrderStatus
			switch statusVal {
			case "PENDING":
				statusEnum = commercepb.OrderStatus_ORDER_STATUS_PENDING
			case "PAID":
				statusEnum = commercepb.OrderStatus_ORDER_STATUS_PAID
			case "SHIPPED":
				statusEnum = commercepb.OrderStatus_ORDER_STATUS_SHIPPED
			case "COMPLETED":
				statusEnum = commercepb.OrderStatus_ORDER_STATUS_COMPLETED
			case "CANCELLED":
				statusEnum = commercepb.OrderStatus_ORDER_STATUS_CANCELLED
			case "REFUNDED":
				statusEnum = commercepb.OrderStatus_ORDER_STATUS_REFUNDED
			default:
				statusEnum = commercepb.OrderStatus_ORDER_STATUS_UNSPECIFIED
			}
			protoReq := &commercepb.UpdateOrderStatusRequest{
				OrderId: orderID,
				Status:  statusEnum,
			}
			resp, err := commerceSvc.UpdateOrderStatus(ctx, protoReq)
			if err != nil {
				log.Error("Failed to update order status", zap.Error(err))
				http.Error(w, "failed to update order status", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (update_order_status)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_transactions":
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in list_transactions", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			page := int32(1)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			protoReq := &commercepb.ListTransactionsRequest{
				UserId:   userID,
				Page:     page,
				PageSize: pageSize,
			}
			resp, err := commerceSvc.ListTransactions(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list transactions", zap.Error(err))
				http.Error(w, "failed to list transactions", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list_transactions)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_balances":
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in list_balances", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			protoReq := &commercepb.ListBalancesRequest{
				UserId: userID,
			}
			resp, err := commerceSvc.ListBalances(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list balances", zap.Error(err))
				http.Error(w, "failed to list balances", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list_balances)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_events":
			entityID, ok := req["entity_id"].(string)
			if !ok {
				log.Error("Missing or invalid entity_id in list_events", zap.Any("value", req["entity_id"]))
				http.Error(w, "missing or invalid entity_id", http.StatusBadRequest)
				return
			}
			entityType, ok := req["entity_type"].(string)
			if !ok {
				log.Error("Missing or invalid entity_type in list_events", zap.Any("value", req["entity_type"]))
				http.Error(w, "missing or invalid entity_type", http.StatusBadRequest)
				return
			}
			page := int32(1)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &commercepb.ListEventsRequest{
				EntityId:   entityID,
				EntityType: entityType,
				Page:       page,
				PageSize:   pageSize,
				CampaignId: campaignID,
			}
			resp, err := commerceSvc.ListEvents(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list events", zap.Error(err))
				http.Error(w, "failed to list events", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list_events)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "create_investment_account":
			ownerID, ok := req["owner_id"].(string)
			if !ok {
				log.Error("Missing or invalid owner_id in create_investment_account", zap.Any("value", req["owner_id"]))
				http.Error(w, "missing or invalid owner_id", http.StatusBadRequest)
				return
			}
			currency, ok := req["currency"].(string)
			if !ok {
				log.Error("Missing or invalid currency in create_investment_account", zap.Any("value", req["currency"]))
				http.Error(w, "missing or invalid currency", http.StatusBadRequest)
				return
			}
			typeVal, ok := req["type"].(string)
			if !ok {
				log.Error("Missing or invalid type in request", zap.Any("value", req["type"]))
				http.Error(w, "missing or invalid type", http.StatusBadRequest)
				return
			}
			balance := 0.0
			if v, ok := req["balance"].(float64); ok {
				balance = v
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
					http.Error(w, "invalid metadata", http.StatusBadRequest)
					return
				}
				meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &commercepb.CreateInvestmentAccountRequest{
				OwnerId:    ownerID,
				Type:       typeVal,
				Currency:   currency,
				Balance:    balance,
				Metadata:   meta,
				CampaignId: campaignID,
			}
			resp, err := commerceSvc.CreateInvestmentAccount(ctx, protoReq)
			if err != nil {
				log.Error("Failed to create investment account", zap.Error(err))
				http.Error(w, "failed to create investment account", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (create_investment_account)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "place_investment_order":
			accountID, ok := req["account_id"].(string)
			if !ok {
				log.Error("Missing or invalid account_id in place_investment_order", zap.Any("value", req["account_id"]))
				http.Error(w, "missing or invalid account_id", http.StatusBadRequest)
				return
			}
			assetID, ok := req["asset_id"].(string)
			if !ok {
				log.Error("Missing or invalid asset_id in place_investment_order", zap.Any("value", req["asset_id"]))
				http.Error(w, "missing or invalid asset_id", http.StatusBadRequest)
				return
			}
			quantity, ok := req["quantity"].(float64)
			if !ok {
				log.Error("Missing or invalid quantity in place_investment_order", zap.Any("value", req["quantity"]))
				http.Error(w, "missing or invalid quantity", http.StatusBadRequest)
				return
			}
			price, ok := req["price"].(float64)
			if !ok {
				log.Error("Missing or invalid price in place_investment_order", zap.Any("value", req["price"]))
				http.Error(w, "missing or invalid price", http.StatusBadRequest)
				return
			}
			orderType, ok := req["order_type"].(string)
			if !ok {
				log.Error("Missing or invalid order_type in place_investment_order", zap.Any("value", req["order_type"]))
				http.Error(w, "missing or invalid order_type", http.StatusBadRequest)
				return
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
					http.Error(w, "invalid metadata", http.StatusBadRequest)
					return
				}
				meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &commercepb.PlaceInvestmentOrderRequest{
				AccountId:  accountID,
				AssetId:    assetID,
				Quantity:   quantity,
				Price:      price,
				OrderType:  orderType,
				Metadata:   meta,
				CampaignId: campaignID,
			}
			resp, err := commerceSvc.PlaceInvestmentOrder(ctx, protoReq)
			if err != nil {
				log.Error("Failed to place investment order", zap.Error(err))
				http.Error(w, "failed to place investment order", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (place_investment_order)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_portfolio":
			portfolioID, ok := req["portfolio_id"].(string)
			if !ok {
				log.Error("Missing or invalid portfolio_id in get_portfolio", zap.Any("value", req["portfolio_id"]))
				http.Error(w, "missing or invalid portfolio_id", http.StatusBadRequest)
				return
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &commercepb.GetPortfolioRequest{PortfolioId: portfolioID, CampaignId: campaignID}
			resp, err := commerceSvc.GetPortfolio(ctx, protoReq)
			if err != nil {
				log.Error("Failed to get portfolio", zap.Error(err))
				http.Error(w, "failed to get portfolio", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (get_portfolio)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_investment_account":
			accountID, ok := req["account_id"].(string)
			if !ok {
				log.Error("Missing or invalid account_id in get_investment_account", zap.Any("value", req["account_id"]))
				http.Error(w, "missing or invalid account_id", http.StatusBadRequest)
				return
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &commercepb.GetInvestmentAccountRequest{AccountId: accountID, CampaignId: campaignID}
			resp, err := commerceSvc.GetInvestmentAccount(ctx, protoReq)
			if err != nil {
				log.Error("Failed to get investment account", zap.Error(err))
				http.Error(w, "failed to get investment account", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (get_investment_account)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_portfolios":
			accountID, ok := req["account_id"].(string)
			if !ok {
				log.Error("Missing or invalid account_id in list_portfolios", zap.Any("value", req["account_id"]))
				http.Error(w, "missing or invalid account_id", http.StatusBadRequest)
				return
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &commercepb.ListPortfoliosRequest{AccountId: accountID, CampaignId: campaignID}
			resp, err := commerceSvc.ListPortfolios(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list portfolios", zap.Error(err))
				http.Error(w, "failed to list portfolios", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list_portfolios)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "create_exchange_pair":
			pairID, ok := req["pair_id"].(string)
			if !ok {
				log.Error("Missing or invalid pair_id in create_exchange_pair", zap.Any("value", req["pair_id"]))
				http.Error(w, "missing or invalid pair_id", http.StatusBadRequest)
				return
			}
			baseAsset, ok := req["base_asset"].(string)
			if !ok {
				log.Error("Missing or invalid base_asset in create_exchange_pair", zap.Any("value", req["base_asset"]))
				http.Error(w, "missing or invalid base_asset", http.StatusBadRequest)
				return
			}
			quoteAsset, ok := req["quote_asset"].(string)
			if !ok {
				log.Error("Missing or invalid quote_asset in create_exchange_pair", zap.Any("value", req["quote_asset"]))
				http.Error(w, "missing or invalid quote_asset", http.StatusBadRequest)
				return
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
					http.Error(w, "invalid metadata", http.StatusBadRequest)
					return
				}
				meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			resp, err := commerceSvc.CreateExchangePair(ctx, &commercepb.CreateExchangePairRequest{
				PairId:     pairID,
				BaseAsset:  baseAsset,
				QuoteAsset: quoteAsset,
				Metadata:   meta,
			})
			if err != nil {
				log.Error("Failed to create exchange pair", zap.Error(err))
				http.Error(w, "failed to create exchange pair", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"pair": resp.Pair}); err != nil {
				log.Error("Failed to write JSON response (create_exchange_pair)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_exchange_pairs":
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in list_exchange_pairs", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
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
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			resp, err := commerceSvc.ListExchangePairs(ctx, &commercepb.ListExchangePairsRequest{
				UserId:     userID,
				Page:       page,
				PageSize:   pageSize,
				CampaignId: campaignID,
			})
			if err != nil {
				log.Error("Failed to list exchange pairs", zap.Error(err))
				http.Error(w, "failed to list exchange pairs", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"pairs": resp.Pairs}); err != nil {
				log.Error("Failed to write JSON response (list_exchange_pairs)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		default:
			log.Error("Unknown action in commerce_ops", zap.Any("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
}
