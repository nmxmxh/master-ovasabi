package finance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// CachedRepository wraps a Repository with caching capabilities.
type CachedRepository struct {
	repo   Repository
	cache  *redis.Cache
	logger *zap.Logger
}

// NewCachedRepository creates a new cached finance repository.
func NewCachedRepository(repo Repository, cache *redis.Cache, logger *zap.Logger) Repository {
	return &CachedRepository{
		repo:   repo,
		cache:  cache,
		logger: logger.With(zap.String("repository", "finance_cached")),
	}
}

// generateBalanceKey creates a cache key for user balance.
func (r *CachedRepository) generateBalanceKey(userID uuid.UUID) string {
	return fmt.Sprintf("%s:%s:balance:%s",
		redis.NamespaceCache,
		redis.ContextFinance,
		userID.String())
}

// generateTransactionKey creates a cache key for transaction.
func (r *CachedRepository) generateTransactionKey(txID uuid.UUID) string {
	return fmt.Sprintf("%s:%s:transaction:%s",
		redis.NamespaceCache,
		redis.ContextFinance,
		txID.String())
}

// GetBalance retrieves the balance from cache or repository.
func (r *CachedRepository) GetBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	key := r.generateBalanceKey(userID)

	// Try to get from cache
	var balance float64
	err := r.cache.Get(ctx, key, "", &balance)
	if err == nil {
		return balance, nil
	}

	// Get from repository
	balance, err = r.repo.GetBalance(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Cache the result
	if err := r.cache.Set(ctx, key, "", balance, redis.TTLFinanceBalance); err != nil {
		r.logger.Warn("failed to cache balance",
			zap.String("user_id", userID.String()),
			zap.Error(err))
	}

	return balance, nil
}

// UpdateBalance updates the balance and invalidates cache.
func (r *CachedRepository) UpdateBalance(ctx context.Context, userID uuid.UUID, amount float64) error {
	if err := r.repo.UpdateBalance(ctx, userID, amount); err != nil {
		return err
	}

	// Invalidate balance cache
	key := r.generateBalanceKey(userID)
	if err := r.cache.Delete(ctx, key, ""); err != nil {
		r.logger.Warn("failed to invalidate balance cache",
			zap.String("user_id", userID.String()),
			zap.Error(err))
	}

	return nil
}

// CreateTransaction creates a transaction and caches it.
func (r *CachedRepository) CreateTransaction(ctx context.Context, tx *TransactionModel) error {
	if err := r.repo.CreateTransaction(ctx, tx); err != nil {
		return err
	}

	// Cache the transaction
	key := r.generateTransactionKey(tx.ID)
	if err := r.cache.Set(ctx, key, "", tx, redis.TTLFinanceTransaction); err != nil {
		r.logger.Warn("failed to cache transaction",
			zap.String("transaction_id", tx.ID.String()),
			zap.Error(err))
	}

	return nil
}

// GetTransaction retrieves a transaction from cache or repository.
func (r *CachedRepository) GetTransaction(ctx context.Context, id uuid.UUID) (*TransactionModel, error) {
	key := r.generateTransactionKey(id)

	// Try to get from cache
	var tx TransactionModel
	err := r.cache.Get(ctx, key, "", &tx)
	if err == nil {
		return &tx, nil
	}

	// Get from repository
	tx2, err := r.repo.GetTransaction(ctx, id)
	if err != nil {
		return nil, err
	}

	if tx2 == nil {
		return nil, ErrTransactionNotFound
	}

	// Cache the result
	if err := r.cache.Set(ctx, key, "", tx2, redis.TTLFinanceTransaction); err != nil {
		r.logger.Warn("failed to cache transaction",
			zap.String("transaction_id", id.String()),
			zap.Error(err))
	}

	return tx2, nil
}

// ListTransactions retrieves transactions from repository (no caching for lists).
func (r *CachedRepository) ListTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*TransactionModel, error) {
	cacheKey := fmt.Sprintf("transactions:%s:%d:%d", userID.String(), limit, offset)
	cacheNamespace := "transactions"

	// Try to get from cache
	var transactions []*TransactionModel
	if err := r.cache.Get(ctx, cacheNamespace, cacheKey, &transactions); err == nil {
		return transactions, nil
	}

	// Get from repository
	transactions, err := r.repo.ListTransactions(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if err := r.cache.Set(ctx, cacheNamespace, cacheKey, transactions, time.Hour); err != nil {
		r.logger.Warn("failed to cache transactions", zap.Error(err))
	}

	return transactions, nil
}
