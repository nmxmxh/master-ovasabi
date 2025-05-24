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

### PricingMetadata

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

### BuildProductMetadata

BuildProductMetadata builds a canonical product metadata struct for storage, analytics, and
extensibility.

### ExtractAndEnrichProductMetadata

ExtractAndEnrichProductMetadata extracts, validates, and enriches product metadata.

### Register

Register registers the product service with the DI container and event bus support.

### ServiceMetadataToStruct

ServiceMetadataToStruct converts ServiceMetadata to structpb.Struct.

### StartEventSubscribers
