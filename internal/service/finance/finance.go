package finance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	financepb "github.com/nmxmxh/master-ovasabi/api/protos/finance/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/repository/finance"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Transaction types
	TransactionTypeDeposit  = "deposit"
	TransactionTypeWithdraw = "withdraw"
	TransactionTypeTransfer = "transfer"

	// Transaction statuses
	TransactionStatusPending  = "pending"
	TransactionStatusComplete = "complete"
	TransactionStatusFailed   = "failed"
)

// Service defines the interface for finance operations
type Service interface {
	GetBalanceInternal(ctx context.Context, userID uuid.UUID) (float64, error)
	DepositInternal(ctx context.Context, userID uuid.UUID, amount float64) (*finance.TransactionModel, error)
	WithdrawInternal(ctx context.Context, userID uuid.UUID, amount float64) (*finance.TransactionModel, error)
	TransferInternal(ctx context.Context, fromUserID, toUserID uuid.UUID, amount float64) (*finance.TransactionModel, error)
	GetTransactionInternal(ctx context.Context, transactionID uuid.UUID) (*finance.TransactionModel, error)
	ListTransactionsInternal(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*finance.TransactionModel, error)
}

// service implements both the Service interface and gRPC service
type service struct {
	financepb.UnimplementedFinanceServiceServer
	repo   finance.Repository
	master repository.MasterRepository
	cache  *redis.Cache
	logger *zap.Logger
}

// Compile-time check
var _ financepb.FinanceServiceServer = (*service)(nil)

// New creates a new finance service instance
func New(repo finance.Repository, master repository.MasterRepository, cache *redis.Cache, logger *zap.Logger) *service {
	return &service{
		repo:   repo,
		master: master,
		cache:  cache,
		logger: logger,
	}
}

// GetBalanceInternal retrieves the current balance for a user
func (s *service) GetBalanceInternal(ctx context.Context, userID uuid.UUID) (float64, error) {
	balance, err := s.repo.GetBalance(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}
	return balance, nil
}

// DepositInternal adds funds to a user's account
func (s *service) DepositInternal(ctx context.Context, userID uuid.UUID, amount float64) (*finance.TransactionModel, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("deposit amount must be positive")
	}

	tx := &finance.TransactionModel{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      TransactionTypeDeposit,
		Amount:    amount,
		Status:    TransactionStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to create deposit transaction: %w", err)
	}

	if err := s.repo.UpdateBalance(ctx, userID, amount); err != nil {
		tx.Status = TransactionStatusFailed
		if updateErr := s.repo.CreateTransaction(ctx, tx); updateErr != nil {
			s.logger.Error("failed to update failed transaction status", zap.Error(updateErr))
		}
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	tx.Status = TransactionStatusComplete
	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		s.logger.Error("failed to update completed transaction status", zap.Error(err))
	}

	return tx, nil
}

// WithdrawInternal removes funds from a user's account
func (s *service) WithdrawInternal(ctx context.Context, userID uuid.UUID, amount float64) (*finance.TransactionModel, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("withdrawal amount must be positive")
	}

	balance, err := s.GetBalanceInternal(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	if balance < amount {
		return nil, fmt.Errorf("insufficient funds")
	}

	tx := &finance.TransactionModel{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      TransactionTypeWithdraw,
		Amount:    -amount,
		Status:    TransactionStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to create withdrawal transaction: %w", err)
	}

	if err := s.repo.UpdateBalance(ctx, userID, -amount); err != nil {
		tx.Status = TransactionStatusFailed
		if updateErr := s.repo.CreateTransaction(ctx, tx); updateErr != nil {
			s.logger.Error("failed to update failed transaction status", zap.Error(updateErr))
		}
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	tx.Status = TransactionStatusComplete
	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		s.logger.Error("failed to update completed transaction status", zap.Error(err))
	}

	return tx, nil
}

// TransferInternal moves funds between two user accounts
func (s *service) TransferInternal(ctx context.Context, fromUserID, toUserID uuid.UUID, amount float64) (*finance.TransactionModel, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("transfer amount must be positive")
	}

	if fromUserID == toUserID {
		return nil, fmt.Errorf("cannot transfer to same account")
	}

	balance, err := s.GetBalanceInternal(ctx, fromUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	if balance < amount {
		return nil, fmt.Errorf("insufficient funds")
	}

	tx := &finance.TransactionModel{
		ID:        uuid.New(),
		UserID:    fromUserID,
		ToUserID:  toUserID,
		Type:      TransactionTypeTransfer,
		Amount:    -amount,
		Status:    TransactionStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to create transfer transaction: %w", err)
	}

	if err := s.repo.UpdateBalance(ctx, fromUserID, -amount); err != nil {
		tx.Status = TransactionStatusFailed
		if updateErr := s.repo.CreateTransaction(ctx, tx); updateErr != nil {
			s.logger.Error("failed to update failed transaction status", zap.Error(updateErr))
		}
		return nil, fmt.Errorf("failed to update sender balance: %w", err)
	}

	if err := s.repo.UpdateBalance(ctx, toUserID, amount); err != nil {
		// Rollback sender's balance
		if rollbackErr := s.repo.UpdateBalance(ctx, fromUserID, amount); rollbackErr != nil {
			s.logger.Error("failed to rollback sender balance", zap.Error(rollbackErr))
		}
		tx.Status = TransactionStatusFailed
		if updateErr := s.repo.CreateTransaction(ctx, tx); updateErr != nil {
			s.logger.Error("failed to update failed transaction status", zap.Error(updateErr))
		}
		return nil, fmt.Errorf("failed to update recipient balance: %w", err)
	}

	tx.Status = TransactionStatusComplete
	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		s.logger.Error("failed to update completed transaction status", zap.Error(err))
	}

	return tx, nil
}

// GetTransactionInternal retrieves a specific transaction
func (s *service) GetTransactionInternal(ctx context.Context, transactionID uuid.UUID) (*finance.TransactionModel, error) {
	tx, err := s.repo.GetTransaction(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	return tx, nil
}

// ListTransactionsInternal retrieves a paginated list of transactions for a user
func (s *service) ListTransactionsInternal(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*finance.TransactionModel, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	txs, err := s.repo.ListTransactions(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	return txs, nil
}

// gRPC service implementation
func (s *service) GetBalance(ctx context.Context, req *financepb.GetBalanceRequest) (*financepb.GetBalanceResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	balance, err := s.GetBalanceInternal(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get balance: %v", err)
	}

	return &financepb.GetBalanceResponse{
		Balance: &financepb.Balance{
			UserId:    req.UserId,
			Amount:    balance,
			UpdatedAt: timestamppb.Now(),
		},
	}, nil
}

// Deposit adds funds to a user's account
func (s *service) Deposit(ctx context.Context, req *financepb.DepositRequest) (*financepb.DepositResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	_, err = s.DepositInternal(ctx, userID, req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process deposit: %v", err)
	}

	return &financepb.DepositResponse{}, nil
}

// Withdraw removes funds from a user's account
func (s *service) Withdraw(ctx context.Context, req *financepb.WithdrawRequest) (*financepb.WithdrawResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	_, err = s.WithdrawInternal(ctx, userID, req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process withdrawal: %v", err)
	}

	return &financepb.WithdrawResponse{}, nil
}

// Transfer moves funds between two user accounts
func (s *service) Transfer(ctx context.Context, req *financepb.TransferRequest) (*financepb.TransferResponse, error) {
	fromUserID, err := uuid.Parse(req.FromUserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid from_user_id: %v", err)
	}

	toUserID, err := uuid.Parse(req.ToUserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid to_user_id: %v", err)
	}

	_, err = s.TransferInternal(ctx, fromUserID, toUserID, req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process transfer: %v", err)
	}

	return &financepb.TransferResponse{}, nil
}

// GetTransaction retrieves a transaction
func (s *service) GetTransaction(ctx context.Context, req *financepb.GetTransactionRequest) (*financepb.GetTransactionResponse, error) {
	txID, err := uuid.Parse(req.TransactionId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid transaction ID: %v", err)
	}

	_, err = s.GetTransactionInternal(ctx, txID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get transaction: %v", err)
	}

	return &financepb.GetTransactionResponse{}, nil
}

func (s *service) ListTransactions(ctx context.Context, req *financepb.ListTransactionsRequest) (*financepb.ListTransactionsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	txs, err := s.ListTransactionsInternal(ctx, userID, int(req.PageSize), int(req.Page)*int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list transactions: %v", err)
	}

	protoTxs := make([]*financepb.Transaction, 0, len(txs))
	for _, tx := range txs {
		protoTxs = append(protoTxs, convertTransactionToProto(tx))
	}

	return &financepb.ListTransactionsResponse{
		Transactions: protoTxs,
		HasMore:      len(txs) == int(req.PageSize),
		TotalCount:   int32(len(txs)),
	}, nil
}

// convertTransactionToProto converts repository transaction model to proto transaction
func convertTransactionToProto(tx *finance.TransactionModel) *financepb.Transaction {
	protoTx := &financepb.Transaction{
		Id:          tx.ID.String(),
		UserId:      tx.UserID.String(),
		Type:        tx.Type,
		Amount:      tx.Amount,
		Description: tx.Description,
		Status:      tx.Status,
		CreatedAt:   timestamppb.New(tx.CreatedAt),
		UpdatedAt:   timestamppb.New(tx.UpdatedAt),
	}

	if tx.ToUserID != uuid.Nil {
		toUserID := tx.ToUserID.String()
		protoTx.ToUserId = &toUserID
	}

	return protoTx
}
