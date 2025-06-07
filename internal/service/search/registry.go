package search

import (
	"reflect"
)

type LocalSearchableField struct {
	Name string
	Type string
	Tag  reflect.StructTag
}

// Example entity definitions (replace with real ones as needed).
type Content struct {
	Title  string `search:"true"`
	Body   string
	Author string `search:"true"`
}

type User struct {
	Username string `search:"true"`
	Email    string `search:"true"`
}

// Add all core entities here.
var searchableEntities = []interface{}{
	Content{},
	User{},
	// Add Campaign{}, etc.
}

func BuildSearchFieldRegistry() map[string][]LocalSearchableField {
	result := make(map[string][]LocalSearchableField)
	for _, entity := range searchableEntities {
		typeName := reflect.TypeOf(entity).Name()
		result[typeName] = DiscoverSearchableFields(entity)
	}
	return result
}

var SearchFieldRegistry = BuildSearchFieldRegistry()
