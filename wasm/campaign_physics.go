package main

import (
	"encoding/json"
	"fmt"
	"log"
)

// CampaignPhysicsManager manages physics rules and settings for different campaigns
type CampaignPhysicsManager struct {
	campaigns map[string]*CampaignPhysicsRules
	rules     map[string]*CampaignPhysicsRules
}

// NewCampaignPhysicsManager creates a new campaign physics manager
func NewCampaignPhysicsManager() *CampaignPhysicsManager {
	cpm := &CampaignPhysicsManager{
		campaigns: make(map[string]*CampaignPhysicsRules),
		rules:     make(map[string]*CampaignPhysicsRules),
	}

	// Initialize with default rules
	cpm.initializeDefaultRules()

	return cpm
}

// initializeDefaultRules initializes default physics rules
func (cpm *CampaignPhysicsManager) initializeDefaultRules() {
	// Default physics rules
	defaultRules := &CampaignPhysicsRules{
		CampaignID:  "default",
		Gravity:     Vector3{X: 0, Y: -9.81, Z: 0},
		PhysicsRate: 60.0,
		MaxEntities: 1000,
		WorldBounds: BoundingBox{
			Min: Vector3{X: -100, Y: -100, Z: -100},
			Max: Vector3{X: 100, Y: 100, Z: 100},
		},
		CollisionRules: make(map[string]CollisionRule),
		EntityTypes:    make(map[string]EntityType),
		LODDistances:   []float32{50.0, 100.0, 200.0, 500.0},
		ChunkSize:      50.0,
		MaxChunks:      100,
		Properties:     make(map[string]interface{}),
	}

	// Add default collision rules
	cpm.addDefaultCollisionRules(defaultRules)

	// Add default entity types
	cpm.addDefaultEntityTypes(defaultRules)

	cpm.rules["default"] = defaultRules
}

// addDefaultCollisionRules adds default collision rules
func (cpm *CampaignPhysicsManager) addDefaultCollisionRules(rules *CampaignPhysicsRules) {
	// Default collision rules
	collisionRules := map[string]CollisionRule{
		"default:default": {
			TypeA:       "default",
			TypeB:       "default",
			CanCollide:  true,
			Restitution: 0.8,
			Friction:    0.5,
			Force:       1.0,
			Properties:  make(map[string]interface{}),
		},
		"player:player": {
			TypeA:       "player",
			TypeB:       "player",
			CanCollide:  false,
			Restitution: 0.0,
			Friction:    0.0,
			Force:       0.0,
			Properties:  make(map[string]interface{}),
		},
		"player:environment": {
			TypeA:       "player",
			TypeB:       "environment",
			CanCollide:  true,
			Restitution: 0.0,
			Friction:    0.8,
			Force:       1.0,
			Properties:  make(map[string]interface{}),
		},
		"projectile:environment": {
			TypeA:       "projectile",
			TypeB:       "environment",
			CanCollide:  true,
			Restitution: 0.2,
			Friction:    0.3,
			Force:       2.0,
			Properties:  make(map[string]interface{}),
		},
	}

	for key, rule := range collisionRules {
		rules.CollisionRules[key] = rule
	}
}

// addDefaultEntityTypes adds default entity types
func (cpm *CampaignPhysicsManager) addDefaultEntityTypes(rules *CampaignPhysicsRules) {
	// Default entity types
	entityTypes := map[string]EntityType{
		"default": {
			Name:          "default",
			Shape:         "box",
			Mass:          1.0,
			Restitution:   0.8,
			Friction:      0.5,
			MaxVelocity:   50.0,
			MaxAngularVel: 10.0,
			LODLevels: []LODLevel{
				{Level: 0, Distance: 50.0, PolygonCount: 1000, TextureSize: 512, Compressed: false},
				{Level: 1, Distance: 100.0, PolygonCount: 500, TextureSize: 256, Compressed: false},
				{Level: 2, Distance: 200.0, PolygonCount: 250, TextureSize: 128, Compressed: true},
				{Level: 3, Distance: 500.0, PolygonCount: 100, TextureSize: 64, Compressed: true},
			},
			Properties: make(map[string]interface{}),
		},
		"player": {
			Name:          "player",
			Shape:         "capsule",
			Mass:          70.0,
			Restitution:   0.1,
			Friction:      0.8,
			MaxVelocity:   20.0,
			MaxAngularVel: 5.0,
			LODLevels: []LODLevel{
				{Level: 0, Distance: 50.0, PolygonCount: 2000, TextureSize: 1024, Compressed: false},
				{Level: 1, Distance: 100.0, PolygonCount: 1000, TextureSize: 512, Compressed: false},
				{Level: 2, Distance: 200.0, PolygonCount: 500, TextureSize: 256, Compressed: true},
				{Level: 3, Distance: 500.0, PolygonCount: 200, TextureSize: 128, Compressed: true},
			},
			Properties: make(map[string]interface{}),
		},
		"environment": {
			Name:          "environment",
			Shape:         "mesh",
			Mass:          0.0, // Static
			Restitution:   0.3,
			Friction:      0.8,
			MaxVelocity:   0.0,
			MaxAngularVel: 0.0,
			LODLevels: []LODLevel{
				{Level: 0, Distance: 50.0, PolygonCount: 5000, TextureSize: 1024, Compressed: false},
				{Level: 1, Distance: 100.0, PolygonCount: 2500, TextureSize: 512, Compressed: false},
				{Level: 2, Distance: 200.0, PolygonCount: 1000, TextureSize: 256, Compressed: true},
				{Level: 3, Distance: 500.0, PolygonCount: 500, TextureSize: 128, Compressed: true},
			},
			Properties: make(map[string]interface{}),
		},
		"projectile": {
			Name:          "projectile",
			Shape:         "sphere",
			Mass:          0.1,
			Restitution:   0.6,
			Friction:      0.2,
			MaxVelocity:   100.0,
			MaxAngularVel: 20.0,
			LODLevels: []LODLevel{
				{Level: 0, Distance: 50.0, PolygonCount: 200, TextureSize: 256, Compressed: false},
				{Level: 1, Distance: 100.0, PolygonCount: 100, TextureSize: 128, Compressed: false},
				{Level: 2, Distance: 200.0, PolygonCount: 50, TextureSize: 64, Compressed: true},
				{Level: 3, Distance: 500.0, PolygonCount: 20, TextureSize: 32, Compressed: true},
			},
			Properties: make(map[string]interface{}),
		},
	}

	for key, entityType := range entityTypes {
		rules.EntityTypes[key] = entityType
	}
}

// GetCampaignRules gets physics rules for a specific campaign
func (cpm *CampaignPhysicsManager) GetCampaignRules(campaignID string) *CampaignPhysicsRules {
	// Check if campaign-specific rules exist
	if rules, exists := cpm.campaigns[campaignID]; exists {
		return rules
	}

	// Return default rules
	return cpm.rules["default"]
}

// SetCampaignRules sets physics rules for a specific campaign
func (cpm *CampaignPhysicsManager) SetCampaignRules(campaignID string, rules *CampaignPhysicsRules) {
	cpm.campaigns[campaignID] = rules
	log.Printf("[CampaignPhysicsManager] Set rules for campaign %s", campaignID)
}

// CreateCampaignRules creates new physics rules for a campaign
func (cpm *CampaignPhysicsManager) CreateCampaignRules(campaignID string, config map[string]interface{}) *CampaignPhysicsRules {
	// Start with default rules
	rules := cpm.copyRules(cpm.rules["default"])
	rules.CampaignID = campaignID

	// Apply custom configuration
	cpm.applyConfiguration(rules, config)

	// Store rules
	cpm.campaigns[campaignID] = rules

	log.Printf("[CampaignPhysicsManager] Created rules for campaign %s", campaignID)
	return rules
}

// copyRules creates a deep copy of physics rules
func (cpm *CampaignPhysicsManager) copyRules(original *CampaignPhysicsRules) *CampaignPhysicsRules {
	// Create a deep copy (simplified implementation)
	rules := &CampaignPhysicsRules{
		CampaignID:     original.CampaignID,
		Gravity:        original.Gravity,
		PhysicsRate:    original.PhysicsRate,
		MaxEntities:    original.MaxEntities,
		WorldBounds:    original.WorldBounds,
		CollisionRules: make(map[string]CollisionRule),
		EntityTypes:    make(map[string]EntityType),
		LODDistances:   make([]float32, len(original.LODDistances)),
		ChunkSize:      original.ChunkSize,
		MaxChunks:      original.MaxChunks,
		Properties:     make(map[string]interface{}),
	}

	// Copy collision rules
	for key, rule := range original.CollisionRules {
		rules.CollisionRules[key] = rule
	}

	// Copy entity types
	for key, entityType := range original.EntityTypes {
		rules.EntityTypes[key] = entityType
	}

	// Copy LOD distances
	copy(rules.LODDistances, original.LODDistances)

	// Copy properties
	for key, value := range original.Properties {
		rules.Properties[key] = value
	}

	return rules
}

// applyConfiguration applies custom configuration to rules
func (cpm *CampaignPhysicsManager) applyConfiguration(rules *CampaignPhysicsRules, config map[string]interface{}) {
	// Apply gravity
	if gravity, ok := config["gravity"].([]interface{}); ok && len(gravity) == 3 {
		rules.Gravity = Vector3{
			X: float32(gravity[0].(float64)),
			Y: float32(gravity[1].(float64)),
			Z: float32(gravity[2].(float64)),
		}
	}

	// Apply physics rate
	if rate, ok := config["physics_rate"].(float64); ok {
		rules.PhysicsRate = float32(rate)
	}

	// Apply max entities
	if max, ok := config["max_entities"].(float64); ok {
		rules.MaxEntities = int(max)
	}

	// Apply world bounds
	if bounds, ok := config["world_bounds"].(map[string]interface{}); ok {
		if min, ok := bounds["min"].([]interface{}); ok && len(min) == 3 {
			rules.WorldBounds.Min = Vector3{
				X: float32(min[0].(float64)),
				Y: float32(min[1].(float64)),
				Z: float32(min[2].(float64)),
			}
		}
		if max, ok := bounds["max"].([]interface{}); ok && len(max) == 3 {
			rules.WorldBounds.Max = Vector3{
				X: float32(max[0].(float64)),
				Y: float32(max[1].(float64)),
				Z: float32(max[2].(float64)),
			}
		}
	}

	// Apply LOD distances
	if lodDistances, ok := config["lod_distances"].([]interface{}); ok {
		rules.LODDistances = make([]float32, len(lodDistances))
		for i, dist := range lodDistances {
			rules.LODDistances[i] = float32(dist.(float64))
		}
	}

	// Apply chunk size
	if chunkSize, ok := config["chunk_size"].(float64); ok {
		rules.ChunkSize = float32(chunkSize)
	}

	// Apply max chunks
	if maxChunks, ok := config["max_chunks"].(float64); ok {
		rules.MaxChunks = int(maxChunks)
	}

	// Apply custom properties
	if properties, ok := config["properties"].(map[string]interface{}); ok {
		for key, value := range properties {
			rules.Properties[key] = value
		}
	}
}

// GetEntityType gets entity type configuration
func (cpm *CampaignPhysicsManager) GetEntityType(campaignID, entityType string) *EntityType {
	rules := cpm.GetCampaignRules(campaignID)
	if entityType, exists := rules.EntityTypes[entityType]; exists {
		return &entityType
	}
	return nil
}

// GetCollisionRule gets collision rule between two entity types
func (cpm *CampaignPhysicsManager) GetCollisionRule(campaignID, typeA, typeB string) *CollisionRule {
	rules := cpm.GetCampaignRules(campaignID)

	// Try direct lookup
	key := typeA + ":" + typeB
	if rule, exists := rules.CollisionRules[key]; exists {
		return &rule
	}

	// Try reverse lookup
	key = typeB + ":" + typeA
	if rule, exists := rules.CollisionRules[key]; exists {
		return &rule
	}

	return nil
}

// AddCollisionRule adds a collision rule
func (cpm *CampaignPhysicsManager) AddCollisionRule(campaignID, typeA, typeB string, rule CollisionRule) {
	rules := cpm.GetCampaignRules(campaignID)
	key := typeA + ":" + typeB
	rules.CollisionRules[key] = rule
	log.Printf("[CampaignPhysicsManager] Added collision rule %s for campaign %s", key, campaignID)
}

// AddEntityType adds an entity type
func (cpm *CampaignPhysicsManager) AddEntityType(campaignID, typeName string, entityType EntityType) {
	rules := cpm.GetCampaignRules(campaignID)
	rules.EntityTypes[typeName] = entityType
	log.Printf("[CampaignPhysicsManager] Added entity type %s for campaign %s", typeName, campaignID)
}

// ValidateRules validates physics rules
func (cpm *CampaignPhysicsManager) ValidateRules(rules *CampaignPhysicsRules) error {
	// Validate gravity
	if rules.Gravity.X < -100 || rules.Gravity.X > 100 ||
		rules.Gravity.Y < -100 || rules.Gravity.Y > 100 ||
		rules.Gravity.Z < -100 || rules.Gravity.Z > 100 {
		return fmt.Errorf("invalid gravity values")
	}

	// Validate physics rate
	if rules.PhysicsRate < 1 || rules.PhysicsRate > 120 {
		return fmt.Errorf("invalid physics rate: %f", rules.PhysicsRate)
	}

	// Validate max entities
	if rules.MaxEntities < 1 || rules.MaxEntities > 10000 {
		return fmt.Errorf("invalid max entities: %d", rules.MaxEntities)
	}

	// Validate world bounds
	if rules.WorldBounds.Min.X >= rules.WorldBounds.Max.X ||
		rules.WorldBounds.Min.Y >= rules.WorldBounds.Max.Y ||
		rules.WorldBounds.Min.Z >= rules.WorldBounds.Max.Z {
		return fmt.Errorf("invalid world bounds")
	}

	// Validate LOD distances
	if len(rules.LODDistances) != 4 {
		return fmt.Errorf("invalid LOD distances count")
	}

	for i, dist := range rules.LODDistances {
		if dist <= 0 || (i > 0 && dist <= rules.LODDistances[i-1]) {
			return fmt.Errorf("invalid LOD distance at index %d: %f", i, dist)
		}
	}

	return nil
}

// ExportRules exports rules to JSON
func (cpm *CampaignPhysicsManager) ExportRules(campaignID string) ([]byte, error) {
	rules := cpm.GetCampaignRules(campaignID)
	return json.MarshalIndent(rules, "", "  ")
}

// ImportRules imports rules from JSON
func (cpm *CampaignPhysicsManager) ImportRules(campaignID string, data []byte) error {
	var rules CampaignPhysicsRules
	if err := json.Unmarshal(data, &rules); err != nil {
		return err
	}

	// Validate rules
	if err := cpm.ValidateRules(&rules); err != nil {
		return err
	}

	// Set rules
	cpm.campaigns[campaignID] = &rules
	log.Printf("[CampaignPhysicsManager] Imported rules for campaign %s", campaignID)

	return nil
}

// GetCampaignList returns list of all campaigns with rules
func (cpm *CampaignPhysicsManager) GetCampaignList() []string {
	var campaigns []string
	for campaignID := range cpm.campaigns {
		campaigns = append(campaigns, campaignID)
	}
	return campaigns
}

// DeleteCampaignRules deletes rules for a campaign
func (cpm *CampaignPhysicsManager) DeleteCampaignRules(campaignID string) {
	delete(cpm.campaigns, campaignID)
	log.Printf("[CampaignPhysicsManager] Deleted rules for campaign %s", campaignID)
}

// GetRulesSummary returns a summary of rules for a campaign
func (cpm *CampaignPhysicsManager) GetRulesSummary(campaignID string) map[string]interface{} {
	rules := cpm.GetCampaignRules(campaignID)

	return map[string]interface{}{
		"campaign_id":     rules.CampaignID,
		"gravity":         rules.Gravity,
		"physics_rate":    rules.PhysicsRate,
		"max_entities":    rules.MaxEntities,
		"world_bounds":    rules.WorldBounds,
		"collision_rules": len(rules.CollisionRules),
		"entity_types":    len(rules.EntityTypes),
		"lod_distances":   rules.LODDistances,
		"chunk_size":      rules.ChunkSize,
		"max_chunks":      rules.MaxChunks,
		"properties":      rules.Properties,
	}
}
