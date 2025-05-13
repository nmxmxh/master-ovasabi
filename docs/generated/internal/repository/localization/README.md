# Package localization

## Variables

### ErrTranslationNotFound

## Types

### Locale

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

### Translation

--- Data Models ---.
