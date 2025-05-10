# Package finance

## Variables

### ErrTransactionNotFound

## Types

### CachedRepository

CachedRepository wraps a Repository with caching capabilities.

#### Methods

##### CreateTransaction

CreateTransaction creates a transaction and caches it.

##### GetBalance

GetBalance retrieves the balance from cache or repository.

##### GetTransaction

GetTransaction retrieves a transaction from cache or repository.

##### ListTransactions

ListTransactions retrieves transactions from repository (no caching for lists).

##### UpdateBalance

UpdateBalance updates the balance and invalidates cache.

### Repository

Repository defines the interface for finance operations.

### Transaction

Transaction represents a financial transaction.

### TransactionModel

TransactionModel represents a financial transaction in the database.
