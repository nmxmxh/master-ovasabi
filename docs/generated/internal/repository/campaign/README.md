# Package campaign

## Variables

### ErrCampaignNotFound

## Types

### Campaign

(move from shared repository types if needed).

### LeaderboardEntry

LeaderboardEntry represents a single entry in the campaign leaderboard.

### RankingColumn

### RankingFormula

Example: "referral_count DESC, username ASC".

#### Methods

##### ToSQL

ToSQL returns the SQL ORDER BY clause for the validated formula.

### Repository

Repository handles database operations for campaigns.

#### Methods

##### CreateWithTransaction

CreateWithTransaction creates a new campaign within a transaction.

##### Delete

Delete deletes a campaign by ID.

##### GetBySlug

GetBySlug retrieves a campaign by its slug.

##### GetLeaderboard

GetLeaderboard returns the leaderboard for a campaign, applying the ranking formula.

##### List

List retrieves a paginated list of campaigns.

##### Update

Update updates an existing campaign.

## Functions

### FlattenMetadataToVars

FlattenMetadataToVars extracts primitive fields from campaign metadata into the variables map.
