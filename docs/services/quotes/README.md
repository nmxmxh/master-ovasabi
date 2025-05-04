# Quotes Service

The Quotes service manages financial quotes, pricing data, and related analytics in the OVASABI
platform. It provides real-time and historical quote data with support for various financial
instruments.

## Architecture

```mermaid
graph TD
    A[gRPC API] -->|Implements| B[Service Layer]
    B -->|Uses| C[Repository Layer]
    C -->|Writes| D[Postgres]
    C -->|Caches| E[Redis]
    G[Master Repository] -->|References| D
    H[Event Log] -->|Records| D
    I[Campaign Service] -->|References| B
    J[Finance Service] -->|Uses| B
```

## Features

1. **Quote Management**

   - Quote creation and updates
   - Real-time price tracking
   - Volume tracking
   - Historical data storage

2. **Data Access**

   - Symbol-based retrieval
   - Historical data queries
   - Batch quote retrieval
   - Pagination support

3. **Analytics Support**

   - Price history analysis
   - Volume analysis
   - Metadata tracking
   - Time-series data

4. **Integration Features**
   - Campaign integration
   - Finance service integration
   - Master-client pattern
   - Event logging

## API Reference

### Proto Definition

```protobuf
service QuotesService {
  rpc CreateQuote(CreateQuoteRequest) returns (CreateQuoteResponse) {}
  rpc GetQuote(GetQuoteRequest) returns (GetQuoteResponse) {}
  rpc ListQuotes(ListQuotesRequest) returns (ListQuotesResponse) {}
}

message BillingQuote {
  int32 id = 1;
  int32 master_id = 2;
  int32 campaign_id = 3;
  string description = 4;
  string author = 5;
  map<string, string> metadata = 6;
  google.protobuf.Timestamp created_at = 7;
  double amount = 8;
  string currency = 9;
}
```

### Methods

#### CreateQuote

Creates a new quote record.

```go
func (s *service) CreateQuote(ctx context.Context, req *pb.CreateQuoteRequest) (*pb.CreateQuoteResponse, error)
```

#### GetQuote

Retrieves a specific quote by ID.

```go
func (s *service) GetQuote(ctx context.Context, req *pb.GetQuoteRequest) (*pb.GetQuoteResponse, error)
```

#### ListQuotes

Retrieves a paginated list of quotes.

```go
func (s *service) ListQuotes(ctx context.Context, req *pb.ListQuotesRequest) (*pb.ListQuotesResponse, error)
```

## Data Model

### Quote Model

```go
type Quote struct {
    ID        int64     `db:"id"`
    MasterID  int64     `db:"master_id"`
    Symbol    string    `db:"symbol"`
    Price     float64   `db:"price"`
    Volume    int64     `db:"volume"`
    Metadata  string    `db:"metadata"`
    Timestamp time.Time `db:"timestamp"`
    CreatedAt time.Time `db:"created_at"`
    UpdatedAt time.Time `db:"updated_at"`
}
```

### Database Schema

```sql
CREATE TABLE service_quote (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL,
    price DECIMAL(20,8) NOT NULL,
    volume BIGINT NOT NULL,
    metadata JSONB DEFAULT '{}',
    timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_quote_symbol ON service_quote(symbol);
CREATE INDEX idx_quote_timestamp ON service_quote(timestamp);
CREATE INDEX idx_quote_master_id ON service_quote(master_id);
CREATE INDEX idx_quote_metadata ON service_quote USING gin(metadata);
```

## Knowledge Graph Integration

### Capabilities

```go
type QuoteCapabilities struct {
    QuoteManagement struct {
        Creation     bool `json:"creation"`
        Updates      bool `json:"updates"`
        Retrieval    bool `json:"retrieval"`
        History      bool `json:"history"`
    } `json:"quote_management"`

    DataAccess struct {
        SymbolBased  bool `json:"symbol_based"`
        Historical   bool `json:"historical"`
        Batch        bool `json:"batch"`
        Pagination   bool `json:"pagination"`
    } `json:"data_access"`

    Analytics struct {
        PriceHistory bool `json:"price_history"`
        Volume       bool `json:"volume"`
        Metadata     bool `json:"metadata"`
        TimeSeries   bool `json:"time_series"`
    } `json:"analytics"`
}
```

## Caching Strategy

### Key Structure

- Quote: `cache:quote:{quote_id}`
- Symbol Latest: `cache:quote:symbol:{symbol}:latest`
- Symbol History: `cache:quote:symbol:{symbol}:history`
- Batch Quotes: `cache:quote:batch:{symbols_hash}`

### TTL Values

- Quote: 5 minutes
- Symbol Latest: 1 minute
- Symbol History: 15 minutes
- Batch Quotes: 2 minutes

## Error Handling

### Error Types

1. **QuoteError**

   - Invalid symbol
   - Price validation
   - Volume validation
   - Timestamp validation

2. **DataAccessError**
   - Not found
   - Invalid range
   - Invalid pagination
   - Batch size limit

## Monitoring

### Metrics

1. **Quote Metrics**

   - Quote creation rate
   - Quote retrieval rate
   - Cache hit ratio
   - Error rates

2. **Performance Metrics**
   - Response times
   - Database load
   - Cache performance
   - Resource usage

## Security

1. **Access Control**

   - Symbol permissions
   - Rate limiting
   - Audit logging
   - Data validation

2. **Data Protection**
   - Price integrity
   - Volume validation
   - Timestamp validation
   - Metadata sanitization

## Future Improvements

1. **Phase 1 - Enhanced Data Access**

   - Advanced filtering
   - Aggregation support
   - Real-time updates
   - WebSocket support

2. **Phase 2 - Analytics Enhancement**

   - Price analytics
   - Volume analytics
   - Trend analysis
   - Pattern detection

3. **Phase 3 - Integration Expansion**
   - More financial instruments
   - External data sources
   - Advanced caching
   - ML integration

## Dependencies

- Campaign Service (v1)
- Finance Service (v1)
- Redis Cache
- Postgres Database

## Configuration

```yaml
quotes:
  database:
    pool_size: 20
    max_idle: 10
    max_lifetime: 1h
  cache:
    ttl:
      quote: 5m
      symbol_latest: 1m
      symbol_history: 15m
      batch_quotes: 2m
  data:
    max_batch_size: 100
    max_history_days: 365
    price_precision: 8
    volume_min: 0
  rate_limits:
    create: 100/minute
    get: 1000/minute
    list: 500/minute
```

## Deployment

### Resource Requirements

- CPU: 2 cores
- Memory: 4GB
- Storage: 50GB
- Cache: 2GB Redis

### Environment Variables

```bash
QUOTES_DB_URL=postgres://user:pass@host:5432/db
QUOTES_REDIS_URL=redis://host:6379
QUOTES_LOG_LEVEL=info
QUOTES_API_PORT=50059
```

## Dependency Injection & Provider Pattern

- The Quotes service is registered and resolved via the central Provider using a DI container
  (`internal/service/provider.go`).
- Modular registration ensures the service is only registered once.
- Health and metrics are managed centrally and exposed for observability.
- Amadeus registration is performed at service startup for capability tracking.

## Babel & Location-Based Pricing

- The Quotes service integrates with the Babel service for dynamic, location-based pricing rules and
  i18n support.
- Quote generation uses Babel to determine the best pricing rule for a user's location and context.

## Amadeus Integration & Observability

- The service registers its capabilities and dependencies with Amadeus at startup.
- Health checks and metrics are exposed and tracked centrally.
