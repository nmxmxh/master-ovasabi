package main

import (
	"encoding/json"
	"math"
	"time"
)

// PhysicsEventType defines all possible physics event types
type PhysicsEventType string

const (
	// Entity Events
	PhysicsEntitySpawn     PhysicsEventType = "physics:entity:spawn"
	PhysicsEntityUpdate    PhysicsEventType = "physics:entity:update"
	PhysicsEntityDestroy   PhysicsEventType = "physics:entity:destroy"
	PhysicsEntityCollision PhysicsEventType = "physics:entity:collision"

	// Environment Events
	PhysicsEnvironmentUpdate PhysicsEventType = "physics:environment:update"
	PhysicsEnvironmentChunk  PhysicsEventType = "physics:environment:chunk"

	// Proximity Events
	PhysicsProximityEnter  PhysicsEventType = "physics:proximity:enter"
	PhysicsProximityExit   PhysicsEventType = "physics:proximity:exit"
	PhysicsProximityUpdate PhysicsEventType = "physics:proximity:update"

	// Campaign Events
	PhysicsCampaignJoin  PhysicsEventType = "physics:campaign:join"
	PhysicsCampaignLeave PhysicsEventType = "physics:campaign:leave"
	PhysicsCampaignRules PhysicsEventType = "physics:campaign:rules"

	// System Events
	PhysicsSystemStart  PhysicsEventType = "physics:system:start"
	PhysicsSystemStop   PhysicsEventType = "physics:system:stop"
	PhysicsSystemPause  PhysicsEventType = "physics:system:pause"
	PhysicsSystemResume PhysicsEventType = "physics:system:resume"
)

// Vector3 represents a 3D vector
type Vector3 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

// Quaternion represents a 3D rotation
type Quaternion struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
	W float32 `json:"w"`
}

// BoundingBox represents a 3D bounding box
type BoundingBox struct {
	Min Vector3 `json:"min"`
	Max Vector3 `json:"max"`
}

// PhysicsEntity represents a physics object in the world
type PhysicsEntity struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Position    Vector3                `json:"position"`
	Rotation    Quaternion             `json:"rotation"`
	Velocity    Vector3                `json:"velocity"`
	AngularVel  Vector3                `json:"angular_velocity"`
	Scale       Vector3                `json:"scale"`
	Mass        float32                `json:"mass"`
	Restitution float32                `json:"restitution"`
	Friction    float32                `json:"friction"`
	Shape       string                 `json:"shape"` // "sphere", "box", "plane", "mesh"
	MeshData    map[string]interface{} `json:"mesh_data,omitempty"`
	Properties  map[string]interface{} `json:"properties"`
	Bounds      BoundingBox            `json:"bounds"`
	LOD         int                    `json:"lod"` // Level of Detail
	Active      bool                   `json:"active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// PhysicsEvent represents a physics event in the system
type PhysicsEvent struct {
	Type       PhysicsEventType       `json:"type"`
	CampaignID string                 `json:"campaign_id"`
	EntityID   string                 `json:"entity_id,omitempty"`
	Position   Vector3                `json:"position,omitempty"`
	Rotation   Quaternion             `json:"rotation,omitempty"`
	Velocity   Vector3                `json:"velocity,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	FrameID    uint64                 `json:"frame_id"`
	Priority   int                    `json:"priority"` // 0=low, 1=normal, 2=high, 3=critical
	Source     string                 `json:"source"`   // "godot", "wasm", "frontend", "backend"
}

// PhysicsCollision represents a collision between entities
type PhysicsCollision struct {
	EntityA    string                 `json:"entity_a"`
	EntityB    string                 `json:"entity_b"`
	Position   Vector3                `json:"position"`
	Normal     Vector3                `json:"normal"`
	Force      float32                `json:"force"`
	Properties map[string]interface{} `json:"properties"`
	Timestamp  time.Time              `json:"timestamp"`
}

// EnvironmentChunk represents a chunk of environment data
type EnvironmentChunk struct {
	ID           string                 `json:"id"`
	CampaignID   string                 `json:"campaign_id"`
	Position     Vector3                `json:"position"`
	Bounds       BoundingBox            `json:"bounds"`
	LOD          int                    `json:"lod"`
	Data         []byte                 `json:"data"`
	Dependencies []string               `json:"dependencies"`
	Size         int64                  `json:"size"`
	Compressed   bool                   `json:"compressed"`
	Checksum     string                 `json:"checksum"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Properties   map[string]interface{} `json:"properties"`
}

// ProximityEvent represents a proximity-based event
type ProximityEvent struct {
	EntityID  string    `json:"entity_id"`
	TargetID  string    `json:"target_id"`
	Distance  float32   `json:"distance"`
	Position  Vector3   `json:"position"`
	TargetPos Vector3   `json:"target_position"`
	EventType string    `json:"event_type"` // "enter", "exit", "update"
	Timestamp time.Time `json:"timestamp"`
}

// CampaignPhysicsRules represents physics rules for a campaign
type CampaignPhysicsRules struct {
	CampaignID     string                   `json:"campaign_id"`
	Gravity        Vector3                  `json:"gravity"`
	PhysicsRate    float32                  `json:"physics_rate"`
	MaxEntities    int                      `json:"max_entities"`
	WorldBounds    BoundingBox              `json:"world_bounds"`
	CollisionRules map[string]CollisionRule `json:"collision_rules"`
	EntityTypes    map[string]EntityType    `json:"entity_types"`
	LODDistances   []float32                `json:"lod_distances"`
	ChunkSize      float32                  `json:"chunk_size"`
	MaxChunks      int                      `json:"max_chunks"`
	Properties     map[string]interface{}   `json:"properties"`
}

// CollisionRule defines collision behavior between entity types
type CollisionRule struct {
	TypeA       string                 `json:"type_a"`
	TypeB       string                 `json:"type_b"`
	CanCollide  bool                   `json:"can_collide"`
	Restitution float32                `json:"restitution"`
	Friction    float32                `json:"friction"`
	Force       float32                `json:"force"`
	Properties  map[string]interface{} `json:"properties"`
}

// EntityType defines properties for different entity types
type EntityType struct {
	Name          string                 `json:"name"`
	Shape         string                 `json:"shape"`
	Mass          float32                `json:"mass"`
	Restitution   float32                `json:"restitution"`
	Friction      float32                `json:"friction"`
	MaxVelocity   float32                `json:"max_velocity"`
	MaxAngularVel float32                `json:"max_angular_velocity"`
	LODLevels     []LODLevel             `json:"lod_levels"`
	Properties    map[string]interface{} `json:"properties"`
}

// LODLevel defines level of detail for an entity type
type LODLevel struct {
	Level        int                    `json:"level"`
	Distance     float32                `json:"distance"`
	PolygonCount int                    `json:"polygon_count"`
	TextureSize  int                    `json:"texture_size"`
	Compressed   bool                   `json:"compressed"`
	Properties   map[string]interface{} `json:"properties"`
}

// PhysicsEventBatch represents a batch of physics events
type PhysicsEventBatch struct {
	Events     []PhysicsEvent `json:"events"`
	CampaignID string         `json:"campaign_id"`
	FrameID    uint64         `json:"frame_id"`
	Timestamp  time.Time      `json:"timestamp"`
	Source     string         `json:"source"`
}

// NewPhysicsEvent creates a new physics event
func NewPhysicsEvent(eventType PhysicsEventType, campaignID string, frameID uint64) *PhysicsEvent {
	return &PhysicsEvent{
		Type:       eventType,
		CampaignID: campaignID,
		FrameID:    frameID,
		Timestamp:  time.Now(),
		Priority:   1, // Normal priority
		Source:     "wasm",
		Properties: make(map[string]interface{}),
		Data:       make(map[string]interface{}),
	}
}

// NewPhysicsEntity creates a new physics entity
func NewPhysicsEntity(id, entityType string, position Vector3) *PhysicsEntity {
	now := time.Now()
	return &PhysicsEntity{
		ID:          id,
		Type:        entityType,
		Position:    position,
		Rotation:    Quaternion{W: 1.0}, // Identity quaternion
		Velocity:    Vector3{},
		AngularVel:  Vector3{},
		Scale:       Vector3{X: 1.0, Y: 1.0, Z: 1.0},
		Mass:        1.0,
		Restitution: 0.8,
		Friction:    0.5,
		Shape:       "sphere",
		Properties:  make(map[string]interface{}),
		Bounds:      BoundingBox{Min: position, Max: position},
		LOD:         0,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// ToJSON converts a physics event to JSON
func (pe *PhysicsEvent) ToJSON() ([]byte, error) {
	return json.Marshal(pe)
}

// FromJSON creates a physics event from JSON
func (pe *PhysicsEvent) FromJSON(data []byte) error {
	return json.Unmarshal(data, pe)
}

// ToJSON converts a physics entity to JSON
func (pe *PhysicsEntity) ToJSON() ([]byte, error) {
	return json.Marshal(pe)
}

// FromJSON creates a physics entity from JSON
func (pe *PhysicsEntity) FromJSON(data []byte) error {
	return json.Unmarshal(data, pe)
}

// UpdatePosition updates the entity position and bounds
func (pe *PhysicsEntity) UpdatePosition(position Vector3) {
	pe.Position = position
	pe.Bounds = BoundingBox{
		Min: Vector3{
			X: position.X - pe.Scale.X/2,
			Y: position.Y - pe.Scale.Y/2,
			Z: position.Z - pe.Scale.Z/2,
		},
		Max: Vector3{
			X: position.X + pe.Scale.X/2,
			Y: position.Y + pe.Scale.Y/2,
			Z: position.Z + pe.Scale.Z/2,
		},
	}
	pe.UpdatedAt = time.Now()
}

// UpdateVelocity updates the entity velocity
func (pe *PhysicsEntity) UpdateVelocity(velocity Vector3) {
	pe.Velocity = velocity
	pe.UpdatedAt = time.Now()
}

// UpdateRotation updates the entity rotation
func (pe *PhysicsEntity) UpdateRotation(rotation Quaternion) {
	pe.Rotation = rotation
	pe.UpdatedAt = time.Now()
}

// IsNear checks if this entity is near another entity
func (pe *PhysicsEntity) IsNear(other *PhysicsEntity, distance float32) bool {
	dx := pe.Position.X - other.Position.X
	dy := pe.Position.Y - other.Position.Y
	dz := pe.Position.Z - other.Position.Z
	dist := dx*dx + dy*dy + dz*dz
	return dist <= distance*distance
}

// GetDistance calculates distance to another entity
func (pe *PhysicsEntity) GetDistance(other *PhysicsEntity) float32 {
	dx := pe.Position.X - other.Position.X
	dy := pe.Position.Y - other.Position.Y
	dz := pe.Position.Z - other.Position.Z
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}
