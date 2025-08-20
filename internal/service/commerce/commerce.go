package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	commercepb "github.com/nmxmxh/master-ovasabi/api/protos/commerce/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func handleQuoteAction(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	_, state := parseActionAndState(event.GetEventType())
	switch state {
	case "create":
		var req commercepb.CreateQuoteRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal quote create event payload", zap.Error(err))
				return
			}
		}
		if _, err := svc.CreateQuote(ctx, &req); err != nil {
			svc.log.Error("CreateQuote failed", zap.Error(err))
		}
	case "update":
		// Implement update logic if needed
	case "delete":
		// Implement delete logic if needed
	}
}

func handleOrderAction(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	_, state := parseActionAndState(event.GetEventType())
	switch state {
	case "create":
		var req commercepb.CreateOrderRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal order create event payload", zap.Error(err))
				return
			}
		}
		if _, err := svc.CreateOrder(ctx, &req); err != nil {
			svc.log.Error("CreateOrder failed", zap.Error(err))
		}
	case "update":
		var req commercepb.UpdateOrderStatusRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal order update event payload", zap.Error(err))
				return
			}
		}
		if _, err := svc.UpdateOrderStatus(ctx, &req); err != nil {
			svc.log.Error("UpdateOrderStatus failed", zap.Error(err))
		}
	case "delete":
		// Implement delete logic if needed
	}
}

func handlePaymentAction(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	_, state := parseActionAndState(event.GetEventType())
	switch state {
	case "initiate":
		var req commercepb.InitiatePaymentRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal payment initiate event payload", zap.Error(err))
				return
			}
		}
		if _, err := svc.InitiatePayment(ctx, &req); err != nil {
			svc.log.Error("InitiatePayment failed", zap.Error(err))
		}
	case "confirm":
		var req commercepb.ConfirmPaymentRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal payment confirm event payload", zap.Error(err))
				return
			}
		}
		if _, err := svc.ConfirmPayment(ctx, &req); err != nil {
			svc.log.Error("ConfirmPayment failed", zap.Error(err))
		}
	case "refund":
		var req commercepb.RefundPaymentRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal payment refund event payload", zap.Error(err))
				return
			}
		}
		if _, err := svc.RefundPayment(ctx, &req); err != nil {
			svc.log.Error("RefundPayment failed", zap.Error(err))
		}
	}
}

func handleTransactionAction(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	_, state := parseActionAndState(event.GetEventType())
	switch state {
	case "get":
		var req commercepb.GetTransactionRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal get transaction event payload", zap.Error(err))
				return
			}
		}
		if _, err := svc.GetTransaction(ctx, &req); err != nil {
			svc.log.Error("GetTransaction failed", zap.Error(err))
		}
	case "list":
		var req commercepb.ListTransactionsRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal list transactions event payload", zap.Error(err))
				return
			}
		}
		if _, err := svc.ListTransactions(ctx, &req); err != nil {
			svc.log.Error("ListTransactions failed", zap.Error(err))
		}
	}
}

func handlePortfolioAction(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	_, state := parseActionAndState(event.GetEventType())
	switch state {
	case "get":
		var req commercepb.GetPortfolioRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal get portfolio event payload", zap.Error(err))
				return
			}
		}
		if _, err := svc.GetPortfolio(ctx, &req); err != nil {
			svc.log.Error("GetPortfolio failed", zap.Error(err))
		}
	case "list":
		var req commercepb.ListPortfoliosRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal list portfolios event payload", zap.Error(err))
				return
			}
		}
		if _, err := svc.ListPortfolios(ctx, &req); err != nil {
			svc.log.Error("ListPortfolios failed", zap.Error(err))
		}
	}
}

func handleListingAction(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	// Reference unused ctx for diagnostics/cancellation
	if ctx != nil && ctx.Err() != nil {
		svc.log.Warn("Context error in handleListingAction", zap.Error(ctx.Err()))
	}
	_, state := parseActionAndState(event.GetEventType())
	switch state {
	case "create":
		var req commercepb.CreateListingRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal create listing event payload", zap.Error(err))
				return
			}
		}
	case "list":
		var req commercepb.ListListingsRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal list listings event payload", zap.Error(err))
				return
			}
		}
	}
}

func handleBalanceAction(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	_, state := parseActionAndState(event.GetEventType())
	switch state {
	case "get":
		var req commercepb.GetBalanceRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal get balance event payload", zap.Error(err))
				return
			}
		}
		if _, err := svc.GetBalance(ctx, &req); err != nil {
			svc.log.Error("GetBalance failed", zap.Error(err))
		}
	case "list":
		var req commercepb.ListBalancesRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal list balances event payload", zap.Error(err))
				return
			}
		}
		if _, err := svc.ListBalances(ctx, &req); err != nil {
			svc.log.Error("ListBalances failed", zap.Error(err))
		}
	}
}

func handleAssetAction(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	// Reference unused ctx for diagnostics/cancellation
	if ctx != nil && ctx.Err() != nil {
		svc.log.Warn("Context error in handleAssetAction", zap.Error(ctx.Err()))
	}
	_, state := parseActionAndState(event.GetEventType())
	if state == "list" {
		var req commercepb.ListAssetsRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal list assets event payload", zap.Error(err))
				return
			}
		}
		// Method not implemented
	}
}

func handleExchangeRateAction(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	// Reference unused ctx for diagnostics/cancellation
	if ctx != nil && ctx.Err() != nil {
		svc.log.Warn("Context error in handleExchangeRateAction", zap.Error(ctx.Err()))
	}
	_, state := parseActionAndState(event.GetEventType())
	switch state {
	case "get":
		var req commercepb.GetExchangeRateRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal get exchange rate event payload", zap.Error(err))
				return
			}
		}
		// Method not implemented
	case "create":
		var req commercepb.CreateExchangeRateRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal create exchange rate event payload", zap.Error(err))
				return
			}
		}
		// Method not implemented
	}
}

func handleOfferAction(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	// Reference unused ctx for diagnostics/cancellation
	if ctx != nil && ctx.Err() != nil {
		svc.log.Warn("Context error in handleOfferAction", zap.Error(ctx.Err()))
	}
	_, state := parseActionAndState(event.GetEventType())
	if state == "make" {
		var req commercepb.MakeOfferRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal make offer event payload", zap.Error(err))
				return
			}
		}
		// Method not implemented
	}
}

func handleEventAction(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	_, state := parseActionAndState(event.GetEventType())
	if state == "list" {
		var req commercepb.ListEventsRequest
		if event.Payload != nil && event.Payload.Data != nil {
			b, err := protojson.Marshal(event.Payload.Data)
			if err == nil {
				err = protojson.Unmarshal(b, &req)
			}
			if err != nil {
				svc.log.Error("Failed to unmarshal list events event payload", zap.Error(err))
				return
			}
		}
		if _, err := svc.ListEvents(ctx, &req); err != nil {
			svc.log.Error("ListEvents failed", zap.Error(err))
		}
	}
}

// Register all canonical commerce action handlers.
func init() {
	RegisterActionHandler("quote", handleQuoteAction)
	RegisterActionHandler("order", handleOrderAction)
	RegisterActionHandler("payment", handlePaymentAction)
	RegisterActionHandler("transaction", handleTransactionAction)
	RegisterActionHandler("portfolio", handlePortfolioAction)
	RegisterActionHandler("listing", handleListingAction)
	RegisterActionHandler("balance", handleBalanceAction)
	RegisterActionHandler("asset", handleAssetAction)
	RegisterActionHandler("exchange_rate", handleExchangeRateAction)
	RegisterActionHandler("offer", handleOfferAction)
	RegisterActionHandler("event", handleEventAction)
}

// Provider/DI Registration Pattern (Modern, Extensible, DRY)
// ---------------------------------------------------------
// This file implements the centralized Provider pattern for service registration and dependency injection (DI) across the platform.
// It also implements the Graceful Orchestration Standard for error and success handling, as required by the OVASABI platform.
// See docs/amadeus/amadeus_context.md for details.

type Service struct {
	commercepb.UnimplementedCommerceServiceServer
	log          *zap.Logger
	repo         Repository
	Cache        *redis.Cache
	eventEmitter events.EventEmitter
	eventEnabled bool
	handler      *graceful.Handler
}

func NewService(log *zap.Logger, repo Repository, cache *redis.Cache, eventEmitter events.EventEmitter, eventEnabled bool) commercepb.CommerceServiceServer {
	graceful.RegisterErrorMap(map[error]graceful.ErrorMapEntry{
		sql.ErrNoRows: {Code: codes.NotFound, Message: "not found"},
		// Add more domain-specific errors here as needed
	})
	return &Service{
		log:          log,
		repo:         repo,
		Cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
		handler:      graceful.NewHandler(log, eventEmitter, cache, "commerce", "v1", eventEnabled),
	}
}

// generateQuoteID generates a unique quote ID based on userID, productID, and timestamp.
func generateQuoteID(userID, productID string) string {
	return userID + ":" + productID + ":" + time.Now().Format("20060102150405.000")
}

// Helper to convert string to QuoteStatus enum.
func toQuoteStatus(statusStr string) commercepb.QuoteStatus {
	switch statusStr {
	case "PENDING":
		return commercepb.QuoteStatus_QUOTE_STATUS_PENDING
	case "ACCEPTED":
		return commercepb.QuoteStatus_QUOTE_STATUS_ACCEPTED
	case "REJECTED":
		return commercepb.QuoteStatus_QUOTE_STATUS_REJECTED
	case "EXPIRED":
		return commercepb.QuoteStatus_QUOTE_STATUS_EXPIRED
	default:
		return commercepb.QuoteStatus_QUOTE_STATUS_UNSPECIFIED
	}
}

// Helper to convert string to OrderStatus enum.
func toOrderStatus(statusStr string) commercepb.OrderStatus {
	switch statusStr {
	case "PENDING":
		return commercepb.OrderStatus_ORDER_STATUS_PENDING
	case "PAID":
		return commercepb.OrderStatus_ORDER_STATUS_PAID
	case "SHIPPED":
		return commercepb.OrderStatus_ORDER_STATUS_SHIPPED
	case "COMPLETED":
		return commercepb.OrderStatus_ORDER_STATUS_COMPLETED
	case "CANCELLED":
		return commercepb.OrderStatus_ORDER_STATUS_CANCELLED
	case "REFUNDED":
		return commercepb.OrderStatus_ORDER_STATUS_REFUNDED
	default:
		return commercepb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

// Helper to convert string to TransactionType enum.
func toTransactionType(t string) commercepb.TransactionType {
	switch t {
	case "DEBIT":
		return commercepb.TransactionType_TRANSACTION_TYPE_DEBIT
	case "CREDIT":
		return commercepb.TransactionType_TRANSACTION_TYPE_CREDIT
	default:
		return commercepb.TransactionType_TRANSACTION_TYPE_UNSPECIFIED
	}
}

// Helper to convert string to TransactionStatus enum.
func toTransactionStatus(s string) commercepb.TransactionStatus {
	switch s {
	case "PENDING":
		return commercepb.TransactionStatus_TRANSACTION_STATUS_PENDING
	case "COMPLETED":
		return commercepb.TransactionStatus_TRANSACTION_STATUS_COMPLETED
	case "FAILED":
		return commercepb.TransactionStatus_TRANSACTION_STATUS_FAILED
	default:
		return commercepb.TransactionStatus_TRANSACTION_STATUS_UNSPECIFIED
	}
}

func (s *Service) CreateQuote(ctx context.Context, req *commercepb.CreateQuoteRequest) (*commercepb.CreateQuoteResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "create_quote", codes.InvalidArgument, "request is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	meta := req.Metadata
	quote := &Quote{
		QuoteID:    generateQuoteID(req.UserId, req.ProductId),
		UserID:     req.UserId,
		ProductID:  req.ProductId,
		Amount:     req.Amount,
		Currency:   req.Currency,
		Status:     "PENDING",
		Metadata:   meta,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		CampaignID: req.CampaignId,
	}
	if err := s.repo.CreateQuote(ctx, quote); err != nil {
		s.handler.Error(ctx, "create_quote", codes.Internal, "failed to create quote", err, meta, quote.QuoteID)
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.CreateQuoteResponse{
		Quote: &commercepb.Quote{
			QuoteId:    quote.QuoteID,
			UserId:     quote.UserID,
			ProductId:  quote.ProductID,
			Amount:     quote.Amount,
			Currency:   quote.Currency,
			Status:     toQuoteStatus(quote.Status),
			Metadata:   quote.Metadata,
			CreatedAt:  timestamppb.New(quote.CreatedAt),
			UpdatedAt:  timestamppb.New(quote.UpdatedAt),
			CampaignId: quote.CampaignID,
		},
	}
	s.handler.Success(ctx, "create_quote", codes.OK, "quote created", resp, quote.Metadata, quote.QuoteID, nil)
	return resp, nil
}

func (s *Service) GetQuote(ctx context.Context, req *commercepb.GetQuoteRequest) (*commercepb.GetQuoteResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "get_quote", codes.InvalidArgument, "request is required", err, nil, req.GetQuoteId())
		return nil, graceful.ToStatusError(err)
	}
	quote, err := s.repo.GetQuote(ctx, req.QuoteId)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to get quote", err)
		s.handler.Error(ctx, "get_quote", codes.Internal, "failed to get quote", err, nil, req.GetQuoteId())
		return nil, graceful.ToStatusError(err)
	}
	if quote == nil {
		err := graceful.WrapErr(ctx, codes.NotFound, "quote not found", nil)
		s.handler.Error(ctx, "get_quote", codes.NotFound, "quote not found", err, nil, req.GetQuoteId())
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.GetQuoteResponse{
		Quote: &commercepb.Quote{
			QuoteId:   quote.QuoteID,
			UserId:    quote.UserID,
			ProductId: quote.ProductID,
			Amount:    quote.Amount,
			Currency:  quote.Currency,
			Status:    toQuoteStatus(quote.Status),
			Metadata:  quote.Metadata,
			CreatedAt: timestamppb.New(quote.CreatedAt),
			UpdatedAt: timestamppb.New(quote.UpdatedAt),
		},
	}
	s.handler.Success(ctx, "get_quote", codes.OK, "quote fetched", resp, quote.Metadata, quote.QuoteID, nil)
	return resp, nil
}

func (s *Service) ListQuotes(ctx context.Context, req *commercepb.ListQuotesRequest) (*commercepb.ListQuotesResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "list_quotes", codes.InvalidArgument, "request is required", err, nil, req.GetUserId())
		return nil, graceful.ToStatusError(err)
	}
	quotes, total, err := s.repo.ListQuotes(ctx, req.UserId, req.CampaignId, int(req.Page), int(req.PageSize))
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to list quotes", err)
		s.handler.Error(ctx, "list_quotes", codes.Internal, "failed to list quotes", err, nil, req.GetUserId())
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.ListQuotesResponse{
		Quotes: make([]*commercepb.Quote, len(quotes)),
		Total:  utils.ToInt32(total),
	}
	for i, q := range quotes {
		resp.Quotes[i] = &commercepb.Quote{
			QuoteId:   q.QuoteID,
			UserId:    q.UserID,
			ProductId: q.ProductID,
			Amount:    q.Amount,
			Currency:  q.Currency,
			Status:    toQuoteStatus(q.Status),
			Metadata:  q.Metadata,
			CreatedAt: timestamppb.New(q.CreatedAt),
			UpdatedAt: timestamppb.New(q.UpdatedAt),
		}
	}
	s.handler.Success(ctx, "list_quotes", codes.OK, "quotes listed", resp, nil, req.GetUserId(), nil)
	return resp, nil
}

// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
func (s *Service) CreateOrder(ctx context.Context, req *commercepb.CreateOrderRequest) (*commercepb.CreateOrderResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "create_order", codes.InvalidArgument, "request is required", err, nil, req.GetUserId())
		return nil, graceful.ToStatusError(err)
	}
	if req.Metadata == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "metadata is required", nil)
		s.handler.Error(ctx, "create_order", codes.InvalidArgument, "metadata is required", err, nil, req.GetUserId())
		return nil, graceful.ToStatusError(err)
	}
	meta := req.Metadata
	orderID := req.UserId + ":order:" + time.Now().Format("20060102150405.000")
	order := &Order{
		OrderID:    orderID,
		UserID:     req.UserId,
		Total:      0,
		Currency:   req.Currency,
		Status:     "PENDING",
		Metadata:   meta,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		CampaignID: req.CampaignId, // Pass CampaignId from request
	}
	repoItems := make([]OrderItem, len(req.Items))
	for i, item := range req.Items {
		repoItems[i] = OrderItem{
			ProductID: item.ProductId,
			Quantity:  int(item.Quantity),
			Price:     item.Price,
			Metadata:  item.Metadata,
		}
	}
	for _, item := range req.Items {
		order.Total += item.Price * float64(item.Quantity)
	}
	if err := s.repo.CreateOrder(ctx, order, repoItems); err != nil {
		s.handler.Error(ctx, "create_order", codes.Internal, "failed to create order", err, meta, orderID)
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.CreateOrderResponse{
		Order: &commercepb.Order{
			OrderId:    order.OrderID,
			UserId:     order.UserID,
			Total:      order.Total,
			Currency:   order.Currency,
			Status:     toOrderStatus(order.Status),
			Metadata:   order.Metadata,
			CreatedAt:  timestamppb.New(order.CreatedAt),
			CampaignId: order.CampaignID, // Include CampaignId in response
			UpdatedAt:  timestamppb.New(order.UpdatedAt),
		},
	}
	s.handler.Success(ctx, "create_order", codes.OK, "order created", resp, order.Metadata, order.OrderID, nil)
	return resp, nil
}

func (s *Service) GetOrder(ctx context.Context, req *commercepb.GetOrderRequest) (*commercepb.GetOrderResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "get_order", codes.InvalidArgument, "request is required", err, nil, req.GetOrderId())
		return nil, graceful.ToStatusError(err)
	}
	order, err := s.repo.GetOrder(ctx, req.OrderId)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to get order", err)
		s.handler.Error(ctx, "get_order", codes.Internal, "failed to get order", err, nil, req.GetOrderId())
		return nil, graceful.ToStatusError(err)
	}
	if order == nil {
		err := graceful.WrapErr(ctx, codes.NotFound, "order not found", nil)
		s.handler.Error(ctx, "get_order", codes.NotFound, "order not found", err, nil, req.GetOrderId())
		return nil, graceful.ToStatusError(err)
	}
	// Load order items
	orderItems, err := s.repo.ListOrderItems(ctx, order.OrderID)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to list order items", err)
		s.handler.Error(ctx, "commerce:order:v1:failed", codes.Internal, "failed to list order items", err, nil, order.OrderID)
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.GetOrderResponse{
		Order: &commercepb.Order{
			OrderId:    order.OrderID,
			UserId:     order.UserID,
			Total:      order.Total,
			Currency:   order.Currency,
			Status:     toOrderStatus(order.Status),
			Metadata:   order.Metadata,
			CreatedAt:  timestamppb.New(order.CreatedAt),
			CampaignId: order.CampaignID, // Include CampaignId in response
			Items: func() []*commercepb.OrderItem {
				protoItems := make([]*commercepb.OrderItem, len(orderItems))
				for i, item := range orderItems {
					var safeQty int32
					const int32Max = int32(^uint32(0) >> 1)
					const int32Min = -int32Max - 1
					switch {
					case item.Quantity > int(int32Max):
						safeQty = int32Max
					case item.Quantity < int(int32Min):
						safeQty = int32Min
					default:
						safeQty = int32(item.Quantity)
					}
					protoItems[i] = &commercepb.OrderItem{
						ProductId: item.ProductID,
						Quantity:  safeQty,
						Price:     item.Price,
					}
				}
				return protoItems
			}(),
			UpdatedAt: timestamppb.New(order.UpdatedAt),
		},
	}
	s.handler.Success(ctx, "get_order", codes.OK, "order fetched", resp, order.Metadata, order.OrderID, nil)
	return resp, nil
}

func (s *Service) ListOrders(ctx context.Context, req *commercepb.ListOrdersRequest) (*commercepb.ListOrdersResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "list_orders", codes.InvalidArgument, "request is required", err, nil, req.GetUserId())
		return nil, graceful.ToStatusError(err)
	}
	orders, total, err := s.repo.ListOrders(ctx, req.UserId, req.CampaignId, int(req.Page), int(req.PageSize))
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to list orders", err)
		s.handler.Error(ctx, "list_orders", codes.Internal, "failed to list orders", err, nil, req.GetUserId())
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.ListOrdersResponse{
		Orders: make([]*commercepb.Order, len(orders)),
		Total:  utils.ToInt32(total),
	}
	for i, order := range orders {
		resp.Orders[i] = &commercepb.Order{
			OrderId:    order.OrderID,
			UserId:     order.UserID,
			Total:      order.Total,
			Currency:   order.Currency,
			Status:     toOrderStatus(order.Status),
			Metadata:   order.Metadata,
			CreatedAt:  timestamppb.New(order.CreatedAt),
			CampaignId: order.CampaignID, // Include CampaignId in response
			UpdatedAt:  timestamppb.New(order.UpdatedAt),
		}
	}
	s.handler.Success(ctx, "list_orders", codes.OK, "orders listed", resp, nil, req.GetUserId(), nil)
	return resp, nil
}

// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
func (s *Service) UpdateOrderStatus(ctx context.Context, req *commercepb.UpdateOrderStatusRequest) (*commercepb.UpdateOrderStatusResponse, error) {
	log := s.log.With(zap.String("operation", "update_order_status"), zap.String("order_id", req.GetOrderId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.Status == commercepb.OrderStatus_ORDER_STATUS_UNSPECIFIED {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "status is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	err := s.repo.UpdateOrderStatus(ctx, req.OrderId, req.Status.String())
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to update order status", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	order, err := s.repo.GetOrder(ctx, req.OrderId)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to fetch updated order", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.UpdateOrderStatusResponse{
		Order: &commercepb.Order{
			OrderId:   order.OrderID,
			UserId:    order.UserID,
			Total:     order.Total,
			Currency:  order.Currency,
			Status:    toOrderStatus(order.Status),
			Metadata:  order.Metadata,
			CreatedAt: timestamppb.New(order.CreatedAt),
			UpdatedAt: timestamppb.New(order.UpdatedAt),
		},
	}
	s.handler.Success(ctx, "update_order_status", codes.OK, "order status updated", resp, order.Metadata, order.OrderID, nil)
	return resp, nil
}

func (s *Service) InitiatePayment(ctx context.Context, req *commercepb.InitiatePaymentRequest) (*commercepb.InitiatePaymentResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "initiate_payment", codes.InvalidArgument, "request is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	if req.OrderId == "" || req.UserId == "" || req.Amount <= 0 || req.Currency == "" || req.Method == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid payment fields", nil)
		s.handler.Error(ctx, "initiate_payment", codes.InvalidArgument, "missing or invalid payment fields", err, nil, req.GetOrderId())
		return nil, graceful.ToStatusError(err)
	}
	meta := req.Metadata
	payment := &Payment{
		PaymentID:  req.OrderId + ":payment:" + time.Now().Format("20060102150405.000"),
		OrderID:    req.OrderId,
		UserID:     req.UserId,
		Amount:     req.Amount,
		Currency:   req.Currency,
		Method:     req.Method,
		Status:     "PENDING",
		Metadata:   meta,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		CampaignID: req.CampaignId, // Pass CampaignId from request
	}
	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		s.handler.Error(ctx, "initiate_payment", codes.Internal, "failed to create payment", err, meta, payment.PaymentID)
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.InitiatePaymentResponse{
		Payment: &commercepb.Payment{
			PaymentId:  payment.PaymentID,
			OrderId:    payment.OrderID,
			UserId:     payment.UserID,
			Amount:     payment.Amount,
			Currency:   payment.Currency,
			Method:     payment.Method,
			Status:     commercepb.PaymentStatus_PAYMENT_STATUS_PENDING,
			Metadata:   payment.Metadata,
			CreatedAt:  timestamppb.New(payment.CreatedAt),
			CampaignId: payment.CampaignID, // Include CampaignId in response
			UpdatedAt:  timestamppb.New(payment.UpdatedAt),
		},
	}
	s.handler.Success(ctx, "initiate_payment", codes.OK, "payment initiated", resp, payment.Metadata, payment.PaymentID, nil)
	return resp, nil
}

func (s *Service) ConfirmPayment(ctx context.Context, req *commercepb.ConfirmPaymentRequest) (*commercepb.ConfirmPaymentResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "confirm_payment", codes.InvalidArgument, "request is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	if req.PaymentId == "" || req.UserId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid payment fields", nil)
		s.handler.Error(ctx, "confirm_payment", codes.InvalidArgument, "missing or invalid payment fields", err, nil, req.GetPaymentId())
		return nil, graceful.ToStatusError(err)
	}
	_ = metadata.ExtractServiceVariables(req.Metadata, "commerce")
	err := s.repo.UpdatePaymentStatus(ctx, req.PaymentId, "SUCCEEDED")
	if err != nil {
		s.handler.Error(ctx, "confirm_payment", codes.Internal, "failed to update payment status", err, nil, req.GetPaymentId())
		return nil, graceful.ToStatusError(err)
	}
	payment, err := s.repo.GetPayment(ctx, req.PaymentId)
	if err != nil {
		s.handler.Error(ctx, "confirm_payment", codes.Internal, "failed to fetch payment", err, nil, req.GetPaymentId())
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.ConfirmPaymentResponse{
		Payment: &commercepb.Payment{
			PaymentId:  payment.PaymentID,
			OrderId:    payment.OrderID,
			UserId:     payment.UserID,
			Amount:     payment.Amount,
			Currency:   payment.Currency,
			Method:     payment.Method,
			Status:     commercepb.PaymentStatus_PAYMENT_STATUS_SUCCEEDED,
			Metadata:   payment.Metadata,
			CreatedAt:  timestamppb.New(payment.CreatedAt),
			CampaignId: payment.CampaignID, // Include CampaignId in response
			UpdatedAt:  timestamppb.New(payment.UpdatedAt),
		},
	}
	s.handler.Success(ctx, "confirm_payment", codes.OK, "payment confirmed", resp, payment.Metadata, payment.PaymentID, nil)
	return resp, nil
}

func (s *Service) RefundPayment(ctx context.Context, req *commercepb.RefundPaymentRequest) (*commercepb.RefundPaymentResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "refund_payment", codes.InvalidArgument, "request is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	if req.PaymentId == "" || req.UserId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid payment fields", nil)
		s.handler.Error(ctx, "refund_payment", codes.InvalidArgument, "missing or invalid payment fields", err, nil, req.GetPaymentId())
		return nil, graceful.ToStatusError(err)
	}
	_ = metadata.ExtractServiceVariables(req.Metadata, "commerce")
	err := s.repo.UpdatePaymentStatus(ctx, req.PaymentId, "REFUNDED")
	if err != nil {
		s.handler.Error(ctx, "refund_payment", codes.Internal, "failed to update payment status", err, nil, req.GetPaymentId())
		return nil, graceful.ToStatusError(err)
	}
	payment, err := s.repo.GetPayment(ctx, req.PaymentId)
	if err != nil {
		s.handler.Error(ctx, "refund_payment", codes.Internal, "failed to fetch payment", err, nil, req.GetPaymentId())
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.RefundPaymentResponse{
		Payment: &commercepb.Payment{
			PaymentId:  payment.PaymentID,
			OrderId:    payment.OrderID,
			UserId:     payment.UserID,
			Amount:     payment.Amount,
			Currency:   payment.Currency,
			Method:     payment.Method,
			Status:     commercepb.PaymentStatus_PAYMENT_STATUS_REFUNDED,
			Metadata:   payment.Metadata,
			CreatedAt:  timestamppb.New(payment.CreatedAt),
			CampaignId: payment.CampaignID, // Include CampaignId in response
			UpdatedAt:  timestamppb.New(payment.UpdatedAt),
		},
	}
	s.handler.Success(ctx, "refund_payment", codes.OK, "payment refunded", resp, payment.Metadata, payment.PaymentID, nil)
	return resp, nil
}

func (s *Service) GetTransaction(ctx context.Context, req *commercepb.GetTransactionRequest) (*commercepb.GetTransactionResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "get_transaction", codes.InvalidArgument, "request is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	if req.TransactionId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "transaction_id is required", nil)
		s.handler.Error(ctx, "get_transaction", codes.InvalidArgument, "transaction_id is required", err, nil, req.GetTransactionId())
		return nil, graceful.ToStatusError(err)
	}
	transaction, err := s.repo.GetTransaction(ctx, req.TransactionId)
	if err != nil {
		s.handler.Error(ctx, "get_transaction", codes.Internal, "failed to get transaction", err, nil, req.GetTransactionId())
		return nil, graceful.ToStatusError(err)
	}
	if transaction == nil {
		err := graceful.WrapErr(ctx, codes.NotFound, "transaction not found", nil)
		s.handler.Error(ctx, "get_transaction", codes.NotFound, "transaction not found", err, nil, req.GetTransactionId())
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.GetTransactionResponse{
		Transaction: &commercepb.Transaction{
			TransactionId: transaction.TransactionID,
			PaymentId:     transaction.PaymentID,
			UserId:        transaction.UserID,
			Type:          toTransactionType(transaction.Type),
			Amount:        transaction.Amount,
			Currency:      transaction.Currency,
			Status:        toTransactionStatus(transaction.Status),
			Metadata:      transaction.Metadata,
			CreatedAt:     timestamppb.New(transaction.CreatedAt),
			CampaignId:    transaction.CampaignID, // Include CampaignId in response
			UpdatedAt:     timestamppb.New(transaction.UpdatedAt),
		},
	}
	s.handler.Success(ctx, "get_transaction", codes.OK, "transaction fetched", resp, transaction.Metadata, transaction.TransactionID, nil)
	return resp, nil
}

func (s *Service) ListTransactions(ctx context.Context, req *commercepb.ListTransactionsRequest) (*commercepb.ListTransactionsResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "list_transactions", codes.InvalidArgument, "request is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	if req.UserId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "user_id is required", nil)
		s.handler.Error(ctx, "list_transactions", codes.InvalidArgument, "user_id is required", err, nil, req.GetUserId())
		return nil, graceful.ToStatusError(err)
	}
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 20
	}
	transactions, total, err := s.repo.ListTransactions(ctx, req.UserId, req.CampaignId, page, pageSize)
	if err != nil {
		s.handler.Error(ctx, "list_transactions", codes.Internal, "failed to list transactions", err, nil, req.GetUserId())
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.ListTransactionsResponse{
		Transactions: make([]*commercepb.Transaction, len(transactions)),
		Total:        utils.ToInt32(total),
	}
	for i, tx := range transactions {
		resp.Transactions[i] = &commercepb.Transaction{
			TransactionId: tx.TransactionID,
			PaymentId:     tx.PaymentID,
			UserId:        tx.UserID,
			Type:          toTransactionType(tx.Type),
			Amount:        tx.Amount,
			Currency:      tx.Currency,
			Status:        toTransactionStatus(tx.Status),
			Metadata:      tx.Metadata,
			CreatedAt:     timestamppb.New(tx.CreatedAt),
			CampaignId:    tx.CampaignID, // Include CampaignId in response
			UpdatedAt:     timestamppb.New(tx.UpdatedAt),
		}
	}
	s.handler.Success(ctx, "list_transactions", codes.OK, "transactions listed", resp, nil, req.GetUserId(), nil)
	return resp, nil
}

func (s *Service) GetBalance(ctx context.Context, req *commercepb.GetBalanceRequest) (*commercepb.GetBalanceResponse, error) {
	log := s.log.With(zap.String("operation", "get_balance"), zap.String("user_id", req.GetUserId()), zap.String("currency", req.GetCurrency()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.UserId == "" || req.Currency == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "user_id and currency are required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	balance, err := s.repo.GetBalance(ctx, req.UserId, req.Currency)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to get balance", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if balance == nil {
		err := graceful.WrapErr(ctx, codes.NotFound, "balance not found", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.GetBalanceResponse{
		Balance: &commercepb.Balance{
			UserId:     balance.UserID,
			Currency:   balance.Currency,
			Amount:     balance.Amount,
			UpdatedAt:  timestamppb.New(balance.UpdatedAt),
			Metadata:   balance.Metadata,
			CampaignId: balance.CampaignID, // Include CampaignId in response
		},
	}
	s.handler.Success(ctx, "get_balance", codes.OK, "balance fetched", resp, balance.Metadata, balance.UserID+":"+balance.Currency, nil)
	return resp, nil
}

func (s *Service) ListBalances(ctx context.Context, req *commercepb.ListBalancesRequest) (*commercepb.ListBalancesResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "commerce:balance:v1:failed", codes.InvalidArgument, "request is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	if req.UserId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "user_id is required", nil)
		s.handler.Error(ctx, "commerce:balance:v1:failed", codes.InvalidArgument, "user_id is required", err, nil, req.GetUserId())
		return nil, graceful.ToStatusError(err)
	}
	balances, err := s.repo.ListBalances(ctx, req.UserId)
	if err != nil {
		s.handler.Error(ctx, "commerce:balance:v1:failed", codes.Internal, "failed to list balances", err, nil, req.GetUserId())
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.ListBalancesResponse{
		Balances: make([]*commercepb.Balance, len(balances)),
	}
	for i, b := range balances {
		resp.Balances[i] = &commercepb.Balance{
			UserId:     b.UserID,
			Currency:   b.Currency,
			Amount:     b.Amount,
			UpdatedAt:  timestamppb.New(b.UpdatedAt),
			Metadata:   b.Metadata,
			CampaignId: b.CampaignID, // Include CampaignId in response
		}
	}
	s.handler.Success(ctx, "list_balances", codes.OK, "balances listed", resp, nil, req.GetUserId(), nil)
	return resp, nil
}

func (s *Service) ListEvents(ctx context.Context, req *commercepb.ListEventsRequest) (*commercepb.ListEventsResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "commerce:list_events:v1:failed", codes.InvalidArgument, "request is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	if req.EntityId == "" || req.EntityType == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "entity_id and entity_type are required", nil)
		s.handler.Error(ctx, "commerce:list_events:v1:failed", codes.InvalidArgument, "entity_id and entity_type are required", err, nil, req.GetEntityId())
		return nil, graceful.ToStatusError(err)
	}
	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page < 1 { // Ensure page is at least 1
		page = 1
	}
	if pageSize < 1 { // Ensure pageSize is at least 1
		pageSize = 20
	}
	eventList, total, err := s.repo.ListEvents(ctx, req.EntityId, req.EntityType, req.CampaignId, page, pageSize) // Pass CampaignId
	if err != nil {
		s.handler.Error(ctx, "commerce:list_events:v1:failed", codes.Internal, "failed to list events", err, nil, req.GetEntityId())
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.ListEventsResponse{
		Events: make([]*commercepb.CommerceEvent, len(eventList)),
		Total:  utils.ToInt32(total),
	}
	for i, e := range eventList {
		resp.Events[i] = &commercepb.CommerceEvent{
			EventId:    fmt.Sprintf("%d", e.ID),
			EntityId:   e.EntityID,
			EntityType: e.EntityType,
			EventType:  e.EventType,
			Payload:    toProtoStruct(e.Payload),
			CreatedAt:  timestamppb.New(e.CreatedAt),
			CampaignId: e.CampaignID, // Include CampaignId in response
			Metadata:   e.Metadata,
		}
	}
	s.handler.Success(ctx, "list_events", codes.OK, "events listed", resp, nil, req.GetEntityId(), nil)
	return resp, nil
}

// --- Investment ---.
func (s *Service) CreateInvestmentAccount(ctx context.Context, req *commercepb.CreateInvestmentAccountRequest) (*commercepb.CreateInvestmentAccountResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "commerce:investment_account:v1:failed", codes.InvalidArgument, "request is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	if req.OwnerId == "" || req.Currency == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "owner_id and currency are required", nil)
		s.handler.Error(ctx, "commerce:investment_account:v1:failed", codes.InvalidArgument, "owner_id and currency are required", err, nil, req.GetOwnerId())
		return nil, graceful.ToStatusError(err)
	}
	meta := req.Metadata
	accountID := req.OwnerId + ":investment_account:" + time.Now().Format("20060102150405.000")
	account := &InvestmentAccount{
		AccountID:  accountID,
		OwnerID:    req.OwnerId,
		Type:       req.Type,
		Currency:   req.Currency,
		Balance:    req.Balance,
		Metadata:   meta,
		CreatedAt:  time.Now(),
		CampaignID: req.CampaignId, // Pass CampaignId from request
		UpdatedAt:  time.Now(),
	}
	if err := s.repo.CreateInvestmentAccount(ctx, account); err != nil {
		s.handler.Error(ctx, "create_investment_account", codes.Internal, "failed to create investment account", err, meta, accountID)
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.CreateInvestmentAccountResponse{
		Account: &commercepb.InvestmentAccount{
			AccountId:  account.AccountID,
			OwnerId:    account.OwnerID,
			Type:       account.Type,
			Currency:   account.Currency,
			Balance:    account.Balance,
			CampaignId: account.CampaignID, // Include CampaignId in response
			Metadata:   account.Metadata,
		},
	}
	s.handler.Success(ctx, "create_investment_account", codes.OK, "investment account created", resp, account.Metadata, account.AccountID, nil)
	return resp, nil
}

func (s *Service) PlaceInvestmentOrder(ctx context.Context, req *commercepb.PlaceInvestmentOrderRequest) (*commercepb.PlaceInvestmentOrderResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "commerce:investment_order:v1:failed", codes.InvalidArgument, "request is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	if req.AccountId == "" || req.AssetId == "" || req.Quantity <= 0 || req.Price <= 0 || req.OrderType == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "account_id, asset_id, quantity, price, and order_type are required", nil)
		s.handler.Error(ctx, "commerce:investment_order:v1:failed", codes.InvalidArgument, "account_id, asset_id, quantity, price, and order_type are required", err, nil, req.GetAccountId())
		return nil, graceful.ToStatusError(err)
	}
	meta := req.Metadata
	orderID := req.AccountId + ":investment_order:" + time.Now().Format("20060102150405.000")
	order := &InvestmentOrder{
		OrderID:    orderID,
		AccountID:  req.AccountId,
		AssetID:    req.AssetId,
		Quantity:   req.Quantity,
		Price:      req.Price,
		OrderType:  req.OrderType,
		Status:     1,
		Metadata:   meta,
		CreatedAt:  time.Now(),
		CampaignID: req.CampaignId, // Pass CampaignId from request
		UpdatedAt:  time.Now(),
	}
	if err := s.repo.CreateInvestmentOrder(ctx, order); err != nil {
		s.handler.Error(ctx, "place_investment_order", codes.Internal, "failed to create investment order", err, meta, orderID)
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.PlaceInvestmentOrderResponse{
		Order: &commercepb.InvestmentOrder{
			OrderId:    order.OrderID,
			AccountId:  order.AccountID,
			AssetId:    order.AssetID,
			Quantity:   order.Quantity,
			Price:      order.Price,
			OrderType:  order.OrderType,
			Status:     commercepb.InvestmentOrderStatus_INVESTMENT_ORDER_STATUS_PENDING,
			CampaignId: order.CampaignID, // Include CampaignId in response
			Metadata:   order.Metadata,
			CreatedAt:  timestamppb.New(order.CreatedAt),
		},
	}
	s.handler.Success(ctx, "place_investment_order", codes.OK, "investment order placed", resp, order.Metadata, order.OrderID, nil)
	return resp, nil
}

func (s *Service) GetPortfolio(ctx context.Context, req *commercepb.GetPortfolioRequest) (*commercepb.GetPortfolioResponse, error) {
	if req == nil || req.PortfolioId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "portfolio_id is required", nil)
		s.handler.Error(ctx, "commerce:portfolio:v1:failed", codes.InvalidArgument, "portfolio_id is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	portfolio, err := s.repo.GetPortfolio(ctx, req.PortfolioId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.handler.Error(ctx, "commerce:portfolio:v1:failed", codes.NotFound, "portfolio not found", err, nil, req.GetPortfolioId())
			return nil, graceful.ToStatusError(err)
		}
		s.handler.Error(ctx, "commerce:portfolio:v1:failed", codes.Internal, "failed to get portfolio", err, nil, req.GetPortfolioId())
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.GetPortfolioResponse{
		Portfolio: &commercepb.Portfolio{
			PortfolioId: portfolio.PortfolioID,
			AccountId:   portfolio.AccountID,
			Metadata:    portfolio.Metadata,
			CreatedAt:   timestamppb.New(portfolio.CreatedAt),
			CampaignId:  portfolio.CampaignID, // Include CampaignId in response
			UpdatedAt:   timestamppb.New(portfolio.UpdatedAt),
		},
	}
	s.handler.Success(ctx, "get_portfolio", codes.OK, "portfolio fetched", resp, portfolio.Metadata, portfolio.PortfolioID, nil)
	return resp, nil
}

// --- Investment/Account/Asset Service Methods ---.
func (s *Service) GetInvestmentAccount(ctx context.Context, req *commercepb.GetInvestmentAccountRequest) (*commercepb.GetInvestmentAccountResponse, error) {
	if req == nil || req.AccountId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "account_id is required", nil)
		s.handler.Error(ctx, "commerce:investment_account:v1:failed", codes.InvalidArgument, "account_id is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	account, err := s.repo.GetInvestmentAccount(ctx, req.AccountId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.handler.Error(ctx, "commerce:investment_account:v1:failed", codes.NotFound, "investment account not found", err, nil, req.GetAccountId())
			return nil, graceful.ToStatusError(err)
		}
		s.handler.Error(ctx, "commerce:investment_account:v1:failed", codes.Internal, "failed to get investment account", err, nil, req.GetAccountId())
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.GetInvestmentAccountResponse{
		Account: &commercepb.InvestmentAccount{
			AccountId:  account.AccountID,
			OwnerId:    account.OwnerID,
			Type:       account.Type,
			Currency:   account.Currency,
			Balance:    account.Balance,
			Metadata:   account.Metadata,
			CreatedAt:  timestamppb.New(account.CreatedAt),
			CampaignId: account.CampaignID, // Include CampaignId in response
			UpdatedAt:  timestamppb.New(account.UpdatedAt),
		},
	}
	s.handler.Success(ctx, "get_investment_account", codes.OK, "investment account fetched", resp, account.Metadata, account.AccountID, nil)
	return resp, nil
}

func (s *Service) ListPortfolios(ctx context.Context, req *commercepb.ListPortfoliosRequest) (*commercepb.ListPortfoliosResponse, error) {
	if req == nil || req.AccountId == "" || req.CampaignId == 0 {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "account_id and campaign_id are required", nil)
		s.handler.Error(ctx, "commerce:portfolio:v1:failed", codes.InvalidArgument, "account_id and campaign_id are required", err, nil, req.GetAccountId())
		return nil, graceful.ToStatusError(err)
	}
	portfolios, err := s.repo.ListPortfolios(ctx, req.AccountId, req.CampaignId)
	if err != nil {
		s.handler.Error(ctx, "commerce:portfolio:v1:failed", codes.Internal, "failed to list portfolios", err, nil, req.GetAccountId())
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.ListPortfoliosResponse{
		Portfolios: make([]*commercepb.Portfolio, len(portfolios)),
	}
	for i, p := range portfolios {
		resp.Portfolios[i] = &commercepb.Portfolio{
			PortfolioId: p.PortfolioID,
			AccountId:   p.AccountID,
			Metadata:    p.Metadata,
			CreatedAt:   timestamppb.New(p.CreatedAt),
			CampaignId:  p.CampaignID, // Include CampaignId in response
			UpdatedAt:   timestamppb.New(p.UpdatedAt),
		}
	}
	s.handler.Success(ctx, "list_portfolios", codes.OK, "portfolios listed", resp, nil, req.GetAccountId(), nil)
	return resp, nil
}

func toProtoStruct(payload map[string]interface{}) *structpb.Struct {
	structValue, err := structpb.NewStruct(payload)
	if err != nil {
		return nil
	}
	return structValue
}

func (s *Service) CreateExchangePair(ctx context.Context, req *commercepb.CreateExchangePairRequest) (*commercepb.CreateExchangePairResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "commerce:exchange_pair:v1:failed", codes.InvalidArgument, "request is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	if req.PairId == "" || req.BaseAsset == "" || req.QuoteAsset == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "pair_id, base_asset, and quote_asset are required", nil)
		s.handler.Error(ctx, "commerce:exchange_pair:v1:failed", codes.InvalidArgument, "pair_id, base_asset, and quote_asset are required", err, nil, req.GetPairId())
		return nil, graceful.ToStatusError(err)
	}
	meta := req.Metadata
	pair := &ExchangePair{
		PairID:     req.PairId,
		BaseAsset:  req.BaseAsset,
		QuoteAsset: req.QuoteAsset,
		Metadata:   meta,
	}
	if err := s.repo.CreateExchangePair(ctx, pair); err != nil {
		s.handler.Error(ctx, "create_exchange_pair", codes.Internal, "failed to create exchange pair", err, meta, pair.PairID)
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.CreateExchangePairResponse{
		Pair: &commercepb.ExchangePair{
			PairId:     pair.PairID,
			BaseAsset:  pair.BaseAsset,
			QuoteAsset: pair.QuoteAsset,
			CampaignId: pair.CampaignID, // Include CampaignId in response
			Metadata:   pair.Metadata,
		},
	}
	s.handler.Success(ctx, "create_exchange_pair", codes.OK, "exchange pair created", resp, pair.Metadata, pair.PairID, nil)
	return resp, nil
}

func (s *Service) CreateExchangeRate(ctx context.Context, req *commercepb.CreateExchangeRateRequest) (*commercepb.CreateExchangeRateResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		s.handler.Error(ctx, "commerce:exchange_rate:v1:failed", codes.InvalidArgument, "request is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	if req.PairId == "" || req.Rate == 0 {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "pair_id and rate are required", nil)
		s.handler.Error(ctx, "commerce:exchange_rate:v1:failed", codes.InvalidArgument, "pair_id and rate are required", err, nil, req.GetPairId())
		return nil, graceful.ToStatusError(err)
	}
	rate := &ExchangeRate{
		PairID: req.PairId,
		Rate:   req.Rate,
	}
	if err := s.repo.CreateExchangeRate(ctx, rate); err != nil {
		s.handler.Error(ctx, "create_exchange_rate", codes.Internal, "failed to create exchange rate", err, nil, rate.PairID)
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.CreateExchangeRateResponse{
		Rate: &commercepb.ExchangeRate{
			PairId:     rate.PairID,
			Rate:       rate.Rate,
			Timestamp:  req.Timestamp,
			CampaignId: rate.CampaignID, // Include CampaignId in response
			Metadata:   rate.Metadata,
		},
	}
	s.handler.Success(ctx, "create_exchange_rate", codes.OK, "exchange rate created", resp, rate.Metadata, rate.PairID, nil)
	return resp, nil
}
