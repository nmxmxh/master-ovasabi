package nexus

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
)

// RelationType defines the type of relationship between entities.
type RelationType string

// Common relationship types.
const (
	RelationTypeOwner    RelationType = "owner"
	RelationTypeMember   RelationType = "member"
	RelationTypeLinked   RelationType = "linked"
	RelationTypeParent   RelationType = "parent"
	RelationTypeChild    RelationType = "child"
	RelationTypeReferral RelationType = "referral"
)

// Relationship represents a connection between two master records.
type Relationship struct {
	ID         int64                  `json:"id" db:"id"`
	ParentID   int64                  `json:"parent_id" db:"parent_id"`
	ChildID    int64                  `json:"child_id" db:"child_id"`
	Type       RelationType           `json:"type" db:"type"`
	EntityType repository.EntityType  `json:"entity_type" db:"entity_type"`
	Metadata   map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at" db:"updated_at"`
	IsActive   bool                   `json:"is_active" db:"is_active"`
	Version    int                    `json:"version" db:"version"`
}

// Event represents a cross-service event.
type Event struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	MasterID    int64                  `json:"master_id" db:"master_id"`
	EntityType  repository.EntityType  `json:"entity_type" db:"entity_type"`
	EventType   string                 `json:"event_type" db:"event_type"`
	Payload     map[string]interface{} `json:"payload" db:"payload"`
	Status      string                 `json:"status" db:"status"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	ProcessedAt *time.Time             `json:"processed_at" db:"processed_at"`
}

// Repository defines the interface for Nexus operations.
type Repository interface {
	// Relationship operations
	CreateRelationship(ctx context.Context, parentID, childID int64, relType RelationType, metadata map[string]interface{}) (*Relationship, error)
	GetRelationship(ctx context.Context, id int64) (*Relationship, error)
	UpdateRelationship(ctx context.Context, rel *Relationship) error
	DeleteRelationship(ctx context.Context, id int64) error
	ListRelationships(ctx context.Context, masterID int64, relType RelationType) ([]*Relationship, error)

	// Event operations
	PublishEvent(ctx context.Context, event *Event) error
	ProcessEvent(ctx context.Context, eventID uuid.UUID) error
	GetPendingEvents(ctx context.Context, entityType repository.EntityType) ([]*Event, error)

	// Graph operations
	GetRelatedEntities(ctx context.Context, masterID int64, depth int) ([]*repository.Master, error)
	FindPath(ctx context.Context, fromID, toID int64) ([]*Relationship, error)
	GetEntityGraph(ctx context.Context, masterID int64, maxDepth int) (*Graph, error)
}

// Graph represents a relationship graph.
type Graph struct {
	Nodes []*repository.Master `json:"nodes"`
	Edges []*Relationship      `json:"edges"`
}
