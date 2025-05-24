# Package handlers

## Types

### NexusOpsHandler

NexusOpsHandler handles /api/nexus/ops requests.

#### Methods

##### ServeHTTP

## Functions

### AdminOpsHandler

AdminOpsHandler: Composable, robust handler for admin operations.

### AnalyticsOpsHandler

AnalyticsOpsHandler is a composable endpoint for all analytics operations.

### CampaignHandler

CampaignHandler returns an http.HandlerFunc for campaign operations (composable endpoint).

### CommerceOpsHandler

CommerceOpsHandler: Composable, robust handler for commerce operations.

### ContentModerationOpsHandler

ContentModerationOpsHandler: composable handler for content moderation operations.

### ContentOpsHandler

ContentOpsHandler handles content-related actions via the "action" field.

@Summary Content Operations @Description Handles content-related actions using the "action" field in
the request body. Each action (e.g., create_content, update_content, etc.) has its own
required/optional fields. All requests must include a 'metadata' field following the robust metadata
pattern (see docs/services/metadata.md). @Tags content @Accept json @Produce json @Param request
body object true "Composable request with 'action', required fields for the action, and 'metadata'
(see docs/services/metadata.md for schema)" @Success 200 {object} object "Response depends on
action" @Failure 400 {object} ErrorResponse @Router /api/content_ops [post].

### LocalizationOpsHandler

LocalizationOpsHandler: Composable, robust handler for localization operations.

### MediaOpsHandler

### MessagingOpsHandler

MessagingOpsHandler: Handles messaging-related actions using the composable API pattern.

### NotificationHandler

NotificationHandler handles notification-related actions (send, list, acknowledge, etc.).

### ProductOpsHandler

### ReferralOpsHandler

ReferralOpsHandler handles referral-related actions via the "action" field.

@Summary Referral Operations @Description Handles referral-related actions using the "action" field
in the request body. Each action (e.g., create_referral, get_referral, etc.) has its own
required/optional fields. All requests must include a 'metadata' field following the robust metadata
pattern (see docs/services/metadata.md). @Tags referral @Accept json @Produce json @Param request
body object true "Composable request with 'action', required fields for the action, and 'metadata'
(see docs/services/metadata.md for schema)" @Success 200 {object} object "Response depends on
action" @Failure 400 {object} ErrorResponse @Router /api/referral_ops [post].

### SchedulerOpsHandler

### SearchOpsHandler

SearchOpsHandler: Composable, robust handler for search operations.

### TalentOpsHandler

TalentOpsHandler: Composable, robust handler for talent operations.

### UserOpsHandler

UserOpsHandler: Robust request parsing and error handling

All request fields must be parsed with type assertions and error checks. For required fields, if the
assertion fails, log and return HTTP 400. For optional fields, only use if present and valid. This
prevents linter/runtime errors and ensures robust, predictable APIs.

Example:

    username, ok := req["username"].(string)
    if !ok { log.Error(...); http.Error(...); return }

This pattern is enforced for all handler files.
