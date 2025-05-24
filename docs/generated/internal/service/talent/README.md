# Package talent

## Variables

### TalentEventRegistry

## Types

### AccessibilityMetadata

### AuditMetadata

### ComplianceMetadata

### DiversityMetadata

### EventEmitter

EventEmitter defines the interface for emitting events (canonical platform interface).

### EventHandlerFunc

### EventRegistry

### EventSubscription

### Metadata

ServiceMetadata for talent, with diversity, inclusion, and industry-standard fields.

### Profile

### Repository

#### Methods

##### BookTalent

##### CreateTalentProfile

##### DeleteTalentProfile

##### GetBooking

##### GetTalentProfile

##### ListBookings

##### ListTalentProfiles

##### SearchTalentProfiles

##### UpdateTalentProfile

### Service

#### Methods

##### BookTalent

##### CreateTalentProfile

##### DeleteTalentProfile

##### GetTalentProfile

##### ListBookings

##### ListTalentProfiles

##### SearchTalentProfiles

##### UpdateTalentProfile

## Functions

### BuildTalentMetadata

BuildTalentMetadata builds a commonpb.Metadata from ServiceMetadata and tags.

### ExtractAndEnrichTalentMetadata

ExtractAndEnrichTalentMetadata extracts, validates, and enriches talent metadata.

### NewService

### Register

Register registers the talent service with the DI container and event bus support.

### StartEventSubscribers
