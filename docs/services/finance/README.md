# Finance Service

The Finance service manages all financial transactions, balances, and accounting operations in the
OVASABI platform.

## Architecture

```mermaid
graph TD
    A[gRPC API] -->|Implements| B[Service Layer]
    B -->|Uses| C[Repository Layer]
    C -->|Writes| D[Postgres]
    C -->|Caches| E[Redis]
    F[Master Repository] -->|References| D
    G[Event Log] -->|Records| D
    H[Auth Service] -->|Validates| B
    I[User Service] -->|References| B
```

## Features

1. **Transaction Management**

   - Transaction creation
   - Transaction processing
   - Status tracking
   - History and audit

2. **Balance Management**

   - Account balances
   - Balance updates
   - Currency conversion
   - Balance history

3. **Payment Processing**

   - Payment methods
   - Payment execution
   - Payment verification
   - Refund handling

4. **Financial Reporting**
   - Transaction reports
   - Balance statements
   - Audit trails
   - Analytics

## API Reference

### Proto Definition

```protobuf
service FinanceService {
    rpc CreateTransaction(CreateTransactionRequest) returns (TransactionResponse);
    rpc GetTransaction(GetTransactionRequest) returns (TransactionResponse);
    rpc ListTransactions(ListTransactionsRequest) returns (ListTransactionsResponse);
    rpc ProcessPayment(ProcessPaymentRequest) returns (PaymentResponse);
    rpc GetBalance(GetBalanceRequest) returns (BalanceResponse);
    rpc UpdateBalance(UpdateBalanceRequest) returns (BalanceResponse);
    rpc GenerateReport(GenerateReportRequest) returns (ReportResponse);
}
```

### Methods

#### CreateTransaction

Creates a new financial transaction.

```go
func (s *service) CreateTransaction(ctx context.Context, req *pb.CreateTransactionRequest) (*pb.TransactionResponse, error)
```

#### ProcessPayment

Processes a payment transaction.

```go
func (s *service) ProcessPayment(ctx context.Context, req *pb.ProcessPaymentRequest) (*pb.PaymentResponse, error)
```

## Data Model

### Transaction Model

```go
type TransactionModel struct {
    ID              uuid.UUID
    UserID          uuid.UUID
    Type            string // credit, debit, transfer
    Amount          decimal.Decimal
    Currency        string
    Status          string // pending, completed, failed
    PaymentMethod   string
    Reference       string
    Metadata        map[string]interface{} // JSONB in DB
    CreatedAt       time.Time
    UpdatedAt       time.Time
    CompletedAt     *time.Time
}
```

### Database Schema

```sql
CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id),
    user_id UUID NOT NULL,
    type TEXT NOT NULL,
    amount DECIMAL NOT NULL,
    currency TEXT NOT NULL,
    status TEXT NOT NULL,
    payment_method TEXT,
    reference TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);
CREATE INDEX idx_transactions_metadata ON transactions USING gin(metadata);
```

## Knowledge Graph

### Capabilities

```go
type FinanceCapabilities struct {
    TransactionManagement struct {
        Creation     bool `json:"creation"`
        Processing   bool `json:"processing"`
        Tracking     bool `json:"tracking"`
        Audit        bool `json:"audit"`
    } `json:"transaction_management"`

    BalanceManagement struct {
        Tracking     bool `json:"tracking"`
        Updates      bool `json:"updates"`
        Conversion   bool `json:"conversion"`
        History      bool `json:"history"`
    } `json:"balance_management"`

    PaymentProcessing struct {
        Methods      bool `json:"methods"`
        Execution    bool `json:"execution"`
        Verification bool `json:"verification"`
        Refunds      bool `json:"refunds"`
    } `json:"payment_processing"`

    Reporting struct {
        Transactions bool `json:"transactions"`
        Balances     bool `json:"balances"`
        Audit        bool `json:"audit"`
        Analytics    bool `json:"analytics"`
    } `json:"reporting"`
}
```

### Growth Patterns

```mermaid
graph TD
    A[Basic Transactions] -->|Add Processing| B[Smart Transactions]
    B -->|Add Analytics| C[Intelligent Transactions]
    C -->|Add ML| D[Predictive Transactions]

    E[Basic Balance] -->|Add History| F[Tracked Balance]
    F -->|Add Conversion| G[Multi-Currency]
    G -->|Add Analytics| H[Smart Balance]

    I[Basic Payments] -->|Add Methods| J[Multi-Payment]
    J -->|Add Security| K[Secure Payment]
    K -->|Add AI| L[Smart Payment]

    M[Basic Reports] -->|Add Analytics| N[Smart Reports]
    N -->|Add ML| O[Predictive Reports]
    O -->|Add AI| P[Intelligent Reports]
```

### Evolution Tracking

```go
type FinanceEvolution struct {
    TransactionCapabilities []string `json:"transaction_capabilities"`
    BalanceCapabilities    []string `json:"balance_capabilities"`
    PaymentCapabilities    []string `json:"payment_capabilities"`
    ReportingFeatures      []string `json:"reporting_features"`
    Version               string    `json:"version"`
    LastUpdated           time.Time `json:"last_updated"`
}
```

## Caching Strategy

### Key Structure

- Transaction: `cache:finance:transaction:{transaction_id}`
- User Balance: `cache:finance:balance:{user_id}:{currency}`
- Payment Status: `cache:finance:payment:{payment_id}`
- Report Cache: `cache:finance:report:{report_id}`

### TTL Values

- Transaction: 1 hour
- Balance: 5 minutes
- Payment Status: 30 minutes
- Report Cache: 24 hours

## Error Handling

### Error Types

1. **TransactionError**

   - Invalid amount
   - Invalid currency
   - Processing failure
   - Insufficient funds

2. **PaymentError**

   - Payment failed
   - Invalid method
   - Verification failed
   - Timeout error

3. **BalanceError**
   - Update failed
   - Conversion error
   - Invalid operation

## Monitoring

### Metrics

1. **Transaction Metrics**

   - Transaction volume
   - Success/failure rates
   - Processing time
   - Amount distribution

2. **Performance Metrics**
   - API latency
   - Cache hit ratio
   - Error rates
   - Database load

## Security

1. **Transaction Security**

   - Amount validation
   - Currency validation
   - Double-entry verification
   - Audit logging

2. **Access Control**
   - Role-based access
   - Amount limits
   - Operation limits
   - IP verification

## Future Improvements

1. **Phase 1 - Enhanced Processing**

   - Multi-currency support
   - Advanced payment methods
   - Real-time processing
   - Smart routing

2. **Phase 2 - Advanced Analytics**

   - ML-based fraud detection
   - Predictive analytics
   - Risk assessment
   - Pattern detection

3. **Phase 3 - Smart Features**
   - Automated reconciliation
   - Smart reporting
   - AI-powered insights
   - Predictive balancing

## Dependencies

- Auth Service (v1)
- User Service (v1)
- Redis Cache
- Postgres Database

## Configuration

```yaml
finance:
  database:
    pool_size: 20
    max_idle: 10
    max_lifetime: 1h
  cache:
    ttl:
      transaction: 1h
      balance: 5m
      payment: 30m
      report: 24h
  limits:
    max_amount: 1000000
    min_amount: 0.01
    currencies: ['USD', 'EUR', 'GBP']
  security:
    verify_balance: true
    double_entry: true
    audit_all: true
```

## Deployment

### Resource Requirements

- CPU: 4 cores
- Memory: 8GB
- Storage: 100GB
- Cache: 4GB Redis

### Environment Variables

```bash
FINANCE_DB_URL=postgres://user:pass@host:5432/db
FINANCE_REDIS_URL=redis://host:6379
FINANCE_LOG_LEVEL=info
FINANCE_API_PORT=50054
FINANCE_PAYMENT_KEY=your-payment-key
```

## Dependency Injection & Provider Pattern

- The Finance service is registered and resolved via the central Provider using a DI container
  (`internal/service/provider.go`).
- Modular registration ensures the service is only registered once.
- Health and metrics are managed centrally and exposed for observability.
- Amadeus registration is performed at service startup for capability tracking.

## Babel & Location-Based Pricing

- The Finance service integrates with the Babel service for dynamic, location-based pricing rules
  and i18n support.
- Pricing logic is driven by metadata and can be updated via the knowledge graph.

## Amadeus Integration & Observability

- The service registers its capabilities and dependencies with Amadeus at startup.
- Health checks and metrics are exposed and tracked centrally.
