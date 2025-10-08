package metadata

import (
	"fmt"
	"reflect"

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

	// Try to convert problematic types first
	convertedMap := convertForStructpb(m, log)

	// Ensure we have the correct type for structpb.NewStruct
	var finalMap map[string]interface{}
	if convertedMap != nil {
		if mapVal, ok := convertedMap.(map[string]interface{}); ok {
			finalMap = mapVal
		} else {
			if log != nil {
				log.Error("converted data is not a map[string]interface{}",
					zap.String("type", fmt.Sprintf("%T", convertedMap)),
					zap.Any("value", convertedMap))
			}
			return &structpb.Struct{Fields: map[string]*structpb.Value{}}
		}
	} else {
		finalMap = make(map[string]interface{})
	}

	s, err := structpb.NewStruct(finalMap)
	if err != nil {
		if log != nil {
			log.Error("failed to create structpb.Struct",
				zap.Error(err),
				zap.Any("input_map", m),
				zap.Any("converted_map", finalMap),
				zap.Int("map_size", len(m)))
		}
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	return s
}

// convertForStructpb converts problematic Go types to structpb-compatible types.
func convertForStructpb(data interface{}, log *zap.Logger) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = convertForStructpb(val, log)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = convertForStructpb(val, log)
		}
		return result
	case []string:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = val
		}
		return result
	case []int:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = val
		}
		return result
	case []float64:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = val
		}
		return result
	case []bool:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = val
		}
		return result
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	case string, float64, bool, nil:
		return v
	default:
		// For complex types like slices of maps, try to convert recursively
		// if log != nil {
		// 	log.Info("Converting complex type recursively",
		// 		zap.String("type", fmt.Sprintf("%T", v)),
		// 		zap.Any("value", v))
		// }

		// Try to convert slices of maps to []interface{}
		if reflect.TypeOf(v).Kind() == reflect.Slice {
			sliceValue := reflect.ValueOf(v)
			result := make([]interface{}, sliceValue.Len())
			for i := 0; i < sliceValue.Len(); i++ {
				result[i] = convertForStructpb(sliceValue.Index(i).Interface(), log)
			}
			return result
		}

		// Try to convert maps to map[string]interface{}
		if reflect.TypeOf(v).Kind() == reflect.Map {
			mapValue := reflect.ValueOf(v)
			result := make(map[string]interface{})
			for _, key := range mapValue.MapKeys() {
				keyStr := fmt.Sprintf("%v", key.Interface())
				result[keyStr] = convertForStructpb(mapValue.MapIndex(key).Interface(), log)
			}
			return result
		}

		// Last resort: convert to string
		if log != nil {
			log.Warn("Converting unknown type to string as fallback",
				zap.String("type", fmt.Sprintf("%T", v)),
				zap.Any("value", v))
		}
		return fmt.Sprintf("%v", v)
	}
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
