# Package messaging

## Variables

### ErrMessageNotFound

### MessagingEventRegistry

## Types

### AttachmentMetadata

### AuditMetadata

### ChatGroup

ChatGroup represents a group chat.

### ComplianceMetadata

### Conversation

Conversation represents a messaging conversation.

### DeliveryMetadata

### EventEmitter

EventEmitter defines the interface for emitting events (canonical platform interface).

### EventHandlerFunc

### EventRegistry

### EventSubscription

### Message

Message represents a message in the messaging system.

### MessageEvent

MessageEvent represents an event for analytics/audit.

### Metadata

MessagingMetadata is the canonical struct for messaging-specific metadata.

#### Methods

##### ToStruct

ToStruct converts MessagingMetadata to a structpb.Struct for storage in
Metadata.service_specific.messaging.

##### UpdateDeliveryStatus

UpdateDeliveryStatus updates the delivery/read/ack status for a user.

### ReactionMetadata

### Repository

MessagingRepository handles operations on the messaging tables.

#### Methods

##### AcknowledgeMessage

AcknowledgeMessage acknowledges a message for a user.

##### AddChatGroupMember

AddChatGroupMember adds a member to a chat group.

##### CreateChatGroupWithRequest

CreateChatGroup creates a new chat group.

##### CreateMessage

CreateMessage inserts a new message record.

##### DeleteMessage

DeleteMessage removes a message and its master record.

##### DeleteMessageByRequest

DeleteMessageByRequest marks a message as deleted or removes it (RPC-aligned).

##### EditMessage

EditMessage updates the content/attachments of a message.

##### GetChatGroupByID

GetChatGroupByID fetches a chat group by its ID.

##### GetMessage

GetMessage retrieves a message by ID.

##### GetMessageByID

GetMessage retrieves a message by ID (RPC-aligned).

##### GetMessagingPreferences

GetMessagingPreferences fetches preferences for a user.

##### ListChatGroups

ListChatGroups retrieves a paginated list of chat groups.

##### ListConversations

ListConversations retrieves a paginated list of conversations.

##### ListConversationsByUser

ListConversations retrieves conversations for a user or context (RPC-aligned).

##### ListMessageEvents

ListMessageEvents retrieves a paginated list of message events for analytics/audit.

##### ListMessageEventsByUser

ListMessageEventsByUser retrieves a paginated list of message events for a user.

##### ListMessages

ListMessages retrieves a paginated list of messages for a thread or conversation.

##### ListMessagesByFilter

ListMessages retrieves messages for a thread, conversation, or group (RPC-aligned).

##### ListThreads

ListThreads retrieves a paginated list of threads.

##### ListThreadsByUser

ListThreads retrieves threads for a user or context (RPC-aligned).

##### MarkAsDelivered

MarkAsDelivered marks a message as delivered for a user.

##### MarkAsRead

MarkAsRead marks a message as read for a user.

##### ReactToMessage

ReactToMessage adds or updates a reaction on a message.

##### RemoveChatGroupMember

RemoveChatGroupMember removes a member from a chat group.

##### SendGroupMessage

SendGroupMessage creates and persists a new group message.

##### SendMessage

SendMessage creates and persists a new direct or thread message.

##### UpdateMessage

UpdateMessage updates a message record.

##### UpdateMessagingPreferences

UpdateMessagingPreferences upserts preferences for a user.

### Service

Service implements the MessagingService gRPC interface.

#### Methods

##### AcknowledgeMessage

##### AddChatGroupMember

##### CreateChatGroup

##### DeleteMessage

##### EditMessage

##### GetMessage

##### ListChatGroupMembers

##### ListConversations

##### ListMessageEvents

##### ListMessages

##### ListThreads

##### MarkAsDelivered

##### MarkAsRead

##### ReactToMessage

##### RemoveChatGroupMember

##### SendGroupMessage

##### SendMessage

##### StreamMessages

##### StreamPresence

##### StreamTyping

##### UpdateMessagingPreferences

### Thread

Thread represents a messaging thread.

### VersioningMetadata

## Functions

### NewMessagingClient

NewMessagingClient creates a new gRPC client connection and returns a MessagingServiceClient and a
cleanup function.

### NewService

NewService creates a new MessagingService instance with event bus support.

### Register

Register registers the messaging service with the DI container and event bus support.

### StartEventSubscribers

### ValidateMessagingMetadata

ValidateMessagingMetadata validates the structure and required fields.
