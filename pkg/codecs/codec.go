package codecs

import "github.com/smoxy-io/proto2fixed/pkg/generator"

type Codec interface {
	// Encode accepts a JSON encoded byte array matching the codec's schema and returns the fixed-binary encoded representation. returns an error if the input data is invalid.
	Encode(data []byte) ([]byte, error)
	// Decode accepts a fixed-binary encoded byte array matching the codec's schema and returns the JSON encoded representation. returns an error if the input data is invalid.
	Decode(data []byte) ([]byte, error)
	// Schema returns the JSON schema for the codec
	Schema() generator.JSONSchema
}
