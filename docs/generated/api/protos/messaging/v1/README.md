# Package messagingpb

## Constants

### MessagingService_SendMessage_FullMethodName

## Variables

### MessageType_name

Enum value maps for MessageType.

### MessageStatus_name

Enum value maps for MessageStatus.

### File_messaging_v1_messaging_proto

### MessagingService_ServiceDesc

MessagingService_ServiceDesc is the grpc.ServiceDesc for MessagingService service. It's only
intended for direct use with grpc.RegisterService, and not to be introspected or modified (even as a
copy)

## Types

### AcknowledgeMessageRequest

#### Methods

##### Descriptor

Deprecated: Use AcknowledgeMessageRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetMessageId

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AcknowledgeMessageResponse

#### Methods

##### Descriptor

Deprecated: Use AcknowledgeMessageResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AddChatGroupMemberRequest

#### Methods

##### Descriptor

Deprecated: Use AddChatGroupMemberRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetChatGroupId

##### GetMetadata

##### GetRole

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AddChatGroupMemberResponse

#### Methods

##### Descriptor

Deprecated: Use AddChatGroupMemberResponse.ProtoReflect.Descriptor instead.

##### GetChatGroup

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Attachment

#### Methods

##### Descriptor

Deprecated: Use Attachment.ProtoReflect.Descriptor instead.

##### GetFilename

##### GetId

##### GetMetadata

##### GetSize

##### GetType

##### GetUrl

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ChatGroup

#### Methods

##### Descriptor

Deprecated: Use ChatGroup.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetDescription

##### GetId

##### GetMemberIds

##### GetMetadata

##### GetName

##### GetRoles

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Conversation

#### Methods

##### Descriptor

Deprecated: Use Conversation.ProtoReflect.Descriptor instead.

##### GetChatGroupId

##### GetCreatedAt

##### GetId

##### GetMetadata

##### GetParticipantIds

##### GetThreadIds

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateChatGroupRequest

--- Group Management ---

#### Methods

##### Descriptor

Deprecated: Use CreateChatGroupRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetDescription

##### GetMemberIds

##### GetMetadata

##### GetName

##### GetRoles

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateChatGroupResponse

#### Methods

##### Descriptor

Deprecated: Use CreateChatGroupResponse.ProtoReflect.Descriptor instead.

##### GetChatGroup

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteMessageRequest

#### Methods

##### Descriptor

Deprecated: Use DeleteMessageRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetMessageId

##### GetMetadata

##### GetRequesterId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteMessageResponse

#### Methods

##### Descriptor

Deprecated: Use DeleteMessageResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### EditMessageRequest

#### Methods

##### Descriptor

Deprecated: Use EditMessageRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetEditorId

##### GetMessageId

##### GetMetadata

##### GetNewAttachments

##### GetNewContent

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### EditMessageResponse

#### Methods

##### Descriptor

Deprecated: Use EditMessageResponse.ProtoReflect.Descriptor instead.

##### GetMessage

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetMessageRequest

#### Methods

##### Descriptor

Deprecated: Use GetMessageRequest.ProtoReflect.Descriptor instead.

##### GetMessageId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetMessageResponse

#### Methods

##### Descriptor

Deprecated: Use GetMessageResponse.ProtoReflect.Descriptor instead.

##### GetMessage

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListChatGroupMembersRequest

#### Methods

##### Descriptor

Deprecated: Use ListChatGroupMembersRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetChatGroupId

##### GetMetadata

##### GetPage

##### GetPageSize

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListChatGroupMembersResponse

#### Methods

##### Descriptor

Deprecated: Use ListChatGroupMembersResponse.ProtoReflect.Descriptor instead.

##### GetMemberIds

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListConversationsRequest

#### Methods

##### Descriptor

Deprecated: Use ListConversationsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetFilters

##### GetMetadata

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListConversationsResponse

#### Methods

##### Descriptor

Deprecated: Use ListConversationsResponse.ProtoReflect.Descriptor instead.

##### GetConversations

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListMessageEventsRequest

--- Analytics/Events ---

#### Methods

##### Descriptor

Deprecated: Use ListMessageEventsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetFilters

##### GetMetadata

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListMessageEventsResponse

#### Methods

##### Descriptor

Deprecated: Use ListMessageEventsResponse.ProtoReflect.Descriptor instead.

##### GetEvents

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListMessagesRequest

#### Methods

##### Descriptor

Deprecated: Use ListMessagesRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetChatGroupId

##### GetConversationId

##### GetFilters

##### GetMetadata

##### GetPage

##### GetPageSize

##### GetThreadId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListMessagesResponse

#### Methods

##### Descriptor

Deprecated: Use ListMessagesResponse.ProtoReflect.Descriptor instead.

##### GetMessages

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListThreadsRequest

#### Methods

##### Descriptor

Deprecated: Use ListThreadsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetFilters

##### GetMetadata

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListThreadsResponse

#### Methods

##### Descriptor

Deprecated: Use ListThreadsResponse.ProtoReflect.Descriptor instead.

##### GetPage

##### GetThreads

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MarkAsDeliveredRequest

#### Methods

##### Descriptor

Deprecated: Use MarkAsDeliveredRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetMessageId

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MarkAsDeliveredResponse

#### Methods

##### Descriptor

Deprecated: Use MarkAsDeliveredResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MarkAsReadRequest

--- Read/Delivery/Ack ---

#### Methods

##### Descriptor

Deprecated: Use MarkAsReadRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetMessageId

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MarkAsReadResponse

#### Methods

##### Descriptor

Deprecated: Use MarkAsReadResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Message

--- Core Entities ---

#### Methods

##### Descriptor

Deprecated: Use Message.ProtoReflect.Descriptor instead.

##### GetAttachments

##### GetCampaignId

##### GetChatGroupId

##### GetContent

##### GetConversationId

##### GetCreatedAt

##### GetDeleted

##### GetEdited

##### GetId

##### GetMetadata

##### GetReactions

##### GetRecipientIds

##### GetSenderId

##### GetStatus

##### GetThreadId

##### GetType

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MessageEvent

#### Methods

##### Descriptor

Deprecated: Use MessageEvent.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetChatGroupId

##### GetConversationId

##### GetCreatedAt

##### GetEventId

##### GetEventType

##### GetMessageId

##### GetPayload

##### GetThreadId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MessageStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use MessageStatus.Descriptor instead.

##### Number

##### String

##### Type

### MessageType

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use MessageType.Descriptor instead.

##### Number

##### String

##### Type

### MessagingPreferences

--- Preferences ---

#### Methods

##### Descriptor

Deprecated: Use MessagingPreferences.ProtoReflect.Descriptor instead.

##### GetArchive

##### GetCampaignId

##### GetMetadata

##### GetMute

##### GetNotificationTypes

##### GetQuietHours

##### GetTimezone

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MessagingServiceClient

MessagingServiceClient is the client API for MessagingService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

--- Messaging Service: Robust, Extensible, Real-Time ---

### MessagingServiceServer

MessagingServiceServer is the server API for MessagingService service. All implementations must
embed UnimplementedMessagingServiceServer for forward compatibility.

--- Messaging Service: Robust, Extensible, Real-Time ---

### MessagingService_StreamMessagesClient

This type alias is provided for backwards compatibility with existing code that references the prior
non-generic stream type by name.

### MessagingService_StreamMessagesServer

This type alias is provided for backwards compatibility with existing code that references the prior
non-generic stream type by name.

### MessagingService_StreamPresenceClient

This type alias is provided for backwards compatibility with existing code that references the prior
non-generic stream type by name.

### MessagingService_StreamPresenceServer

This type alias is provided for backwards compatibility with existing code that references the prior
non-generic stream type by name.

### MessagingService_StreamTypingClient

This type alias is provided for backwards compatibility with existing code that references the prior
non-generic stream type by name.

### MessagingService_StreamTypingServer

This type alias is provided for backwards compatibility with existing code that references the prior
non-generic stream type by name.

### PresenceEvent

#### Methods

##### Descriptor

Deprecated: Use PresenceEvent.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetStatus

##### GetTimestamp

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ReactToMessageRequest

#### Methods

##### Descriptor

Deprecated: Use ReactToMessageRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetEmoji

##### GetMessageId

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ReactToMessageResponse

#### Methods

##### Descriptor

Deprecated: Use ReactToMessageResponse.ProtoReflect.Descriptor instead.

##### GetMessage

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Reaction

#### Methods

##### Descriptor

Deprecated: Use Reaction.ProtoReflect.Descriptor instead.

##### GetEmoji

##### GetMetadata

##### GetReactedAt

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RemoveChatGroupMemberRequest

#### Methods

##### Descriptor

Deprecated: Use RemoveChatGroupMemberRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetChatGroupId

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RemoveChatGroupMemberResponse

#### Methods

##### Descriptor

Deprecated: Use RemoveChatGroupMemberResponse.ProtoReflect.Descriptor instead.

##### GetChatGroup

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SendGroupMessageRequest

#### Methods

##### Descriptor

Deprecated: Use SendGroupMessageRequest.ProtoReflect.Descriptor instead.

##### GetAttachments

##### GetCampaignId

##### GetChatGroupId

##### GetContent

##### GetMetadata

##### GetSenderId

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SendGroupMessageResponse

#### Methods

##### Descriptor

Deprecated: Use SendGroupMessageResponse.ProtoReflect.Descriptor instead.

##### GetMessage

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SendMessageRequest

--- Requests/Responses ---

#### Methods

##### Descriptor

Deprecated: Use SendMessageRequest.ProtoReflect.Descriptor instead.

##### GetAttachments

##### GetCampaignId

##### GetChatGroupId

##### GetContent

##### GetConversationId

##### GetMetadata

##### GetRecipientIds

##### GetSenderId

##### GetThreadId

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SendMessageResponse

#### Methods

##### Descriptor

Deprecated: Use SendMessageResponse.ProtoReflect.Descriptor instead.

##### GetMessage

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### StreamMessagesRequest

--- Real-Time Streaming ---

#### Methods

##### Descriptor

Deprecated: Use StreamMessagesRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetChatGroupIds

##### GetConversationIds

##### GetFilters

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### StreamPresenceRequest

#### Methods

##### Descriptor

Deprecated: Use StreamPresenceRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### StreamTypingRequest

#### Methods

##### Descriptor

Deprecated: Use StreamTypingRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetChatGroupId

##### GetConversationId

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Thread

#### Methods

##### Descriptor

Deprecated: Use Thread.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetId

##### GetMessageIds

##### GetMetadata

##### GetParticipantIds

##### GetSubject

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TypingEvent

#### Methods

##### Descriptor

Deprecated: Use TypingEvent.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetChatGroupId

##### GetConversationId

##### GetIsTyping

##### GetTimestamp

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedMessagingServiceServer

UnimplementedMessagingServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

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

### UnsafeMessagingServiceServer

UnsafeMessagingServiceServer may be embedded to opt out of forward compatibility for this service.
Use of this interface is not recommended, as added methods to MessagingServiceServer will result in
compilation errors.

### UpdateMessagingPreferencesRequest

#### Methods

##### Descriptor

Deprecated: Use UpdateMessagingPreferencesRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetPreferences

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateMessagingPreferencesResponse

#### Methods

##### Descriptor

Deprecated: Use UpdateMessagingPreferencesResponse.ProtoReflect.Descriptor instead.

##### GetPreferences

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterMessagingServiceServer
