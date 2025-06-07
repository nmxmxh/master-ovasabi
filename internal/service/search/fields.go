package search

import (
	"reflect"
)

// DiscoverSearchableFields returns all fields with `search:"true"` in the struct.
func DiscoverSearchableFields(entity interface{}) []LocalSearchableField {
	var fields []LocalSearchableField
	val := reflect.ValueOf(entity)
	typ := val.Type()
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Tag.Get("search") == "true" {
			fields = append(fields, LocalSearchableField{
				Name: field.Name,
				Type: field.Type.String(),
				Tag:  field.Tag,
			})
		}
	}
	return fields
}
