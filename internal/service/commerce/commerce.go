package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	commercepb "github.com/nmxmxh/master-ovasabi/api/protos/commerce/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	commercepb.UnimplementedCommerceServiceServer
	log          *zap.Logger
	repo         Repository
	Cache        *redis.Cache
	eventEmitter EventEmitter
	eventEnabled bool
}

func NewService(log *zap.Logger, repo Repository, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) commercepb.CommerceServiceServer {
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
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	meta, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.UserId, true)
	if err != nil {
		// Emit failure event
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.quote_create_failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.quote_create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
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
		// Emit failure event
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.quote_create_failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.quote_create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create quote: %v", err)
	}
	// Emit success event
	if s.eventEnabled && s.eventEmitter != nil {
		errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.quote_created", quote.QuoteID, quote.Metadata)
		if errEmit != nil {
			s.log.Warn("Failed to emit commerce.quote_created event", zap.Error(errEmit))
		}
	}
	// Pattern helpers
	if s.Cache != nil && quote.Metadata != nil {
		err := pattern.CacheMetadata(ctx, s.log, s.Cache, "quote", quote.QuoteID, quote.Metadata, 10*time.Minute)
		if err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	err = pattern.RegisterSchedule(ctx, s.log, "quote", quote.QuoteID, quote.Metadata)
	if err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	err = pattern.EnrichKnowledgeGraph(ctx, s.log, "quote", quote.QuoteID, quote.Metadata)
	if err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "quote", quote.Metadata)
	if err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	// Map to proto
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
	return resp, nil
}

func (s *Service) GetQuote(ctx context.Context, req *commercepb.GetQuoteRequest) (*commercepb.GetQuoteResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	quote, err := s.repo.GetQuote(ctx, req.QuoteId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get quote: %v", err)
	}
	if quote == nil {
		return nil, status.Error(codes.NotFound, "quote not found")
	}
	// Map to proto
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
	return resp, nil
}

func (s *Service) ListQuotes(ctx context.Context, req *commercepb.ListQuotesRequest) (*commercepb.ListQuotesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	quotes, total, err := s.repo.ListQuotes(ctx, req.UserId, req.CampaignId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list quotes: %v", err)
	}
	resp := &commercepb.ListQuotesResponse{
		Quotes: make([]*commercepb.Quote, 0, len(quotes)),
		Total:  utils.ToInt32(total),
	}
	for _, q := range quotes {
		resp.Quotes = append(resp.Quotes, &commercepb.Quote{
			QuoteId:    q.QuoteID,
			UserId:     q.UserID,
			ProductId:  q.ProductID,
			Amount:     q.Amount,
			Currency:   q.Currency,
			Status:     toQuoteStatus(q.Status),
			Metadata:   q.Metadata,
			CreatedAt:  timestamppb.New(q.CreatedAt),
			UpdatedAt:  timestamppb.New(q.UpdatedAt),
			CampaignId: q.CampaignID,
		})
	}
	return resp, nil
}

// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
func (s *Service) CreateOrder(ctx context.Context, req *commercepb.CreateOrderRequest) (*commercepb.CreateOrderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "metadata is required")
	}
	orderID := req.UserId + ":order:" + time.Now().Format("20060102150405.000")
	order := &Order{
		OrderID:   orderID,
		UserID:    req.UserId,
		Total:     0,
		Currency:  req.Currency,
		Status:    "PENDING",
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	for _, item := range req.Items {
		order.Total += item.Price * float64(item.Quantity)
	}
	err := s.repo.CreateOrder(ctx, order, nil)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.order_create_failed", orderID, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.order_create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create order: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.order_created", orderID, order.Metadata)
		if errEmit != nil {
			s.log.Warn("Failed to emit commerce.order_created event", zap.Error(errEmit))
		}
	}
	// Pattern helpers
	if s.Cache != nil && order.Metadata != nil {
		err := pattern.CacheMetadata(ctx, s.log, s.Cache, "order", order.OrderID, order.Metadata, 10*time.Minute)
		if err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	err = pattern.RegisterSchedule(ctx, s.log, "order", order.OrderID, order.Metadata)
	if err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	err = pattern.EnrichKnowledgeGraph(ctx, s.log, "order", order.OrderID, order.Metadata)
	if err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "order", order.Metadata)
	if err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
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
	return resp, nil
}

func (s *Service) GetOrder(ctx context.Context, req *commercepb.GetOrderRequest) (*commercepb.GetOrderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	order, err := s.repo.GetOrder(ctx, req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get order: %v", err)
	}
	if order == nil {
		return nil, status.Error(codes.NotFound, "order not found")
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
	return resp, nil
}

func (s *Service) ListOrders(ctx context.Context, req *commercepb.ListOrdersRequest) (*commercepb.ListOrdersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	orders, total, err := s.repo.ListOrders(ctx, req.UserId, req.CampaignId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list orders: %v", err)
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
	return resp, nil
}

// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
func (s *Service) UpdateOrderStatus(ctx context.Context, req *commercepb.UpdateOrderStatusRequest) (*commercepb.UpdateOrderStatusResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.Status == commercepb.OrderStatus_ORDER_STATUS_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "status is required")
	}
	err := s.repo.UpdateOrderStatus(ctx, req.OrderId, req.Status.String())
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.order_update_failed", req.OrderId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.order_update_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to update order status: %v", err)
	}
	order, err := s.repo.GetOrder(ctx, req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch updated order: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.order_updated", order.OrderID, order.Metadata)
		if errEmit != nil {
			s.log.Warn("Failed to emit commerce.order_updated event", zap.Error(errEmit))
		}
	}
	// Pattern helpers
	if s.Cache != nil && order.Metadata != nil {
		err := pattern.CacheMetadata(ctx, s.log, s.Cache, "order", order.OrderID, order.Metadata, 10*time.Minute)
		if err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	err = pattern.RegisterSchedule(ctx, s.log, "order", order.OrderID, order.Metadata)
	if err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	err = pattern.EnrichKnowledgeGraph(ctx, s.log, "order", order.OrderID, order.Metadata)
	if err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "order", order.Metadata)
	if err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
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
	return resp, nil
}

func (s *Service) InitiatePayment(ctx context.Context, req *commercepb.InitiatePaymentRequest) (*commercepb.InitiatePaymentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.OrderId == "" || req.UserId == "" || req.Amount <= 0 || req.Currency == "" || req.Method == "" {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid payment fields")
	}
	meta, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.UserId, true)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.payment_initiate_failed", req.OrderId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.payment_initiate_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
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
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.payment_initiate_failed", req.OrderId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.payment_initiate_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create payment: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.payment_initiated", payment.PaymentID, payment.Metadata)
		if errEmit != nil {
			s.log.Warn("Failed to emit commerce.payment_initiated event", zap.Error(errEmit))
		}
	}
	// Pattern helpers
	if s.Cache != nil && payment.Metadata != nil {
		err := pattern.CacheMetadata(ctx, s.log, s.Cache, "payment", payment.PaymentID, payment.Metadata, 10*time.Minute)
		if err != nil {
			s.log.Error("failed to cache payment metadata", zap.Error(err))
		}
	}
	err = pattern.RegisterSchedule(ctx, s.log, "payment", payment.PaymentID, payment.Metadata)
	if err != nil {
		s.log.Error("failed to register payment schedule", zap.Error(err))
	}
	err = pattern.EnrichKnowledgeGraph(ctx, s.log, "payment", payment.PaymentID, payment.Metadata)
	if err != nil {
		s.log.Error("failed to enrich payment knowledge graph", zap.Error(err))
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "payment", payment.Metadata)
	if err != nil {
		s.log.Error("failed to register payment with nexus", zap.Error(err))
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
	return resp, nil
}

func (s *Service) ConfirmPayment(ctx context.Context, req *commercepb.ConfirmPaymentRequest) (*commercepb.ConfirmPaymentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.PaymentId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid payment fields")
	}
	if req.Metadata != nil {
		if _, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.UserId, false); err != nil {
			if s.eventEnabled && s.eventEmitter != nil {
				errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
				if err != nil {
					s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
					return nil, status.Error(codes.Internal, "internal error")
				}
				errMeta := &commonpb.Metadata{}
				errMeta.ServiceSpecific = errStruct
				errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.payment_confirm_failed", req.PaymentId, errMeta)
				if errEmit != nil {
					s.log.Warn("Failed to emit commerce.payment_confirm_failed event", zap.Error(errEmit))
				}
			}
			return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
		}
	}
	err := s.repo.UpdatePaymentStatus(ctx, req.PaymentId, "SUCCEEDED")
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.payment_confirm_failed", req.PaymentId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.payment_confirm_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to update payment status: %v", err)
	}
	payment, err := s.repo.GetPayment(ctx, req.PaymentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch payment: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.payment_confirmed", payment.PaymentID, payment.Metadata)
		if errEmit != nil {
			s.log.Warn("Failed to emit commerce.payment_confirmed event", zap.Error(errEmit))
		}
	}
	// Pattern helpers
	if s.Cache != nil && payment.Metadata != nil {
		err := pattern.CacheMetadata(ctx, s.log, s.Cache, "payment", payment.PaymentID, payment.Metadata, 10*time.Minute)
		if err != nil {
			s.log.Error("failed to cache payment metadata", zap.Error(err))
		}
	}
	err = pattern.RegisterSchedule(ctx, s.log, "payment", payment.PaymentID, payment.Metadata)
	if err != nil {
		s.log.Error("failed to register payment schedule", zap.Error(err))
	}
	err = pattern.EnrichKnowledgeGraph(ctx, s.log, "payment", payment.PaymentID, payment.Metadata)
	if err != nil {
		s.log.Error("failed to enrich payment knowledge graph", zap.Error(err))
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "payment", payment.Metadata)
	if err != nil {
		s.log.Error("failed to register payment with nexus", zap.Error(err))
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
	return resp, nil
}

func (s *Service) RefundPayment(ctx context.Context, req *commercepb.RefundPaymentRequest) (*commercepb.RefundPaymentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.PaymentId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid payment fields")
	}
	if req.Metadata != nil {
		if _, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.UserId, false); err != nil {
			if s.eventEnabled && s.eventEmitter != nil {
				errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
				if err != nil {
					s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
					return nil, status.Error(codes.Internal, "internal error")
				}
				errMeta := &commonpb.Metadata{}
				errMeta.ServiceSpecific = errStruct
				errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.payment_refund_failed", req.PaymentId, errMeta)
				if errEmit != nil {
					s.log.Warn("Failed to emit commerce.payment_refund_failed event", zap.Error(errEmit))
				}
			}
			return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
		}
	}
	err := s.repo.UpdatePaymentStatus(ctx, req.PaymentId, "REFUNDED")
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.payment_refund_failed", req.PaymentId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.payment_refund_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to update payment status: %v", err)
	}
	payment, err := s.repo.GetPayment(ctx, req.PaymentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch payment: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.payment_refunded", payment.PaymentID, payment.Metadata)
		if errEmit != nil {
			s.log.Warn("Failed to emit commerce.payment_refunded event", zap.Error(errEmit))
		}
	}
	// Pattern helpers
	if s.Cache != nil && payment.Metadata != nil {
		err := pattern.CacheMetadata(ctx, s.log, s.Cache, "payment", payment.PaymentID, payment.Metadata, 10*time.Minute)
		if err != nil {
			s.log.Error("failed to cache payment metadata", zap.Error(err))
		}
	}
	err = pattern.RegisterSchedule(ctx, s.log, "payment", payment.PaymentID, payment.Metadata)
	if err != nil {
		s.log.Error("failed to register payment schedule", zap.Error(err))
	}
	err = pattern.EnrichKnowledgeGraph(ctx, s.log, "payment", payment.PaymentID, payment.Metadata)
	if err != nil {
		s.log.Error("failed to enrich payment knowledge graph", zap.Error(err))
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "payment", payment.Metadata)
	if err != nil {
		s.log.Error("failed to register payment with nexus", zap.Error(err))
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
	return resp, nil
}

func (s *Service) GetTransaction(ctx context.Context, req *commercepb.GetTransactionRequest) (*commercepb.GetTransactionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.TransactionId == "" {
		return nil, status.Error(codes.InvalidArgument, "transaction_id is required")
	}
	transaction, err := s.repo.GetTransaction(ctx, req.TransactionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get transaction: %v", err)
	}
	if transaction == nil {
		return nil, status.Error(codes.NotFound, "transaction not found")
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
	return resp, nil
}

func (s *Service) ListTransactions(ctx context.Context, req *commercepb.ListTransactionsRequest) (*commercepb.ListTransactionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
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
		return nil, status.Errorf(codes.Internal, "failed to list transactions: %v", err)
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
	return resp, nil
}

func (s *Service) GetBalance(ctx context.Context, req *commercepb.GetBalanceRequest) (*commercepb.GetBalanceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.UserId == "" || req.Currency == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and currency are required")
	}
	balance, err := s.repo.GetBalance(ctx, req.UserId, req.Currency)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get balance: %v", err)
	}
	if balance == nil {
		return nil, status.Error(codes.NotFound, "balance not found")
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
	return resp, nil
}

func (s *Service) ListBalances(ctx context.Context, req *commercepb.ListBalancesRequest) (*commercepb.ListBalancesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	balances, err := s.repo.ListBalances(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list balances: %v", err)
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
	return resp, nil
}

func (s *Service) ListEvents(ctx context.Context, req *commercepb.ListEventsRequest) (*commercepb.ListEventsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.EntityId == "" || req.EntityType == "" {
		return nil, status.Error(codes.InvalidArgument, "entity_id and entity_type are required")
	}
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 20
	}
	events, total, err := s.repo.ListEvents(ctx, req.EntityId, req.EntityType, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list events: %v", err)
	}
	resp := &commercepb.ListEventsResponse{
		Events: make([]*commercepb.CommerceEvent, len(events)),
		Total:  utils.ToInt32(total),
	}
	for i, e := range events {
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
	return resp, nil
}

// --- Investment ---.
func (s *Service) CreateInvestmentAccount(ctx context.Context, req *commercepb.CreateInvestmentAccountRequest) (*commercepb.CreateInvestmentAccountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.OwnerId == "" || req.Currency == "" {
		return nil, status.Error(codes.InvalidArgument, "owner_id and currency are required")
	}
	meta, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.OwnerId, true)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.investment_account_create_failed", req.OwnerId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.investment_account_create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
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
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.investment_account_create_failed", req.OwnerId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.investment_account_create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create investment account: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.investment_account_created", account.AccountID, account.Metadata)
		if errEmit != nil {
			s.log.Warn("Failed to emit commerce.investment_account_created event", zap.Error(errEmit))
		}
	}
	// Pattern helpers
	if s.Cache != nil && account.Metadata != nil {
		err := pattern.CacheMetadata(ctx, s.log, s.Cache, "investment_account", account.AccountID, account.Metadata, 10*time.Minute)
		if err != nil {
			s.log.Error("failed to cache investment account metadata", zap.Error(err))
		}
	}
	err = pattern.RegisterSchedule(ctx, s.log, "investment_account", account.AccountID, account.Metadata)
	if err != nil {
		s.log.Error("failed to register investment account schedule", zap.Error(err))
	}
	err = pattern.EnrichKnowledgeGraph(ctx, s.log, "investment_account", account.AccountID, account.Metadata)
	if err != nil {
		s.log.Error("failed to enrich investment account knowledge graph", zap.Error(err))
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "investment_account", account.Metadata)
	if err != nil {
		s.log.Error("failed to register investment account with nexus", zap.Error(err))
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
	return resp, nil
}

func (s *Service) PlaceInvestmentOrder(ctx context.Context, req *commercepb.PlaceInvestmentOrderRequest) (*commercepb.PlaceInvestmentOrderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.AccountId == "" || req.AssetId == "" || req.Quantity <= 0 || req.Price <= 0 || req.OrderType == "" {
		return nil, status.Error(codes.InvalidArgument, "account_id, asset_id, quantity, price, and order_type are required")
	}
	meta, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.AccountId, true)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.investment_order_create_failed", req.AccountId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.investment_order_create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
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
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.investment_order_create_failed", req.AccountId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.investment_order_create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create investment order: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.investment_order_placed", order.OrderID, order.Metadata)
		if errEmit != nil {
			s.log.Warn("Failed to emit commerce.investment_order_placed event", zap.Error(errEmit))
		}
	}
	// Pattern helpers
	if s.Cache != nil && order.Metadata != nil {
		err := pattern.CacheMetadata(ctx, s.log, s.Cache, "investment_order", order.OrderID, order.Metadata, 10*time.Minute)
		if err != nil {
			s.log.Error("failed to cache investment order metadata", zap.Error(err))
		}
	}
	err = pattern.RegisterSchedule(ctx, s.log, "investment_order", order.OrderID, order.Metadata)
	if err != nil {
		s.log.Error("failed to register investment order schedule", zap.Error(err))
	}
	err = pattern.EnrichKnowledgeGraph(ctx, s.log, "investment_order", order.OrderID, order.Metadata)
	if err != nil {
		s.log.Error("failed to enrich investment order knowledge graph", zap.Error(err))
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "investment_order", order.Metadata)
	if err != nil {
		s.log.Error("failed to register investment order with nexus", zap.Error(err))
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
	return resp, nil
}

func (s *Service) GetPortfolio(ctx context.Context, req *commercepb.GetPortfolioRequest) (*commercepb.GetPortfolioResponse, error) {
	if req == nil || req.PortfolioId == "" {
		return nil, status.Error(codes.InvalidArgument, "portfolio_id is required")
	}
	portfolio, err := s.repo.GetPortfolio(ctx, req.PortfolioId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.Warn("portfolio not found", zap.String("portfolio_id", req.PortfolioId))
			return nil, status.Error(codes.NotFound, "portfolio not found")
		}
		s.log.Error("failed to get portfolio", zap.String("portfolio_id", req.PortfolioId), zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get portfolio: %v", err)
	}
	return &commercepb.GetPortfolioResponse{
		Portfolio: &commercepb.Portfolio{
			PortfolioId: portfolio.PortfolioID,
			AccountId:   portfolio.AccountID,
			Metadata:    portfolio.Metadata,
			CreatedAt:   timestamppb.New(portfolio.CreatedAt),
			UpdatedAt:   timestamppb.New(portfolio.UpdatedAt),
		},
	}, nil
}

// --- Investment/Account/Asset Service Methods ---.
func (s *Service) GetInvestmentAccount(ctx context.Context, req *commercepb.GetInvestmentAccountRequest) (*commercepb.GetInvestmentAccountResponse, error) {
	if req == nil || req.AccountId == "" {
		return nil, status.Error(codes.InvalidArgument, "account_id is required")
	}
	account, err := s.repo.GetInvestmentAccount(ctx, req.AccountId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.Warn("investment account not found", zap.String("account_id", req.AccountId))
			return nil, status.Error(codes.NotFound, "investment account not found")
		}
		s.log.Error("failed to get investment account", zap.String("account_id", req.AccountId), zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get investment account: %v", err)
	}
	return &commercepb.GetInvestmentAccountResponse{
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
	}, nil
}

func (s *Service) ListPortfolios(ctx context.Context, req *commercepb.ListPortfoliosRequest) (*commercepb.ListPortfoliosResponse, error) {
	if req == nil || req.AccountId == "" {
		return nil, status.Error(codes.InvalidArgument, "account_id is required")
	}
	portfolios, err := s.repo.ListPortfolios(ctx, req.AccountId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list portfolios: %v", err)
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
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.PairId == "" || req.BaseAsset == "" || req.QuoteAsset == "" {
		return nil, status.Error(codes.InvalidArgument, "pair_id, base_asset, and quote_asset are required")
	}
	meta, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.PairId, true)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.exchange_pair_create_failed", req.PairId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.exchange_pair_create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	pair := &ExchangePair{
		PairID:     req.PairId,
		MasterID:   0,
		BaseAsset:  req.BaseAsset,
		QuoteAsset: req.QuoteAsset,
		Metadata:   meta,
	}
	if err := s.repo.CreateExchangePair(ctx, pair); err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.exchange_pair_create_failed", req.PairId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.exchange_pair_create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create exchange pair: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.marketplace_listing_created", pair.PairID, pair.Metadata)
		if errEmit != nil {
			s.log.Warn("Failed to emit commerce.marketplace_listing_created event", zap.Error(errEmit))
		}
	}
	resp := &commercepb.CreateExchangePairResponse{
		Pair: &commercepb.ExchangePair{
			PairId:     pair.PairID,
			BaseAsset:  pair.BaseAsset,
			QuoteAsset: pair.QuoteAsset,
			Metadata:   pair.Metadata,
		},
	}
	return resp, nil
}

func (s *Service) CreateExchangeRate(ctx context.Context, req *commercepb.CreateExchangeRateRequest) (*commercepb.CreateExchangeRateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.PairId == "" || req.Rate == 0 {
		return nil, status.Error(codes.InvalidArgument, "pair_id and rate are required")
	}
	meta, err := ExtractAndEnrichCommerceMetadata(s.log, req.Metadata, req.PairId, true)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.exchange_rate_create_failed", req.PairId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.exchange_rate_create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	rate := &ExchangeRate{
		RateID:    0,
		MasterID:  0,
		PairID:    req.PairId,
		Rate:      req.Rate,
		Timestamp: req.Timestamp.AsTime(),
		Metadata:  meta,
	}
	if err := s.repo.CreateExchangeRate(ctx, rate); err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for commerce event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.exchange_rate_create_failed", req.PairId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit commerce.exchange_rate_create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create exchange rate: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEmit := s.eventEmitter.EmitEvent(ctx, "commerce.exchange_rate_updated", rate.PairID, rate.Metadata)
		if errEmit != nil {
			s.log.Warn("Failed to emit commerce.exchange_rate_updated event", zap.Error(errEmit))
		}
	}
	resp := &commercepb.CreateExchangeRateResponse{
		Rate: &commercepb.ExchangeRate{
			PairId:    rate.PairID,
			Rate:      rate.Rate,
			Timestamp: req.Timestamp,
			Metadata:  rate.Metadata,
		},
	}
	return resp, nil
}
