# Package talent

## Variables

### TalentEventRegistry

## Types

### AccessibilityMetadata

### AuditMetadata

### Badge

### CampaignParticipation

### ComplianceMetadata

### DiversityMetadata

### EventEmitter

EventEmitter defines the interface for emitting events (canonical platform interface).

### EventHandlerFunc

### EventRegistry

### EventSubscription

### GamifiedMetadata

### Guild

### Metadata

ServiceMetadata for talent, with diversity, inclusion, and industry-standard fields.

### Party

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

### ExtractTalentLevel

ExtractTalentLevel extracts level from metadata.

### ExtractTalentRoles

ExtractTalentRoles extracts roles from metadata.

### ExtractTalentTeamworkScore

ExtractTalentTeamworkScore extracts teamwork score from metadata.

### NewService

### Register

Register registers the talent service with the DI container and event bus support.

### StartEventSubscribers
