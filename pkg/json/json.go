package json

import jsoniter "github.com/json-iterator/go"

var (
	// JSON is the instance of jsoniter.API that should be used throughout the codebase
	JSON = jsoniter.ConfigCompatibleWithStandardLibrary

	// Marshal is a shorthand for JSON.Marshal
	Marshal = JSON.Marshal

	// Unmarshal is a shorthand for JSON.Unmarshal
	Unmarshal = JSON.Unmarshal

	// NewDecoder is a shorthand for JSON.NewDecoder
	NewDecoder = JSON.NewDecoder

	// NewEncoder is a shorthand for JSON.NewEncoder
	NewEncoder = JSON.NewEncoder
)
