package nexus

import (
	"context"
	"fmt"

	"github.com/nmxmxh/master-ovasabi/amadeus/nexus/pattern"
	"github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"
	assetpb "github.com/nmxmxh/master-ovasabi/api/protos/asset/v0"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v0"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v0"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v0"
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

// ExecutePattern executes a registered pattern
func (s *Service) ExecutePattern(ctx context.Context, req *nexuspb.ExecutePatternRequest) (*nexuspb.ExecutePatternResponse, error) {
	s.logger.Info("Executing pattern",
		zap.String("pattern_name", req.PatternName))

	// Convert parameters to the expected format
	params := make(map[string]interface{})
	for k, v := range req.Parameters {
		params[k] = v
	}

	var result map[string]interface{}
	var err error

	// Execute the appropriate pattern
	switch req.PatternName {
	case "user_creation":
		result, err = s.userPattern.Execute(ctx, params)
	case "knowledge_graph":
		result, err = s.kgPattern.Execute(ctx, params)
	default:
		return nil, fmt.Errorf("unknown pattern: %s", req.PatternName)
	}

	if err != nil {
		s.logger.Error("Failed to execute pattern",
			zap.String("pattern_name", req.PatternName),
			zap.Error(err))
		return nil, fmt.Errorf("failed to execute pattern: %w", err)
	}

	// Convert result to response format
	response := &nexuspb.ExecutePatternResponse{
		Status: "success",
		Result: make(map[string]string),
	}
	for k, v := range result {
		response.Result[k] = fmt.Sprintf("%v", v)
	}

	return response, nil
}

// RegisterPattern registers a new pattern
func (s *Service) RegisterPattern(ctx context.Context, req *nexuspb.RegisterPatternRequest) (*nexuspb.RegisterPatternResponse, error) {
	s.logger.Info("Registering pattern",
		zap.String("pattern_name", req.PatternName),
		zap.String("pattern_type", req.PatternType))

	// Get the knowledge graph
	kg := kg.DefaultKnowledgeGraph()

	// Add pattern to knowledge graph
	patternInfo := make(map[string]interface{})
	for k, v := range req.PatternConfig {
		patternInfo[k] = v
	}

	err := kg.AddPattern(req.PatternType, req.PatternName, patternInfo)
	if err != nil {
		s.logger.Error("Failed to register pattern",
			zap.String("pattern_name", req.PatternName),
			zap.Error(err))
		return nil, fmt.Errorf("failed to register pattern: %w", err)
	}

	return &nexuspb.RegisterPatternResponse{
		Status:  "success",
		Message: fmt.Sprintf("Pattern '%s' registered successfully", req.PatternName),
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
