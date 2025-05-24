# Package quote

## Variables

### ErrQuoteNotFound

## Types

### Quote

Quote represents a financial quote in the service_quote table.

### QuoteRepository

QuoteRepository handles operations on the service_quote table.

#### Methods

##### Create

Create inserts a new quote record.

##### Delete

Delete removes a quote and its master record.

##### GetByID

GetByID retrieves a quote by ID.

##### GetBySymbol

GetBySymbol retrieves the latest quote for a symbol.

##### GetLatestQuotes

GetLatestQuotes retrieves the latest quotes for multiple symbols.

##### GetQuoteHistory

GetQuoteHistory retrieves quotes for a symbol within a time range.

##### ListBySymbol

ListBySymbol retrieves a paginated list of quotes for a symbol.

## Functions

### SetLogger
