package examples

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/internal/nexus"
	baseRepo "github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/repository/commerce"
	userRepo "github.com/nmxmxh/master-ovasabi/internal/repository/user"
	"github.com/nmxmxh/master-ovasabi/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

var logInstance logger.Logger

func init() {
	var err error
	logInstance, err = logger.NewDefault()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
}

// UserFinanceManager demonstrates how to use Nexus for user-finance relationships.
type UserFinanceManager struct {
	nexusRepo   nexus.Repository
	userRepo    *userRepo.UserRepository
	financeRepo commerce.Repository
	masterRepo  baseRepo.MasterRepository
}

// TransactionWithMetadata combines transaction data with relationship metadata.
type TransactionWithMetadata struct {
	*commerce.Transaction
	RelationshipMetadata map[string]interface{}
}

// CreateUserWithWallet demonstrates creating a user with an associated wallet.
func (m *UserFinanceManager) CreateUserWithWallet(ctx context.Context, username, email string) error {
	// 1. Create user
	newUser := &userRepo.User{
		Username: username,
		Email:    email,
	}
	createdUser, err := m.userRepo.Create(ctx, newUser)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// 2. Create wallet master record
	walletMasterID, err := m.masterRepo.Create(ctx, baseRepo.EntityTypeFinance,
		fmt.Sprintf("wallet_%s", username))
	if err != nil {
		return fmt.Errorf("failed to create wallet master: %w", err)
	}

	// 3. Create relationship between user and wallet
	metadata := map[string]interface{}{
		"wallet_type": "primary",
		"created_at":  time.Now(),
		"currency":    "USD",
	}
	masterID, err := strconv.ParseInt(createdUser.MasterID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid master_id: %w", err)
	}
	_, err = m.nexusRepo.CreateRelationship(ctx,
		masterID,
		walletMasterID,
		nexus.RelationTypeOwner,
		metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create relationship: %w", err)
	}

	return nil
}

// TODO: Implement logic to get user transactions (fetch master record, call financeRepo.ListTransactions, map to response). Use userID and limit in the real implementation.
func (m *UserFinanceManager) GetUserTransactions(_ context.Context, _ uuid.UUID, _ int) ([]*TransactionWithMetadata, error) {
	// Pseudocode:
	// 1. Get user's master record
	// 2. Call financeRepo.ListTransactions with userID
	// 3. Map transactions to TransactionWithMetadata
	// 4. Return result
	return nil, nil // TODO: Replace with real implementation
}

// TransferBetweenUsers demonstrates a complex operation using Nexus.
func (m *UserFinanceManager) TransferBetweenUsers(ctx context.Context, fromUserID, toUserID uuid.UUID, amount float64) error {
	// 1. Get relationship graph for both users
	fromMaster, err := m.masterRepo.GetByUUID(ctx, fromUserID)
	if err != nil {
		return fmt.Errorf("failed to get from user master: %w", err)
	}

	toMaster, err := m.masterRepo.GetByUUID(ctx, toUserID)
	if err != nil {
		return fmt.Errorf("failed to get to user master: %w", err)
	}

	fromGraph, err := m.nexusRepo.GetEntityGraph(ctx, fromMaster.ID, 2)
	if err != nil {
		return fmt.Errorf("failed to get from user graph: %w", err)
	}

	toGraph, err := m.nexusRepo.GetEntityGraph(ctx, toMaster.ID, 2)
	if err != nil {
		return fmt.Errorf("failed to get to user graph: %w", err)
	}

	// 2. Create transfer transaction
	// Build service-specific metadata for commerce
	serviceSpecificMap := map[string]interface{}{
		"commerce": map[string]interface{}{
			"to_user_id": toUserID.String(),
		},
	}
	serviceSpecificStruct, err := structpb.NewStruct(serviceSpecificMap)
	if err != nil {
		logInstance.Warn("failed to create structpb for serviceSpecificStruct", zap.Error(err))
		return err
	}
	tx := &commerce.Transaction{
		TransactionID: uuid.New().String(),
		UserID:        fromUserID.String(),
		Type:          "transfer",
		Amount:        amount,
		Currency:      "USD",
		Status:        "pending",
		Metadata: &commonpb.Metadata{
			ServiceSpecific: serviceSpecificStruct,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 3. Create transaction record
	if err := m.financeRepo.CreateTransaction(ctx, tx); err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// 4. Create relationships for the transaction
	metadata := map[string]interface{}{
		"transfer_type": "user_to_user",
		"timestamp":     time.Now(),
	}

	// Get transaction master record
	txMaster, err := m.masterRepo.Create(ctx, baseRepo.EntityTypeFinance, tx.TransactionID)
	if err != nil {
		return fmt.Errorf("failed to create transaction master: %w", err)
	}

	// Link transaction to both users
	for _, userGraph := range []*nexus.Graph{fromGraph, toGraph} {
		for _, node := range userGraph.Nodes {
			if node.Type == baseRepo.EntityTypeUser {
				_, err = m.nexusRepo.CreateRelationship(ctx,
					node.ID,
					txMaster,
					nexus.RelationTypeLinked,
					metadata,
				)
				if err != nil {
					return fmt.Errorf("failed to create relationship: %w", err)
				}
			}
		}
	}

	// 5. Publish event for processing
	event := &nexus.Event{
		ID:         uuid.New(),
		MasterID:   txMaster,
		EntityType: baseRepo.EntityTypeFinance,
		EventType:  "transfer_created",
		Payload: map[string]interface{}{
			"from_user_id": fromUserID.String(),
			"to_user_id":   toUserID.String(),
			"amount":       amount,
		},
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	if err := m.nexusRepo.PublishEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// GetUserFinancialGraph demonstrates getting a complete financial relationship graph.
func (m *UserFinanceManager) GetUserFinancialGraph(ctx context.Context, userID uuid.UUID) (*nexus.Graph, error) {
	// Get user's master record first
	userMaster, err := m.masterRepo.GetByUUID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user master: %w", err)
	}

	// Get complete graph with depth 3 (user -> wallet -> transactions -> related users)
	graph, err := m.nexusRepo.GetEntityGraph(ctx, userMaster.ID, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity graph: %w", err)
	}

	return graph, nil
}
