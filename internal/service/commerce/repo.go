package commerce

import (
	"context"
	"database/sql"
	"encoding/json"
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
	ID        int                `db:"id"`
	OrderID   string             `db:"order_id"`
	ProductID string             `db:"product_id"`
	Quantity  int                `db:"quantity"`
	Price     float64            `db:"price"`
	Metadata  *commonpb.Metadata `db:"metadata"`
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
	MasterID   int64              `db:"master_id"`
	MasterUUID string             `db:"master_uuid"`
}

type Event struct {
	EventID    int64                  `db:"event_id"`
	EntityID   int64                  `db:"master_id"`
	EntityType string                 `db:"entity_type"`
	EventType  string                 `db:"event_type"`
	Payload    map[string]interface{} `db:"payload"`
	Metadata   *commonpb.Metadata     `db:"metadata"`
	CreatedAt  time.Time              `db:"occurred_at"`
	MasterID   int64                  `db:"master_id"`
	MasterUUID string                 `db:"master_uuid"`
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
}

type Portfolio struct {
	PortfolioID string             `db:"portfolio_id"`
	AccountID   string             `db:"account_id"`
	Metadata    *commonpb.Metadata `db:"metadata"`
	CreatedAt   time.Time          `db:"created_at"`
	UpdatedAt   time.Time          `db:"updated_at"`
	MasterID    int64              `db:"master_id"`
	MasterUUID  string             `db:"master_uuid"`
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
	MasterUUID    string             `db:"master_uuid"`
}

type BankStatement struct {
	StatementID int64              `db:"statement_id"`
	MasterID    int64              `db:"master_id"`
	AccountID   string             `db:"account_id"`
	Metadata    *commonpb.Metadata `db:"metadata"`
	CreatedAt   time.Time          `db:"created_at"`
	MasterUUID  string             `db:"master_uuid"`
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
	MasterUUID string             `db:"master_uuid"`
}

type ExchangeRate struct {
	RateID     int64              `db:"rate_id"`
	MasterID   int64              `db:"master_id"`
	PairID     string             `db:"pair_id"`
	Rate       float64            `db:"rate"`
	Timestamp  time.Time          `db:"timestamp"`
	Metadata   *commonpb.Metadata `db:"metadata"`
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
	ListBalances(ctx context.Context, userID string) ([]*Balance, error)
	UpdateBalance(ctx context.Context, userID, currency string, amount float64) error
	LogEvent(ctx context.Context, e *Event) error
	ListEvents(ctx context.Context, entityID, entityType string, page, pageSize int) ([]*Event, int, error)
	CreateInvestmentAccount(ctx context.Context, a *InvestmentAccount) error
	CreateInvestmentOrder(ctx context.Context, o *InvestmentOrder) error
	CreatePortfolio(ctx context.Context, p *Portfolio) error
	CreateAssetPosition(ctx context.Context, pos *AssetPosition) error
	CreateBankAccount(ctx context.Context, a *BankAccount) error
	CreateBankTransfer(ctx context.Context, t *BankTransfer) error
	CreateBankStatement(ctx context.Context, s *BankStatement) error
	CreateMarketplaceListing(ctx context.Context, l *MarketplaceListing) error
	CreateMarketplaceOrder(ctx context.Context, o *MarketplaceOrder) error
	CreateMarketplaceOffer(ctx context.Context, o *MarketplaceOffer) error
	CreateExchangeOrder(ctx context.Context, o *ExchangeOrder) error
	CreateExchangePair(ctx context.Context, p *ExchangePair) error
	CreateExchangeRate(ctx context.Context, r8 *ExchangeRate) error
	GetInvestmentAccount(ctx context.Context, accountID string) (*InvestmentAccount, error)
	ListInvestmentAccounts(ctx context.Context, ownerID string) ([]*InvestmentAccount, error)
	GetPortfolio(ctx context.Context, portfolioID string) (*Portfolio, error)
	ListPortfolios(ctx context.Context, accountID string) ([]*Portfolio, error)
	GetAssetPosition(ctx context.Context, id int64) (*AssetPosition, error)
	ListAssetPositions(ctx context.Context, portfolioID int64) ([]*AssetPosition, error)
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
	query := `
		INSERT INTO service_commerce_quote (
			quote_id, user_id, product_id, amount, currency, status, metadata, created_at, updated_at, campaign_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, now(), now(), $8
		)
	`
	_, err = r.GetDB().ExecContext(ctx, query,
		q.QuoteID,
		q.UserID,
		q.ProductID,
		q.Amount,
		q.Currency,
		q.Status,
		metaJSON,
		q.CampaignID,
	)
	return err
}

func (r *repository) GetQuote(ctx context.Context, quoteID string) (*Quote, error) {
	query := `
		SELECT quote_id, user_id, product_id, amount, currency, status, metadata, created_at, updated_at, master_id, master_uuid, campaign_id
		FROM quotes
		WHERE quote_id = $1
	`
	var quote Quote
	err := r.GetDB().QueryRowContext(ctx, query, quoteID).Scan(
		&quote.QuoteID,
		&quote.UserID,
		&quote.ProductID,
		&quote.Amount,
		&quote.Currency,
		&quote.Status,
		&quote.Metadata,
		&quote.CreatedAt,
		&quote.UpdatedAt,
		&quote.MasterID,
		&quote.MasterUUID,
		&quote.CampaignID,
	)
	if err != nil {
		return nil, err
	}
	return &quote, nil
}

func (r *repository) ListQuotes(ctx context.Context, userID string, campaignID int64, page, pageSize int) ([]*Quote, int, error) {
	query := `
		SELECT quote_id, user_id, product_id, amount, currency, status, metadata, created_at, updated_at, master_id, master_uuid, campaign_id
		FROM quotes
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
			&quote.UserID,
			&quote.ProductID,
			&quote.Amount,
			&quote.Currency,
			&quote.Status,
			&quote.Metadata,
			&quote.CreatedAt,
			&quote.UpdatedAt,
			&quote.MasterID,
			&quote.MasterUUID,
			&quote.CampaignID,
		); err != nil {
			return nil, 0, graceful.WrapErr(ctx, codes.Internal, "error scanning quote", err)
		}
		quotes = append(quotes, &quote)
	}

	countQuery := `
		SELECT COUNT(*)
		FROM quotes
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
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback failed: %v", rbErr)
			}
			log.Printf("panic occurred: %v", p)
			err = fmt.Errorf("panic occurred: %v", p)
			return
		}
	}()

	// 1. Insert into master_entity
	var masterID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO master_entity (type, created_at, updated_at)
		VALUES ($1, now(), now()) RETURNING id
	`, "order").Scan(&masterID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}

	// Defensive: ensure metadata is always valid JSON
	if o.Metadata == nil {
		o.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}

	// 2. Marshal metadata
	metaJSON, err := json.Marshal(o.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}

	// 3. Insert into service_commerce_order
	_, err = tx.ExecContext(ctx, `
		INSERT INTO service_commerce_order (order_id, master_id, user_id, total, currency, status, metadata, created_at, updated_at, campaign_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now(), $8)
	`, o.OrderID, masterID, o.UserID, o.Total, o.Currency, o.Status, metaJSON, o.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}

	// 4. Insert order items
	for _, item := range items {
		itemMetaJSON, err := json.Marshal(item.Metadata)
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback failed: %v", rbErr)
			}
			return err
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO service_commerce_order_item (order_id, product_id, quantity, price, metadata, created_at)
			VALUES ($1, $2, $3, $4, $5, now())
		`, o.OrderID, item.ProductID, item.Quantity, item.Price, itemMetaJSON)
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback failed: %v", rbErr)
			}
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *repository) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	var o Order
	var metaJSON []byte
	var masterID int64
	err := r.GetDB().QueryRowContext(ctx, `
		SELECT order_id, master_id, user_id, total, currency, status, metadata, created_at, updated_at, campaign_id
		FROM service_commerce_order
		WHERE order_id = $1
	`, orderID).Scan(&o.OrderID, &masterID, &o.UserID, &o.Total, &o.Currency, &o.Status, &metaJSON, &o.CreatedAt, &o.UpdatedAt, &o.CampaignID)
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
	rows, err := r.GetDB().QueryContext(ctx, `SELECT order_id, master_id, user_id, total, currency, status, metadata, created_at, updated_at, campaign_id FROM service_commerce_order WHERE user_id = $1 AND campaign_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`, userID, campaignID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "failed to query orders", err)
	}
	defer rows.Close()
	var orders []*Order
	for rows.Next() {
		var o Order
		var metaJSON []byte
		var masterID int64
		if err := rows.Scan(&o.OrderID, &masterID, &o.UserID, &o.Total, &o.Currency, &o.Status, &metaJSON, &o.CreatedAt, &o.UpdatedAt, &o.CampaignID); err != nil {
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
	rows, err := r.GetDB().QueryContext(ctx, `SELECT id, order_id, product_id, quantity, price, metadata FROM service_commerce_order_item WHERE order_id = $1`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*OrderItem
	for rows.Next() {
		var item OrderItem
		var metaJSON []byte
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.Price, &metaJSON); err != nil {
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
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback failed: %v", rbErr)
			}
			log.Printf("panic occurred: %v", p)
			err = fmt.Errorf("panic occurred: %v", p)
			return
		}
	}()

	// 1. Insert into master_entity
	var masterID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO master_entity (type, created_at, updated_at)
		VALUES ($1, now(), now()) RETURNING id
	`, "payment").Scan(&masterID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}

	// Defensive: ensure metadata is always valid JSON
	if p.Metadata == nil {
		p.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}

	// 2. Marshal metadata
	metaJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}

	// 3. Insert into service_commerce_payment
	_, err = tx.ExecContext(ctx, `
		INSERT INTO service_commerce_payment (payment_id, master_id, order_id, user_id, amount, currency, method, status, metadata, created_at, updated_at, campaign_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now(), $10)
	`, p.PaymentID, masterID, p.OrderID, p.UserID, p.Amount, p.Currency, p.Method, p.Status, metaJSON, p.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *repository) GetPayment(ctx context.Context, paymentID string) (*Payment, error) {
	var p Payment
	var metaJSON []byte
	var masterID int64
	err := r.GetDB().QueryRowContext(ctx, `SELECT payment_id, master_id, order_id, user_id, amount, currency, method, status, metadata, created_at, updated_at, campaign_id FROM service_commerce_payment WHERE payment_id = $1`, paymentID).Scan(&p.PaymentID, &masterID, &p.OrderID, &p.UserID, &p.Amount, &p.Currency, &p.Method, &p.Status, &metaJSON, &p.CreatedAt, &p.UpdatedAt, &p.CampaignID)
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
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback failed: %v", rbErr)
			}
			log.Printf("panic occurred: %v", p)
			err = fmt.Errorf("panic occurred: %v", p)
			return
		}
	}()

	// 1. Insert into master_entity
	var masterID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO master_entity (type, created_at, updated_at)
		VALUES ($1, now(), now()) RETURNING id
	`, "transaction").Scan(&masterID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}

	// Defensive: ensure metadata is always valid JSON
	if t.Metadata == nil {
		t.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}

	// 2. Marshal metadata
	metaJSON, err := json.Marshal(t.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}

	// 3. Insert into service_commerce_transaction
	_, err = tx.ExecContext(ctx, `
		INSERT INTO service_commerce_transaction (transaction_id, master_id, payment_id, user_id, type, amount, currency, status, metadata, created_at, updated_at, campaign_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now(), $10)
	`, t.TransactionID, masterID, t.PaymentID, t.UserID, t.Type, t.Amount, t.Currency, t.Status, metaJSON, t.CampaignID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *repository) GetTransaction(ctx context.Context, transactionID string) (*Transaction, error) {
	var t Transaction
	var metaJSON []byte
	var masterID int64
	err := r.GetDB().QueryRowContext(ctx, `SELECT transaction_id, master_id, payment_id, user_id, type, amount, currency, status, metadata, created_at, updated_at, campaign_id FROM service_commerce_transaction WHERE transaction_id = $1`, transactionID).Scan(&t.TransactionID, &masterID, &t.PaymentID, &t.UserID, &t.Type, &t.Amount, &t.Currency, &t.Status, &metaJSON, &t.CreatedAt, &t.UpdatedAt, &t.CampaignID)
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
	rows, err := r.GetDB().QueryContext(ctx, `SELECT transaction_id, master_id, payment_id, user_id, type, amount, currency, status, metadata, created_at, updated_at, campaign_id FROM service_commerce_transaction WHERE user_id = $1 AND campaign_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`, userID, campaignID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "failed to query transactions", err)
	}
	defer rows.Close()
	var txs []*Transaction
	for rows.Next() {
		var t Transaction
		var metaJSON []byte
		var masterID int64
		if err := rows.Scan(&t.TransactionID, &masterID, &t.PaymentID, &t.UserID, &t.Type, &t.Amount, &t.Currency, &t.Status, &metaJSON, &t.CreatedAt, &t.UpdatedAt, &t.CampaignID); err != nil {
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
	err := r.GetDB().QueryRowContext(ctx, `SELECT user_id, currency, amount, metadata, updated_at FROM service_commerce_balance WHERE user_id = $1 AND currency = $2`, userID, currency).Scan(&b.UserID, &b.Currency, &b.Amount, &b.Metadata, &b.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &b, nil
}

func (r *repository) ListBalances(ctx context.Context, userID string) ([]*Balance, error) {
	rows, err := r.GetDB().QueryContext(ctx, `SELECT user_id, currency, amount, metadata, updated_at FROM service_commerce_balance WHERE user_id = $1`, userID)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to query balances", err)
	}
	defer rows.Close()
	var balances []*Balance
	for rows.Next() {
		var b Balance
		if err := rows.Scan(&b.UserID, &b.Currency, &b.Amount, &b.Metadata, &b.UpdatedAt); err != nil {
			return nil, graceful.WrapErr(ctx, codes.Internal, "error scanning balance", err)
		}
		balances = append(balances, &b)
	}
	if err := rows.Err(); err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "error iterating balances", err)
	}
	return balances, nil
}

func (r *repository) UpdateBalance(ctx context.Context, userID, currency string, amount float64) error {
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE service_commerce_balance SET amount = $1, updated_at = now() WHERE user_id = $2 AND currency = $3`, amount, userID, currency)
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

// Events.
func (r *repository) LogEvent(ctx context.Context, e *Event) error {
	// Defensive: ensure metadata is always valid JSON
	if e.Metadata == nil {
		e.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}

	metaJSON, err := json.Marshal(e.Payload)
	if err != nil {
		return err
	}
	_, err = r.GetDB().ExecContext(ctx, `INSERT INTO service_commerce_event (master_id, entity_type, event_type, payload, metadata, occurred_at) VALUES ($1, $2, $3, $4, $5, now())`, e.EntityID, e.EntityType, e.EventType, metaJSON, e.Metadata)
	return err
}

func (r *repository) ListEvents(ctx context.Context, entityID, entityType string, page, pageSize int) ([]*Event, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var total int
	err := r.GetDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM service_commerce_event WHERE master_id = $1 AND entity_type = $2`, entityID, entityType).Scan(&total)
	if err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "failed to count events", err)
	}
	rows, err := r.GetDB().QueryContext(ctx, `SELECT event_id, master_id, entity_type, event_type, payload, metadata, occurred_at FROM service_commerce_event WHERE master_id = $1 AND entity_type = $2 ORDER BY occurred_at DESC LIMIT $3 OFFSET $4`, entityID, entityType, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, graceful.WrapErr(ctx, codes.Internal, "failed to query events", err)
	}
	defer rows.Close()
	var events []*Event
	for rows.Next() {
		var e Event
		var payloadJSON []byte
		var metaJSON []byte
		if err := rows.Scan(&e.EventID, &e.EntityID, &e.EntityType, &e.EventType, &payloadJSON, &metaJSON, &e.CreatedAt); err != nil {
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
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback failed: %v", rbErr)
			}
			log.Printf("panic occurred: %v", p)
			err = fmt.Errorf("panic occurred: %v", p)
			return
		}
	}()
	var masterID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO master_entity (type, created_at, updated_at) VALUES ($1, now(), now()) RETURNING id`, "investment_account").Scan(&masterID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	metaJSON, err := json.Marshal(a.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_investment_account (account_id, master_id, owner_id, type, currency, balance, metadata, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now())`, a.AccountID, masterID, a.OwnerID, a.Type, a.Currency, a.Balance, metaJSON)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
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
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback failed: %v", rbErr)
			}
			log.Printf("panic occurred: %v", p)
			err = fmt.Errorf("panic occurred: %v", p)
			return
		}
	}()
	var masterID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO master_entity (type, created_at, updated_at) VALUES ($1, now(), now()) RETURNING id`, "investment_order").Scan(&masterID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	metaJSON, err := json.Marshal(o.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_investment_order (order_id, master_id, account_id, asset_id, quantity, price, order_type, status, metadata, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now())`, o.OrderID, masterID, o.AccountID, o.AssetID, o.Quantity, o.Price, o.OrderType, o.Status, metaJSON)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
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
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback failed: %v", rbErr)
			}
			log.Printf("panic occurred: %v", p)
			err = fmt.Errorf("panic occurred: %v", p)
			return
		}
	}()
	var masterID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO master_entity (type, created_at, updated_at) VALUES ($1, now(), now()) RETURNING id`, "portfolio").Scan(&masterID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	metaJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_portfolio (master_id, account_id, metadata, created_at, updated_at) VALUES ($1, $2, $3, now(), now())`, masterID, p.AccountID, metaJSON)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	return tx.Commit()
}

func (r *repository) CreateAssetPosition(ctx context.Context, pos *AssetPosition) error {
	// Defensive: ensure metadata is always valid JSON
	if pos.Metadata == nil {
		pos.Metadata = &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(nil, nil)}
	}
	metaJSON, err := json.Marshal(pos.Metadata)
	if err != nil {
		return err
	}
	_, err = r.GetDB().ExecContext(ctx, `INSERT INTO service_commerce_asset_position (portfolio_id, asset_id, quantity, average_price, metadata) VALUES ($1, $2, $3, $4, $5)`, pos.PortfolioID, pos.AssetID, pos.Quantity, pos.AveragePrice, metaJSON)
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
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback failed: %v", rbErr)
			}
			log.Printf("panic occurred: %v", p)
			err = fmt.Errorf("panic occurred: %v", p)
			return
		}
	}()
	var masterID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO master_entity (type, created_at, updated_at) VALUES ($1, now(), now()) RETURNING id`, "bank_account").Scan(&masterID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	metaJSON, err := json.Marshal(a.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_bank_account (account_id, master_id, user_id, iban, bic, currency, balance, metadata, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), now())`, a.AccountID, masterID, a.UserID, a.IBAN, a.BIC, a.Currency, a.Balance, metaJSON)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
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
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback failed: %v", rbErr)
			}
			log.Printf("panic occurred: %v", p)
			err = fmt.Errorf("panic occurred: %v", p)
			return
		}
	}()
	var masterID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO master_entity (type, created_at, updated_at) VALUES ($1, now(), now()) RETURNING id`, "bank_transfer").Scan(&masterID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	metaJSON, err := json.Marshal(t.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_bank_transfer (transfer_id, master_id, from_account_id, to_account_id, amount, currency, status, metadata, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())`, t.TransferID, masterID, t.FromAccountID, t.ToAccountID, t.Amount, t.Currency, t.Status, metaJSON)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
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
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback failed: %v", rbErr)
			}
			log.Printf("panic occurred: %v", p)
			err = fmt.Errorf("panic occurred: %v", p)
			return
		}
	}()
	var masterID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO master_entity (type, created_at, updated_at) VALUES ($1, now(), now()) RETURNING id`, "bank_statement").Scan(&masterID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	metaJSON, err := json.Marshal(s.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_bank_statement (master_id, account_id, metadata, created_at) VALUES ($1, $2, $3, now())`, masterID, s.AccountID, metaJSON)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
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
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback failed: %v", rbErr)
			}
			log.Printf("panic occurred: %v", p)
			err = fmt.Errorf("panic occurred: %v", p)
			return
		}
	}()
	var masterID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO master_entity (type, created_at) VALUES ($1, now()) RETURNING id`, "marketplace_listing").Scan(&masterID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	metaJSON, err := json.Marshal(l.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_marketplace_listing (listing_id, master_id, seller_id, product_id, price, currency, status, metadata, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())`, l.ListingID, masterID, l.SellerID, l.ProductID, l.Price, l.Currency, l.Status, metaJSON)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
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
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback failed: %v", rbErr)
			}
			log.Printf("panic occurred: %v", p)
			err = fmt.Errorf("panic occurred: %v", p)
			return
		}
	}()
	var masterID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO master_entity (type, created_at) VALUES ($1, now()) RETURNING id`, "marketplace_order").Scan(&masterID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	metaJSON, err := json.Marshal(o.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_marketplace_order (order_id, master_id, listing_id, buyer_id, price, currency, status, metadata, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())`, o.OrderID, masterID, o.ListingID, o.BuyerID, o.Price, o.Currency, o.Status, metaJSON)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
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
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback failed: %v", rbErr)
			}
			log.Printf("panic occurred: %v", p)
			err = fmt.Errorf("panic occurred: %v", p)
			return
		}
	}()
	var masterID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO master_entity (type, created_at) VALUES ($1, now()) RETURNING id`, "marketplace_offer").Scan(&masterID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	metaJSON, err := json.Marshal(o.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_marketplace_offer (offer_id, master_id, listing_id, buyer_id, offer_price, currency, status, metadata, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())`, o.OfferID, masterID, o.ListingID, o.BuyerID, o.OfferPrice, o.Currency, o.Status, metaJSON)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
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
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback failed: %v", rbErr)
			}
			log.Printf("panic occurred: %v", p)
			err = fmt.Errorf("panic occurred: %v", p)
			return
		}
	}()
	var masterID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO master_entity (type, created_at) VALUES ($1, now()) RETURNING id`, "exchange_order").Scan(&masterID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	metaJSON, err := json.Marshal(o.Metadata)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_commerce_exchange_order (order_id, master_id, account_id, pair, amount, price, order_type, status, metadata, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())`, o.OrderID, masterID, o.AccountID, o.Pair, o.Amount, o.Price, o.OrderType, o.Status, metaJSON)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("rollback failed: %v", rbErr)
		}
		return err
	}
	return tx.Commit()
}

func (r *repository) CreateExchangePair(ctx context.Context, p *ExchangePair) error {
	// Defensive: ensure metadata is always valid JSON
	if p.Metadata == nil {
		p.Metadata = &commonpb.Metadata{}
	}
	// Marshal metadata to JSON
	metaJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return err
	}
	// Insert into service_commerce_exchange_pair
	_, err = r.GetDB().ExecContext(ctx, `
		INSERT INTO service_commerce_exchange_pair (pair_id, master_id, base_asset, quote_asset, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`, p.PairID, p.MasterID, p.BaseAsset, p.QuoteAsset, metaJSON)
	return err
}

func (r *repository) CreateExchangeRate(ctx context.Context, r8 *ExchangeRate) error {
	// Defensive: ensure metadata is always valid JSON
	if r8.Metadata == nil {
		r8.Metadata = &commonpb.Metadata{}
	}
	// Marshal metadata to JSON
	metaJSON, err := json.Marshal(r8.Metadata)
	if err != nil {
		return err
	}
	// Insert into service_commerce_exchange_rate
	_, err = r.GetDB().ExecContext(ctx, `
		INSERT INTO service_commerce_exchange_rate (rate_id, master_id, pair_id, rate, timestamp, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, r8.RateID, r8.MasterID, r8.PairID, r8.Rate, r8.Timestamp, metaJSON)
	return err
}

// --- Investment/Account/Asset Reads ---.
func (r *repository) GetInvestmentAccount(ctx context.Context, accountID string) (*InvestmentAccount, error) {
	query := `
		SELECT account_id, master_id, owner_id, type, currency, balance, metadata, created_at, updated_at, master_uuid
		FROM investment_accounts
		WHERE account_id = $1
	`
	var account InvestmentAccount
	err := r.GetDB().QueryRowContext(ctx, query, accountID).Scan(
		&account.AccountID,
		&account.MasterID,
		&account.OwnerID,
		&account.Type,
		&account.Currency,
		&account.Balance,
		&account.Metadata,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.MasterUUID,
	)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to query investment account", err)
	}
	return &account, nil
}

func (r *repository) ListInvestmentAccounts(ctx context.Context, ownerID string) ([]*InvestmentAccount, error) {
	query := `
		SELECT account_id, master_id, owner_id, type, currency, balance, metadata, created_at, updated_at, master_uuid
		FROM investment_accounts
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
			&account.AccountID,
			&account.MasterID,
			&account.OwnerID,
			&account.Type,
			&account.Currency,
			&account.Balance,
			&account.Metadata,
			&account.CreatedAt,
			&account.UpdatedAt,
			&account.MasterUUID,
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
		SELECT portfolio_id, account_id, metadata, created_at, updated_at, master_id, master_uuid
		FROM portfolios
		WHERE portfolio_id = $1
	`
	var portfolio Portfolio
	err := r.GetDB().QueryRowContext(ctx, query, portfolioID).Scan(
		&portfolio.PortfolioID,
		&portfolio.AccountID,
		&portfolio.Metadata,
		&portfolio.CreatedAt,
		&portfolio.UpdatedAt,
		&portfolio.MasterID,
		&portfolio.MasterUUID,
	)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to query portfolio", err)
	}
	return &portfolio, nil
}

func (r *repository) ListPortfolios(ctx context.Context, accountID string) ([]*Portfolio, error) {
	query := `
		SELECT portfolio_id, account_id, metadata, created_at, updated_at, master_id, master_uuid
		FROM portfolios
		WHERE account_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.GetDB().QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to query portfolios", err)
	}
	defer rows.Close()

	var portfolios []*Portfolio
	for rows.Next() {
		var portfolio Portfolio
		if err := rows.Scan(
			&portfolio.PortfolioID,
			&portfolio.AccountID,
			&portfolio.Metadata,
			&portfolio.CreatedAt,
			&portfolio.UpdatedAt,
			&portfolio.MasterID,
			&portfolio.MasterUUID,
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
		SELECT id, portfolio_id, asset_id, quantity, average_price, metadata, master_id, master_uuid
		FROM asset_positions
		WHERE id = $1
	`
	var position AssetPosition
	err := r.GetDB().QueryRowContext(ctx, query, id).Scan(
		&position.ID,
		&position.PortfolioID,
		&position.AssetID,
		&position.Quantity,
		&position.AveragePrice,
		&position.Metadata,
		&position.MasterID,
		&position.MasterUUID,
	)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to query asset position", err)
	}
	return &position, nil
}

func (r *repository) ListAssetPositions(ctx context.Context, portfolioID int64) ([]*AssetPosition, error) {
	query := `
		SELECT id, portfolio_id, asset_id, quantity, average_price, metadata, master_id, master_uuid
		FROM asset_positions
		WHERE portfolio_id = $1
		ORDER BY id
	`
	rows, err := r.GetDB().QueryContext(ctx, query, portfolioID)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to query asset positions", err)
	}
	defer rows.Close()

	var positions []*AssetPosition
	for rows.Next() {
		var position AssetPosition
		if err := rows.Scan(
			&position.ID,
			&position.PortfolioID,
			&position.AssetID,
			&position.Quantity,
			&position.AveragePrice,
			&position.Metadata,
			&position.MasterID,
			&position.MasterUUID,
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
	query := `
		INSERT INTO products (
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
