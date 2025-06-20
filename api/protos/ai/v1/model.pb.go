// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.29.3
// source: ai/v1/model.proto

package aipb

import (
	v1 "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// ModelUpdate represents a federated learning update with metadata and hash for auditability.
type ModelUpdate struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Data          []byte                 `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"` // Model update data (weights, gradients, etc.)
	Meta          *v1.Metadata           `protobuf:"bytes,2,opt,name=meta,proto3" json:"meta,omitempty"` // Canonical metadata (versioning, peer info, round, etc.)
	Hash          string                 `protobuf:"bytes,3,opt,name=hash,proto3" json:"hash,omitempty"` // Unique, tamper-evident identifier
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ModelUpdate) Reset() {
	*x = ModelUpdate{}
	mi := &file_ai_v1_model_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ModelUpdate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModelUpdate) ProtoMessage() {}

func (x *ModelUpdate) ProtoReflect() protoreflect.Message {
	mi := &file_ai_v1_model_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ModelUpdate.ProtoReflect.Descriptor instead.
func (*ModelUpdate) Descriptor() ([]byte, []int) {
	return file_ai_v1_model_proto_rawDescGZIP(), []int{0}
}

func (x *ModelUpdate) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *ModelUpdate) GetMeta() *v1.Metadata {
	if x != nil {
		return x.Meta
	}
	return nil
}

func (x *ModelUpdate) GetHash() string {
	if x != nil {
		return x.Hash
	}
	return ""
}

// Model represents the current AI model state with metadata and hash for auditability.
type Model struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Data          []byte                 `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`                               // Model weights, parameters, or state
	Meta          *v1.Metadata           `protobuf:"bytes,2,opt,name=meta,proto3" json:"meta,omitempty"`                               // Canonical metadata (versioning, training params, performance, etc.)
	Hash          string                 `protobuf:"bytes,3,opt,name=hash,proto3" json:"hash,omitempty"`                               // Unique, tamper-evident identifier
	Version       string                 `protobuf:"bytes,4,opt,name=version,proto3" json:"version,omitempty"`                         // Model version string
	ParentHash    string                 `protobuf:"bytes,5,opt,name=parent_hash,json=parentHash,proto3" json:"parent_hash,omitempty"` // (Optional) for lineage/ancestry tracking
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Model) Reset() {
	*x = Model{}
	mi := &file_ai_v1_model_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Model) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Model) ProtoMessage() {}

func (x *Model) ProtoReflect() protoreflect.Message {
	mi := &file_ai_v1_model_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Model.ProtoReflect.Descriptor instead.
func (*Model) Descriptor() ([]byte, []int) {
	return file_ai_v1_model_proto_rawDescGZIP(), []int{1}
}

func (x *Model) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *Model) GetMeta() *v1.Metadata {
	if x != nil {
		return x.Meta
	}
	return nil
}

func (x *Model) GetHash() string {
	if x != nil {
		return x.Hash
	}
	return ""
}

func (x *Model) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *Model) GetParentHash() string {
	if x != nil {
		return x.ParentHash
	}
	return ""
}

var File_ai_v1_model_proto protoreflect.FileDescriptor

const file_ai_v1_model_proto_rawDesc = "" +
	"\n" +
	"\x11ai/v1/model.proto\x12\x05ai.v1\x1a\x18common/v1/metadata.proto\"[\n" +
	"\vModelUpdate\x12\x12\n" +
	"\x04data\x18\x01 \x01(\fR\x04data\x12$\n" +
	"\x04meta\x18\x02 \x01(\v2\x10.common.MetadataR\x04meta\x12\x12\n" +
	"\x04hash\x18\x03 \x01(\tR\x04hash\"\x90\x01\n" +
	"\x05Model\x12\x12\n" +
	"\x04data\x18\x01 \x01(\fR\x04data\x12$\n" +
	"\x04meta\x18\x02 \x01(\v2\x10.common.MetadataR\x04meta\x12\x12\n" +
	"\x04hash\x18\x03 \x01(\tR\x04hash\x12\x18\n" +
	"\aversion\x18\x04 \x01(\tR\aversion\x12\x1f\n" +
	"\vparent_hash\x18\x05 \x01(\tR\n" +
	"parentHashB8Z6github.com/nmxmxh/master-ovasabi/api/protos/ai/v1;aipbb\x06proto3"

var (
	file_ai_v1_model_proto_rawDescOnce sync.Once
	file_ai_v1_model_proto_rawDescData []byte
)

func file_ai_v1_model_proto_rawDescGZIP() []byte {
	file_ai_v1_model_proto_rawDescOnce.Do(func() {
		file_ai_v1_model_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_ai_v1_model_proto_rawDesc), len(file_ai_v1_model_proto_rawDesc)))
	})
	return file_ai_v1_model_proto_rawDescData
}

var file_ai_v1_model_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_ai_v1_model_proto_goTypes = []any{
	(*ModelUpdate)(nil), // 0: ai.v1.ModelUpdate
	(*Model)(nil),       // 1: ai.v1.Model
	(*v1.Metadata)(nil), // 2: common.Metadata
}
var file_ai_v1_model_proto_depIdxs = []int32{
	2, // 0: ai.v1.ModelUpdate.meta:type_name -> common.Metadata
	2, // 1: ai.v1.Model.meta:type_name -> common.Metadata
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_ai_v1_model_proto_init() }
func file_ai_v1_model_proto_init() {
	if File_ai_v1_model_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_ai_v1_model_proto_rawDesc), len(file_ai_v1_model_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_ai_v1_model_proto_goTypes,
		DependencyIndexes: file_ai_v1_model_proto_depIdxs,
		MessageInfos:      file_ai_v1_model_proto_msgTypes,
	}.Build()
	File_ai_v1_model_proto = out.File
	file_ai_v1_model_proto_goTypes = nil
	file_ai_v1_model_proto_depIdxs = nil
}
