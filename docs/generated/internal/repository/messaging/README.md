# Package repository

## Variables

### ErrMessageNotFound

## Types

### ChatGroup

ChatGroup represents a group chat.

### Conversation

Conversation represents a messaging conversation.

### Message

Message represents a message in the messaging system.

### MessageEvent

MessageEvent represents an event for analytics/audit.

### MessagingRepository

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

### Thread

Thread represents a messaging thread.
