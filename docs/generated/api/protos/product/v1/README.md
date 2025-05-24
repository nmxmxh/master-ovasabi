# Package productpb

## Constants

### ProductService_CreateProduct_FullMethodName

## Variables

### ProductType_name

Enum value maps for ProductType.

### ProductStatus_name

Enum value maps for ProductStatus.

### File_product_v1_product_proto

### ProductService_ServiceDesc

ProductService_ServiceDesc is the grpc.ServiceDesc for ProductService service. It's only intended
for direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### CreateProductRequest

#### Methods

##### Descriptor

Deprecated: Use CreateProductRequest.ProtoReflect.Descriptor instead.

##### GetProduct

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateProductResponse

#### Methods

##### Descriptor

Deprecated: Use CreateProductResponse.ProtoReflect.Descriptor instead.

##### GetProduct

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteProductRequest

#### Methods

##### Descriptor

Deprecated: Use DeleteProductRequest.ProtoReflect.Descriptor instead.

##### GetProductId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteProductResponse

#### Methods

##### Descriptor

Deprecated: Use DeleteProductResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetProductRequest

#### Methods

##### Descriptor

Deprecated: Use GetProductRequest.ProtoReflect.Descriptor instead.

##### GetProductId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetProductResponse

#### Methods

##### Descriptor

Deprecated: Use GetProductResponse.ProtoReflect.Descriptor instead.

##### GetProduct

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListProductVariantsRequest

#### Methods

##### Descriptor

Deprecated: Use ListProductVariantsRequest.ProtoReflect.Descriptor instead.

##### GetProductId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListProductVariantsResponse

#### Methods

##### Descriptor

Deprecated: Use ListProductVariantsResponse.ProtoReflect.Descriptor instead.

##### GetVariants

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListProductsRequest

#### Methods

##### Descriptor

Deprecated: Use ListProductsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetOwnerId

##### GetPage

##### GetPageSize

##### GetStatus

##### GetTags

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListProductsResponse

#### Methods

##### Descriptor

Deprecated: Use ListProductsResponse.ProtoReflect.Descriptor instead.

##### GetPage

##### GetProducts

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Product

#### Methods

##### Descriptor

Deprecated: Use Product.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetCreatedAt

##### GetDescription

##### GetGalleryImageUrls

##### GetId

##### GetMainImageUrl

##### GetMasterId

##### GetMasterUuid

##### GetMetadata

##### GetName

##### GetOwnerId

##### GetStatus

##### GetTags

##### GetType

##### GetUpdatedAt

##### GetVariants

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ProductServiceClient

ProductServiceClient is the client API for ProductService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### ProductServiceServer

ProductServiceServer is the server API for ProductService service. All implementations must embed
UnimplementedProductServiceServer for forward compatibility.

### ProductStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use ProductStatus.Descriptor instead.

##### Number

##### String

##### Type

### ProductType

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use ProductType.Descriptor instead.

##### Number

##### String

##### Type

### ProductVariant

#### Methods

##### Descriptor

Deprecated: Use ProductVariant.ProtoReflect.Descriptor instead.

##### GetAttributes

##### GetCompareAtPrice

##### GetCreatedAt

##### GetCurrency

##### GetId

##### GetInventory

##### GetIsDefault

##### GetMetadata

##### GetName

##### GetPaymentType

##### GetPrice

##### GetProductId

##### GetSku

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SearchProductsRequest

#### Methods

##### Descriptor

Deprecated: Use SearchProductsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetPage

##### GetPageSize

##### GetQuery

##### GetStatus

##### GetTags

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SearchProductsResponse

#### Methods

##### Descriptor

Deprecated: Use SearchProductsResponse.ProtoReflect.Descriptor instead.

##### GetPage

##### GetProducts

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedProductServiceServer

UnimplementedProductServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### CreateProduct

##### DeleteProduct

##### GetProduct

##### ListProductVariants

##### ListProducts

##### SearchProducts

##### UpdateInventory

##### UpdateProduct

### UnsafeProductServiceServer

UnsafeProductServiceServer may be embedded to opt out of forward compatibility for this service. Use
of this interface is not recommended, as added methods to ProductServiceServer will result in
compilation errors.

### UpdateInventoryRequest

#### Methods

##### Descriptor

Deprecated: Use UpdateInventoryRequest.ProtoReflect.Descriptor instead.

##### GetDelta

##### GetVariantId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateInventoryResponse

#### Methods

##### Descriptor

Deprecated: Use UpdateInventoryResponse.ProtoReflect.Descriptor instead.

##### GetVariant

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateProductRequest

#### Methods

##### Descriptor

Deprecated: Use UpdateProductRequest.ProtoReflect.Descriptor instead.

##### GetProduct

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateProductResponse

#### Methods

##### Descriptor

Deprecated: Use UpdateProductResponse.ProtoReflect.Descriptor instead.

##### GetProduct

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterProductServiceServer
