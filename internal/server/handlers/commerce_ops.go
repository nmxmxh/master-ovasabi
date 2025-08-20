package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	commercepb "github.com/nmxmxh/master-ovasabi/api/protos/commerce/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/shield"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
func CommerceOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var commerceSvc commercepb.CommerceServiceServer
		if err := container.Resolve(&commerceSvc); err != nil {
			log.Error("Failed to resolve CommerceService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err) // Already correct
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		// Extract authentication context
		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		isGuest := userID == "" || (len(authCtx.Roles) == 1 && authCtx.Roles[0] == "guest")
		if isGuest {
			httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", nil)
			return
		}
		meta := shield.BuildRequestMetadata(r, userID, isGuest)
		ctx = contextx.WithMetadata(ctx, meta)
		var securitySvc securitypb.SecurityServiceClient
		if err := container.Resolve(&securitySvc); err != nil {
			log.Error("Failed to resolve SecurityService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err) // Already correct
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil, zap.Any("value", req["action"]))
			return
		}
		if err := shield.CheckPermission(ctx, securitySvc, action, "commerce", shield.WithMetadata(meta)); err != nil {
			httputil.HandleShieldError(w, log, err)
			return
		}

		actionHandlers := map[string]func(){
			"create_quote": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.CreateQuoteRequest{}, commerceSvc.CreateQuote)
			},
			"get_quote": func() { handleCommerceAction(ctx, w, log, req, &commercepb.GetQuoteRequest{}, commerceSvc.GetQuote) },
			"create_order": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.CreateOrderRequest{}, commerceSvc.CreateOrder)
			},
			"get_order": func() { handleCommerceAction(ctx, w, log, req, &commercepb.GetOrderRequest{}, commerceSvc.GetOrder) },
			"initiate_payment": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.InitiatePaymentRequest{}, commerceSvc.InitiatePayment)
			},
			"confirm_payment": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.ConfirmPaymentRequest{}, commerceSvc.ConfirmPayment)
			},
			"refund_payment": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.RefundPaymentRequest{}, commerceSvc.RefundPayment)
			},
			"get_transaction": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.GetTransactionRequest{}, commerceSvc.GetTransaction)
			},
			"get_balance": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.GetBalanceRequest{}, commerceSvc.GetBalance)
			},
			"list_quotes": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.ListQuotesRequest{}, commerceSvc.ListQuotes)
			},
			"list_orders": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.ListOrdersRequest{}, commerceSvc.ListOrders)
			},
			"update_order_status": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.UpdateOrderStatusRequest{}, commerceSvc.UpdateOrderStatus)
			},
			"list_transactions": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.ListTransactionsRequest{}, commerceSvc.ListTransactions)
			},
			"list_balances": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.ListBalancesRequest{}, commerceSvc.ListBalances)
			},
			"list_events": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.ListEventsRequest{}, commerceSvc.ListEvents)
			},
			"create_investment_account": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.CreateInvestmentAccountRequest{}, commerceSvc.CreateInvestmentAccount)
			},
			"place_investment_order": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.PlaceInvestmentOrderRequest{}, commerceSvc.PlaceInvestmentOrder)
			},
			"get_portfolio": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.GetPortfolioRequest{}, commerceSvc.GetPortfolio)
			},
			"get_investment_account": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.GetInvestmentAccountRequest{}, commerceSvc.GetInvestmentAccount)
			},
			"list_portfolios": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.ListPortfoliosRequest{}, commerceSvc.ListPortfolios)
			},
			"create_exchange_pair": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.CreateExchangePairRequest{}, commerceSvc.CreateExchangePair)
			},
			"list_exchange_pairs": func() {
				handleCommerceAction(ctx, w, log, req, &commercepb.ListExchangePairsRequest{}, commerceSvc.ListExchangePairs)
			},
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil, zap.String("action", action))
		}
	}
}

// handleCommerceAction is a generic helper to reduce boilerplate in CommerceOpsHandler.
func handleCommerceAction[T proto.Message, U proto.Message](
	ctx context.Context,
	w http.ResponseWriter,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoCommerce(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		log.Error("commerce service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil)
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProtoCommerce converts a map[string]interface{} to a proto.Message using JSON as an intermediate.
func mapToProtoCommerce(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonBytes, v)
}
