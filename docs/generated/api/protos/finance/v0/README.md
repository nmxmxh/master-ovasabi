# Package financepb

## Constants

### FinanceService_GetBalance_FullMethodName

## Variables

### File_api_protos_finance_v0_finance_proto

### FinanceService_ServiceDesc

FinanceService_ServiceDesc is the grpc.ServiceDesc for FinanceService service. It's only intended
for direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### Balance

Balance represents a user's current financial balance

#### Methods

##### Descriptor

Deprecated: Use Balance.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetLockedAmount

##### GetUpdatedAt

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DepositRequest

DepositRequest is used to add funds to a user's account

#### Methods

##### Descriptor

Deprecated: Use DepositRequest.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetDescription

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### FinanceServiceClient

FinanceServiceClient is the client API for FinanceService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### FinanceServiceServer

FinanceServiceServer is the server API for FinanceService service. All implementations must embed
UnimplementedFinanceServiceServer for forward compatibility

### GetBalanceRequest

GetBalanceRequest is used to request a user's balance

#### Methods

##### Descriptor

Deprecated: Use GetBalanceRequest.ProtoReflect.Descriptor instead.

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetBalanceResponse

GetBalanceResponse returns the user's current balance

#### Methods

##### Descriptor

Deprecated: Use GetBalanceResponse.ProtoReflect.Descriptor instead.

##### GetBalance

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetTransactionRequest

GetTransactionRequest is used to retrieve a specific transaction

#### Methods

##### Descriptor

Deprecated: Use GetTransactionRequest.ProtoReflect.Descriptor instead.

##### GetTransactionId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListTransactionsRequest

ListTransactionsRequest is used to retrieve a list of transactions

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

ListTransactionsResponse returns a paginated list of transactions

#### Methods

##### Descriptor

Deprecated: Use ListTransactionsResponse.ProtoReflect.Descriptor instead.

##### GetHasMore

##### GetTotalCount

##### GetTransactions

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Transaction

Transaction represents a financial transaction

#### Methods

##### Descriptor

Deprecated: Use Transaction.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetCreatedAt

##### GetDescription

##### GetId

##### GetStatus

##### GetToUserId

##### GetType

##### GetUpdatedAt

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TransactionResponse

TransactionResponse returns details of a transaction

#### Methods

##### Descriptor

Deprecated: Use TransactionResponse.ProtoReflect.Descriptor instead.

##### GetTransaction

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TransferRequest

TransferRequest is used to move funds between accounts

#### Methods

##### Descriptor

Deprecated: Use TransferRequest.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetDescription

##### GetFromUserId

##### GetToUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedFinanceServiceServer

UnimplementedFinanceServiceServer must be embedded to have forward compatible implementations.

#### Methods

##### Deposit

##### GetBalance

##### GetTransaction

##### ListTransactions

##### Transfer

##### Withdraw

### UnsafeFinanceServiceServer

UnsafeFinanceServiceServer may be embedded to opt out of forward compatibility for this service. Use
of this interface is not recommended, as added methods to FinanceServiceServer will result in
compilation errors.

### WithdrawRequest

WithdrawRequest is used to remove funds from a user's account

#### Methods

##### Descriptor

Deprecated: Use WithdrawRequest.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetDescription

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterFinanceServiceServer
