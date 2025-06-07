# Package referral

## Constants

### ReferralMetaFraudSignals

Canonical keys for referral metadata (for onboarding, extensibility, and analytics).

## Variables

### ErrReferralNotFound

### ReferralEventRegistry

## Types

### EventEmitter

EventEmitter defines the interface for emitting events in the referral service.

### EventHandlerFunc

### EventRegistry

### EventSubscription

### Referral

Referral represents a referral record.

### Repository

Repository handles database operations for referrals.

#### Methods

##### Create

Create inserts a new referral record.

##### GetByCode

GetByCode retrieves a referral by referral_code.

##### GetByID

GetByID retrieves a referral by ID.

##### GetStats

GetStats retrieves referral statistics for a user.

##### UpdateReferredMasterID

UpdateReferredMasterID updates the referred_master_id for a referral.

### Service

Service struct implements the ReferralService interface.

#### Methods

##### CreateReferral

CreateReferral creates a new referral code following the Master-Client-Service-Event pattern.

##### GetReferral

GetReferral retrieves a specific referral by code.

##### GetReferralStats

GetReferralStats retrieves referral statistics.

##### UpdateReferredMasterID

UpdateReferredMasterID updates the referred master ID for a referral.

### Stats

Stats represents referral statistics.

## Functions

### NewService

NewService creates a new instance of ReferralService.

### Register

Register registers the Referral service with the DI container and event bus (self-registration
pattern).

### StartEventSubscribers
