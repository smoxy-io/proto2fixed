package codecs

import (
	"testing"

	"github.com/smoxy-io/proto2fixed/pkg/generator"
)

// TestCodecInterface tests that Codec interface is properly defined
func TestCodecInterface(t *testing.T) {
	// Create a test implementation to verify interface compliance
	var _ Codec = (*testCodec)(nil)
}

// testCodec is a minimal implementation for testing interface compliance
type testCodec struct{}

func (c *testCodec) Encode(data []byte) ([]byte, error) {
	return data, nil
}

func (c *testCodec) Decode(data []byte) ([]byte, error) {
	return data, nil
}

func (c *testCodec) Schema() generator.JSONSchema {
	return generator.JSONSchema{
		Version:  "1.0.0",
		Endian:   "little",
		Messages: make(map[string]*generator.JSONMessage),
	}
}
