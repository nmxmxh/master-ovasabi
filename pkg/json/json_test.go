package json

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	Name    string   `json:"name"`
	Age     int      `json:"age"`
	Hobbies []string `json:"hobbies"`
}

func TestMarshalUnmarshal(t *testing.T) {
	original := testStruct{
		Name:    "John Doe",
		Age:     30,
		Hobbies: []string{"reading", "coding"},
	}

	// Test Marshal
	data, err := Marshal(original)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"name":"John Doe"`)
	assert.Contains(t, string(data), `"age":30`)
	assert.Contains(t, string(data), `"hobbies":["reading","coding"]`)

	// Test Unmarshal
	var decoded testStruct
	err = Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)

	// Test invalid JSON
	err = Unmarshal([]byte(`{"invalid`), &decoded)
	assert.Error(t, err)
}

func TestEncoderDecoder(t *testing.T) {
	original := testStruct{
		Name:    "Jane Smith",
		Age:     25,
		Hobbies: []string{"painting", "music"},
	}

	// Test Encoder
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	err := encoder.Encode(original)
	require.NoError(t, err)

	// Test Decoder
	var decoded testStruct
	decoder := NewDecoder(bytes.NewReader(buf.Bytes()))
	err = decoder.Decode(&decoded)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)

	// Test decoding invalid JSON
	invalidDecoder := NewDecoder(bytes.NewReader([]byte(`{"invalid`)))
	err = invalidDecoder.Decode(&decoded)
	assert.Error(t, err)
}

func TestNilHandling(t *testing.T) {
	// Test marshaling nil
	data, err := Marshal(nil)
	require.NoError(t, err)
	assert.Equal(t, "null", string(data))

	// Test unmarshaling null
	var result interface{}
	err = Unmarshal([]byte("null"), &result)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestSpecialCharacters(t *testing.T) {
	type SpecialStruct struct {
		Text string `json:"text"`
	}

	original := SpecialStruct{
		Text: "Hello\n\"World\"\t🌍",
	}

	// Test Marshal with special characters
	data, err := Marshal(original)
	require.NoError(t, err)

	var decoded SpecialStruct
	err = Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}
