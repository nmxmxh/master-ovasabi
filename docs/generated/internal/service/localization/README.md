# Package localization

## Variables

### ErrTranslationNotFound

### LocalizationEventRegistry

## Types

### AccessibilityMetadata

### AuditMetadata

### ComplianceIssue

### ComplianceMetadata

### ComplianceStandard

### EventEmitter

EventEmitter defines the interface for emitting events in the localization service.

### EventHandlerFunc

### EventRegistry

### EventSubscription

### Locale

### Localization

### PricingRule

### Repository

Repository handles translations, pricing rules, and locale metadata for the unified
LocalizationService.

#### Methods

##### BatchTranslate

BatchTranslate returns translations for multiple keys in a given locale.

##### CreateLocale

--- Locale CRUD ---.

##### CreatePricingRule

--- PricingRule CRUD ---.

##### CreateTranslation

CreateTranslation creates a new translation entry.

##### DeleteLocale

##### DeletePricingRule

##### DeleteTranslation

##### GetLocaleMetadata

GetLocaleMetadata returns metadata for a locale.

##### GetPricingRule

GetPricingRule retrieves a pricing rule for a location.

##### GetTranslation

GetTranslation retrieves a translation by ID.

##### ListLocales

ListLocales returns all supported locales.

##### ListPricingRules

ListPricingRules lists pricing rules for a country/region with pagination.

##### ListTranslations

ListTranslations lists translations for a language with pagination.

##### SetPricingRule

SetPricingRule creates or updates a pricing rule.

##### Translate

Translate returns a translation for a given key and locale.

##### UpdateLocale

##### UpdatePricingRule

##### UpdateTranslation

--- Translation CRUD ---.

### Service

#### Methods

##### BatchTranslate

BatchTranslate returns translations for multiple keys in a given locale.

##### CreateTranslation

CreateTranslation creates a new translation entry.

##### GetLocaleMetadata

GetLocaleMetadata returns metadata for a locale.

##### GetPricingRule

GetPricingRule retrieves a pricing rule for a location.

##### GetTranslation

GetTranslation retrieves a translation by ID.

##### ListLocales

ListLocales returns all supported locales.

##### ListPricingRules

ListPricingRules lists pricing rules for a country/region with pagination.

##### ListTranslations

ListTranslations lists translations for a language with pagination.

##### SetPricingRule

SetPricingRule creates or updates a pricing rule.

##### Translate

Translate returns a translation for a given key and locale.

### ServiceMetadata

ServiceMetadata holds all localization service-specific metadata fields. This struct documents all
fields expected under metadata.service_specific["localization"] in the common.Metadata proto.
Reference: docs/services/metadata.md, docs/amadeus/amadeus_context.md All extraction and mutation
must use canonical helpers from pkg/metadata.

### Translation

--- Data Models ---.

### TranslationProvenanceMetadata

TranslationProvenanceMetadata describes how a translation was produced.

### VersioningMetadata

## Functions

### NewService

### Register

Register registers the localization service with the DI container and event bus support.

### StartEventSubscribers
