package commerce

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	productpb "github.com/nmxmxh/master-ovasabi/api/protos/product/v1"
	repo "github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

type Quote struct {
	QuoteID    string             `db:"quote_id"`
	UserID     string             `db:"user_id"`
	ProductID  string             `db:"product_id"`
	Amount     float64            `db:"amount"`
	Currency   string             `db:"currency"`
	Status     string             `db:"status"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	CreatedAt  time.Time          `db:"created_at"`
	UpdatedAt  time.Time          `db:"updated_at"`
	MasterID   int64              `db:"master_id"`
	MasterUUID string             `db:"master_uuid"`
	CampaignID int64              `db:"campaign_id"`
}

type Order struct {
	OrderID    string             `db:"order_id"`
	UserID     string             `db:"user_id"`
	Items      []OrderItem        // Not directly mapped, loaded separately
	Total      float64            `db:"total"`
	Currency   string             `db:"currency"`
	Status     string             `db:"status"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	CreatedAt  time.Time          `db:"created_at"`
	UpdatedAt  time.Time          `db:"updated_at"`
	MasterID   int64              `db:"master_id"`
	MasterUUID string             `db:"master_uuid"`
	CampaignID int64              `db:"campaign_id"`
}

type OrderItem struct {
	ID         int                `db:"id"`
	OrderID    string             `db:"order_id"`
	ProductID  string             `db:"product_id"`
	Quantity   int                `db:"quantity"`
	Price      float64            `db:"price"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	MasterID   int64              `db:"master_id"`   // NEW: For consistency with master-client pattern
	MasterUUID string             `db:"master_uuid"` // NEW: For consistency with master-client pattern
	CampaignID int64              `db:"campaign_id"` // NEW: For campaign-specific architecture
}

type Payment struct {
	PaymentID  string             `db:"payment_id"`
	OrderID    string             `db:"order_id"`
	UserID     string             `db:"user_id"`
	Amount     float64            `db:"amount"`
	Currency   string             `db:"currency"`
	Method     string             `db:"method"`
	Status     string             `db:"status"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	CreatedAt  time.Time          `db:"created_at"`
	UpdatedAt  time.Time          `db:"updated_at"`
	MasterID   int64              `db:"master_id"`
	MasterUUID string             `db:"master_uuid"`
	CampaignID int64              `db:"campaign_id"`
}

type Transaction struct {
	TransactionID string             `db:"transaction_id"`
	PaymentID     string             `db:"payment_id"`
	UserID        string             `db:"user_id"`
	Type          string             `db:"type"`
	Amount        float64            `db:"amount"`
	Currency      string             `db:"currency"`
	Status        string             `db:"status"`
	Metadata      *commonpb.Metadata `db:"metadata"`
	CreatedAt     time.Time          `db:"created_at"`
	UpdatedAt     time.Time          `db:"updated_at"`
	MasterID      int64              `db:"master_id"`
	MasterUUID    string             `db:"master_uuid"`
	CampaignID    int64              `db:"campaign_id"`
}

type Balance struct {
	UserID     string             `db:"user_id"`
	Currency   string             `db:"currency"`
	Amount     float64            `db:"amount"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	UpdatedAt  time.Time          `db:"updated_at"`
	CampaignID int64              `db:"campaign_id"`
	MasterID   int64              `db:"master_id"`
	MasterUUID string             `db:"master_uuid"`
}

type Event struct {
	ID         int64                  `db:"id"`        // Renamed from EventID for clarity and common PK naming
	EntityID   string                 `db:"entity_id"` // The ID of the entity the event is about
	EntityType string                 `db:"entity_type"`
	EventType  string                 `db:"event_type"`
	Payload    map[string]interface{} `db:"payload"`
	Metadata   *commonpb.Metadata     `db:"metadata"`
	CampaignID int64                  `db:"campaign_id"`
	CreatedAt  time.Time              `db:"occurred_at"`
	// NOTE: It is strongly recommended to use the centralized `service_event` table for all event logging.
	// This struct is aligned with the provided migration for `service_commerce_event`.
	MasterID   int64  `db:"master_id"`
	MasterUUID string `db:"master_uuid"`
}

// --- Investment ---.
type InvestmentAccount struct {
	AccountID  string             `db:"account_id"`
	MasterID   int64              `db:"master_id"`
	OwnerID    string             `db:"owner_id"`
	Type       string             `db:"type"`
	Currency   string             `db:"currency"`
	Balance    float64            `db:"balance"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	CreatedAt  time.Time          `db:"created_at"`
	UpdatedAt  time.Time          `db:"updated_at"`
	MasterUUID string             `db:"master_uuid"`
	CampaignID int64              `db:"campaign_id"`
}

type InvestmentOrder struct {
	OrderID    string             `db:"order_id"`
	MasterID   int64              `db:"master_id"`
	AccountID  string             `db:"account_id"`
	AssetID    string             `db:"asset_id"`
	Quantity   float64            `db:"quantity"`
	Price      float64            `db:"price"`
	OrderType  string             `db:"order_type"`
	Status     int                `db:"status"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	CreatedAt  time.Time          `db:"created_at"`
	UpdatedAt  time.Time          `db:"updated_at"`
	MasterUUID string             `db:"master_uuid"`
	CampaignID int64              `db:"campaign_id"`
}

type Portfolio struct {
	PortfolioID string             `db:"portfolio_id"`
	AccountID   string             `db:"account_id"`
	Metadata    *commonpb.Metadata `db:"metadata"`
	CreatedAt   time.Time          `db:"created_at"`
	UpdatedAt   time.Time          `db:"updated_at"`
	MasterID    int64              `db:"master_id"`
	MasterUUID  string             `db:"master_uuid"`
	CampaignID  int64              `db:"campaign_id"`
}

type AssetPosition struct {
	ID           int64              `db:"id"`
	PortfolioID  int64              `db:"portfolio_id"`
	AssetID      string             `db:"asset_id"`
	Quantity     float64            `db:"quantity"`
	AveragePrice float64            `db:"average_price"`
	Metadata     *commonpb.Metadata `db:"metadata"`
	MasterID     int64              `db:"master_id"`
	MasterUUID   string             `db:"master_uuid"`
	CampaignID   int64              `db:"campaign_id"`
}

// --- Banking ---.
type BankAccount struct {
	AccountID  string             `db:"account_id"`
	MasterID   int64              `db:"master_id"`
	UserID     string             `db:"user_id"`
	IBAN       string             `db:"iban"`
	BIC        string             `db:"bic"`
	Currency   string             `db:"currency"`
	Balance    float64            `db:"balance"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	CreatedAt  time.Time          `db:"created_at"`
	UpdatedAt  time.Time          `db:"updated_at"`
	MasterUUID string             `db:"master_uuid"`
	CampaignID int64              `db:"campaign_id"`
}

type BankTransfer struct {
	TransferID    string             `db:"transfer_id"`
	MasterID      int64              `db:"master_id"`
	FromAccountID string             `db:"from_account_id"`
	ToAccountID   string             `db:"to_account_id"`
	Amount        float64            `db:"amount"`
	Currency      string             `db:"currency"`
	Status        int                `db:"status"`
	Metadata      *commonpb.Metadata `db:"metadata"`
	CreatedAt     time.Time          `db:"created_at"`
	CampaignID    int64              `db:"campaign_id"`
	MasterUUID    string             `db:"master_uuid"`
}

type BankStatement struct {
	StatementID int64              `db:"statement_id"`
	MasterID    int64              `db:"master_id"`
	AccountID   string             `db:"account_id"`
	Metadata    *commonpb.Metadata `db:"metadata"`
	CreatedAt   time.Time          `db:"created_at"`
	MasterUUID  string             `db:"master_uuid"`
	CampaignID  int64              `db:"campaign_id"`
}

// --- Marketplace ---.
type MarketplaceListing struct {
	ListingID  string             `db:"listing_id"`
	MasterID   int64              `db:"master_id"`
	SellerID   string             `db:"seller_id"`
	ProductID  string             `db:"product_id"`
	Price      float64            `db:"price"`
	Currency   string             `db:"currency"`
	Status     int                `db:"status"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	CreatedAt  time.Time          `db:"created_at"`
	CampaignID int64              `db:"campaign_id"`
	MasterUUID string             `db:"master_uuid"`
}

type MarketplaceOrder struct {
	OrderID    string             `db:"order_id"`
	MasterID   int64              `db:"master_id"`
	ListingID  string             `db:"listing_id"`
	BuyerID    string             `db:"buyer_id"`
	Price      float64            `db:"price"`
	Currency   string             `db:"currency"`
	Status     int                `db:"status"`
	CampaignID int64              `db:"campaign_id"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	CreatedAt  time.Time          `db:"created_at"`
	MasterUUID string             `db:"master_uuid"`
}

type MarketplaceOffer struct {
	OfferID    string             `db:"offer_id"`
	MasterID   int64              `db:"master_id"`
	ListingID  string             `db:"listing_id"`
	BuyerID    string             `db:"buyer_id"`
	OfferPrice float64            `db:"offer_price"`
	Currency   string             `db:"currency"`
	CampaignID int64              `db:"campaign_id"`
	Status     int                `db:"status"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	CreatedAt  time.Time          `db:"created_at"`
	MasterUUID string             `db:"master_uuid"`
}

// --- Exchange ---.
type ExchangeOrder struct {
	OrderID    string             `db:"order_id"`
	MasterID   int64              `db:"master_id"`
	AccountID  string             `db:"account_id"`
	Pair       string             `db:"pair"`
	Amount     float64            `db:"amount"`
	Price      float64            `db:"price"`
	OrderType  string             `db:"order_type"`
	Status     int                `db:"status"`
	CampaignID int64              `db:"campaign_id"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	CreatedAt  time.Time          `db:"created_at"`
	MasterUUID string             `db:"master_uuid"`
}

type ExchangePair struct {
	PairID     string             `db:"pair_id"`
	MasterID   int64              `db:"master_id"`
	BaseAsset  string             `db:"base_asset"`
	QuoteAsset string             `db:"quote_asset"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	CampaignID int64              `db:"campaign_id"`
	MasterUUID string             `db:"master_uuid"`
}

type ExchangeRate struct {
	RateID     int64              `db:"rate_id"`
	MasterID   int64              `db:"master_id"`
	PairID     string             `db:"pair_id"`
	Rate       float64            `db:"rate"`
	Timestamp  time.Time          `db:"timestamp"`
	Metadata   *commonpb.Metadata `db:"metadata"`
	CampaignID int64              `db:"campaign_id"`
	MasterUUID string             `db:"master_uuid"`
}

type repository struct {
	*repo.BaseRepository
	masterRepo repo.MasterRepository
	log        *zap.Logger
}

func NewRepository(db *sql.DB, masterRepo repo.MasterRepository, logger *zap.Logger) Repository {
	return &repository{
		BaseRepository: repo.NewBaseRepository(db, logger),
		masterRepo:     masterRepo,
		log:            logger,
	}
}

// RepositoryInterface defines all public methods for the commerce repository.
type Repository interface {
	CreateQuote(ctx context.Context, q *Quote) error
	GetQuote(ctx context.Context, quoteID string) (*Quote, error)
	ListQuotes(ctx context.Context, userID string, campaignID int64, page, pageSize int) ([]*Quote, int, error)
	CreateOrder(ctx context.Context, o *Order, items []OrderItem) error
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	ListOrders(ctx context.Context, userID string, campaignID int64, page, pageSize int) ([]*Order, int, error)
	UpdateOrderStatus(ctx context.Context, orderID, status string) error
	ListOrderItems(ctx context.Context, orderID string) ([]*OrderItem, error)
	CreatePayment(ctx context.Context, p *Payment) error
	GetPayment(ctx context.Context, paymentID string) (*Payment, error)
	UpdatePaymentStatus(ctx context.Context, paymentID, status string) error
	CreateTransaction(ctx context.Context, t *Transaction) error
	GetTransaction(ctx context.Context, transactionID string) (*Transaction, error)
	ListTransactions(ctx context.Context, userID string, campaignID int64, page, pageSize int) ([]*Transaction, int, error)
	GetBalance(ctx context.Context, userID, currency string) (*Balance, error)
	ListBalances(ctx context.Context, userID string) ([]*Balance, error) // This method was already correct
	UpdateBalance(ctx context.Context, userID, currency string, amount float64, campaignID int64) error
	LogEvent(ctx context.Context, e *Event) error
	ListEvents(ctx context.Context, entityID, entityType string, campaignID int64, page, pageSize int) ([]*Event, int, error)
	CreateInvestmentAccount(ctx context.Context, a *InvestmentAccount) error
	CreateInvestmentOrder(ctx context.Context, o *InvestmentOrder) error
	CreatePortfolio(ctx context.Context, p *Portfolio) error
	CreateAssetPosition(ctx context.Context, pos *AssetPosition) error
	CreateBankAccount(ctx context.Context, a *BankAccount) error
	CreateBankTransfer(ctx context.Context, t *BankTransfer) error
	CreateBankStatement(ctx context.Context, s *BankStatement) error
	CreateMarketplaceListing(ctx context.Context, l *MarketplaceListing) error
	CreateMarketplaceOrder(ctx context.Context, o *MarketplaceOrder) error // CampaignID added to struct
	CreateMarketplaceOffer(ctx context.Context, o *MarketplaceOffer) error
	CreateExchangeOrder(ctx context.Context, o *ExchangeOrder) error
	CreateExchangePair(ctx context.Context, p *ExchangePair) error
	CreateExchangeRate(ctx context.Context, r8 *ExchangeRate) error
	GetInvestmentAccount(ctx context.Context, accountID string) (*InvestmentAccount, error)
	ListInvestmentAccounts(ctx context.Context, ownerID string) ([]*InvestmentAccount, error)
	GetPortfolio(ctx context.Context, portfolioID string) (*Portfolio, error)
	ListPortfolios(ctx context.Context, accountID string, campaignID int64) ([]*Portfolio, error) // CampaignID added
	GetAssetPosition(ctx context.Context, id int64) (*AssetPosition, error)
	ListAssetPositions(ctx context.Context, portfolioID int64, campaignID int64) ([]*AssetPosition, error) // CampaignID added
	CreateProduct(ctx context.Context, p *productpb.Product) (*productpb.Product, error)
}

// GetDB returns the database instance.
func (r *repository) GetDB() *sql.DB {
	return r.BaseRepository.GetDB()
}

// convertMetadataFields converts map[string]*structpb.Value to map[string]interface{}.
func convertMetadataFields(fields map[string]*structpb.Value) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range fields {
		result[k] = v.AsInterface()
	}
	return result
}

// Quotes.
func (r *repository) CreateQuote(ctx context.Context, q *Quote) error {
	if q.Metadata == nil {
		q.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}
	metaJSON, err := repo.ToJSONB(convertMetadataFields(q.Metadata.ServiceSpecific.Fields))
	if err != nil {
		return err
	}

	// 1. Create master entity for the quote
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", err)
		}
	}()

	masterID, masterUUID, err := r.masterRepo.Create(ctx, tx, repo.EntityType("quote"), q.QuoteID)
	if err != nil {
		return fmt.Errorf("failed to create master entity for quote: %w", err)
	}
	q.MasterID = masterID
	q.MasterUUID = masterUUID

	query := `
		INSERT INTO service_commerce_quote (
			quote_id, master_id, master_uuid, user_id, product_id, amount, currency, status, metadata, created_at, updated_at, campaign_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now(), $10
		)
	`
	_, err = tx.ExecContext(ctx, query,
		q.QuoteID,
		q.MasterID,
		q.MasterUUID,
		q.UserID,
		q.ProductID,
		q.Amount,
		q.Currency,
		q.Status,
		metaJSON,
		q.CampaignID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *repository) GetQuote(ctx context.Context, quoteID string) (*Quote, error) {
	query := `
		SELECT quote_id, master_id, master_uuid, user_id, product_id, amount, currency, status, metadata, created_at, updated_at, campaign_id
		FROM service_commerce_quote
		WHERE quote_id = $1
	`
	var quote Quote
	err := r.GetDB().QueryRowContext(ctx, query, quoteID).Scan(
		&quote.QuoteID,
		&quote.MasterID,
		&quote.MasterUUID,
		&quote.UserID,
		&quote.ProductID,
		&quote.Amount,
		&quote.Currency,
		&quote.Status,
		&quote.Metadata,
		&quote.CreatedAt,
		&quote.UpdatedAt,
		&quote.CampaignID,
	)
	if err != nil {
		return nil, err
	}
	return &quote, nil
}

func (r *repository) ListQuotes(ctx context.Context, userID string, campaignID int64, page, pageSize int) ([]*Quote, int, error) {
	query := `
		SELECT quote_id, master_id, master_uuid, user_id, product_id, amount, currency, status, metadata, created_at, updated_at, campaign_id
		FROM service_commerce_quote
		WHERE user_id = $1 AND campaign_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.GetDB().QueryContext(ctx, query, userID, campaignID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "failed to query quotes", err)
	}
	defer rows.Close()

	var quotes []*Quote
	for rows.Next() {
		var quote Quote
		if err := rows.Scan(
			&quote.QuoteID,
			&quote.MasterID,
			&quote.MasterUUID,
			&quote.UserID,
			&quote.ProductID,
			&quote.Amount,
			&quote.Currency,
			&quote.Status,
			&quote.Metadata,
			&quote.CreatedAt,
			&quote.UpdatedAt,
			&quote.CampaignID,
		); err != nil {
			return nil, 0, graceful.WrapErr(ctx, codes.Internal, "error scanning quote", err)
		}
		quotes = append(quotes, &quote)
	}

	countQuery := `
		SELECT COUNT(*) FROM service_commerce_quote
		WHERE user_id = $1 AND campaign_id = $2
	`
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, userID, campaignID).Scan(&total); err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "failed to count quotes", err)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "error iterating quotes", err)
	}

	return quotes, total, nil
}

// Orders.
func (r *repository) CreateOrder(ctx context.Context, o *Order, items []OrderItem) error {
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 1. Use masterRepo to create the master entity record
	masterID, masterUUID, err := r.masterRepo.Create(ctx, tx, repo.EntityType("order"), o.OrderID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for order: %w", err)
	}
	o.MasterID = masterID
	o.MasterUUID = masterUUID

	// 2. Use canonical metadata marshaling
	metaJSON, err := metadata.MarshalCanonical(o.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to marshal order metadata: %w", err)
	}

	// 3. Insert into service_commerce_order using the new master_id
	_, err = tx.ExecContext(ctx, `
		INSERT INTO service_commerce_order (order_id, master_id, user_id, total, currency, status, metadata, created_at, updated_at, campaign_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now(), $8)
	`, o.OrderID, o.MasterID, o.UserID, o.Total, o.Currency, o.Status, metaJSON, o.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to insert into service_commerce_order: %w", err)
	}

	// 4. Insert order items
	for _, item := range items {
		// Create master entity for each order item
		itemMasterID, itemMasterUUID, err := r.masterRepo.Create(ctx, tx, repo.EntityType("order_item"), fmt.Sprintf("%s:%s", o.OrderID, item.ProductID))
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
			return fmt.Errorf("failed to create master entity for order item: %w", err)
		}
		item.MasterID = itemMasterID
		item.MasterUUID = itemMasterUUID

		itemMetaJSON, err := metadata.MarshalCanonical(item.Metadata)
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction for order item metadata: %v", rbErr)
			}
			return fmt.Errorf("failed to marshal order item metadata: %w", err)
		}
		_, err = tx.ExecContext(ctx, `
		INSERT INTO service_commerce_order_item (order_id, master_id, master_uuid, product_id, quantity, price, metadata, created_at, updated_at, campaign_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now(), $8)
	`, o.OrderID, item.MasterID, item.MasterUUID, item.ProductID, item.Quantity, item.Price, itemMetaJSON, o.CampaignID) // Pass campaign_id from the order
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("failed to rollback transaction for order item: %v", rbErr)
			}
			return fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for order: %w", err)
	}
	return nil
}

func (r *repository) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	var o Order
	var metaJSON []byte
	err := r.GetDB().QueryRowContext(ctx, `
		SELECT order_id, master_id, master_uuid, user_id, total, currency, status, metadata, created_at, updated_at, campaign_id
		FROM service_commerce_order
		WHERE order_id = $1
	`, orderID).Scan(&o.OrderID, &o.MasterID, &o.MasterUUID, &o.UserID, &o.Total, &o.Currency, &o.Status, &metaJSON, &o.CreatedAt, &o.UpdatedAt, &o.CampaignID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	if len(metaJSON) > 0 {
		var meta commonpb.Metadata
		if err := json.Unmarshal(metaJSON, &meta); err == nil {
			o.Metadata = &meta
		}
	}
	// Optionally: join with master_entity for more info if needed
	return &o, nil
}

func (r *repository) ListOrders(ctx context.Context, userID string, campaignID int64, page, pageSize int) ([]*Order, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var total int
	err := r.GetDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM service_commerce_order WHERE user_id = $1 AND campaign_id = $2`, userID, campaignID).Scan(&total)
	if err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "failed to count orders", err)
	}
	rows, err := r.GetDB().QueryContext(ctx, `SELECT order_id, master_id, master_uuid, user_id, total, currency, status, metadata, created_at, updated_at, campaign_id FROM service_commerce_order WHERE user_id = $1 AND campaign_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`, userID, campaignID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "failed to query orders", err)
	}
	defer rows.Close()
	var orders []*Order
	for rows.Next() {
		var o Order
		var metaJSON []byte
		if err := rows.Scan(&o.OrderID, &o.MasterID, &o.MasterUUID, &o.UserID, &o.Total, &o.Currency, &o.Status, &metaJSON, &o.CreatedAt, &o.UpdatedAt, &o.CampaignID); err != nil {
			return nil, 0, graceful.WrapErr(ctx, codes.Internal, "error scanning order", err)
		}
		if len(metaJSON) > 0 {
			var meta commonpb.Metadata
			if err := json.Unmarshal(metaJSON, &meta); err == nil {
				o.Metadata = &meta
			}
		}
		orders = append(orders, &o)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "error iterating orders", err)
	}
	return orders, total, nil
}

func (r *repository) UpdateOrderStatus(ctx context.Context, orderID, status string) error {
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE service_commerce_order SET status = $1, updated_at = now() WHERE order_id = $2`, status, orderID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	if n == 0 {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func (r *repository) ListOrderItems(ctx context.Context, orderID string) ([]*OrderItem, error) {
	rows, err := r.GetDB().QueryContext(ctx, `SELECT id, order_id, master_id, master_uuid, product_id, quantity, price, metadata, campaign_id FROM service_commerce_order_item WHERE order_id = $1`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*OrderItem
	for rows.Next() {
		var item OrderItem
		var metaJSON []byte
		if err := rows.Scan(&item.ID, &item.OrderID, &item.MasterID, &item.MasterUUID, &item.ProductID, &item.Quantity, &item.Price, &metaJSON, &item.CampaignID); err != nil {
			return nil, err
		}
		if len(metaJSON) > 0 {
			var meta commonpb.Metadata
			if err := json.Unmarshal(metaJSON, &meta); err == nil {
				item.Metadata = &meta
			}
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// Payments.
func (r *repository) CreatePayment(ctx context.Context, p *Payment) error {
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction on panic: %v", rbErr)
			}
			panic(r) // Re-throw panic
		}
	}()

	// 1. Use masterRepo to create the master entity record
	p.MasterID, p.MasterUUID, err = r.masterRepo.Create(ctx, tx, repo.EntityType("payment"), p.PaymentID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for payment: %w", err)
	}

	// Defensive: ensure metadata is always valid JSON
	if p.Metadata == nil {
		p.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}

	// 2. Marshal metadata
	metaJSON, err := metadata.MarshalCanonical(p.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to marshal payment metadata: %w", err)
	}

	// 3. Insert into service_commerce_payment
	_, err = tx.ExecContext(ctx, `
		INSERT INTO service_commerce_payment (payment_id, master_id, master_uuid, order_id, user_id, amount, currency, method, status, metadata, created_at, updated_at, campaign_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), now(), $11)
	`, p.PaymentID, p.MasterID, p.MasterUUID, p.OrderID, p.UserID, p.Amount, p.Currency, p.Method, p.Status, metaJSON, p.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to insert into service_commerce_payment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *repository) GetPayment(ctx context.Context, paymentID string) (*Payment, error) {
	var p Payment
	var metaJSON []byte // Added master_uuid to SELECT and Scan
	err := r.GetDB().QueryRowContext(ctx, `SELECT payment_id, master_id, master_uuid, order_id, user_id, amount, currency, method, status, metadata, created_at, updated_at, campaign_id FROM service_commerce_payment WHERE payment_id = $1`, paymentID).Scan(&p.PaymentID, &p.MasterID, &p.MasterUUID, &p.OrderID, &p.UserID, &p.Amount, &p.Currency, &p.Method, &p.Status, &metaJSON, &p.CreatedAt, &p.UpdatedAt, &p.CampaignID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	if len(metaJSON) > 0 {
		var meta commonpb.Metadata
		if err := json.Unmarshal(metaJSON, &meta); err == nil {
			p.Metadata = &meta
		}
	}
	return &p, nil
}

func (r *repository) UpdatePaymentStatus(ctx context.Context, paymentID, status string) error {
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE service_commerce_payment SET status = $1, updated_at = now() WHERE payment_id = $2`, status, paymentID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	if n == 0 {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return sql.ErrNoRows
	}
	return tx.Commit()
}

// Transactions.
func (r *repository) CreateTransaction(ctx context.Context, t *Transaction) error {
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if rec := recover(); rec != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction on panic: %v", rbErr)
			}
			panic(rec) // Re-throw panic
		}
	}()

	// 1. Use masterRepo to create the master entity record
	t.MasterID, t.MasterUUID, err = r.masterRepo.Create(ctx, tx, repo.EntityType("transaction"), t.TransactionID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for transaction: %w", err)
	}

	// Defensive: ensure metadata is always valid JSON
	if t.Metadata == nil {
		t.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}

	// 2. Marshal metadata
	metaJSON, err := metadata.MarshalCanonical(t.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to marshal transaction metadata: %w", err)
	}

	// 3. Insert into service_commerce_transaction
	_, err = tx.ExecContext(ctx, `
		INSERT INTO service_commerce_transaction (transaction_id, master_id, master_uuid, payment_id, user_id, type, amount, currency, status, metadata, created_at, updated_at, campaign_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), now(), $11)
	`, t.TransactionID, t.MasterID, t.MasterUUID, t.PaymentID, t.UserID, t.Type, t.Amount, t.Currency, t.Status, metaJSON, t.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to insert into service_commerce_transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *repository) GetTransaction(ctx context.Context, transactionID string) (*Transaction, error) {
	var t Transaction
	var metaJSON []byte // Added master_uuid to SELECT and Scan
	err := r.GetDB().QueryRowContext(ctx, `SELECT transaction_id, master_id, master_uuid, payment_id, user_id, type, amount, currency, status, metadata, created_at, updated_at, campaign_id FROM service_commerce_transaction WHERE transaction_id = $1`, transactionID).Scan(&t.TransactionID, &t.MasterID, &t.MasterUUID, &t.PaymentID, &t.UserID, &t.Type, &t.Amount, &t.Currency, &t.Status, &metaJSON, &t.CreatedAt, &t.UpdatedAt, &t.CampaignID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	if len(metaJSON) > 0 {
		var meta commonpb.Metadata
		if err := json.Unmarshal(metaJSON, &meta); err == nil {
			t.Metadata = &meta
		}
	}
	return &t, nil
}

func (r *repository) ListTransactions(ctx context.Context, userID string, campaignID int64, page, pageSize int) ([]*Transaction, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var total int
	err := r.GetDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM service_commerce_transaction WHERE user_id = $1 AND campaign_id = $2`, userID, campaignID).Scan(&total)
	if err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "failed to count transactions", err)
	}
	rows, err := r.GetDB().QueryContext(ctx, `SELECT transaction_id, master_id, master_uuid, payment_id, user_id, type, amount, currency, status, metadata, created_at, updated_at, campaign_id FROM service_commerce_transaction WHERE user_id = $1 AND campaign_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`, userID, campaignID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "failed to query transactions", err)
	}
	defer rows.Close()
	var txs []*Transaction
	for rows.Next() {
		var t Transaction
		var metaJSON []byte
		if err := rows.Scan(&t.TransactionID, &t.MasterID, &t.MasterUUID, &t.PaymentID, &t.UserID, &t.Type, &t.Amount, &t.Currency, &t.Status, &metaJSON, &t.CreatedAt, &t.UpdatedAt, &t.CampaignID); err != nil {
			return nil, 0, graceful.WrapErr(ctx, codes.Internal, "error scanning transaction", err)
		}
		if len(metaJSON) > 0 {
			var meta commonpb.Metadata
			if err := json.Unmarshal(metaJSON, &meta); err == nil {
				t.Metadata = &meta
			}
		}
		txs = append(txs, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "error iterating transactions", err)
	}
	return txs, total, nil
}

// Balances.
func (r *repository) GetBalance(ctx context.Context, userID, currency string) (*Balance, error) {
	var b Balance
	err := r.GetDB().QueryRowContext(ctx, `SELECT user_id, currency, amount, metadata, updated_at, master_id, master_uuid, campaign_id FROM service_commerce_balance WHERE user_id = $1 AND currency = $2`, userID, currency).Scan(&b.UserID, &b.Currency, &b.Amount, &b.Metadata, &b.UpdatedAt, &b.MasterID, &b.MasterUUID, &b.CampaignID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &b, nil
}

func (r *repository) ListBalances(ctx context.Context, userID string) ([]*Balance, error) {
	rows, err := r.GetDB().QueryContext(ctx, `SELECT user_id, currency, amount, metadata, updated_at, master_id, master_uuid, campaign_id FROM service_commerce_balance WHERE user_id = $1`, userID)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to query balances", err)
	}
	defer rows.Close()
	var balances []*Balance
	for rows.Next() {
		var b Balance
		if err := rows.Scan(&b.UserID, &b.Currency, &b.Amount, &b.Metadata, &b.UpdatedAt, &b.MasterID, &b.MasterUUID, &b.CampaignID); err != nil {
			return nil, graceful.WrapErr(ctx, codes.Internal, "error scanning balance", err)
		}
		balances = append(balances, &b)
	}
	if err := rows.Err(); err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "error iterating balances", err)
	}
	return balances, nil
}

func (r *repository) UpdateBalance(ctx context.Context, userID, currency string, amount float64, campaignID int64) error {
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil { // Filter by campaign_id
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE service_commerce_balance SET amount = $1, updated_at = now() WHERE user_id = $2 AND currency = $3 AND campaign_id = $4`, amount, userID, currency, campaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	if n == 0 {
		if rbErr := tx.Rollback(); rbErr != nil { // Filter by campaign_id
			log.Printf("rollback failed: %v", rbErr)
		}
		return sql.ErrNoRows // Filter by campaign_id
	}
	return tx.Commit()
}

// Events.
func (r *repository) LogEvent(ctx context.Context, e *Event) error {
	// Defensive: ensure metadata is always valid JSON
	if e.Metadata == nil {
		e.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}

	payloadJSON, err := json.Marshal(e.Payload) // Renamed variable for clarity
	if err != nil {
		return err
	}
	metaJSON, err := metadata.MarshalCanonical(e.Metadata) // Corrected metadata marshalling
	if err != nil {
		return err
	}
	_, err = r.GetDB().ExecContext(ctx, `INSERT INTO service_commerce_event (master_id, master_uuid, entity_id, entity_type, event_type, payload, metadata, occurred_at, campaign_id) VALUES ($1, $2, $3, $4, $5, $6, $7, now(), $8)`, e.MasterID, e.MasterUUID, e.EntityID, e.EntityType, e.EventType, payloadJSON, metaJSON, e.CampaignID)
	return err
}

func (r *repository) ListEvents(ctx context.Context, entityID, entityType string, campaignID int64, page, pageSize int) ([]*Event, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var total int
	err := r.GetDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM service_commerce_event WHERE entity_id = $1 AND entity_type = $2 AND campaign_id = $3`, entityID, entityType, campaignID).Scan(&total)
	if err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "failed to count events", err)
	}
	rows, err := r.GetDB().QueryContext(ctx, `SELECT id, master_id, master_uuid, entity_id, entity_type, event_type, payload, metadata, occurred_at, campaign_id FROM service_commerce_event WHERE entity_id = $1 AND entity_type = $2 AND campaign_id = $3 ORDER BY occurred_at DESC LIMIT $4 OFFSET $5`, entityID, entityType, campaignID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "failed to query events", err)
	}
	defer rows.Close()
	var events []*Event
	for rows.Next() {
		var e Event
		var payloadJSON []byte
		var metaJSON []byte
		if err := rows.Scan(&e.ID, &e.MasterID, &e.MasterUUID, &e.EntityID, &e.EntityType, &e.EventType, &payloadJSON, &metaJSON, &e.CreatedAt, &e.CampaignID); err != nil {
			return nil, 0, graceful.WrapErr(ctx, codes.Internal, "error scanning event", err)
		}
		if len(payloadJSON) > 0 {
			if err := json.Unmarshal(payloadJSON, &e.Payload); err != nil {
				e.Payload = map[string]interface{}{"error": "unmarshal failed"}
			}
		}
		if len(metaJSON) > 0 {
			if err := json.Unmarshal(metaJSON, &e.Metadata); err != nil {
				e.Metadata = nil
			}
		}
		events = append(events, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "error iterating events", err)
	}
	return events, total, nil
}

// --- Investment ---.
func (r *repository) CreateInvestmentAccount(ctx context.Context, a *InvestmentAccount) error {
	// Defensive: ensure metadata is always valid JSON
	if a.Metadata == nil {
		a.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
			panic(r) // Re-throw panic
		}
	}()
	a.MasterID, a.MasterUUID, err = r.masterRepo.Create(ctx, tx, repo.EntityType("investment_account"), a.AccountID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for investment account: %w", err)
	}
	metaJSON, err := metadata.MarshalCanonical(a.Metadata) // Added campaign_id to INSERT
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to marshal investment account metadata: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_investment_account (account_id, master_id, master_uuid, owner_id, type, currency, balance, metadata, created_at, updated_at, campaign_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), now(), $9)`, a.AccountID, a.MasterID, a.MasterUUID, a.OwnerID, a.Type, a.Currency, a.Balance, metaJSON, a.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to insert into service_commerce_investment_account: %w", err)
	}
	return tx.Commit()
}

func (r *repository) CreateInvestmentOrder(ctx context.Context, o *InvestmentOrder) error {
	// Defensive: ensure metadata is always valid JSON
	if o.Metadata == nil {
		o.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
			panic(r) // Re-throw panic
		}
	}()
	o.MasterID, o.MasterUUID, err = r.masterRepo.Create(ctx, tx, repo.EntityType("investment_order"), o.OrderID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for investment order: %w", err)
	}
	metaJSON, err := metadata.MarshalCanonical(o.Metadata) // Added campaign_id to INSERT
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to marshal investment order metadata: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_investment_order (order_id, master_id, master_uuid, account_id, asset_id, quantity, price, order_type, status, metadata, created_at, updated_at, campaign_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), now(), $11)`, o.OrderID, o.MasterID, o.MasterUUID, o.AccountID, o.AssetID, o.Quantity, o.Price, o.OrderType, o.Status, metaJSON, o.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to insert into service_commerce_investment_order: %w", err)
	}
	return tx.Commit()
}

func (r *repository) CreatePortfolio(ctx context.Context, p *Portfolio) error {
	// Defensive: ensure metadata is always valid JSON
	if p.Metadata == nil {
		p.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
			panic(r) // Re-throw panic
		}
	}()
	p.MasterID, p.MasterUUID, err = r.masterRepo.Create(ctx, tx, repo.EntityType("portfolio"), p.PortfolioID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for portfolio: %w", err)
	}
	metaJSON, err := metadata.MarshalCanonical(p.Metadata) // Added campaign_id to INSERT
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to marshal portfolio metadata: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_portfolio (portfolio_id, master_id, master_uuid, account_id, metadata, created_at, updated_at, campaign_id) VALUES ($1, $2, $3, $4, $5, now(), now(), $6)`, p.PortfolioID, p.MasterID, p.MasterUUID, p.AccountID, metaJSON, p.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to insert into service_commerce_portfolio: %w", err)
	}
	return tx.Commit()
}

func (r *repository) CreateAssetPosition(ctx context.Context, pos *AssetPosition) error {
	// Defensive: ensure metadata is always valid JSON
	if pos.Metadata == nil {
		pos.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}

	// 1. Create master entity for the asset position
	masterID, masterUUID, err := r.masterRepo.Create(ctx, tx, repo.EntityType("asset_position"), fmt.Sprintf("%d:%s", pos.PortfolioID, pos.AssetID))
	if err != nil {
		return fmt.Errorf("failed to create master entity for asset position: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
			panic(r) // Re-throw panic
		}
	}()
	pos.MasterID, pos.MasterUUID, err = r.masterRepo.Create(ctx, tx, repo.EntityType("asset_position"), fmt.Sprintf("%d:%s", pos.PortfolioID, pos.AssetID))
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for asset position: %w", err)
	}
	pos.MasterID = masterID
	pos.MasterUUID = masterUUID

	// 2. Marshal metadata
	metaJSON, err := metadata.MarshalCanonical(pos.Metadata) // Added campaign_id to INSERT
	if err != nil {
		return fmt.Errorf("failed to marshal asset position metadata: %w", err)
	}

	// 3. Insert into service_commerce_asset_position
	_, err = r.GetDB().ExecContext(ctx, `INSERT INTO service_commerce_asset_position (portfolio_id, master_id, master_uuid, asset_id, quantity, average_price, metadata, campaign_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, pos.PortfolioID, pos.MasterID, pos.MasterUUID, pos.AssetID, pos.Quantity, pos.AveragePrice, metaJSON, pos.CampaignID)
	return err
}

// --- Banking ---.
func (r *repository) CreateBankAccount(ctx context.Context, a *BankAccount) error {
	// Defensive: ensure metadata is always valid JSON
	if a.Metadata == nil {
		a.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
			panic(r) // Re-throw panic
		}
	}()
	a.MasterID, a.MasterUUID, err = r.masterRepo.Create(ctx, tx, repo.EntityType("bank_account"), a.AccountID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for bank account: %w", err)
	}
	metaJSON, err := metadata.MarshalCanonical(a.Metadata) // Added campaign_id to INSERT
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to marshal bank account metadata: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_bank_account (account_id, master_id, master_uuid, user_id, iban, bic, currency, balance, metadata, created_at, updated_at, campaign_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now(), $10)`, a.AccountID, a.MasterID, a.MasterUUID, a.UserID, a.IBAN, a.BIC, a.Currency, a.Balance, metaJSON, a.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to insert into service_commerce_bank_account: %w", err)
	}
	return tx.Commit()
}

func (r *repository) CreateBankTransfer(ctx context.Context, t *BankTransfer) error {
	// Defensive: ensure metadata is always valid JSON
	if t.Metadata == nil {
		t.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
			panic(r) // Re-throw panic
		}
	}()
	t.MasterID, t.MasterUUID, err = r.masterRepo.Create(ctx, tx, repo.EntityType("bank_transfer"), t.TransferID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for bank transfer: %w", err)
	}
	metaJSON, err := metadata.MarshalCanonical(t.Metadata) // Added campaign_id to INSERT
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to marshal bank transfer metadata: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_bank_transfer (transfer_id, master_id, master_uuid, from_account_id, to_account_id, amount, currency, status, metadata, created_at, campaign_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), $10)`, t.TransferID, t.MasterID, t.MasterUUID, t.FromAccountID, t.ToAccountID, t.Amount, t.Currency, t.Status, metaJSON, t.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to insert into service_commerce_bank_transfer: %w", err)
	}
	return tx.Commit()
}

func (r *repository) CreateBankStatement(ctx context.Context, s *BankStatement) error {
	// Defensive: ensure metadata is always valid JSON
	if s.Metadata == nil {
		s.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
			panic(r) // Re-throw panic
		}
	}()
	s.MasterID, s.MasterUUID, err = r.masterRepo.Create(ctx, tx, repo.EntityType("bank_statement"), fmt.Sprintf("%d", s.StatementID)) // Assuming StatementID is unique enough for name
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for bank statement: %w", err)
	}
	metaJSON, err := metadata.MarshalCanonical(s.Metadata) // Added campaign_id to INSERT
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to marshal bank statement metadata: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_bank_statement (statement_id, master_id, master_uuid, account_id, metadata, created_at, campaign_id) VALUES ($1, $2, $3, $4, $5, now(), $6)`, s.StatementID, s.MasterID, s.MasterUUID, s.AccountID, metaJSON, s.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to insert into service_commerce_bank_statement: %w", err)
	}
	return tx.Commit()
}

// --- Marketplace ---.
func (r *repository) CreateMarketplaceListing(ctx context.Context, l *MarketplaceListing) error {
	// Defensive: ensure metadata is always valid JSON
	if l.Metadata == nil {
		l.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
			panic(r) // Re-throw panic
		}
	}()
	l.MasterID, l.MasterUUID, err = r.masterRepo.Create(ctx, tx, repo.EntityType("marketplace_listing"), l.ListingID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for marketplace listing: %w", err)
	}
	metaJSON, err := metadata.MarshalCanonical(l.Metadata) // Added campaign_id to INSERT
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to marshal marketplace listing metadata: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_marketplace_listing (listing_id, master_id, master_uuid, seller_id, product_id, price, currency, status, metadata, created_at, campaign_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), $10)`, l.ListingID, l.MasterID, l.MasterUUID, l.SellerID, l.ProductID, l.Price, l.Currency, l.Status, metaJSON, l.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to insert into service_commerce_marketplace_listing: %w", err)
	}
	return tx.Commit()
}

func (r *repository) CreateMarketplaceOrder(ctx context.Context, o *MarketplaceOrder) error {
	// Defensive: ensure metadata is always valid JSON
	if o.Metadata == nil {
		o.Metadata = &commonpb.Metadata{}
	}
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
			panic(r) // Re-throw panic
		}
	}()
	o.MasterID, o.MasterUUID, err = r.masterRepo.Create(ctx, tx, repo.EntityType("marketplace_order"), o.OrderID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for marketplace order: %w", err)
	}
	metaJSON, err := metadata.MarshalCanonical(o.Metadata) // Added campaign_id to INSERT
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to marshal marketplace order metadata: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_marketplace_order (order_id, master_id, master_uuid, listing_id, buyer_id, price, currency, status, metadata, created_at, campaign_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), $10)`, o.OrderID, o.MasterID, o.MasterUUID, o.ListingID, o.BuyerID, o.Price, o.Currency, o.Status, metaJSON, o.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to insert into service_commerce_marketplace_order: %w", err)
	}
	return tx.Commit()
}

func (r *repository) CreateMarketplaceOffer(ctx context.Context, o *MarketplaceOffer) error {
	// Defensive: ensure metadata is always valid JSON
	if o.Metadata == nil {
		o.Metadata = &commonpb.Metadata{}
	}
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
			panic(r) // Re-throw panic
		}
	}()
	o.MasterID, o.MasterUUID, err = r.masterRepo.Create(ctx, tx, repo.EntityType("marketplace_offer"), o.OfferID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for marketplace offer: %w", err)
	}
	metaJSON, err := metadata.MarshalCanonical(o.Metadata) // Added campaign_id to INSERT
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to marshal marketplace offer metadata: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_marketplace_offer (offer_id, master_id, master_uuid, listing_id, buyer_id, offer_price, currency, status, metadata, created_at, campaign_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), $10)`, o.OfferID, o.MasterID, o.MasterUUID, o.ListingID, o.BuyerID, o.OfferPrice, o.Currency, o.Status, metaJSON, o.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to insert into service_commerce_marketplace_offer: %w", err)
	}
	return tx.Commit()
}

// --- Exchange ---.
func (r *repository) CreateExchangeOrder(ctx context.Context, o *ExchangeOrder) error {
	// Defensive: ensure metadata is always valid JSON
	if o.Metadata == nil {
		o.Metadata = &commonpb.Metadata{}
	}
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
			panic(r) // Re-throw panic
		}
	}()
	o.MasterID, o.MasterUUID, err = r.masterRepo.Create(ctx, tx, repo.EntityType("exchange_order"), o.OrderID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for exchange order: %w", err)
	}
	metaJSON, err := metadata.MarshalCanonical(o.Metadata) // Added campaign_id to INSERT
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to marshal exchange order metadata: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_exchange_order (order_id, master_id, master_uuid, account_id, pair, amount, price, order_type, status, metadata, created_at, campaign_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), $11)`, o.OrderID, o.MasterID, o.MasterUUID, o.AccountID, o.Pair, o.Amount, o.Price, o.OrderType, o.Status, metaJSON, o.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to insert into service_commerce_exchange_order: %w", err)
	}
	return tx.Commit()
}

func (r *repository) CreateExchangePair(ctx context.Context, p *ExchangePair) error {
	// Defensive: ensure metadata is always valid JSON
	if p.Metadata == nil {
		p.Metadata = &commonpb.Metadata{}
	}
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
			panic(r) // Re-throw panic
		}
	}()
	p.MasterID, p.MasterUUID, err = r.masterRepo.Create(ctx, tx, repo.EntityType("exchange_pair"), p.PairID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for exchange pair: %w", err)
	}
	metaJSON, err := metadata.MarshalCanonical(p.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to marshal exchange pair metadata: %w", err)
	}

	// Insert into service_commerce_exchange_pair
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_exchange_pair (pair_id, master_id, master_uuid, base_asset, quote_asset, metadata, campaign_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`, p.PairID, p.MasterID, p.MasterUUID, p.BaseAsset, p.QuoteAsset, metaJSON, p.CampaignID) // Added campaign_id
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to insert into service_commerce_exchange_pair: %w", err)
	}
	return tx.Commit()
}

func (r *repository) CreateExchangeRate(ctx context.Context, r8 *ExchangeRate) error {
	if r8.Metadata == nil {
		r8.Metadata = &commonpb.Metadata{}
	}
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
			panic(r) // Re-throw panic
		}
	}()
	r8.MasterID, r8.MasterUUID, err = r.masterRepo.Create(ctx, tx, repo.EntityType("exchange_rate"), fmt.Sprintf("%d", r8.RateID)) // Assuming RateID is unique enough for name
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to create master entity for exchange rate: %w", err)
	}
	metaJSON, err := metadata.MarshalCanonical(r8.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to marshal exchange rate metadata: %w", err)
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_exchange_rate (rate_id, master_id, master_uuid, pair_id, rate, timestamp, metadata, campaign_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, r8.RateID, r8.MasterID, r8.MasterUUID, r8.PairID, r8.Rate, r8.Timestamp, metaJSON, r8.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("failed to rollback transaction: %v", rbErr)
		}
		return fmt.Errorf("failed to insert into service_commerce_exchange_rate: %w", err)
	}
	return tx.Commit()
}

// --- Investment/Account/Asset Reads ---.
func (r *repository) GetInvestmentAccount(ctx context.Context, accountID string) (*InvestmentAccount, error) {
	query := `
		SELECT account_id, master_id, master_uuid, owner_id, type, currency, balance, metadata, created_at, updated_at, campaign_id
		FROM service_commerce_investment_account
		WHERE account_id = $1
	`
	var account InvestmentAccount
	err := r.GetDB().QueryRowContext(ctx, query, accountID).Scan(
		&account.AccountID,
		&account.MasterID, // Corrected scan order
		&account.MasterUUID,
		&account.OwnerID,
		&account.Type,
		&account.Currency,
		&account.Balance,
		&account.Metadata,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.CampaignID,
	)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to query investment account", err)
	}
	return &account, nil
}

func (r *repository) ListInvestmentAccounts(ctx context.Context, ownerID string) ([]*InvestmentAccount, error) {
	query := `
		SELECT account_id, master_id, master_uuid, owner_id, type, currency, balance, metadata, created_at, updated_at, campaign_id
		FROM service_commerce_investment_account
		WHERE owner_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.GetDB().QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to query investment accounts", err)
	}
	defer rows.Close()

	var accounts []*InvestmentAccount
	for rows.Next() {
		var account InvestmentAccount
		if err := rows.Scan(
			&account.AccountID, // Corrected scan order
			&account.MasterID,
			&account.MasterUUID,
			&account.OwnerID,
			&account.Type,
			&account.Currency,
			&account.Balance,
			&account.Metadata,
			&account.CreatedAt,
			&account.UpdatedAt,
			&account.CampaignID,
		); err != nil {
			return nil, graceful.WrapErr(ctx, codes.Internal, "error scanning investment account", err)
		}
		accounts = append(accounts, &account)
	}
	if err := rows.Err(); err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "error iterating investment accounts", err)
	}
	return accounts, nil
}

func (r *repository) GetPortfolio(ctx context.Context, portfolioID string) (*Portfolio, error) {
	query := `
		SELECT portfolio_id, master_id, master_uuid, account_id, metadata, created_at, updated_at, campaign_id
		FROM service_commerce_portfolio
		WHERE portfolio_id = $1
	`
	var portfolio Portfolio
	err := r.GetDB().QueryRowContext(ctx, query, portfolioID).Scan(
		&portfolio.PortfolioID,
		&portfolio.MasterID, // Corrected scan order
		&portfolio.MasterUUID,
		&portfolio.AccountID,
		&portfolio.Metadata,
		&portfolio.CreatedAt,
		&portfolio.UpdatedAt,
		&portfolio.CampaignID,
	)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to query portfolio", err)
	}
	return &portfolio, nil
}

func (r *repository) ListPortfolios(ctx context.Context, accountID string, campaignID int64) ([]*Portfolio, error) { // Updated signature
	query := `
		SELECT portfolio_id, master_id, master_uuid, account_id, metadata, created_at, updated_at, campaign_id
		FROM service_commerce_portfolio
		WHERE account_id = $1 AND campaign_id = $2
		ORDER BY created_at DESC
	`
	rows, err := r.GetDB().QueryContext(ctx, query, accountID, campaignID) // Pass campaignID to query
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to query portfolios", err)
	}
	defer rows.Close()

	var portfolios []*Portfolio
	for rows.Next() {
		var portfolio Portfolio
		if err := rows.Scan(
			&portfolio.PortfolioID, // Corrected scan order
			&portfolio.MasterID,
			&portfolio.MasterUUID,
			&portfolio.AccountID,
			&portfolio.Metadata,
			&portfolio.CreatedAt,
			&portfolio.UpdatedAt,
			&portfolio.CampaignID,
		); err != nil {
			return nil, graceful.WrapErr(ctx, codes.Internal, "error scanning portfolio", err)
		}
		portfolios = append(portfolios, &portfolio)
	}
	if err := rows.Err(); err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "error iterating portfolios", err)
	}
	return portfolios, nil
}

func (r *repository) GetAssetPosition(ctx context.Context, id int64) (*AssetPosition, error) {
	query := `
		SELECT id, master_id, master_uuid, portfolio_id, asset_id, quantity, average_price, metadata, campaign_id
		FROM service_commerce_asset_position
		WHERE id = $1
	`
	var position AssetPosition
	err := r.GetDB().QueryRowContext(ctx, query, id).Scan(
		&position.ID,
		&position.MasterID, // Corrected scan order
		&position.MasterUUID,
		&position.PortfolioID,
		&position.AssetID,
		&position.Quantity,
		&position.AveragePrice,
		&position.Metadata,
		&position.CampaignID,
	)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to query asset position", err)
	}
	return &position, nil
}

func (r *repository) ListAssetPositions(ctx context.Context, portfolioID, campaignID int64) ([]*AssetPosition, error) { // Updated signature
	query := `
		SELECT id, master_id, master_uuid, portfolio_id, asset_id, quantity, average_price, metadata, campaign_id
		FROM service_commerce_asset_position
		WHERE portfolio_id = $1 AND campaign_id = $2
		ORDER BY id
	`
	rows, err := r.GetDB().QueryContext(ctx, query, portfolioID, campaignID) // Pass campaignID to query
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to query asset positions", err)
	}
	defer rows.Close()

	var positions []*AssetPosition
	for rows.Next() {
		var position AssetPosition
		if err := rows.Scan(
			&position.ID, // Corrected scan order
			&position.MasterID,
			&position.MasterUUID,
			&position.PortfolioID,
			&position.AssetID,
			&position.Quantity,
			&position.AveragePrice,
			&position.Metadata,
			&position.CampaignID,
		); err != nil {
			return nil, graceful.WrapErr(ctx, codes.Internal, "error scanning asset position", err)
		}
		positions = append(positions, &position)
	}
	if err := rows.Err(); err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "error iterating asset positions", err)
	}
	return positions, nil
}

func (r *repository) CreateProduct(ctx context.Context, p *productpb.Product) (*productpb.Product, error) {
	// NOTE: This function appears to belong to a 'product' service's repository, not 'commerce'.
	query := `
		INSERT INTO service_product_main ( // Renamed table from 'products'
			id, master_id, master_uuid, name, description, type, status, tags, metadata, created_at, updated_at, main_image_url, gallery_image_urls, owner_id, campaign_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		RETURNING id, master_id, master_uuid, name, description, type, status, tags, metadata, created_at, updated_at, main_image_url, gallery_image_urls, owner_id, campaign_id
	`
	var product productpb.Product
	err := r.GetDB().QueryRowContext(ctx, query,
		p.Id,
		p.MasterId,
		p.MasterUuid,
		p.Name,
		p.Description,
		p.Type,
		p.Status,
		p.Tags,
		p.Metadata,
		p.CreatedAt,
		p.UpdatedAt,
		p.MainImageUrl,
		p.GalleryImageUrls,
		p.OwnerId,
		p.CampaignId,
	).Scan(
		&product.Id,
		&product.MasterId,
		&product.MasterUuid,
		&product.Name,
		&product.Description,
		&product.Type,
		&product.Status,
		&product.Tags,
		&product.Metadata,
		&product.CreatedAt,
		&product.UpdatedAt,
		&product.MainImageUrl,
		&product.GalleryImageUrls,
		&product.OwnerId,
		&product.CampaignId,
	)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to create product", err)
	}
	return &product, nil
}
