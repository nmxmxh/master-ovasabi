# Package product

## Variables

### ProductEventRegistry

## Types

### AuditMetadata

### AvailabilityMetadata

### BadActorMetadata

### CategoryMetadata

### ComplianceMetadata

### DimensionsMetadata

### EventEmitter

### EventHandlerFunc

### EventRegistry

### EventSubscription

### IdentifiersMetadata

### ListProductsFilter

### MediaMetadata

### Model

Model uses Go naming conventions for DB operations.

### PricingMetadata

### Product

### Repository

#### Methods

##### CreateProduct

##### DeleteProduct

##### GetDB

##### GetProduct

##### ListProductVariants

##### ListProducts

##### SearchProducts

##### UpdateInventory

##### UpdateProduct

### RepositoryItf

### ReviewsMetadata

### SearchProductsFilter

### Service

#### Methods

##### CreateProduct

##### DeleteProduct

##### GetProduct

##### ListProductVariants

##### ListProducts

##### SearchProducts

##### UpdateInventory

##### UpdateProduct

### ServiceMetadata

ServiceMetadata holds all product service-specific metadata fields (Amazon-style, extensible).

### ShippingMetadata

### TopReview

### WarrantyMetadata

## Functions

### ExtractAndEnrichProductMetadata

ExtractAndEnrichProductMetadata extracts, validates, and enriches product metadata.

### Register

Register registers the product service with the DI container and event bus support.

### StartEventSubscribers
