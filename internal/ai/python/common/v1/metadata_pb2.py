# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# NO CHECKED-IN PROTOBUF GENCODE
# source: common/v1/metadata.proto
# Protobuf Python Version: 6.31.0
"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(
    _runtime_version.Domain.PUBLIC,
    6,
    31,
    0,
    '',
    'common/v1/metadata.proto'
)
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()


from google.protobuf import struct_pb2 as google_dot_protobuf_dot_struct__pb2
from google.protobuf import timestamp_pb2 as google_dot_protobuf_dot_timestamp__pb2


DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x18\x63ommon/v1/metadata.proto\x12\x06\x63ommon\x1a\x1cgoogle/protobuf/struct.proto\x1a\x1fgoogle/protobuf/timestamp.proto\"K\n\tTieredTax\x12\x14\n\x0cmin_projects\x18\x01 \x01(\x05\x12\x14\n\x0cmax_projects\x18\x02 \x01(\x05\x12\x12\n\npercentage\x18\x03 \x01(\x01\"\xe4\x01\n\x11TaxationConnector\x12\x0c\n\x04type\x18\x01 \x01(\t\x12\x11\n\trecipient\x18\x02 \x01(\t\x12\x18\n\x10recipient_wallet\x18\x03 \x01(\t\x12\x12\n\npercentage\x18\x04 \x01(\x01\x12\"\n\x07tiereds\x18\x05 \x03(\x0b\x32\x11.common.TieredTax\x12\x12\n\napplied_on\x18\x06 \x01(\t\x12\x0e\n\x06\x64omain\x18\x07 \x01(\t\x12\x0f\n\x07\x64\x65\x66\x61ult\x18\x08 \x01(\x08\x12\x10\n\x08\x65nforced\x18\t \x01(\x08\x12\x15\n\rjustification\x18\n \x01(\t\"c\n\x08Taxation\x12-\n\nconnectors\x18\x01 \x03(\x0b\x32\x19.common.TaxationConnector\x12\x15\n\rproject_count\x18\x02 \x01(\x05\x12\x11\n\ttotal_tax\x18\x03 \x01(\x01\"8\n\rOwnerMetadata\x12\n\n\x02id\x18\x01 \x01(\t\x12\x0e\n\x06wallet\x18\x02 \x01(\t\x12\x0b\n\x03uri\x18\x03 \x01(\t\";\n\x10ReferralMetadata\x12\n\n\x02id\x18\x01 \x01(\t\x12\x0e\n\x06wallet\x18\x02 \x01(\t\x12\x0b\n\x03uri\x18\x03 \x01(\t\"]\n\x0eKnowledgeGraph\x12\n\n\x02id\x18\x01 \x01(\t\x12\x0c\n\x04name\x18\x02 \x01(\t\x12\r\n\x05nodes\x18\x03 \x03(\t\x12\r\n\x05\x65\x64ges\x18\x04 \x03(\t\x12\x13\n\x0b\x64\x65scription\x18\x05 \x01(\t\"\x94\x06\n\x08Metadata\x12+\n\nscheduling\x18\x01 \x01(\x0b\x32\x17.google.protobuf.Struct\x12\x10\n\x08\x66\x65\x61tures\x18\x02 \x03(\t\x12-\n\x0c\x63ustom_rules\x18\x03 \x01(\x0b\x32\x17.google.protobuf.Struct\x12&\n\x05\x61udit\x18\x04 \x01(\x0b\x32\x17.google.protobuf.Struct\x12\x0c\n\x04tags\x18\x05 \x03(\t\x12\x31\n\x10service_specific\x18\x06 \x01(\x0b\x32\x17.google.protobuf.Struct\x12/\n\x0fknowledge_graph\x18\x07 \x01(\x0b\x32\x16.common.KnowledgeGraph\x12#\n\x08taxation\x18\x08 \x01(\x0b\x32\x11.common.TieredTax\x12$\n\x05owner\x18\t \x01(\x0b\x32\x15.common.OwnerMetadata\x12*\n\x08referral\x18\n \x01(\x0b\x32\x18.common.ReferralMetadata\x12+\n\nversioning\x18\x0b \x01(\x0b\x32\x17.google.protobuf.Struct\x12\x15\n\rai_confidence\x18\x0e \x01(\x02\x12\x14\n\x0c\x65mbedding_id\x18\x0f \x01(\t\x12\x12\n\ncategories\x18\x10 \x03(\t\x12\x31\n\rlast_accessed\x18\x11 \x01(\x0b\x32\x1a.google.protobuf.Timestamp\x12\x15\n\rnexus_channel\x18\x12 \x01(\t\x12\x12\n\nsource_uri\x18\x13 \x01(\t\x12\x33\n\tscheduler\x18\x14 \x01(\x0b\x32 .common.Metadata.SchedulerConfig\x1a\x87\x01\n\x0fSchedulerConfig\x12\x14\n\x0cis_ephemeral\x18\x01 \x01(\x08\x12*\n\x06\x65xpiry\x18\x02 \x01(\x0b\x32\x1a.google.protobuf.Timestamp\x12\x18\n\x10job_dependencies\x18\x03 \x03(\t\x12\x18\n\x10retention_policy\x18\x04 \x01(\tB@Z>github.com/nmxmxh/master-ovasabi/api/protos/common/v1;commonpbb\x06proto3')

_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'common.v1.metadata_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
  _globals['DESCRIPTOR']._loaded_options = None
  _globals['DESCRIPTOR']._serialized_options = b'Z>github.com/nmxmxh/master-ovasabi/api/protos/common/v1;commonpb'
  _globals['_TIEREDTAX']._serialized_start=99
  _globals['_TIEREDTAX']._serialized_end=174
  _globals['_TAXATIONCONNECTOR']._serialized_start=177
  _globals['_TAXATIONCONNECTOR']._serialized_end=405
  _globals['_TAXATION']._serialized_start=407
  _globals['_TAXATION']._serialized_end=506
  _globals['_OWNERMETADATA']._serialized_start=508
  _globals['_OWNERMETADATA']._serialized_end=564
  _globals['_REFERRALMETADATA']._serialized_start=566
  _globals['_REFERRALMETADATA']._serialized_end=625
  _globals['_KNOWLEDGEGRAPH']._serialized_start=627
  _globals['_KNOWLEDGEGRAPH']._serialized_end=720
  _globals['_METADATA']._serialized_start=723
  _globals['_METADATA']._serialized_end=1511
  _globals['_METADATA_SCHEDULERCONFIG']._serialized_start=1376
  _globals['_METADATA_SCHEDULERCONFIG']._serialized_end=1511
# @@protoc_insertion_point(module_scope)
