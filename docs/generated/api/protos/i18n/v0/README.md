# Package i18n

## Constants

### I18NService_CreateTranslation_FullMethodName

## Variables

### File_api_protos_i18n_v0_i18n_proto

### I18NService_ServiceDesc

I18NService_ServiceDesc is the grpc.ServiceDesc for I18NService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### CreateTranslationRequest

#### Methods

##### Descriptor

Deprecated: Use CreateTranslationRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetKey

##### GetLanguage

##### GetMasterId

##### GetMetadata

##### GetValue

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateTranslationResponse

#### Methods

##### Descriptor

Deprecated: Use CreateTranslationResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### GetTranslation

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetTranslationRequest

#### Methods

##### Descriptor

Deprecated: Use GetTranslationRequest.ProtoReflect.Descriptor instead.

##### GetTranslationId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetTranslationResponse

#### Methods

##### Descriptor

Deprecated: Use GetTranslationResponse.ProtoReflect.Descriptor instead.

##### GetTranslation

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### I18NServiceClient

I18NServiceClient is the client API for I18NService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### I18NServiceServer

I18NServiceServer is the server API for I18NService service. All implementations must embed
UnimplementedI18NServiceServer for forward compatibility

### ListTranslationsRequest

#### Methods

##### Descriptor

Deprecated: Use ListTranslationsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetLanguage

##### GetPage

##### GetPageSize

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListTranslationsResponse

#### Methods

##### Descriptor

Deprecated: Use ListTranslationsResponse.ProtoReflect.Descriptor instead.

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### GetTranslations

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TranslateSiteRequest

#### Methods

##### Descriptor

Deprecated: Use TranslateSiteRequest.ProtoReflect.Descriptor instead.

##### GetSourceLang

##### GetTargetLang

##### GetTexts

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TranslateSiteResponse

#### Methods

##### Descriptor

Deprecated: Use TranslateSiteResponse.ProtoReflect.Descriptor instead.

##### GetTranslations

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Translation

#### Methods

##### Descriptor

Deprecated: Use Translation.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetCreatedAt

##### GetId

##### GetKey

##### GetLanguage

##### GetMasterId

##### GetMetadata

##### GetValue

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedI18NServiceServer

UnimplementedI18NServiceServer must be embedded to have forward compatible implementations.

#### Methods

##### CreateTranslation

##### GetTranslation

##### ListTranslations

##### TranslateSite

### UnsafeI18NServiceServer

UnsafeI18NServiceServer may be embedded to opt out of forward compatibility for this service. Use of
this interface is not recommended, as added methods to I18NServiceServer will result in compilation
errors.

## Functions

### RegisterI18NServiceServer
