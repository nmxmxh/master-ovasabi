package main

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// PhysicsProcessor handles physics event processing and entity management
type PhysicsProcessor struct {
	entities      map[string]*PhysicsEntity
	entitiesMutex sync.RWMutex
	eventQueue    chan PhysicsEvent
	proximityGrid *ProximityGrid
	campaignID    string
	rules         *CampaignPhysicsRules
	active        bool
	stopChan      chan bool
	workerCount   int
	lastFrameID   uint64
	performance   *PhysicsPerformance
}

// ProximityGrid manages spatial partitioning for efficient proximity queries
type ProximityGrid struct {
	cellSize    float32
	grid        map[string][]string // cellID -> entityIDs
	gridMutex   sync.RWMutex
	worldBounds BoundingBox
}

// PhysicsPerformance tracks physics system performance
type PhysicsPerformance struct {
	FPS            float32   `json:"fps"`
	FrameTime      float32   `json:"frame_time"`
	EntityCount    int       `json:"entity_count"`
	CollisionCount int       `json:"collision_count"`
	EventCount     int       `json:"event_count"`
	MemoryUsage    int64     `json:"memory_usage"`
	LastUpdate     time.Time `json:"last_update"`
}

// NewPhysicsProcessor creates a new physics processor
func NewPhysicsProcessor(campaignID string, rules *CampaignPhysicsRules) *PhysicsProcessor {
	pp := &PhysicsProcessor{
		entities:      make(map[string]*PhysicsEntity),
		eventQueue:    make(chan PhysicsEvent, 1000),
		proximityGrid: NewProximityGrid(rules.ChunkSize, rules.WorldBounds),
		campaignID:    campaignID,
		rules:         rules,
		active:        false,
		stopChan:      make(chan bool),
		workerCount:   4,
		performance:   &PhysicsPerformance{},
	}

	// Start processing workers
	for i := 0; i < pp.workerCount; i++ {
		go pp.processWorker(i)
	}

	return pp
}

// NewProximityGrid creates a new proximity grid
func NewProximityGrid(cellSize float32, worldBounds BoundingBox) *ProximityGrid {
	return &ProximityGrid{
		cellSize:    cellSize,
		grid:        make(map[string][]string),
		worldBounds: worldBounds,
	}
}

// Start starts the physics processor
func (pp *PhysicsProcessor) Start() {
	pp.active = true
	log.Printf("[PhysicsProcessor] Started for campaign %s", pp.campaignID)
}

// Stop stops the physics processor
func (pp *PhysicsProcessor) Stop() {
	pp.active = false
	close(pp.stopChan)
	log.Printf("[PhysicsProcessor] Stopped for campaign %s", pp.campaignID)
}

// ProcessPhysicsEvent processes a single physics event
func (pp *PhysicsProcessor) ProcessPhysicsEvent(event PhysicsEvent) {
	if !pp.active {
		return
	}

	select {
	case pp.eventQueue <- event:
		// Event queued successfully
	default:
		log.Printf("[PhysicsProcessor] Event queue full, dropping event: %s", event.Type)
	}
}

// processWorker processes events from the queue
func (pp *PhysicsProcessor) processWorker(workerID int) {
	for {
		select {
		case event := <-pp.eventQueue:
			pp.handlePhysicsEvent(event)
		case <-pp.stopChan:
			return
		}
	}
}

// handlePhysicsEvent handles a single physics event
func (pp *PhysicsProcessor) handlePhysicsEvent(event PhysicsEvent) {
	start := time.Now()

	switch event.Type {
	case PhysicsEntitySpawn:
		pp.spawnEntity(event)
	case PhysicsEntityUpdate:
		pp.updateEntity(event)
	case PhysicsEntityDestroy:
		pp.destroyEntity(event)
	case PhysicsEntityCollision:
		pp.handleCollision(event)
	case PhysicsEnvironmentUpdate:
		pp.handleEnvironmentUpdate(event)
	case PhysicsProximityEnter, PhysicsProximityExit, PhysicsProximityUpdate:
		pp.handleProximityEvent(event)
	case PhysicsCampaignJoin:
		pp.handleCampaignJoin(event)
	case PhysicsCampaignLeave:
		pp.handleCampaignLeave(event)
	case PhysicsCampaignRules:
		pp.handleCampaignRules(event)
	default:
		log.Printf("[PhysicsProcessor] Unknown event type: %s", event.Type)
	}

	// Update performance metrics
	pp.updatePerformance(time.Since(start))
}

// spawnEntity spawns a new physics entity
func (pp *PhysicsProcessor) spawnEntity(event PhysicsEvent) {
	pp.entitiesMutex.Lock()
	defer pp.entitiesMutex.Unlock()

	// Create entity from event data
	entity := &PhysicsEntity{
		ID:         event.EntityID,
		Type:       event.Properties["type"].(string),
		Position:   event.Position,
		Rotation:   event.Rotation,
		Velocity:   event.Velocity,
		Properties: event.Properties,
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Set default values if not provided
	if entity.Mass == 0 {
		entity.Mass = 1.0
	}
	if entity.Restitution == 0 {
		entity.Restitution = 0.8
	}
	if entity.Friction == 0 {
		entity.Friction = 0.5
	}
	if entity.Shape == "" {
		entity.Shape = "sphere"
	}

	// Add to entities map
	pp.entities[event.EntityID] = entity

	// Add to proximity grid
	pp.proximityGrid.AddEntity(entity)

	// Send spawn event to frontend
	pp.broadcastEvent(event)

	log.Printf("[PhysicsProcessor] Spawned entity %s at %v", event.EntityID, event.Position)
}

// updateEntity updates an existing physics entity
func (pp *PhysicsProcessor) updateEntity(event PhysicsEvent) {
	pp.entitiesMutex.Lock()
	defer pp.entitiesMutex.Unlock()

	entity, exists := pp.entities[event.EntityID]
	if !exists {
		log.Printf("[PhysicsProcessor] Entity %s not found for update", event.EntityID)
		return
	}

	// Update position if provided
	if event.Position != (Vector3{}) {
		entity.UpdatePosition(event.Position)
	}

	// Update velocity if provided
	if event.Velocity != (Vector3{}) {
		entity.UpdateVelocity(event.Velocity)
	}

	// Update rotation if provided
	if event.Rotation != (Quaternion{}) {
		entity.UpdateRotation(event.Rotation)
	}

	// Update properties
	for key, value := range event.Properties {
		entity.Properties[key] = value
	}

	// Update proximity grid
	pp.proximityGrid.UpdateEntity(entity)

	// Send update event to frontend
	pp.broadcastEvent(event)
}

// destroyEntity destroys a physics entity
func (pp *PhysicsProcessor) destroyEntity(event PhysicsEvent) {
	pp.entitiesMutex.Lock()
	defer pp.entitiesMutex.Unlock()

	entity, exists := pp.entities[event.EntityID]
	if !exists {
		log.Printf("[PhysicsProcessor] Entity %s not found for destruction", event.EntityID)
		return
	}

	// Remove from proximity grid
	pp.proximityGrid.RemoveEntity(entity)

	// Remove from entities map
	delete(pp.entities, event.EntityID)

	// Send destroy event to frontend
	pp.broadcastEvent(event)

	log.Printf("[PhysicsProcessor] Destroyed entity %s", event.EntityID)
}

// handleCollision handles collision events
func (pp *PhysicsProcessor) handleCollision(event PhysicsEvent) {
	pp.entitiesMutex.RLock()
	defer pp.entitiesMutex.RUnlock()

	// Process collision data
	collisionData, ok := event.Data["collision"].(map[string]interface{})
	if !ok {
		return
	}

	entityA, existsA := pp.entities[collisionData["entity_a"].(string)]
	entityB, existsB := pp.entities[collisionData["entity_b"].(string)]

	if !existsA || !existsB {
		return
	}

	// Apply collision response based on rules
	pp.applyCollisionResponse(entityA, entityB, collisionData)

	// Update performance metrics
	pp.performance.CollisionCount++

	// Send collision event to frontend
	pp.broadcastEvent(event)
}

// applyCollisionResponse applies collision response between two entities
func (pp *PhysicsProcessor) applyCollisionResponse(entityA, entityB *PhysicsEntity, collisionData map[string]interface{}) {
	// Get collision rule for these entity types
	rule := pp.getCollisionRule(entityA.Type, entityB.Type)
	if rule == nil || !rule.CanCollide {
		return
	}

	// Calculate collision response
	// This is a simplified implementation
	// In a real physics engine, this would be much more complex

	// Apply restitution
	if rule.Restitution > 0 {
		// Apply restitution to velocities
		entityA.Velocity.X *= rule.Restitution
		entityA.Velocity.Y *= rule.Restitution
		entityA.Velocity.Z *= rule.Restitution

		entityB.Velocity.X *= rule.Restitution
		entityB.Velocity.Y *= rule.Restitution
		entityB.Velocity.Z *= rule.Restitution
	}
}

// getCollisionRule gets collision rule for two entity types
func (pp *PhysicsProcessor) getCollisionRule(typeA, typeB string) *CollisionRule {
	if pp.rules == nil || pp.rules.CollisionRules == nil {
		return nil
	}

	// Try direct lookup
	key := typeA + ":" + typeB
	if rule, exists := pp.rules.CollisionRules[key]; exists {
		return &rule
	}

	// Try reverse lookup
	key = typeB + ":" + typeA
	if rule, exists := pp.rules.CollisionRules[key]; exists {
		return &rule
	}

	return nil
}

// handleProximityEvent handles proximity-based events
func (pp *PhysicsProcessor) handleProximityEvent(event PhysicsEvent) {
	// Get nearby entities
	nearby := pp.GetProximityData(event.Position, 100.0) // 100 unit radius

	// Process proximity events
	for _, nearbyEvent := range nearby {
		pp.processProximityEvent(event, nearbyEvent)
	}
}

// processProximityEvent processes a single proximity event
func (pp *PhysicsProcessor) processProximityEvent(event, nearbyEvent PhysicsEvent) {
	// Calculate distance
	distance := pp.calculateDistance(event.Position, nearbyEvent.Position)

	// Create proximity event
	proximityEvent := ProximityEvent{
		EntityID:  event.EntityID,
		TargetID:  nearbyEvent.EntityID,
		Distance:  distance,
		Position:  event.Position,
		TargetPos: nearbyEvent.Position,
		EventType: string(event.Type),
		Timestamp: time.Now(),
	}

	// Send to frontend
	pp.broadcastProximityEvent(proximityEvent)
}

// calculateDistance calculates distance between two positions
func (pp *PhysicsProcessor) calculateDistance(pos1, pos2 Vector3) float32 {
	dx := pos1.X - pos2.X
	dy := pos1.Y - pos2.Y
	dz := pos1.Z - pos2.Z
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}

// GetProximityData gets entities within a certain radius of a position
func (pp *PhysicsProcessor) GetProximityData(position Vector3, radius float32) []PhysicsEvent {
	pp.entitiesMutex.RLock()
	defer pp.entitiesMutex.RUnlock()

	var events []PhysicsEvent

	// Get nearby entities from proximity grid
	nearbyIDs := pp.proximityGrid.GetNearby(position, radius)

	for _, entityID := range nearbyIDs {
		entity, exists := pp.entities[entityID]
		if !exists {
			continue
		}

		// Check if within radius
		if entity.IsNear(&PhysicsEntity{Position: position}, radius) {
			event := PhysicsEvent{
				Type:       PhysicsProximityUpdate,
				CampaignID: pp.campaignID,
				EntityID:   entityID,
				Position:   entity.Position,
				Rotation:   entity.Rotation,
				Velocity:   entity.Velocity,
				Properties: entity.Properties,
				Timestamp:  time.Now(),
				FrameID:    pp.lastFrameID,
				Source:     "wasm",
			}
			events = append(events, event)
		}
	}

	return events
}

// updatePerformance updates performance metrics
func (pp *PhysicsProcessor) updatePerformance(duration time.Duration) {
	pp.performance.FrameTime = float32(duration.Milliseconds())
	pp.performance.EntityCount = len(pp.entities)
	pp.performance.EventCount++
	pp.performance.LastUpdate = time.Now()

	// Calculate FPS
	if pp.performance.FrameTime > 0 {
		pp.performance.FPS = 1000.0 / pp.performance.FrameTime
	}
}

// broadcastEvent broadcasts an event to the frontend
func (pp *PhysicsProcessor) broadcastEvent(event PhysicsEvent) {
	// This would send the event through the WebSocket connection
	// For now, we'll just log it
	log.Printf("[PhysicsProcessor] Broadcasting event: %s", event.Type)
}

// broadcastProximityEvent broadcasts a proximity event to the frontend
func (pp *PhysicsProcessor) broadcastProximityEvent(event ProximityEvent) {
	// This would send the proximity event through the WebSocket connection
	log.Printf("[PhysicsProcessor] Broadcasting proximity event: %s", event.EventType)
}

// handleEnvironmentUpdate handles environment update events
func (pp *PhysicsProcessor) handleEnvironmentUpdate(event PhysicsEvent) {
	log.Printf("[PhysicsProcessor] Handling environment update: %s", event.Type)
	// TODO: Implement environment update handling
}

// handleCampaignJoin handles campaign join events
func (pp *PhysicsProcessor) handleCampaignJoin(event PhysicsEvent) {
	log.Printf("[PhysicsProcessor] Handling campaign join: %s", event.Type)
	// TODO: Implement campaign join handling
}

// handleCampaignLeave handles campaign leave events
func (pp *PhysicsProcessor) handleCampaignLeave(event PhysicsEvent) {
	log.Printf("[PhysicsProcessor] Handling campaign leave: %s", event.Type)
	// TODO: Implement campaign leave handling
}

// handleCampaignRules handles campaign rules events
func (pp *PhysicsProcessor) handleCampaignRules(event PhysicsEvent) {
	log.Printf("[PhysicsProcessor] Handling campaign rules: %s", event.Type)
	// TODO: Implement campaign rules handling
}

// ProximityGrid methods

// AddEntity adds an entity to the proximity grid
func (pg *ProximityGrid) AddEntity(entity *PhysicsEntity) {
	pg.gridMutex.Lock()
	defer pg.gridMutex.Unlock()

	cellID := pg.getCellID(entity.Position)
	pg.grid[cellID] = append(pg.grid[cellID], entity.ID)
}

// UpdateEntity updates an entity's position in the proximity grid
func (pg *ProximityGrid) UpdateEntity(entity *PhysicsEntity) {
	pg.gridMutex.Lock()
	defer pg.gridMutex.Unlock()

	// Remove from old cell
	oldCellID := pg.getCellID(entity.Position)
	for i, id := range pg.grid[oldCellID] {
		if id == entity.ID {
			pg.grid[oldCellID] = append(pg.grid[oldCellID][:i], pg.grid[oldCellID][i+1:]...)
			break
		}
	}

	// Add to new cell
	newCellID := pg.getCellID(entity.Position)
	pg.grid[newCellID] = append(pg.grid[newCellID], entity.ID)
}

// RemoveEntity removes an entity from the proximity grid
func (pg *ProximityGrid) RemoveEntity(entity *PhysicsEntity) {
	pg.gridMutex.Lock()
	defer pg.gridMutex.Unlock()

	cellID := pg.getCellID(entity.Position)
	for i, id := range pg.grid[cellID] {
		if id == entity.ID {
			pg.grid[cellID] = append(pg.grid[cellID][:i], pg.grid[cellID][i+1:]...)
			break
		}
	}
}

// GetNearby gets entities within a certain radius
func (pg *ProximityGrid) GetNearby(position Vector3, radius float32) []string {
	pg.gridMutex.RLock()
	defer pg.gridMutex.RUnlock()

	var nearby []string

	// Calculate grid cells to check
	cells := pg.getCellsInRadius(position, radius)

	for _, cellID := range cells {
		if entities, exists := pg.grid[cellID]; exists {
			nearby = append(nearby, entities...)
		}
	}

	return nearby
}

// getCellID gets the cell ID for a position
func (pg *ProximityGrid) getCellID(position Vector3) string {
	x := int(position.X / pg.cellSize)
	y := int(position.Y / pg.cellSize)
	z := int(position.Z / pg.cellSize)
	return fmt.Sprintf("%d,%d,%d", x, y, z)
}

// getCellsInRadius gets all cell IDs within a radius
func (pg *ProximityGrid) getCellsInRadius(position Vector3, radius float32) []string {
	var cells []string

	// Calculate grid bounds
	minX := int((position.X - radius) / pg.cellSize)
	maxX := int((position.X + radius) / pg.cellSize)
	minY := int((position.Y - radius) / pg.cellSize)
	maxY := int((position.Y + radius) / pg.cellSize)
	minZ := int((position.Z - radius) / pg.cellSize)
	maxZ := int((position.Z + radius) / pg.cellSize)

	// Add all cells in the radius
	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for z := minZ; z <= maxZ; z++ {
				cells = append(cells, fmt.Sprintf("%d,%d,%d", x, y, z))
			}
		}
	}

	return cells
}
