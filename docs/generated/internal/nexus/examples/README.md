# Package examples

## Constants

### RelationTypeLinked

RelationType and EntityType stubs.

## Types

### OperationStep

### PatternCategory

### PatternExecutionManager

PatternExecutionManager demonstrates how to use the pattern store and executor.

#### Methods

##### ExecuteUserPattern

ExecuteUserPattern demonstrates executing a user-defined pattern.

##### ListSystemPatterns

ListSystemPatterns demonstrates listing system patterns by category.

##### ListUserPatterns

ListUserPatterns demonstrates listing patterns by user.

### PatternExecutor

#### Methods

##### ExecutePattern

### PatternManager

PatternManager demonstrates how to use the pattern executor.

#### Methods

##### ExecuteTransaction

ExecuteTransaction demonstrates how to execute the transaction pattern.

##### ExecuteUserOnboarding

ExecuteUserOnboarding demonstrates how to execute the user onboarding pattern.

### PatternOrigin

--- Minimal type definitions for examples ---.

### PatternStore

#### Methods

##### GetPattern

##### ListPatterns

##### StorePattern

##### UpdatePatternStats

### StoredPattern

### TransactionWithMetadata

TransactionWithMetadata combines transaction data with relationship metadata.

### UserFinanceManager

UserFinanceManager demonstrates how to use Nexus for user-finance relationships.

#### Methods

##### CreateUserWithWallet

CreateUserWithWallet demonstrates creating a user with an associated wallet.

##### GetUserFinancialGraph

GetUserFinancialGraph demonstrates getting a complete financial relationship graph.

##### GetUserTransactions

TODO: Implement logic to get user transactions (fetch master record, call
financeRepo.ListTransactions, map to response). Use userID and limit in the real implementation.

##### TransferBetweenUsers

TransferBetweenUsers demonstrates a complex operation using Nexus.

## Functions

### CreateTransactionPattern

Example of creating and using a pattern for financial transaction.

### CreateUserOnboardingPattern

Example of creating and using a pattern for user onboarding.

### ExamplePatternUsage

Example usage of creating and executing patterns.
