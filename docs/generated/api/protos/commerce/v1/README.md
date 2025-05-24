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

### InvestmentOrderStatus_name

Enum value maps for InvestmentOrderStatus.

### BankTransferStatus_name

Enum value maps for BankTransferStatus.

### ListingStatus_name

Enum value maps for ListingStatus.

### MarketplaceOrderStatus_name

Enum value maps for MarketplaceOrderStatus.

### OfferStatus_name

Enum value maps for OfferStatus.

### ExchangeOrderStatus_name

Enum value maps for ExchangeOrderStatus.

### CommerceService_ServiceDesc

CommerceService_ServiceDesc is the grpc.ServiceDesc for CommerceService service. It's only intended
for direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

### File_commerce_v1_commerce_proto

## Types

### Account

#### Methods

##### Descriptor

Deprecated: Use Account.ProtoReflect.Descriptor instead.

##### GetAccountId

##### GetBalance

##### GetCurrency

##### GetMetadata

##### GetPartyId

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Asset

#### Methods

##### Descriptor

Deprecated: Use Asset.ProtoReflect.Descriptor instead.

##### GetAssetId

##### GetCreatedAt

##### GetMetadata

##### GetName

##### GetSymbol

##### GetType

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AssetPosition

#### Methods

##### Descriptor

Deprecated: Use AssetPosition.ProtoReflect.Descriptor instead.

##### GetAssetId

##### GetAveragePrice

##### GetCreatedAt

##### GetMetadata

##### GetQuantity

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Balance

#### Methods

##### Descriptor

Deprecated: Use Balance.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetCampaignId

##### GetCurrency

##### GetMetadata

##### GetUpdatedAt

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BankAccount

--- Banking ---

#### Methods

##### Descriptor

Deprecated: Use BankAccount.ProtoReflect.Descriptor instead.

##### GetAccountId

##### GetBalance

##### GetBic

##### GetCurrency

##### GetIban

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BankStatement

#### Methods

##### Descriptor

Deprecated: Use BankStatement.ProtoReflect.Descriptor instead.

##### GetAccountId

##### GetMetadata

##### GetTransactions

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BankTransfer

#### Methods

##### Descriptor

Deprecated: Use BankTransfer.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetCreatedAt

##### GetCurrency

##### GetFromAccountId

##### GetMetadata

##### GetStatus

##### GetToAccountId

##### GetTransferId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BankTransferStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use BankTransferStatus.Descriptor instead.

##### Number

##### String

##### Type

### CommerceEvent

#### Methods

##### Descriptor

Deprecated: Use CommerceEvent.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetCreatedAt

##### GetEntityId

##### GetEntityType

##### GetEventId

##### GetEventType

##### GetMetadata

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

##### GetCampaignId

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

### CreateBankAccountRequest

Banking

#### Methods

##### Descriptor

Deprecated: Use CreateBankAccountRequest.ProtoReflect.Descriptor instead.

##### GetBalance

##### GetBic

##### GetCampaignId

##### GetCurrency

##### GetIban

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateBankAccountResponse

#### Methods

##### Descriptor

Deprecated: Use CreateBankAccountResponse.ProtoReflect.Descriptor instead.

##### GetAccount

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateExchangePairRequest

#### Methods

##### Descriptor

Deprecated: Use CreateExchangePairRequest.ProtoReflect.Descriptor instead.

##### GetBaseAsset

##### GetMetadata

##### GetPairId

##### GetQuoteAsset

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateExchangePairResponse

#### Methods

##### Descriptor

Deprecated: Use CreateExchangePairResponse.ProtoReflect.Descriptor instead.

##### GetPair

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateExchangeRateRequest

#### Methods

##### Descriptor

Deprecated: Use CreateExchangeRateRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetMetadata

##### GetPairId

##### GetRate

##### GetTimestamp

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateExchangeRateResponse

#### Methods

##### Descriptor

Deprecated: Use CreateExchangeRateResponse.ProtoReflect.Descriptor instead.

##### GetRate

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateInvestmentAccountRequest

Investment

#### Methods

##### Descriptor

Deprecated: Use CreateInvestmentAccountRequest.ProtoReflect.Descriptor instead.

##### GetBalance

##### GetCampaignId

##### GetCurrency

##### GetMetadata

##### GetOwnerId

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateInvestmentAccountResponse

#### Methods

##### Descriptor

Deprecated: Use CreateInvestmentAccountResponse.ProtoReflect.Descriptor instead.

##### GetAccount

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateListingRequest

Marketplace

#### Methods

##### Descriptor

Deprecated: Use CreateListingRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetCurrency

##### GetMetadata

##### GetPrice

##### GetProductId

##### GetSellerId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateListingResponse

#### Methods

##### Descriptor

Deprecated: Use CreateListingResponse.ProtoReflect.Descriptor instead.

##### GetListing

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateOrderRequest

--- Orders ---

#### Methods

##### Descriptor

Deprecated: Use CreateOrderRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

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

##### GetCampaignId

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

### ExchangeOrder

--- Exchange ---

#### Methods

##### Descriptor

Deprecated: Use ExchangeOrder.ProtoReflect.Descriptor instead.

##### GetAccountId

##### GetAmount

##### GetCreatedAt

##### GetMetadata

##### GetOrderId

##### GetOrderType

##### GetPair

##### GetPrice

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ExchangeOrderStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use ExchangeOrderStatus.Descriptor instead.

##### Number

##### String

##### Type

### ExchangePair

#### Methods

##### Descriptor

Deprecated: Use ExchangePair.ProtoReflect.Descriptor instead.

##### GetBaseAsset

##### GetMetadata

##### GetPairId

##### GetQuoteAsset

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ExchangeRate

#### Methods

##### Descriptor

Deprecated: Use ExchangeRate.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPairId

##### GetRate

##### GetTimestamp

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetBalanceRequest

--- Balances ---

#### Methods

##### Descriptor

Deprecated: Use GetBalanceRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

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

### GetBankStatementRequest

#### Methods

##### Descriptor

Deprecated: Use GetBankStatementRequest.ProtoReflect.Descriptor instead.

##### GetAccountId

##### GetCampaignId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetBankStatementResponse

#### Methods

##### Descriptor

Deprecated: Use GetBankStatementResponse.ProtoReflect.Descriptor instead.

##### GetStatement

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetExchangeRateRequest

#### Methods

##### Descriptor

Deprecated: Use GetExchangeRateRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetPairId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetExchangeRateResponse

#### Methods

##### Descriptor

Deprecated: Use GetExchangeRateResponse.ProtoReflect.Descriptor instead.

##### GetRate

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetInvestmentAccountRequest

#### Methods

##### Descriptor

Deprecated: Use GetInvestmentAccountRequest.ProtoReflect.Descriptor instead.

##### GetAccountId

##### GetCampaignId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetInvestmentAccountResponse

#### Methods

##### Descriptor

Deprecated: Use GetInvestmentAccountResponse.ProtoReflect.Descriptor instead.

##### GetAccount

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetOrderRequest

#### Methods

##### Descriptor

Deprecated: Use GetOrderRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

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

### GetPortfolioRequest

#### Methods

##### Descriptor

Deprecated: Use GetPortfolioRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetPortfolioId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetPortfolioResponse

#### Methods

##### Descriptor

Deprecated: Use GetPortfolioResponse.ProtoReflect.Descriptor instead.

##### GetPortfolio

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetQuoteRequest

#### Methods

##### Descriptor

Deprecated: Use GetQuoteRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

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

##### GetCampaignId

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

### InitiateBankTransferRequest

#### Methods

##### Descriptor

Deprecated: Use InitiateBankTransferRequest.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetCampaignId

##### GetCurrency

##### GetFromAccountId

##### GetMetadata

##### GetToAccountId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### InitiateBankTransferResponse

#### Methods

##### Descriptor

Deprecated: Use InitiateBankTransferResponse.ProtoReflect.Descriptor instead.

##### GetTransfer

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

##### GetCampaignId

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

### InvestmentAccount

--- Investment ---

#### Methods

##### Descriptor

Deprecated: Use InvestmentAccount.ProtoReflect.Descriptor instead.

##### GetAccountId

##### GetBalance

##### GetCampaignId

##### GetCreatedAt

##### GetCurrency

##### GetMetadata

##### GetOwnerId

##### GetType

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### InvestmentOrder

#### Methods

##### Descriptor

Deprecated: Use InvestmentOrder.ProtoReflect.Descriptor instead.

##### GetAccountId

##### GetAssetId

##### GetCampaignId

##### GetCreatedAt

##### GetMetadata

##### GetOrderId

##### GetOrderType

##### GetPrice

##### GetQuantity

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### InvestmentOrderStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use InvestmentOrderStatus.Descriptor instead.

##### Number

##### String

##### Type

### ListAssetsRequest

#### Methods

##### Descriptor

Deprecated: Use ListAssetsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListAssetsResponse

#### Methods

##### Descriptor

Deprecated: Use ListAssetsResponse.ProtoReflect.Descriptor instead.

##### GetAssets

##### GetTotal

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListBalancesRequest

#### Methods

##### Descriptor

Deprecated: Use ListBalancesRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

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

##### GetCampaignId

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

### ListExchangePairsRequest

#### Methods

##### Descriptor

Deprecated: Use ListExchangePairsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListExchangePairsResponse

#### Methods

##### Descriptor

Deprecated: Use ListExchangePairsResponse.ProtoReflect.Descriptor instead.

##### GetPairs

##### GetTotal

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListListingsRequest

#### Methods

##### Descriptor

Deprecated: Use ListListingsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListListingsResponse

#### Methods

##### Descriptor

Deprecated: Use ListListingsResponse.ProtoReflect.Descriptor instead.

##### GetListings

##### GetTotal

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListOrdersRequest

#### Methods

##### Descriptor

Deprecated: Use ListOrdersRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

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

### ListPortfoliosRequest

#### Methods

##### Descriptor

Deprecated: Use ListPortfoliosRequest.ProtoReflect.Descriptor instead.

##### GetAccountId

##### GetCampaignId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListPortfoliosResponse

#### Methods

##### Descriptor

Deprecated: Use ListPortfoliosResponse.ProtoReflect.Descriptor instead.

##### GetPortfolios

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListQuotesRequest

#### Methods

##### Descriptor

Deprecated: Use ListQuotesRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

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

##### GetCampaignId

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

### ListingStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use ListingStatus.Descriptor instead.

##### Number

##### String

##### Type

### MakeOfferRequest

#### Methods

##### Descriptor

Deprecated: Use MakeOfferRequest.ProtoReflect.Descriptor instead.

##### GetBuyerId

##### GetCampaignId

##### GetCurrency

##### GetListingId

##### GetMetadata

##### GetOfferPrice

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MakeOfferResponse

#### Methods

##### Descriptor

Deprecated: Use MakeOfferResponse.ProtoReflect.Descriptor instead.

##### GetOffer

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MarketplaceListing

--- Marketplace ---

#### Methods

##### Descriptor

Deprecated: Use MarketplaceListing.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetCurrency

##### GetListingId

##### GetMetadata

##### GetPrice

##### GetProductId

##### GetSellerId

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MarketplaceOffer

#### Methods

##### Descriptor

Deprecated: Use MarketplaceOffer.ProtoReflect.Descriptor instead.

##### GetBuyerId

##### GetCreatedAt

##### GetCurrency

##### GetListingId

##### GetMetadata

##### GetOfferId

##### GetOfferPrice

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MarketplaceOrder

#### Methods

##### Descriptor

Deprecated: Use MarketplaceOrder.ProtoReflect.Descriptor instead.

##### GetBuyerId

##### GetCreatedAt

##### GetCurrency

##### GetListingId

##### GetMetadata

##### GetOrderId

##### GetPrice

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MarketplaceOrderStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use MarketplaceOrderStatus.Descriptor instead.

##### Number

##### String

##### Type

### OfferStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use OfferStatus.Descriptor instead.

##### Number

##### String

##### Type

### Order

#### Methods

##### Descriptor

Deprecated: Use Order.ProtoReflect.Descriptor instead.

##### GetCampaignId

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

### Party

--- Shared Primitives ---

#### Methods

##### Descriptor

Deprecated: Use Party.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetName

##### GetPartyId

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Payment

#### Methods

##### Descriptor

Deprecated: Use Payment.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetCampaignId

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

### PlaceExchangeOrderRequest

Exchange

#### Methods

##### Descriptor

Deprecated: Use PlaceExchangeOrderRequest.ProtoReflect.Descriptor instead.

##### GetAccountId

##### GetAmount

##### GetCampaignId

##### GetMetadata

##### GetOrderType

##### GetPair

##### GetPrice

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### PlaceExchangeOrderResponse

#### Methods

##### Descriptor

Deprecated: Use PlaceExchangeOrderResponse.ProtoReflect.Descriptor instead.

##### GetOrder

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### PlaceInvestmentOrderRequest

#### Methods

##### Descriptor

Deprecated: Use PlaceInvestmentOrderRequest.ProtoReflect.Descriptor instead.

##### GetAccountId

##### GetAssetId

##### GetCampaignId

##### GetMetadata

##### GetOrderType

##### GetPrice

##### GetQuantity

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### PlaceInvestmentOrderResponse

#### Methods

##### Descriptor

Deprecated: Use PlaceInvestmentOrderResponse.ProtoReflect.Descriptor instead.

##### GetOrder

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### PlaceMarketplaceOrderRequest

#### Methods

##### Descriptor

Deprecated: Use PlaceMarketplaceOrderRequest.ProtoReflect.Descriptor instead.

##### GetBuyerId

##### GetCampaignId

##### GetCurrency

##### GetListingId

##### GetMetadata

##### GetPrice

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### PlaceMarketplaceOrderResponse

#### Methods

##### Descriptor

Deprecated: Use PlaceMarketplaceOrderResponse.ProtoReflect.Descriptor instead.

##### GetOrder

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Portfolio

#### Methods

##### Descriptor

Deprecated: Use Portfolio.ProtoReflect.Descriptor instead.

##### GetAccountId

##### GetCreatedAt

##### GetMetadata

##### GetPortfolioId

##### GetPositions

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Quote

#### Methods

##### Descriptor

Deprecated: Use Quote.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetCampaignId

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

##### GetCampaignId

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

##### GetCampaignId

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

##### CreateBankAccount

##### CreateExchangePair

##### CreateExchangeRate

##### CreateInvestmentAccount

##### CreateListing

##### CreateOrder

##### CreateQuote

##### GetBalance

##### GetBankStatement

##### GetExchangeRate

##### GetInvestmentAccount

##### GetOrder

##### GetPortfolio

##### GetQuote

##### GetTransaction

##### InitiateBankTransfer

##### InitiatePayment

##### ListAssets

##### ListBalances

##### ListEvents

##### ListExchangePairs

##### ListListings

##### ListOrders

##### ListPortfolios

##### ListQuotes

##### ListTransactions

##### MakeOffer

##### PlaceExchangeOrder

##### PlaceInvestmentOrder

##### PlaceMarketplaceOrder

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

##### GetCampaignId

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
