package metadata

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// NewStructFromMap creates a structpb.Struct from a map, optionally merging into an existing struct.
func NewStructFromMap(m map[string]interface{}, log *zap.Logger, existing ...*structpb.Struct) *structpb.Struct {
	if len(existing) > 0 && existing[0] != nil {
		// Merge m into existing struct
		existingMap := existing[0].AsMap()
		for k, v := range m {
			existingMap[k] = v
		}
		s, err := structpb.NewStruct(existingMap)
		if err != nil {
			if log != nil {
				log.Error("failed to merge structpb.Struct", zap.Error(err))
			}
			return &structpb.Struct{Fields: map[string]*structpb.Value{}}
		}
		return s
	}
	if m == nil {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		if log != nil {
			log.Error("failed to create structpb.Struct", zap.Error(err))
		}
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	return s
}

// ToMap safely converts an interface{} to map[string]interface{} if possible, else returns an empty map.
func ToMap(val interface{}) map[string]interface{} {
	if val == nil {
		return map[string]interface{}{}
	}
	m, ok := val.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	return m
}

// MergeStructs merges two structpb.Structs, with fields from b overwriting a.
func MergeStructs(a, b *structpb.Struct, log *zap.Logger) *structpb.Struct {
	if log != nil {
		log.Info("[MergeStructs] Called", zap.Any("a", a), zap.Any("b", b))
	}
	if a == nil && b == nil {
		if log != nil {
			log.Warn("[MergeStructs] Both structs nil, returning empty struct")
		}
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	if a == nil {
		if log != nil {
			log.Info("[MergeStructs] 'a' is nil, returning 'b'", zap.Any("b", b))
		}
		return b
	}
	if b == nil {
		if log != nil {
			log.Info("[MergeStructs] 'b' is nil, returning 'a'", zap.Any("a", a))
		}
		return a
	}
	am := a.AsMap()
	bm := b.AsMap()
	if log != nil {
		log.Info("[MergeStructs] Merging maps", zap.Any("am_before", am), zap.Any("bm", bm))
	}
	for k, v := range bm {
		am[k] = v
	}
	if log != nil {
		log.Info("[MergeStructs] am_after merge", zap.Any("am_after", am))
	}
	s, err := structpb.NewStruct(am)
	if err != nil {
		if log != nil {
			log.Error("failed to merge structpb.Structs", zap.Error(err))
		}
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	if log != nil {
		log.Info("[MergeStructs] Success", zap.Any("result", s))
	}
	return s
}
