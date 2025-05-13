# Package localizationpb

## Constants

### LocalizationService_Translate_FullMethodName

## Variables

### File_localization_v1_localization_proto

### LocalizationService_ServiceDesc

LocalizationService_ServiceDesc is the grpc.ServiceDesc for LocalizationService service. It's only
intended for direct use with grpc.RegisterService, and not to be introspected or modified (even as a
copy)

## Types

### BatchTranslateRequest

#### Methods

##### Descriptor

Deprecated: Use BatchTranslateRequest.ProtoReflect.Descriptor instead.

##### GetKeys

##### GetLocale

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BatchTranslateResponse

#### Methods

##### Descriptor

Deprecated: Use BatchTranslateResponse.ProtoReflect.Descriptor instead.

##### GetValues

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateTranslationRequest

#### Methods

##### Descriptor

Deprecated: Use CreateTranslationRequest.ProtoReflect.Descriptor instead.

##### GetKey

##### GetLanguage

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

### GetLocaleMetadataRequest

#### Methods

##### Descriptor

Deprecated: Use GetLocaleMetadataRequest.ProtoReflect.Descriptor instead.

##### GetLocale

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetLocaleMetadataResponse

#### Methods

##### Descriptor

Deprecated: Use GetLocaleMetadataResponse.ProtoReflect.Descriptor instead.

##### GetLocale

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetPricingRuleRequest

#### Methods

##### Descriptor

Deprecated: Use GetPricingRuleRequest.ProtoReflect.Descriptor instead.

##### GetCity

##### GetCountryCode

##### GetRegion

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetPricingRuleResponse

#### Methods

##### Descriptor

Deprecated: Use GetPricingRuleResponse.ProtoReflect.Descriptor instead.

##### GetRule

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

### ListLocalesRequest

#### Methods

##### Descriptor

Deprecated: Use ListLocalesRequest.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListLocalesResponse

#### Methods

##### Descriptor

Deprecated: Use ListLocalesResponse.ProtoReflect.Descriptor instead.

##### GetLocales

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListPricingRulesRequest

#### Methods

##### Descriptor

Deprecated: Use ListPricingRulesRequest.ProtoReflect.Descriptor instead.

##### GetCountryCode

##### GetPage

##### GetPageSize

##### GetRegion

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListPricingRulesResponse

#### Methods

##### Descriptor

Deprecated: Use ListPricingRulesResponse.ProtoReflect.Descriptor instead.

##### GetPage

##### GetRules

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListTranslationsRequest

#### Methods

##### Descriptor

Deprecated: Use ListTranslationsRequest.ProtoReflect.Descriptor instead.

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

### Locale

#### Methods

##### Descriptor

Deprecated: Use Locale.ProtoReflect.Descriptor instead.

##### GetCode

##### GetCountry

##### GetCurrency

##### GetLanguage

##### GetMetadata

##### GetRegions

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### LocalizationServiceClient

LocalizationServiceClient is the client API for LocalizationService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### LocalizationServiceServer

LocalizationServiceServer is the server API for LocalizationService service. All implementations
must embed UnimplementedLocalizationServiceServer for forward compatibility.

### PricingRule

#### Methods

##### Descriptor

Deprecated: Use PricingRule.ProtoReflect.Descriptor instead.

##### GetAffluenceTier

##### GetBasePrice

##### GetCity

##### GetCountryCode

##### GetCreatedAt

##### GetCurrencyCode

##### GetDemandLevel

##### GetEffectiveFrom

##### GetEffectiveTo

##### GetId

##### GetMultiplier

##### GetNotes

##### GetRegion

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SetPricingRuleRequest

#### Methods

##### Descriptor

Deprecated: Use SetPricingRuleRequest.ProtoReflect.Descriptor instead.

##### GetRule

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SetPricingRuleResponse

#### Methods

##### Descriptor

Deprecated: Use SetPricingRuleResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TranslateRequest

#### Methods

##### Descriptor

Deprecated: Use TranslateRequest.ProtoReflect.Descriptor instead.

##### GetKey

##### GetLocale

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TranslateResponse

#### Methods

##### Descriptor

Deprecated: Use TranslateResponse.ProtoReflect.Descriptor instead.

##### GetValue

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Translation

#### Methods

##### Descriptor

Deprecated: Use Translation.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetId

##### GetKey

##### GetLanguage

##### GetMetadata

##### GetValue

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedLocalizationServiceServer

UnimplementedLocalizationServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### BatchTranslate

##### CreateTranslation

##### GetLocaleMetadata

##### GetPricingRule

##### GetTranslation

##### ListLocales

##### ListPricingRules

##### ListTranslations

##### SetPricingRule

##### Translate

### UnsafeLocalizationServiceServer

UnsafeLocalizationServiceServer may be embedded to opt out of forward compatibility for this
service. Use of this interface is not recommended, as added methods to LocalizationServiceServer
will result in compilation errors.

## Functions

### RegisterLocalizationServiceServer
