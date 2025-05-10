# Package financepb

## Constants

### FinanceService_GetBalance_FullMethodName

## Variables

### File_api_protos_finance_v1_finance_proto

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

### DepositResponse

TODO: Fill in the fields for this message based on the RPC's purpose. Typically, this will wrap the
relevant entity (e.g., Transaction) or include necessary parameters.

#### Methods

##### Descriptor

Deprecated: Use DepositResponse.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### FinanceServiceClient

FinanceServiceClient is the client API for FinanceService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

FinanceService provides methods for handling financial operations

### FinanceServiceServer

FinanceServiceServer is the server API for FinanceService service. All implementations must embed
UnimplementedFinanceServiceServer for forward compatibility.

FinanceService provides methods for handling financial operations

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

### GetTransactionResponse

TODO: Fill in the fields for this message based on the RPC's purpose. Typically, this will wrap the
relevant entity (e.g., Transaction) or include necessary parameters.

#### Methods

##### Descriptor

Deprecated: Use GetTransactionResponse.ProtoReflect.Descriptor instead.

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

### TransferResponse

TODO: Fill in the fields for this message based on the RPC's purpose. Typically, this will wrap the
relevant entity (e.g., Transaction) or include necessary parameters.

#### Methods

##### Descriptor

Deprecated: Use TransferResponse.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedFinanceServiceServer

UnimplementedFinanceServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

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

### WithdrawResponse

TODO: Fill in the fields for this message based on the RPC's purpose. Typically, this will wrap the
relevant entity (e.g., Transaction) or include necessary parameters.

#### Methods

##### Descriptor

Deprecated: Use WithdrawResponse.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterFinanceServiceServer
