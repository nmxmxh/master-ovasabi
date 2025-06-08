package metadata

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// MarshalCanonical marshals a proto.Message using the canonical options for INOS metadata.
func MarshalCanonical(msg proto.Message) ([]byte, error) {
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
		AllowPartial:    true,
	}
	return marshaler.Marshal(msg)
}

// UnmarshalCanonical unmarshals canonical JSON into a proto.Message.
func UnmarshalCanonical(data []byte, msg proto.Message) error {
	return protojson.Unmarshal(data, msg)
}
