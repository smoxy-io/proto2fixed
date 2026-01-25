package dynamic

import (
	"bytes"
	"encoding/json"
	"github.com/rogpeppe/go-internal/diff"
	"github.com/smoxy-io/proto2fixed/pkg/analyzer"
	"github.com/smoxy-io/proto2fixed/pkg/parser"

	"os"
	"path/filepath"
	"testing"
	
	"github.com/smoxy-io/proto2fixed/pkg/codecs"
	"github.com/smoxy-io/proto2fixed/pkg/generator"
)

// TestNew tests creating a new dynamic codec
func TestNew(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"Test": {
				TotalSize:        4,
				MessageId:        1,
				MessageTotalSize: 8,
				Structure:        []*generator.JSONField{},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	if codec == nil {
		t.Fatal("Codec is nil")
	}

	// Verify it implements the interface
	var _ codecs.Codec = codec
}

// TestNew_InvalidSchema tests error handling for invalid schemas
func TestNew_InvalidSchema(t *testing.T) {
	tests := []struct {
		name   string
		schema generator.JSONSchema
		errMsg string
	}{
		{
			name: "invalid endian",
			schema: generator.JSONSchema{
				Version: "1.0.0",
				Endian:  "middle",
				Messages: map[string]*generator.JSONMessage{
					"Test": {TotalSize: 4},
				},
			},
			errMsg: "invalid endian",
		},
		{
			name: "no messages",
			schema: generator.JSONSchema{
				Version:  "1.0.0",
				Endian:   "little",
				Messages: map[string]*generator.JSONMessage{},
			},
			errMsg: "at least one message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.schema)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

// TestEncode_SimpleMessage tests encoding a simple message
func TestEncode_SimpleMessage(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"Simple": {
				TotalSize:        8,
				MessageId:        1,
				MessageTotalSize: 12,
				Structure: []*generator.JSONField{
					{Name: "id", Type: "uint32", Offset: 0, Size: 4},
					{Name: "value", Type: "uint32", Offset: 4, Size: 4},
				},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"Simple": map[string]any{
			"id":    float64(42),
			"value": float64(100),
		},
	}

	inputJSON, _ := json.Marshal(input)
	output, err := codec.Encode(inputJSON)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(output) != 12 {
		t.Errorf("Expected output size 12, got %d", len(output))
	}

	// Verify message ID (first 4 bytes should be 1 in little endian)
	if output[0] != 1 || output[1] != 0 || output[2] != 0 || output[3] != 0 {
		t.Errorf("Incorrect message ID: %v", output[0:4])
	}
	// id = 42 = 0x2A000000
	if output[4] != 0x2A || output[5] != 0x00 || output[6] != 0x00 || output[7] != 0x00 {
		t.Errorf("Incorrect id encoding: %v", output[4:8])
	}
	// value = 100 = 0x64000000
	if output[8] != 0x64 || output[9] != 0x00 || output[10] != 0x00 || output[11] != 0x00 {
		t.Errorf("Incorrect value encoding: %v", output[8:12])
	}
}

// TestDecode_SimpleMessage tests decoding a simple message
func TestDecode_SimpleMessage(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"Simple": {
				TotalSize:        8,
				MessageId:        1,
				MessageTotalSize: 12,
				Structure: []*generator.JSONField{
					{Name: "id", Type: "uint32", Offset: 0, Size: 4},
					{Name: "value", Type: "uint32", Offset: 4, Size: 4},
				},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	// Binary data: message_id=1, id=42, value=100 (little endian)
	input := []byte{0x01, 0x00, 0x00, 0x00, 0x2A, 0x00, 0x00, 0x00, 0x64, 0x00, 0x00, 0x00}

	output, err := codec.Decode(input)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	simple, ok := result["Simple"].(map[string]any)
	if !ok {
		t.Fatal("Expected Simple message in result")
	}

	if simple["id"].(float64) != 42 {
		t.Errorf("Expected id=42, got %v", simple["id"])
	}
	if simple["value"].(float64) != 100 {
		t.Errorf("Expected value=100, got %v", simple["value"])
	}
}

// TestRoundTrip tests encoding then decoding
func TestRoundTrip(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"Test": {
				TotalSize:        17,
				MessageId:        1,
				MessageTotalSize: 21,
				Structure: []*generator.JSONField{
					{Name: "flag", Type: "bool", Offset: 0, Size: 1},
					{Name: "count", Type: "int32", Offset: 1, Size: 4},
					{Name: "ratio", Type: "double", Offset: 5, Size: 8},
					{Name: "total", Type: "uint32", Offset: 13, Size: 4},
				},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	original := map[string]any{
		"Test": map[string]any{
			"flag":  true,
			"count": float64(-42),
			"ratio": 3.14,
			"total": float64(9999),
		},
	}

	// Encode
	inputJSON, _ := json.Marshal(original)
	binary, err := codec.Encode(inputJSON)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Decode
	outputJSON, err := codec.Decode(binary)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(outputJSON, &result); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	test, ok := result["Test"].(map[string]any)
	if !ok {
		t.Fatal("Expected Test message in result")
	}

	if test["flag"].(bool) != true {
		t.Errorf("flag mismatch: got %v", test["flag"])
	}
	if int32(test["count"].(float64)) != -42 {
		t.Errorf("count mismatch: got %v", test["count"])
	}
	if test["total"].(float64) != 9999 {
		t.Errorf("total mismatch: got %v", test["total"])
	}
}

// TestEncode_BigEndian tests encoding with big endian
func TestEncode_BigEndian(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "big",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"Test": {
				TotalSize:        4,
				MessageId:        1,
				MessageTotalSize: 8,
				Structure: []*generator.JSONField{
					{Name: "value", Type: "uint32", Offset: 0, Size: 4},
				},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"Test": map[string]any{
			"value": float64(0x12345678),
		},
	}

	inputJSON, _ := json.Marshal(input)
	output, err := codec.Encode(inputJSON)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Check message ID (big endian)
	if output[0] != 0x00 || output[1] != 0x00 || output[2] != 0x00 || output[3] != 0x01 {
		t.Errorf("Incorrect message ID in big endian: %v", output[0:4])
	}
	// Big endian value: 0x12345678
	if output[4] != 0x12 || output[5] != 0x34 || output[6] != 0x56 || output[7] != 0x78 {
		t.Errorf("Incorrect big endian encoding: %v", output[4:8])
	}
}

// TestEncode_String tests encoding string fields
func TestEncode_String(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"Test": {
				TotalSize:        16,
				MessageId:        1,
				MessageTotalSize: 20,
				Structure: []*generator.JSONField{
					{Name: "name", Type: "string", Offset: 0, Size: 16},
				},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"Test": map[string]any{
			"name": "hello",
		},
	}

	inputJSON, _ := json.Marshal(input)
	output, err := codec.Encode(inputJSON)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Check message ID (first 4 bytes)
	if output[0] != 1 || output[1] != 0 || output[2] != 0 || output[3] != 0 {
		t.Errorf("Incorrect message ID: %v", output[0:4])
	}

	// Check string content (starting at offset 4)
	expected := "hello"
	for i, c := range expected {
		if output[4+i] != byte(c) {
			t.Errorf("Character %d mismatch: expected %c, got %c", i, c, output[4+i])
		}
	}

	// Check null terminator
	if output[9] != 0 {
		t.Error("Expected null terminator after string")
	}
}

// TestDecode_String tests decoding string fields
func TestDecode_String(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"Test": {
				TotalSize:        16,
				MessageId:        1,
				MessageTotalSize: 20,
				Structure: []*generator.JSONField{
					{Name: "name", Type: "string", Offset: 0, Size: 16},
				},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	// Binary data with message ID + null-terminated string
	input := make([]byte, 20)
	input[0] = 1 // message ID
	copy(input[4:], []byte("hello\x00"))

	output, err := codec.Decode(input)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	test := result["Test"].(map[string]any)
	if test["name"].(string) != "hello" {
		t.Errorf("Expected name='hello', got %v", test["name"])
	}
}

// TestEncode_NestedMessage tests encoding nested messages
func TestEncode_NestedMessage(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"Outer": {
				TotalSize:        8,
				MessageId:        1,
				MessageTotalSize: 12,
				Structure: []*generator.JSONField{
					{Name: "id", Type: "uint32", Offset: 0, Size: 4},
					{
						Name:   "inner",
						Type:   "message",
						Offset: 4,
						Size:   4,
						Structure: []*generator.JSONField{
							{Name: "value", Type: "uint32", Offset: 0, Size: 4},
						},
					},
				},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	input := map[string]any{
		"Outer": map[string]any{
			"id": float64(1),
			"inner": map[string]any{
				"value": float64(42),
			},
		},
	}

	inputJSON, _ := json.Marshal(input)
	output, err := codec.Encode(inputJSON)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(output) != 12 {
		t.Errorf("Expected output size 12, got %d", len(output))
	}

	// Verify message ID
	if output[0] != 1 || output[1] != 0 || output[2] != 0 || output[3] != 0 {
		t.Errorf("Incorrect message ID: %v", output[0:4])
	}

	// Verify nested value (at offset 8 now, due to 4 byte message ID header)
	if output[8] != 0x2A || output[9] != 0x00 || output[10] != 0x00 || output[11] != 0x00 {
		t.Errorf("Incorrect nested value encoding: %v", output[8:12])
	}
}

// TestSchema tests the Schema method
func TestSchema(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"Test": {
				TotalSize:        4,
				MessageId:        1,
				MessageTotalSize: 8,
				Structure:        []*generator.JSONField{},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	returnedSchema := codec.Schema()
	if returnedSchema.Version != schema.Version {
		t.Errorf("Schema version mismatch: expected %s, got %s", schema.Version, returnedSchema.Version)
	}
	if returnedSchema.Endian != schema.Endian {
		t.Errorf("Schema endian mismatch: expected %s, got %s", schema.Endian, returnedSchema.Endian)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestEncode_WithMessageId tests encoding with message ID header
func TestEncode_WithMessageId(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"Command": {
				TotalSize:        8, // 2 uint32 fields
				MessageId:        1,
				MessageTotalSize: 12, // 4 bytes header + 8 bytes body
				Structure: []*generator.JSONField{
					{Name: "id", Offset: 0, Size: 4, Type: "uint32"},
					{Name: "value", Offset: 4, Size: 4, Type: "uint32"},
				},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	input := `{"Command": {"id": 42, "value": 100}}`
	encoded, err := codec.Encode([]byte(input))
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Should be 12 bytes total (4 byte message ID + 8 bytes body)
	if len(encoded) != 12 {
		t.Errorf("Expected encoded size 12, got %d", len(encoded))
	}

	// Check message ID (first 4 bytes should be 1 in little endian)
	if encoded[0] != 1 || encoded[1] != 0 || encoded[2] != 0 || encoded[3] != 0 {
		t.Errorf("Expected message ID 1 in first 4 bytes, got %v", encoded[0:4])
	}

	// Check body (id=42 at offset 4)
	expectedId := uint32(42)
	actualId := uint32(encoded[4]) | uint32(encoded[5])<<8 | uint32(encoded[6])<<16 | uint32(encoded[7])<<24
	if actualId != expectedId {
		t.Errorf("Expected id=%d at offset 4, got %d", expectedId, actualId)
	}
}

// TestDecode_WithMessageId tests decoding with message ID header
func TestDecode_WithMessageId(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"Response": {
				TotalSize:        4, // 1 uint32 field
				MessageId:        2,
				MessageTotalSize: 8, // 4 bytes header + 4 bytes body
				Structure: []*generator.JSONField{
					{Name: "status", Offset: 0, Size: 4, Type: "uint32"},
				},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	// Create binary data: [message_id=2 (4 bytes)] + [status=200 (4 bytes)]
	binary := make([]byte, 8)
	binary[0] = 2   // message ID = 2
	binary[4] = 200 // status = 200

	decoded, err := codec.Decode(binary)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(decoded, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	responseData, ok := result["Response"].(map[string]any)
	if !ok {
		t.Fatal("Expected Response message in result")
	}

	status, ok := responseData["status"].(float64)
	if !ok {
		t.Fatal("Expected status field")
	}

	if status != 200 {
		t.Errorf("Expected status=200, got %v", status)
	}
}

// TestRoundTrip_WithMessageId tests encoding then decoding with message IDs
func TestRoundTrip_WithMessageId(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 2, // 2-byte message IDs
		Messages: map[string]*generator.JSONMessage{
			"Data": {
				TotalSize:        8, // uint32 + float
				MessageId:        5,
				MessageTotalSize: 10, // 2 bytes header + 8 bytes body
				Structure: []*generator.JSONField{
					{Name: "count", Offset: 0, Size: 4, Type: "uint32"},
					{Name: "temperature", Offset: 4, Size: 4, Type: "float"},
				},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	// Encode
	input := `{"Data": {"count": 123, "temperature": 25.5}}`
	encoded, err := codec.Encode([]byte(input))
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(encoded) != 10 {
		t.Errorf("Expected encoded size 10, got %d", len(encoded))
	}

	// Decode
	decoded, err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(decoded, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	dataMsg, ok := result["Data"].(map[string]any)
	if !ok {
		t.Fatal("Expected Data message in result")
	}

	count, ok := dataMsg["count"].(float64)
	if !ok {
		t.Fatal("Expected count field")
	}
	if count != 123 {
		t.Errorf("Expected count=123, got %v", count)
	}

	temp, ok := dataMsg["temperature"].(float64)
	if !ok {
		t.Fatal("Expected temperature field")
	}
	// Allow small floating point difference
	if temp < 25.4 || temp > 25.6 {
		t.Errorf("Expected temperature≈25.5, got %v", temp)
	}
}

// TestDecode_MultipleMessagesWithIds tests decoding different messages by ID
func TestDecode_MultipleMessagesWithIds(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 1,
		Messages: map[string]*generator.JSONMessage{
			"Message1": {
				TotalSize:        4,
				MessageId:        1,
				MessageTotalSize: 5,
				Structure: []*generator.JSONField{
					{Name: "value", Offset: 0, Size: 4, Type: "uint32"},
				},
			},
			"Message2": {
				TotalSize:        4,
				MessageId:        2,
				MessageTotalSize: 5,
				Structure: []*generator.JSONField{
					{Name: "data", Offset: 0, Size: 4, Type: "uint32"},
				},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	// Decode Message1 (ID=1)
	binary1 := []byte{1, 0x0A, 0x00, 0x00, 0x00} // ID=1, value=10
	decoded1, err := codec.Decode(binary1)
	if err != nil {
		t.Fatalf("Decode Message1 failed: %v", err)
	}

	var result1 map[string]any
	if err := json.Unmarshal(decoded1, &result1); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	if _, ok := result1["Message1"]; !ok {
		t.Error("Expected Message1 to be decoded")
	}

	// Decode Message2 (ID=2)
	binary2 := []byte{2, 0x14, 0x00, 0x00, 0x00} // ID=2, data=20
	decoded2, err := codec.Decode(binary2)
	if err != nil {
		t.Fatalf("Decode Message2 failed: %v", err)
	}

	var result2 map[string]any
	if err := json.Unmarshal(decoded2, &result2); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	if _, ok := result2["Message2"]; !ok {
		t.Error("Expected Message2 to be decoded")
	}
}

// TestEncode_UnionWithDiscriminator tests encoding a union message with discriminator
func TestEncode_UnionWithDiscriminator(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"UnionTest": {
				TotalSize:           5, // 1 byte discriminator + 4 byte largest field
				MessageId:           1,
				MessageTotalSize:    9,
				Union:               true,
				HasDiscriminator:    true,
				DiscriminatorOffset: 0,
				DiscriminatorSize:   1,
				Structure: []*generator.JSONField{
					{Name: "intValue", FieldNumber: 1, Type: "uint32", Offset: 1, Size: 4},
					{Name: "floatValue", FieldNumber: 2, Type: "float", Offset: 1, Size: 4},
				},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	// Test encoding with intValue
	input := `{"UnionTest": {"intValue": 42}}`
	result, err := codec.Encode([]byte(input))
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Check discriminator byte (should be 1 for field number 1)
	if result[4] != 1 {
		t.Errorf("Expected discriminator 1, got %d", result[4])
	}

	// Check value
	expected := uint32(42)
	actual := uint32(result[5]) | uint32(result[6])<<8 | uint32(result[7])<<16 | uint32(result[8])<<24
	if actual != expected {
		t.Errorf("Expected value %d, got %d", expected, actual)
	}
}

// TestDecode_UnionWithDiscriminator tests decoding a union message with discriminator
func TestDecode_UnionWithDiscriminator(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"UnionTest": {
				TotalSize:           5,
				MessageId:           1,
				MessageTotalSize:    9,
				Union:               true,
				HasDiscriminator:    true,
				DiscriminatorOffset: 0,
				DiscriminatorSize:   1,
				Structure: []*generator.JSONField{
					{Name: "intValue", FieldNumber: 1, Type: "uint32", Offset: 1, Size: 4},
					{Name: "floatValue", FieldNumber: 2, Type: "float", Offset: 1, Size: 4},
				},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	// Binary: message ID (1) + discriminator (2) + float value (3.14)
	binary := []byte{1, 0, 0, 0, 2, 0xc3, 0xf5, 0x48, 0x40} // ID=1, disc=2, float=3.14
	result, err := codec.Decode(binary)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	unionData, ok := decoded["UnionTest"].(map[string]any)
	if !ok {
		t.Fatal("Expected UnionTest in result")
	}

	// Should only have floatValue (discriminator was 2)
	if _, ok := unionData["floatValue"]; !ok {
		t.Error("Expected floatValue in decoded union")
	}

	if _, ok := unionData["intValue"]; ok {
		t.Error("Should not have intValue in decoded union (only active field)")
	}
}

// TestEncode_OneofWithDiscriminator tests encoding a message with oneof discriminator
func TestEncode_OneofWithDiscriminator(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"OneofTest": {
				TotalSize:        9, // 4 (id) + 5 (1 disc + 4 largest variant)
				MessageId:        1,
				MessageTotalSize: 13,
				Structure: []*generator.JSONField{
					{Name: "id", Type: "uint32", Offset: 0, Size: 4},
				},
				Oneofs: []*generator.JSONOneof{
					{
						Name:                "value",
						Offset:              4,
						Size:                5, // 1 byte discriminator + 4 byte largest variant
						DiscriminatorOffset: 4,
						DiscriminatorSize:   1,
						Variants: []*generator.JSONField{
							{Name: "intValue", FieldNumber: 2, Type: "uint32", Offset: 5, Size: 4},
							{Name: "floatValue", FieldNumber: 3, Type: "float", Offset: 5, Size: 4},
						},
					},
				},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	// Test encoding with intValue
	input := `{"OneofTest": {"id": 100, "value": {"intValue": 42}}}`
	result, err := codec.Encode([]byte(input))
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Check oneof discriminator byte (should be 2 for field number 2)
	if result[8] != 2 {
		t.Errorf("Expected oneof discriminator 2, got %d", result[8])
	}

	// Check value
	expected := uint32(42)
	actual := uint32(result[9]) | uint32(result[10])<<8 | uint32(result[11])<<16 | uint32(result[12])<<24
	if actual != expected {
		t.Errorf("Expected value %d, got %d", expected, actual)
	}
}

// TestDecode_OneofWithDiscriminator tests decoding a message with oneof discriminator
func TestDecode_OneofWithDiscriminator(t *testing.T) {
	schema := generator.JSONSchema{
		Version:       "1.0.0",
		Endian:        "little",
		MessageIdSize: 4,
		Messages: map[string]*generator.JSONMessage{
			"OneofTest": {
				TotalSize:        9,
				MessageId:        1,
				MessageTotalSize: 13,
				Structure: []*generator.JSONField{
					{Name: "id", Type: "uint32", Offset: 0, Size: 4},
				},
				Oneofs: []*generator.JSONOneof{
					{
						Name:                "value",
						Offset:              4,
						Size:                5,
						DiscriminatorOffset: 4,
						DiscriminatorSize:   1,
						Variants: []*generator.JSONField{
							{Name: "intValue", FieldNumber: 2, Type: "uint32", Offset: 5, Size: 4},
							{Name: "floatValue", FieldNumber: 3, Type: "float", Offset: 5, Size: 4},
						},
					},
				},
			},
		},
	}

	codec, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to create codec: %v", err)
	}

	// Binary: message ID (1) + id (100) + discriminator (3) + float value (3.14)
	binary := []byte{
		1, 0, 0, 0, // Message ID
		100, 0, 0, 0, // id = 100
		3,                      // discriminator = 3 (floatValue)
		0xc3, 0xf5, 0x48, 0x40, // float = 3.14
	}
	result, err := codec.Decode(binary)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	msgData, ok := decoded["OneofTest"].(map[string]any)
	if !ok {
		t.Fatal("Expected OneofTest in result")
	}

	oneofData, ok := msgData["value"].(map[string]any)
	if !ok {
		t.Fatal("Expected value oneof in result")
	}

	// Should only have floatValue (discriminator was 3)
	if _, ok := oneofData["floatValue"]; !ok {
		t.Error("Expected floatValue in decoded oneof")
	}

	if _, ok := oneofData["intValue"]; ok {
		t.Error("Should not have intValue in decoded oneof (only active variant)")
	}
}

func TestSimpleProto_GenerateSchema(t *testing.T) {
	wanted, wErr := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "codec", "simple.json"))

	if wErr != nil {
		t.Fatalf("Failed to read wanted schema file: %v", wErr)
	}

	schema, sErr := parser.NewParser(filepath.Join("..", "..", "..")).Parse(filepath.Join("..", "..", "..", "testdata", "codec", "simple.proto"))

	if sErr != nil {
		t.Fatalf("Failed to parse proto file: %v", sErr)
	}

	validator := analyzer.NewValidator()

	result, rErr := validator.Validate(schema)

	if rErr != nil {
		t.Fatalf("Failed to validate schema: %v", rErr)
	}

	if result.HasWarnings() {
		t.Errorf("Warnings: %v", result.Warnings)
	}

	if result.HasErrors() {
		t.Fatalf("Errors: %v", result.Errors)
	}

	code, cErr := generator.NewJSONGenerator().Generate(schema, validator.GetAnalyzer().GetAllLayouts())

	if cErr != nil {
		t.Fatalf("Failed to generate JSON schema: %v", cErr)
	}

	if code != string(wanted) {
		t.Errorf("Generated schema does not match wanted schema:\n%s", diff.Diff("wanted", wanted, "got", []byte(code)))
	}
}

func TestSimpleProto(t *testing.T) {
	jsonMsg := `{"Simple":{"temperature":25.5,"value":123}}`
	binaryMsg := []byte{1, 123, 0, 0, 0, 0, 0, 204, 65}

	schemaJson, sjErr := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "codec", "simple.json"))

	if sjErr != nil {
		t.Fatalf("Failed to read json schema file: %v", sjErr)
	}

	c, cErr := NewFromJSON(schemaJson)

	if cErr != nil {
		t.Fatalf("Failed to create codec: %v", cErr)
	}

	b, bErr := c.Encode([]byte(jsonMsg))

	if bErr != nil {
		t.Fatalf("Failed to encode message: %v", bErr)
	}

	if !bytes.Equal(b, binaryMsg) {
		t.Errorf("Encoded message does not match. expected: %s, got: %s", binaryMsg, b)
	}

	decoded, dErr := c.Decode(b)

	if dErr != nil {
		t.Fatalf("Failed to decode message: %v", dErr)
	}

	if string(decoded) != jsonMsg {
		t.Errorf("Decoded message does not match. expected: %s, got: %s", jsonMsg, decoded)
	}
}
