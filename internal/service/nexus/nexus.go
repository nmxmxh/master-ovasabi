package nexus

import (
	"context"
	"fmt"
	"strings"

	"github.com/nmxmxh/master-ovasabi/amadeus/nexus/pattern"
	"github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"
	assetpb "github.com/nmxmxh/master-ovasabi/api/protos/asset/v1"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// Service implements the Nexus service
type Service struct {
	nexuspb.UnimplementedNexusServiceServer
	logger        *zap.Logger
	cache         *redis.Cache
	kgPattern     *pattern.KnowledgeGraphPattern
	userPattern   *pattern.UserCreationPattern
	userService   userpb.UserServiceServer
	assetService  assetpb.AssetServiceServer
	notifyService notificationpb.NotificationServiceServer
}

// NewService creates a new Nexus service instance
func NewService(
	logger *zap.Logger,
	cache *redis.Cache,
	userSvc userpb.UserServiceServer,
	assetSvc assetpb.AssetServiceServer,
	notifySvc notificationpb.NotificationServiceServer,
) *Service {
	return &Service{
		logger:        logger,
		cache:         cache,
		kgPattern:     pattern.NewKnowledgeGraphPattern(),
		userPattern:   pattern.NewUserCreationPattern(userSvc, assetSvc, notifySvc),
		userService:   userSvc,
		assetService:  assetSvc,
		notifyService: notifySvc,
	}
}

// PatternDefinition for in-memory storage
// (You may want to persist this in the knowledge graph or DB)
type PatternDefinition struct {
	PatternName string
	PatternType string
	Steps       []*nexuspb.PatternStep
}

// Add a map to store registered patterns
var patternRegistry = make(map[string]PatternDefinition)

// RegisterPattern registers a new pattern with steps
func (s *Service) RegisterPattern(ctx context.Context, req *nexuspb.RegisterPatternRequest) (*nexuspb.RegisterPatternResponse, error) {
	s.logger.Info("Registering pattern (structured)",
		zap.String("pattern_name", req.PatternName),
		zap.String("pattern_type", req.PatternType),
	)

	patternRegistry[req.PatternName] = PatternDefinition{
		PatternName: req.PatternName,
		PatternType: req.PatternType,
		Steps:       req.Steps,
	}

	return &nexuspb.RegisterPatternResponse{
		Status:  "success",
		Message: "Pattern registered successfully",
	}, nil
}

// ExecutePattern validates provided arguments against required config keys
func (s *Service) ExecutePattern(ctx context.Context, req *nexuspb.ExecutePatternRequest) (*nexuspb.ExecutePatternResponse, error) {
	pattern, ok := patternRegistry[req.PatternName]
	if !ok {
		return &nexuspb.ExecutePatternResponse{
			Status: "pattern_not_found",
			Result: map[string]string{"error": "Pattern not found"},
		}, nil
	}

	// Aggregate all required arguments from all steps
	required := map[string]bool{}
	for _, step := range pattern.Steps {
		for _, arg := range step.RequiredArgs {
			required[arg] = true
		}
	}

	// Check which arguments are missing
	missing := []string{}
	for arg := range required {
		if _, ok := req.Parameters[arg]; !ok {
			missing = append(missing, arg)
		}
	}

	if len(missing) > 0 {
		return &nexuspb.ExecutePatternResponse{
			Status: "missing_arguments",
			Result: map[string]string{
				"error":   "Missing required arguments",
				"missing": strings.Join(missing, ", "),
			},
		}, nil
	}

	// ... Proceed with actual pattern execution logic ...
	// For now, just return success and echo the parameters
	return &nexuspb.ExecutePatternResponse{
		Status: "success",
		Result: req.Parameters,
	}, nil
}

// GetKnowledge retrieves knowledge from the graph
func (s *Service) GetKnowledge(ctx context.Context, req *nexuspb.GetKnowledgeRequest) (*nexuspb.GetKnowledgeResponse, error) {
	s.logger.Info("Getting knowledge",
		zap.String("path", req.Path))

	// Get the knowledge graph
	kg := kg.DefaultKnowledgeGraph()

	// Get knowledge from the graph
	node, err := kg.GetNode(req.Path)
	if err != nil {
		s.logger.Error("Failed to get knowledge",
			zap.String("path", req.Path),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get knowledge: %w", err)
	}

	// Convert node data to string map
	data := make(map[string]string)
	if nodeMap, ok := node.(map[string]interface{}); ok {
		for k, v := range nodeMap {
			data[k] = fmt.Sprintf("%v", v)
		}
	} else {
		data["value"] = fmt.Sprintf("%v", node)
	}

	return &nexuspb.GetKnowledgeResponse{
		Status: "success",
		Data:   data,
	}, nil
}
