package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	commercepb "github.com/nmxmxh/master-ovasabi/api/protos/commerce/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
	eventEmitter EventEmitter
	eventEnabled bool
}

func NewService(log *zap.Logger, repo Repository, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) commercepb.CommerceServiceServer {
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
	log := s.log.With(zap.String("operation", "create_quote"), zap.String("user_id", req.GetUserId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	meta, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.UserId, true)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
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
		err := graceful.WrapErr(ctx, codes.Internal, "failed to create quote", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
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
	success := graceful.WrapSuccess(ctx, codes.OK, "quote created", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     quote.QuoteID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     quote.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.quote_created",
		EventID:      quote.QuoteID,
		PatternType:  "quote",
		PatternID:    quote.QuoteID,
		PatternMeta:  quote.Metadata,
	})
	return resp, nil
}

func (s *Service) GetQuote(ctx context.Context, req *commercepb.GetQuoteRequest) (*commercepb.GetQuoteResponse, error) {
	log := s.log.With(zap.String("operation", "get_quote"), zap.String("quote_id", req.GetQuoteId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	quote, err := s.repo.GetQuote(ctx, req.QuoteId)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to get quote", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if quote == nil {
		err := graceful.WrapErr(ctx, codes.NotFound, "quote not found", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
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
	success := graceful.WrapSuccess(ctx, codes.OK, "quote fetched", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     quote.QuoteID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     quote.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.quote_fetched",
		EventID:      quote.QuoteID,
		PatternType:  "quote",
		PatternID:    quote.QuoteID,
		PatternMeta:  quote.Metadata,
	})
	return resp, nil
}

func (s *Service) ListQuotes(ctx context.Context, req *commercepb.ListQuotesRequest) (*commercepb.ListQuotesResponse, error) {
	log := s.log.With(zap.String("operation", "list_quotes"), zap.String("user_id", req.GetUserId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	quotes, total, err := s.repo.ListQuotes(ctx, req.UserId, req.CampaignId, int(req.Page), int(req.PageSize))
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to list quotes", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
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
	success := graceful.WrapSuccess(ctx, codes.OK, "quotes listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:        log,
		Cache:      s.Cache,
		CacheKey:   fmt.Sprintf("quotes:user:%s", req.UserId),
		CacheValue: resp,
		CacheTTL:   5 * time.Minute,
	})
	return resp, nil
}

// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
func (s *Service) CreateOrder(ctx context.Context, req *commercepb.CreateOrderRequest) (*commercepb.CreateOrderResponse, error) {
	log := s.log.With(zap.String("operation", "create_order"), zap.String("user_id", req.GetUserId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.Metadata == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "metadata is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	meta, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.UserId, true)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	orderID := req.UserId + ":order:" + time.Now().Format("20060102150405.000")
	order := &Order{
		OrderID:   orderID,
		UserID:    req.UserId,
		Total:     0,
		Currency:  req.Currency,
		Status:    "PENDING",
		Metadata:  meta,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	for _, item := range req.Items {
		order.Total += item.Price * float64(item.Quantity)
	}
	err = s.repo.CreateOrder(ctx, order, nil)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to create order", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.CreateOrderResponse{
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
	success := graceful.WrapSuccess(ctx, codes.OK, "order created", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     order.OrderID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     order.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.order_created",
		EventID:      order.OrderID,
		PatternType:  "order",
		PatternID:    order.OrderID,
		PatternMeta:  order.Metadata,
	})
	return resp, nil
}

func (s *Service) GetOrder(ctx context.Context, req *commercepb.GetOrderRequest) (*commercepb.GetOrderResponse, error) {
	log := s.log.With(zap.String("operation", "get_order"), zap.String("order_id", req.GetOrderId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	order, err := s.repo.GetOrder(ctx, req.OrderId)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to get order", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if order == nil {
		err := graceful.WrapErr(ctx, codes.NotFound, "order not found", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.GetOrderResponse{
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
	success := graceful.WrapSuccess(ctx, codes.OK, "order fetched", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     order.OrderID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     order.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.order_fetched",
		EventID:      order.OrderID,
		PatternType:  "order",
		PatternID:    order.OrderID,
		PatternMeta:  order.Metadata,
	})
	return resp, nil
}

func (s *Service) ListOrders(ctx context.Context, req *commercepb.ListOrdersRequest) (*commercepb.ListOrdersResponse, error) {
	log := s.log.With(zap.String("operation", "list_orders"), zap.String("user_id", req.GetUserId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	orders, total, err := s.repo.ListOrders(ctx, req.UserId, req.CampaignId, int(req.Page), int(req.PageSize))
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to list orders", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.ListOrdersResponse{
		Orders: make([]*commercepb.Order, len(orders)),
		Total:  utils.ToInt32(total),
	}
	for i, order := range orders {
		resp.Orders[i] = &commercepb.Order{
			OrderId:   order.OrderID,
			UserId:    order.UserID,
			Total:     order.Total,
			Currency:  order.Currency,
			Status:    toOrderStatus(order.Status),
			Metadata:  order.Metadata,
			CreatedAt: timestamppb.New(order.CreatedAt),
			UpdatedAt: timestamppb.New(order.UpdatedAt),
		}
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "orders listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:        log,
		Cache:      s.Cache,
		CacheKey:   fmt.Sprintf("orders:user:%s", req.UserId),
		CacheValue: resp,
		CacheTTL:   5 * time.Minute,
	})
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
	success := graceful.WrapSuccess(ctx, codes.OK, "order status updated", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     order.OrderID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     order.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.order_status_updated",
		EventID:      order.OrderID,
		PatternType:  "order",
		PatternID:    order.OrderID,
		PatternMeta:  order.Metadata,
	})
	return resp, nil
}

func (s *Service) InitiatePayment(ctx context.Context, req *commercepb.InitiatePaymentRequest) (*commercepb.InitiatePaymentResponse, error) {
	log := s.log.With(zap.String("operation", "initiate_payment"), zap.String("order_id", req.GetOrderId()), zap.String("user_id", req.GetUserId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.OrderId == "" || req.UserId == "" || req.Amount <= 0 || req.Currency == "" || req.Method == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid payment fields", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	meta, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.UserId, true)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	payment := &Payment{
		PaymentID: req.OrderId + ":payment:" + time.Now().Format("20060102150405.000"),
		OrderID:   req.OrderId,
		UserID:    req.UserId,
		Amount:    req.Amount,
		Currency:  req.Currency,
		Method:    req.Method,
		Status:    "PENDING",
		Metadata:  meta,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to create payment", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.InitiatePaymentResponse{
		Payment: &commercepb.Payment{
			PaymentId: payment.PaymentID,
			OrderId:   payment.OrderID,
			UserId:    payment.UserID,
			Amount:    payment.Amount,
			Currency:  payment.Currency,
			Method:    payment.Method,
			Status:    commercepb.PaymentStatus_PAYMENT_STATUS_PENDING,
			Metadata:  payment.Metadata,
			CreatedAt: timestamppb.New(payment.CreatedAt),
			UpdatedAt: timestamppb.New(payment.UpdatedAt),
		},
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "payment initiated", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     payment.PaymentID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     payment.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.payment_initiated",
		EventID:      payment.PaymentID,
		PatternType:  "payment",
		PatternID:    payment.PaymentID,
		PatternMeta:  payment.Metadata,
	})
	return resp, nil
}

func (s *Service) ConfirmPayment(ctx context.Context, req *commercepb.ConfirmPaymentRequest) (*commercepb.ConfirmPaymentResponse, error) {
	log := s.log.With(zap.String("operation", "confirm_payment"), zap.String("payment_id", req.GetPaymentId()), zap.String("user_id", req.GetUserId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.PaymentId == "" || req.UserId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid payment fields", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.Metadata != nil {
		if _, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.UserId, false); err != nil {
			err := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid metadata", err)
			err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			return nil, graceful.ToStatusError(err)
		}
	}
	err := s.repo.UpdatePaymentStatus(ctx, req.PaymentId, "SUCCEEDED")
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to update payment status", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	payment, err := s.repo.GetPayment(ctx, req.PaymentId)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to fetch payment", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.ConfirmPaymentResponse{
		Payment: &commercepb.Payment{
			PaymentId: payment.PaymentID,
			OrderId:   payment.OrderID,
			UserId:    payment.UserID,
			Amount:    payment.Amount,
			Currency:  payment.Currency,
			Method:    payment.Method,
			Status:    commercepb.PaymentStatus_PAYMENT_STATUS_SUCCEEDED,
			Metadata:  payment.Metadata,
			CreatedAt: timestamppb.New(payment.CreatedAt),
			UpdatedAt: timestamppb.New(payment.UpdatedAt),
		},
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "payment confirmed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     payment.PaymentID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     payment.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.payment_confirmed",
		EventID:      payment.PaymentID,
		PatternType:  "payment",
		PatternID:    payment.PaymentID,
		PatternMeta:  payment.Metadata,
	})
	return resp, nil
}

func (s *Service) RefundPayment(ctx context.Context, req *commercepb.RefundPaymentRequest) (*commercepb.RefundPaymentResponse, error) {
	log := s.log.With(zap.String("operation", "refund_payment"), zap.String("payment_id", req.GetPaymentId()), zap.String("user_id", req.GetUserId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.PaymentId == "" || req.UserId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid payment fields", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.Metadata != nil {
		if _, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.UserId, false); err != nil {
			err := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid metadata", err)
			err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			return nil, graceful.ToStatusError(err)
		}
	}
	err := s.repo.UpdatePaymentStatus(ctx, req.PaymentId, "REFUNDED")
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to update payment status", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	payment, err := s.repo.GetPayment(ctx, req.PaymentId)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to fetch payment", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.RefundPaymentResponse{
		Payment: &commercepb.Payment{
			PaymentId: payment.PaymentID,
			OrderId:   payment.OrderID,
			UserId:    payment.UserID,
			Amount:    payment.Amount,
			Currency:  payment.Currency,
			Method:    payment.Method,
			Status:    commercepb.PaymentStatus_PAYMENT_STATUS_REFUNDED,
			Metadata:  payment.Metadata,
			CreatedAt: timestamppb.New(payment.CreatedAt),
			UpdatedAt: timestamppb.New(payment.UpdatedAt),
		},
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "payment refunded", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     payment.PaymentID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     payment.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.payment_refunded",
		EventID:      payment.PaymentID,
		PatternType:  "payment",
		PatternID:    payment.PaymentID,
		PatternMeta:  payment.Metadata,
	})
	return resp, nil
}

func (s *Service) GetTransaction(ctx context.Context, req *commercepb.GetTransactionRequest) (*commercepb.GetTransactionResponse, error) {
	log := s.log.With(zap.String("operation", "get_transaction"), zap.String("transaction_id", req.GetTransactionId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.TransactionId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "transaction_id is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	transaction, err := s.repo.GetTransaction(ctx, req.TransactionId)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to get transaction", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if transaction == nil {
		err := graceful.WrapErr(ctx, codes.NotFound, "transaction not found", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
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
			UpdatedAt:     timestamppb.New(transaction.UpdatedAt),
		},
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "transaction fetched", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     transaction.TransactionID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     transaction.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.transaction_fetched",
		EventID:      transaction.TransactionID,
		PatternType:  "transaction",
		PatternID:    transaction.TransactionID,
		PatternMeta:  transaction.Metadata,
	})
	return resp, nil
}

func (s *Service) ListTransactions(ctx context.Context, req *commercepb.ListTransactionsRequest) (*commercepb.ListTransactionsResponse, error) {
	log := s.log.With(zap.String("operation", "list_transactions"), zap.String("user_id", req.GetUserId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.UserId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "user_id is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
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
		err := graceful.WrapErr(ctx, codes.Internal, "failed to list transactions", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
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
			UpdatedAt:     timestamppb.New(tx.UpdatedAt),
		}
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "transactions listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:        log,
		Cache:      s.Cache,
		CacheKey:   fmt.Sprintf("transactions:user:%s", req.UserId),
		CacheValue: resp,
		CacheTTL:   5 * time.Minute,
	})
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
			UserId:    balance.UserID,
			Currency:  balance.Currency,
			Amount:    balance.Amount,
			UpdatedAt: timestamppb.New(balance.UpdatedAt),
			Metadata:  balance.Metadata,
		},
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "balance fetched", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     fmt.Sprintf("balance:user:%s:currency:%s", req.GetUserId(), req.GetCurrency()),
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     balance.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.balance_fetched",
		EventID:      balance.UserID + ":" + balance.Currency,
		PatternType:  "balance",
		PatternID:    balance.UserID + ":" + balance.Currency,
		PatternMeta:  balance.Metadata,
	})
	return resp, nil
}

func (s *Service) ListBalances(ctx context.Context, req *commercepb.ListBalancesRequest) (*commercepb.ListBalancesResponse, error) {
	log := s.log.With(zap.String("operation", "list_balances"), zap.String("user_id", req.GetUserId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.UserId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "user_id is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	balances, err := s.repo.ListBalances(ctx, req.UserId)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to list balances", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.ListBalancesResponse{
		Balances: make([]*commercepb.Balance, len(balances)),
	}
	for i, b := range balances {
		resp.Balances[i] = &commercepb.Balance{
			UserId:    b.UserID,
			Currency:  b.Currency,
			Amount:    b.Amount,
			UpdatedAt: timestamppb.New(b.UpdatedAt),
			Metadata:  b.Metadata,
		}
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "balances listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:        log,
		Cache:      s.Cache,
		CacheKey:   fmt.Sprintf("balances:user:%s", req.GetUserId()),
		CacheValue: resp,
		CacheTTL:   5 * time.Minute,
	})
	return resp, nil
}

func (s *Service) ListEvents(ctx context.Context, req *commercepb.ListEventsRequest) (*commercepb.ListEventsResponse, error) {
	log := s.log.With(zap.String("operation", "list_events"), zap.String("entity_id", req.GetEntityId()), zap.String("entity_type", req.GetEntityType()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.EntityId == "" || req.EntityType == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "entity_id and entity_type are required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
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
	eventList, total, err := s.repo.ListEvents(ctx, req.EntityId, req.EntityType, page, pageSize)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to list events", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.ListEventsResponse{
		Events: make([]*commercepb.CommerceEvent, len(eventList)),
		Total:  utils.ToInt32(total),
	}
	for i, e := range eventList {
		resp.Events[i] = &commercepb.CommerceEvent{
			EventId:    fmt.Sprintf("%d", e.EventID),
			EntityId:   fmt.Sprintf("%d", e.EntityID),
			EntityType: e.EntityType,
			EventType:  e.EventType,
			Payload:    toProtoStruct(e.Payload),
			CreatedAt:  timestamppb.New(e.CreatedAt),
			Metadata:   e.Metadata,
		}
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "events listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:        log,
		Cache:      s.Cache,
		CacheKey:   fmt.Sprintf("events:entity:%s:type:%s", req.GetEntityId(), req.GetEntityType()),
		CacheValue: resp,
		CacheTTL:   5 * time.Minute,
	})
	return resp, nil
}

// --- Investment ---.
func (s *Service) CreateInvestmentAccount(ctx context.Context, req *commercepb.CreateInvestmentAccountRequest) (*commercepb.CreateInvestmentAccountResponse, error) {
	log := s.log.With(zap.String("operation", "create_investment_account"), zap.String("owner_id", req.GetOwnerId()), zap.String("currency", req.GetCurrency()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.OwnerId == "" || req.Currency == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "owner_id and currency are required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	meta, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.OwnerId, true)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	accountID := req.OwnerId + ":investment_account:" + time.Now().Format("20060102150405.000")
	account := &InvestmentAccount{
		AccountID: accountID,
		OwnerID:   req.OwnerId,
		Type:      req.Type,
		Currency:  req.Currency,
		Balance:   req.Balance,
		Metadata:  meta,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.CreateInvestmentAccount(ctx, account); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to create investment account", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.CreateInvestmentAccountResponse{
		Account: &commercepb.InvestmentAccount{
			AccountId: account.AccountID,
			OwnerId:   account.OwnerID,
			Type:      account.Type,
			Currency:  account.Currency,
			Balance:   account.Balance,
			Metadata:  account.Metadata,
		},
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "investment account created", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     account.AccountID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     account.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.investment_account_created",
		EventID:      account.AccountID,
		PatternType:  "investment_account",
		PatternID:    account.AccountID,
		PatternMeta:  account.Metadata,
	})
	return resp, nil
}

func (s *Service) PlaceInvestmentOrder(ctx context.Context, req *commercepb.PlaceInvestmentOrderRequest) (*commercepb.PlaceInvestmentOrderResponse, error) {
	log := s.log.With(zap.String("operation", "place_investment_order"), zap.String("account_id", req.GetAccountId()), zap.String("asset_id", req.GetAssetId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.AccountId == "" || req.AssetId == "" || req.Quantity <= 0 || req.Price <= 0 || req.OrderType == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "account_id, asset_id, quantity, price, and order_type are required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	meta, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.AccountId, true)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	orderID := req.AccountId + ":investment_order:" + time.Now().Format("20060102150405.000")
	order := &InvestmentOrder{
		OrderID:   orderID,
		AccountID: req.AccountId,
		AssetID:   req.AssetId,
		Quantity:  req.Quantity,
		Price:     req.Price,
		OrderType: req.OrderType,
		Status:    1,
		Metadata:  meta,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.CreateInvestmentOrder(ctx, order); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to create investment order", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.PlaceInvestmentOrderResponse{
		Order: &commercepb.InvestmentOrder{
			OrderId:   order.OrderID,
			AccountId: order.AccountID,
			AssetId:   order.AssetID,
			Quantity:  order.Quantity,
			Price:     order.Price,
			OrderType: order.OrderType,
			Status:    commercepb.InvestmentOrderStatus_INVESTMENT_ORDER_STATUS_PENDING,
			Metadata:  order.Metadata,
			CreatedAt: timestamppb.New(order.CreatedAt),
		},
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "investment order placed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     order.OrderID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     order.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.investment_order_placed",
		EventID:      order.OrderID,
		PatternType:  "investment_order",
		PatternID:    order.OrderID,
		PatternMeta:  order.Metadata,
	})
	return resp, nil
}

func (s *Service) GetPortfolio(ctx context.Context, req *commercepb.GetPortfolioRequest) (*commercepb.GetPortfolioResponse, error) {
	log := s.log.With(zap.String("operation", "get_portfolio"), zap.String("portfolio_id", req.GetPortfolioId()))
	if req == nil || req.PortfolioId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "portfolio_id is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	portfolio, err := s.repo.GetPortfolio(ctx, req.PortfolioId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn("portfolio not found", zap.String("portfolio_id", req.GetPortfolioId()))
			err := graceful.WrapErr(ctx, codes.NotFound, "portfolio not found", err)
			err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			return nil, graceful.ToStatusError(err)
		}
		log.Error("failed to get portfolio", zap.String("portfolio_id", req.GetPortfolioId()), zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to get portfolio", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.GetPortfolioResponse{
		Portfolio: &commercepb.Portfolio{
			PortfolioId: portfolio.PortfolioID,
			AccountId:   portfolio.AccountID,
			Metadata:    portfolio.Metadata,
			CreatedAt:   timestamppb.New(portfolio.CreatedAt),
			UpdatedAt:   timestamppb.New(portfolio.UpdatedAt),
		},
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "portfolio fetched", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     portfolio.PortfolioID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     portfolio.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.portfolio_fetched",
		EventID:      portfolio.PortfolioID,
		PatternType:  "portfolio",
		PatternID:    portfolio.PortfolioID,
		PatternMeta:  portfolio.Metadata,
	})
	return resp, nil
}

// --- Investment/Account/Asset Service Methods ---.
func (s *Service) GetInvestmentAccount(ctx context.Context, req *commercepb.GetInvestmentAccountRequest) (*commercepb.GetInvestmentAccountResponse, error) {
	log := s.log.With(zap.String("operation", "get_investment_account"), zap.String("account_id", req.GetAccountId()))
	if req == nil || req.AccountId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "account_id is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	account, err := s.repo.GetInvestmentAccount(ctx, req.AccountId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn("investment account not found", zap.String("account_id", req.GetAccountId()))
			err := graceful.WrapErr(ctx, codes.NotFound, "investment account not found", err)
			err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			return nil, graceful.ToStatusError(err)
		}
		log.Error("failed to get investment account", zap.String("account_id", req.GetAccountId()), zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to get investment account", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.GetInvestmentAccountResponse{
		Account: &commercepb.InvestmentAccount{
			AccountId: account.AccountID,
			OwnerId:   account.OwnerID,
			Type:      account.Type,
			Currency:  account.Currency,
			Balance:   account.Balance,
			Metadata:  account.Metadata,
			CreatedAt: timestamppb.New(account.CreatedAt),
			UpdatedAt: timestamppb.New(account.UpdatedAt),
		},
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "investment account fetched", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     account.AccountID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     account.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.investment_account_fetched",
		EventID:      account.AccountID,
		PatternType:  "investment_account",
		PatternID:    account.AccountID,
		PatternMeta:  account.Metadata,
	})
	return resp, nil
}

func (s *Service) ListPortfolios(ctx context.Context, req *commercepb.ListPortfoliosRequest) (*commercepb.ListPortfoliosResponse, error) {
	log := s.log.With(zap.String("operation", "list_portfolios"), zap.String("account_id", req.GetAccountId()))
	if req == nil || req.AccountId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "account_id is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	portfolios, err := s.repo.ListPortfolios(ctx, req.AccountId)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to list portfolios", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
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
			UpdatedAt:   timestamppb.New(p.UpdatedAt),
		}
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "portfolios listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:        log,
		Cache:      s.Cache,
		CacheKey:   fmt.Sprintf("portfolios:account:%s", req.GetAccountId()),
		CacheValue: resp,
		CacheTTL:   5 * time.Minute,
	})
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
	log := s.log.With(zap.String("operation", "create_exchange_pair"), zap.String("pair_id", req.GetPairId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.PairId == "" || req.BaseAsset == "" || req.QuoteAsset == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "pair_id, base_asset, and quote_asset are required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	meta, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.PairId, true)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	pair := &ExchangePair{
		PairID:     req.PairId,
		MasterID:   0,
		BaseAsset:  req.BaseAsset,
		QuoteAsset: req.QuoteAsset,
		Metadata:   meta,
	}
	if err := s.repo.CreateExchangePair(ctx, pair); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to create exchange pair", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.CreateExchangePairResponse{
		Pair: &commercepb.ExchangePair{
			PairId:     pair.PairID,
			BaseAsset:  pair.BaseAsset,
			QuoteAsset: pair.QuoteAsset,
			Metadata:   pair.Metadata,
		},
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "exchange pair created", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     pair.PairID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     pair.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.exchange_pair_created",
		EventID:      pair.PairID,
		PatternType:  "exchange_pair",
		PatternID:    pair.PairID,
		PatternMeta:  pair.Metadata,
	})
	return resp, nil
}

func (s *Service) CreateExchangeRate(ctx context.Context, req *commercepb.CreateExchangeRateRequest) (*commercepb.CreateExchangeRateResponse, error) {
	log := s.log.With(zap.String("operation", "create_exchange_rate"), zap.String("pair_id", req.GetPairId()))
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	if req.PairId == "" || req.Rate == 0 {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "pair_id and rate are required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	rate := &ExchangeRate{
		PairID: req.PairId,
		Rate:   req.Rate,
	}
	if err := s.repo.CreateExchangeRate(ctx, rate); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to create exchange rate", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &commercepb.CreateExchangeRateResponse{
		Rate: &commercepb.ExchangeRate{
			PairId:    rate.PairID,
			Rate:      rate.Rate,
			Timestamp: req.Timestamp,
			Metadata:  rate.Metadata,
		},
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "exchange rate created", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.Cache,
		CacheKey:     rate.PairID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     rate.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "commerce.exchange_rate_created",
		EventID:      rate.PairID,
		PatternType:  "exchange_rate",
		PatternID:    rate.PairID,
		PatternMeta:  rate.Metadata,
	})
	return resp, nil
}
