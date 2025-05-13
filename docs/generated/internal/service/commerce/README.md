# Package commerce

## Types

### Service

#### Methods

##### ConfirmPayment

TODO: Implement payment confirmation logic (validate, call repo.ConfirmPayment, handle errors,
return response).

##### CreateOrder

TODO (Amadeus Context): Implement CreateOrder following the canonical metadata pattern. Reference:
docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.

##### CreateQuote

TODO (Amadeus Context): Implement CreateQuote following the canonical metadata pattern. Reference:
docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
Steps: Validate metadata, store as jsonb, call pattern helpers, handle/log all errors.

##### GetBalance

TODO: Implement get balance logic (fetch by user/currency, call repo.GetBalance, return response).

##### GetOrder

TODO: Implement get order logic (fetch by ID, handle not found, return proto).

##### GetQuote

TODO: Implement get quote logic (fetch by ID, handle not found, return proto).

##### GetTransaction

TODO: Implement get transaction logic (fetch by ID, handle not found, return proto).

##### InitiatePayment

TODO: Implement payment initiation logic (validate, call repo.CreatePayment, handle errors, return
response).

##### ListBalances

TODO: Implement list balances logic (fetch all balances for user, call repo.ListBalances, return
response).

##### ListEvents

TODO: Implement list events logic (fetch events for entity, call repo.ListEvents, return response).

##### ListOrders

TODO: Implement list orders logic (pagination, filtering, call repo.ListOrders, return response).

##### ListQuotes

TODO: Implement list quotes logic (pagination, filtering, call repo.ListQuotes, return response).

##### ListTransactions

TODO: Implement list transactions logic (pagination, filtering, call repo.ListTransactions, return
response).

##### RefundPayment

TODO: Implement payment refund logic (validate, call repo.RefundPayment, handle errors, return
response).

##### UpdateOrderStatus

TODO (Amadeus Context): Implement UpdateOrderStatus following the canonical metadata pattern.
Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern
(System-Wide)'.

## Functions

### NewService
