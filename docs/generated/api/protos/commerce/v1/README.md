# Package commercepb

## Constants

### CommerceService_CreateQuote_FullMethodName

## Variables

### QuoteStatus_name

Enum value maps for QuoteStatus.

### OrderStatus_name

Enum value maps for OrderStatus.

### PaymentStatus_name

Enum value maps for PaymentStatus.

### TransactionType_name

Enum value maps for TransactionType.

### TransactionStatus_name

Enum value maps for TransactionStatus.

### CommerceService_ServiceDesc

CommerceService_ServiceDesc is the grpc.ServiceDesc for CommerceService service. It's only intended
for direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

### File_commerce_v1_commerce_proto

## Types

### Balance

#### Methods

##### Descriptor

Deprecated: Use Balance.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetCurrency

##### GetUpdatedAt

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CommerceEvent

#### Methods

##### Descriptor

Deprecated: Use CommerceEvent.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetEntityId

##### GetEntityType

##### GetEventId

##### GetEventType

##### GetPayload

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CommerceServiceClient

CommerceServiceClient is the client API for CommerceService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### CommerceServiceServer

CommerceServiceServer is the server API for CommerceService service. All implementations must embed
UnimplementedCommerceServiceServer for forward compatibility.

### ConfirmPaymentRequest

#### Methods

##### Descriptor

Deprecated: Use ConfirmPaymentRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPaymentId

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ConfirmPaymentResponse

#### Methods

##### Descriptor

Deprecated: Use ConfirmPaymentResponse.ProtoReflect.Descriptor instead.

##### GetPayment

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateOrderRequest

--- Orders ---

#### Methods

##### Descriptor

Deprecated: Use CreateOrderRequest.ProtoReflect.Descriptor instead.

##### GetCurrency

##### GetItems

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateOrderResponse

#### Methods

##### Descriptor

Deprecated: Use CreateOrderResponse.ProtoReflect.Descriptor instead.

##### GetOrder

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateQuoteRequest

--- Quotes ---

#### Methods

##### Descriptor

Deprecated: Use CreateQuoteRequest.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetCurrency

##### GetMetadata

##### GetProductId

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateQuoteResponse

#### Methods

##### Descriptor

Deprecated: Use CreateQuoteResponse.ProtoReflect.Descriptor instead.

##### GetQuote

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetBalanceRequest

--- Balances ---

#### Methods

##### Descriptor

Deprecated: Use GetBalanceRequest.ProtoReflect.Descriptor instead.

##### GetCurrency

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetBalanceResponse

#### Methods

##### Descriptor

Deprecated: Use GetBalanceResponse.ProtoReflect.Descriptor instead.

##### GetBalance

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetOrderRequest

#### Methods

##### Descriptor

Deprecated: Use GetOrderRequest.ProtoReflect.Descriptor instead.

##### GetOrderId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetOrderResponse

#### Methods

##### Descriptor

Deprecated: Use GetOrderResponse.ProtoReflect.Descriptor instead.

##### GetOrder

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetQuoteRequest

#### Methods

##### Descriptor

Deprecated: Use GetQuoteRequest.ProtoReflect.Descriptor instead.

##### GetQuoteId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetQuoteResponse

#### Methods

##### Descriptor

Deprecated: Use GetQuoteResponse.ProtoReflect.Descriptor instead.

##### GetQuote

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetTransactionRequest

--- Transactions ---

#### Methods

##### Descriptor

Deprecated: Use GetTransactionRequest.ProtoReflect.Descriptor instead.

##### GetTransactionId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetTransactionResponse

#### Methods

##### Descriptor

Deprecated: Use GetTransactionResponse.ProtoReflect.Descriptor instead.

##### GetTransaction

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### InitiatePaymentRequest

--- Payments ---

#### Methods

##### Descriptor

Deprecated: Use InitiatePaymentRequest.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetCurrency

##### GetMetadata

##### GetMethod

##### GetOrderId

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### InitiatePaymentResponse

#### Methods

##### Descriptor

Deprecated: Use InitiatePaymentResponse.ProtoReflect.Descriptor instead.

##### GetPayment

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListBalancesRequest

#### Methods

##### Descriptor

Deprecated: Use ListBalancesRequest.ProtoReflect.Descriptor instead.

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListBalancesResponse

#### Methods

##### Descriptor

Deprecated: Use ListBalancesResponse.ProtoReflect.Descriptor instead.

##### GetBalances

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListEventsRequest

--- Events (Analytics/Audit) ---

#### Methods

##### Descriptor

Deprecated: Use ListEventsRequest.ProtoReflect.Descriptor instead.

##### GetEntityId

##### GetEntityType

##### GetPage

##### GetPageSize

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListEventsResponse

#### Methods

##### Descriptor

Deprecated: Use ListEventsResponse.ProtoReflect.Descriptor instead.

##### GetEvents

##### GetTotal

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListOrdersRequest

#### Methods

##### Descriptor

Deprecated: Use ListOrdersRequest.ProtoReflect.Descriptor instead.

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListOrdersResponse

#### Methods

##### Descriptor

Deprecated: Use ListOrdersResponse.ProtoReflect.Descriptor instead.

##### GetOrders

##### GetTotal

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListQuotesRequest

#### Methods

##### Descriptor

Deprecated: Use ListQuotesRequest.ProtoReflect.Descriptor instead.

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListQuotesResponse

#### Methods

##### Descriptor

Deprecated: Use ListQuotesResponse.ProtoReflect.Descriptor instead.

##### GetQuotes

##### GetTotal

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListTransactionsRequest

#### Methods

##### Descriptor

Deprecated: Use ListTransactionsRequest.ProtoReflect.Descriptor instead.

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListTransactionsResponse

#### Methods

##### Descriptor

Deprecated: Use ListTransactionsResponse.ProtoReflect.Descriptor instead.

##### GetTotal

##### GetTransactions

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Order

#### Methods

##### Descriptor

Deprecated: Use Order.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetCurrency

##### GetItems

##### GetMetadata

##### GetOrderId

##### GetStatus

##### GetTotal

##### GetUpdatedAt

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### OrderItem

#### Methods

##### Descriptor

Deprecated: Use OrderItem.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPrice

##### GetProductId

##### GetQuantity

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### OrderStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use OrderStatus.Descriptor instead.

##### Number

##### String

##### Type

### Payment

#### Methods

##### Descriptor

Deprecated: Use Payment.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetCreatedAt

##### GetCurrency

##### GetMetadata

##### GetMethod

##### GetOrderId

##### GetPaymentId

##### GetStatus

##### GetUpdatedAt

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### PaymentStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use PaymentStatus.Descriptor instead.

##### Number

##### String

##### Type

### Quote

#### Methods

##### Descriptor

Deprecated: Use Quote.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetCreatedAt

##### GetCurrency

##### GetMetadata

##### GetProductId

##### GetQuoteId

##### GetStatus

##### GetUpdatedAt

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### QuoteStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use QuoteStatus.Descriptor instead.

##### Number

##### String

##### Type

### RefundPaymentRequest

#### Methods

##### Descriptor

Deprecated: Use RefundPaymentRequest.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetMetadata

##### GetPaymentId

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RefundPaymentResponse

#### Methods

##### Descriptor

Deprecated: Use RefundPaymentResponse.ProtoReflect.Descriptor instead.

##### GetPayment

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Transaction

#### Methods

##### Descriptor

Deprecated: Use Transaction.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetCreatedAt

##### GetCurrency

##### GetMetadata

##### GetPaymentId

##### GetStatus

##### GetTransactionId

##### GetType

##### GetUpdatedAt

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TransactionStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use TransactionStatus.Descriptor instead.

##### Number

##### String

##### Type

### TransactionType

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use TransactionType.Descriptor instead.

##### Number

##### String

##### Type

### UnimplementedCommerceServiceServer

UnimplementedCommerceServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### ConfirmPayment

##### CreateOrder

##### CreateQuote

##### GetBalance

##### GetOrder

##### GetQuote

##### GetTransaction

##### InitiatePayment

##### ListBalances

##### ListEvents

##### ListOrders

##### ListQuotes

##### ListTransactions

##### RefundPayment

##### UpdateOrderStatus

### UnsafeCommerceServiceServer

UnsafeCommerceServiceServer may be embedded to opt out of forward compatibility for this service.
Use of this interface is not recommended, as added methods to CommerceServiceServer will result in
compilation errors.

### UpdateOrderStatusRequest

#### Methods

##### Descriptor

Deprecated: Use UpdateOrderStatusRequest.ProtoReflect.Descriptor instead.

##### GetOrderId

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateOrderStatusResponse

#### Methods

##### Descriptor

Deprecated: Use UpdateOrderStatusResponse.ProtoReflect.Descriptor instead.

##### GetOrder

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterCommerceServiceServer
