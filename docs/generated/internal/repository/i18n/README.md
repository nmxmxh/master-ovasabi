# Package i18n

## Variables

### ErrTranslationNotFound

## Types

### Repository

Repository handles operations on the service_i18n table.

#### Methods

##### Create

Create inserts a new translation record.

##### Delete

Delete removes a translation and its master record.

##### GetByID

GetByID retrieves a translation by ID.

##### GetByKeyAndLocale

GetByKeyAndLocale retrieves a translation by key and locale.

##### List

List retrieves a paginated list of translations.

##### ListByLocale

ListByLocale retrieves all translations for a specific locale.

##### Update

Update updates a translation record.

### Translation

(move from shared repository types if needed).

## Functions

### SetLogger
