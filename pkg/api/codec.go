package api

import (
	"github.com/smoxy-io/proto2fixed/pkg/codecs"
	"github.com/smoxy-io/proto2fixed/pkg/codecs/dynamic"
)

// NewCodec creates a new codec from a JSON schema
func NewCodec(schema []byte) (codecs.Codec, error) {
	return dynamic.NewFromJSON(schema)
}
