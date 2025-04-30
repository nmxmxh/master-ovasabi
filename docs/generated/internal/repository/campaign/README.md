# Package campaign

## Variables

### ErrCampaignNotFound

## Types

### Campaign

Define the Campaign struct here Campaign represents a campaign entity (move from shared repository
types if needed)

### Repository

Repository handles database operations for campaigns

#### Methods

##### CreateWithTransaction

CreateWithTransaction creates a new campaign within a transaction

##### Delete

Delete deletes a campaign by ID

##### GetBySlug

GetBySlug retrieves a campaign by its slug

##### List

List retrieves a paginated list of campaigns

##### Update

Update updates an existing campaign

## Functions

### SetLogger
