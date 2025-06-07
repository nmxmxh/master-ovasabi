# Package commerce

## Variables

### CommerceEventRegistry

## Types

### AssetPosition

### Balance

### BankAccount

--- Banking ---.

### BankStatement

### BankTransfer

### Event

### EventEmitter

EventEmitter defines the interface for emitting events in the commerce service.

### EventHandlerFunc

### EventRegistry

### EventSubscription

### ExchangeOrder

--- Exchange ---.

### ExchangePair

### ExchangeRate

### InvestmentAccount

--- Investment ---.

### InvestmentOrder

### MarketplaceListing

--- Marketplace ---.

### MarketplaceOffer

### MarketplaceOrder

### Metadata

CommerceServiceMetadata is the service_specific.commerce metadata struct.

### Order

### OrderItem

### Payment

### PaymentPartnerMetadata

PaymentPartnerMetadata describes a payment partner suggestion for a given context.

### Portfolio

### Quote

### Repository

RepositoryInterface defines all public methods for the commerce repository.

### Service

#### Methods

##### ConfirmPayment

##### CreateExchangePair

##### CreateExchangeRate

##### CreateInvestmentAccount

--- Investment ---.

##### CreateOrder

Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern
(System-Wide)'.

##### CreateQuote

##### GetBalance

##### GetInvestmentAccount

--- Investment/Account/Asset Service Methods ---.

##### GetOrder

##### GetPortfolio

##### GetQuote

##### GetTransaction

##### InitiatePayment

##### ListBalances

##### ListEvents

##### ListOrders

##### ListPortfolios

##### ListQuotes

##### ListTransactions

##### PlaceInvestmentOrder

##### RefundPayment

##### UpdateOrderStatus

Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern
(System-Wide)'.

### Transaction

## Functions

### NewService

### Register

Register registers the commerce service with the DI container and event bus support.

### StartEventSubscribers
